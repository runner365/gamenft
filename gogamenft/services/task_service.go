package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

// TaskService manages the async task queue for on-chain operations.
type TaskService struct {
	db         *db.DBHandle
	ethService *EthService
	wsHub      *WsHub
	taskChan   chan *models.Task
	quit       chan struct{}
}

// NewTaskService creates a TaskService, loads pending tasks from DB,
// and starts the worker goroutine.
func NewTaskService(dbh *db.DBHandle, ethSvc *EthService, hub *WsHub) *TaskService {
	ts := &TaskService{
		db:         dbh,
		ethService: ethSvc,
		wsHub:      hub,
		taskChan:   make(chan *models.Task, 100),
		quit:       make(chan struct{}),
	}

	// Recover unfinished tasks from previous run
	pending, err := dbh.GetPendingTasks()
	if err != nil {
		logger.LogErrorf("TaskService: failed to load pending tasks: %v", err)
	} else {
		logger.LogInfof("TaskService: recovering %d pending tasks", len(pending))
		for _, t := range pending {
			ts.taskChan <- t
		}
	}

	go ts.worker()
	logger.LogInfof("TaskService: worker started")
	return ts
}

// Shutdown stops the worker gracefully.
func (ts *TaskService) Shutdown() {
	close(ts.quit)
}

// EnqueueMint creates a mint_reward task and submits it to the queue.
// Returns error if the user already has a pending task for the same item type.
func (ts *TaskService) EnqueueMint(userAddress, itemType string, tokenID int64) (*models.Task, error) {
	has, err := ts.db.HasPendingTask(userAddress, itemType)
	if err != nil {
		logger.LogErrorf("TaskService: failed to check pending task: %v", err)
		return nil, fmt.Errorf("check pending task: %w", err)
	}
	if has {
		logger.LogErrorf("TaskService: a mint for %s is already in progress", itemType)
		return nil, fmt.Errorf("a mint for %s is already in progress", itemType)
	}

	task := &models.Task{
		TaskID:      uuid.New().String(),
		TaskType:    "mint_reward",
		UserAddress: userAddress,
		ItemType:    itemType,
		TokenID:     tokenID,
		Amount:      1,
		Status:      "pending",
		MaxRetries:  3,
	}

	if err := ts.db.CreateTask(task); err != nil {
		logger.LogErrorf("TaskService: failed to create task: %v", err)
		return nil, fmt.Errorf("create task: %w", err)
	}

	logger.LogInfof("TaskService: enqueued task_id=%s user=%s item=%s", task.TaskID, userAddress, itemType)
	ts.taskChan <- task
	return task, nil
}

// GetTask returns task by ID.
func (ts *TaskService) GetTask(taskID string) (*models.Task, error) {
	return ts.db.GetTaskByTaskID(taskID)
}

// ConfirmTask is called by the event listener when a TransferSingle mint event
// is detected on-chain. It updates the task status and pushes a WS notification.
func (ts *TaskService) ConfirmTask(txHash string) {
	task, err := ts.db.GetTaskByTxHash(txHash)
	if err != nil {
		// Not every mint has a task — direct contract interactions won't.
		logger.LogErrorf("TaskService: no task found for tx %s: %v", txHash, err)
		return
	}
	if task.Status == "confirmed" || task.Status == "failed" {
		logger.LogErrorf("TaskService: task %s already confirmed or failed, status:%s", task.TaskID, task.Status)
		return
	}

	_ = ts.db.UpdateTaskStatus(task.TaskID, "confirmed", "", "", task.RetryCount)
	logger.LogInfof("TaskService: task confirmed task_id=%s user=%s item=%s tx=%s",
		task.TaskID, task.UserAddress, task.ItemType, txHash)

	if ts.wsHub != nil {
		ts.wsHub.Publish(task.UserAddress, WsMessage{
			Type:     "reward_confirmed",
			TaskID:   task.TaskID,
			ItemType: task.ItemType,
			Quantity: int(task.Amount),
			TxHash:   txHash,
		})
	}
}

// worker is the background goroutine that processes tasks from the channel.
func (ts *TaskService) worker() {
	for {
		select {
		case <-ts.quit:
			logger.LogInfof("TaskService: worker stopped")
			return
		case task := <-ts.taskChan:
			ts.processTask(task)
		}
	}
}

func (ts *TaskService) processTask(task *models.Task) {
	logger.LogInfof("TaskService: processing task_id=%s user=%s item=%s retry=%d",
		task.TaskID, task.UserAddress, task.ItemType, task.RetryCount)

	// Mark as processing
	if err := ts.db.UpdateTaskStatus(task.TaskID, "processing", "", "", task.RetryCount); err != nil {
		logger.LogErrorf("TaskService: update to processing failed task_id=%s err=%v", task.TaskID, err)
		return
	}

	txHash, err := ts.ethService.MintGameItem(
		context.Background(),
		common.HexToAddress(task.UserAddress),
		task.TokenID,
		task.Amount,
	)
	if err != nil {
		logger.LogErrorf("TaskService: mint failed task_id=%s err=%v", task.TaskID, err)
		newRetry := task.RetryCount + 1
		if newRetry <= task.MaxRetries {
			_ = ts.db.UpdateTaskStatus(task.TaskID, "pending", "", err.Error(), newRetry)
			task.RetryCount = newRetry
			// Exponential backoff: 2s, 4s, 8s
			delay := time.Duration(1<<newRetry) * time.Second
			logger.LogInfof("TaskService: retry %d/%d task_id=%s delay=%v",
				newRetry, task.MaxRetries, task.TaskID, delay)
			time.AfterFunc(delay, func() {
				ts.taskChan <- task
			})
		} else {
			_ = ts.db.UpdateTaskStatus(task.TaskID, "failed", "", err.Error(), newRetry)
			logger.LogErrorf("TaskService: task exhausted retries task_id=%s", task.TaskID)
			if ts.wsHub != nil {
				ts.wsHub.Publish(task.UserAddress, WsMessage{
					Type:     "reward_failed",
					TaskID:   task.TaskID,
					ItemType: task.ItemType,
					Error:    err.Error(),
				})
			}
		}
		return
	}

	// tx sent successfully, wait for event listener to confirm
	_ = ts.db.UpdateTaskStatus(task.TaskID, "tx_sent", txHash, "", task.RetryCount)
	logger.LogInfof("TaskService: tx sent task_id=%s tx=%s", task.TaskID, txHash)
}
