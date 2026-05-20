// db/db_handle.go
package db

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	logger "github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

// DBHandle 数据库操作句柄
type DBHandle struct {
	conn *gorm.DB
}

// NewDBHandle 创建数据库操作句柄
func NewDBHandle(conn *gorm.DB) *DBHandle {
	return &DBHandle{conn: conn}
}

// ========== User 相关操作 ==========

// CreateUser 创建用户
func (h *DBHandle) CreateUser(user *models.User) error {
	jsonData, _ := json.Marshal(user)
	logger.LogInfof("Creating user: %s", string(jsonData))
	return h.conn.Create(user).Error
}

// GetUserByAddress 根据地址获取用户
func (h *DBHandle) GetUserByAddress(address string) (*models.User, error) {
	var user models.User
	result := h.conn.Where("eth_address = ?", strings.ToLower(address)).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logger.LogInfof("GetUserByAddress: record not found for address %s", address)
		} else {
			logger.LogErrorf("GetUserByAddress error: %v", result.Error)
		}
		return nil, result.Error
	}

	jsonData, _ := json.Marshal(user)
	logger.LogInfof("GetUserByAddress result: %s", string(jsonData))
	return &user, nil
}

// GetUserWithListedItems 获取用户及其在售物品
func (h *DBHandle) GetUserWithListedItems(address string) (*models.User, error) {
	var user models.User
	result := h.conn.Preload("Items", "is_listed = ?", true).
		Where("eth_address = ?", strings.ToLower(address)).
		First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func (h *DBHandle) UpdateUser(user *models.User) error {
	return h.conn.Save(user).Error
}

// UpdateUserNonce 更新用户nonce
func (h *DBHandle) UpdateUserNonce(address string, nonce string) error {
	return h.conn.Model(&models.User{}).
		Where("eth_address = ?", strings.ToLower(address)).
		Update("nonce", nonce).Error
}

// UpdateUserLastLogin 更新用户最后登录时间
func (h *DBHandle) UpdateUserLastLogin(address string) error {
	return h.conn.Model(&models.User{}).
		Where("eth_address = ?", strings.ToLower(address)).
		Update("last_login", time.Now()).Error
}

// UpdateUserProfile 更新用户资料
func (h *DBHandle) UpdateUserProfile(user *models.User, updates map[string]interface{}) error {
	return h.conn.Model(user).Updates(updates).Error
}

// CheckUsernameExists 检查用户名是否已存在
func (h *DBHandle) CheckUsernameExists(username string, excludeID uint) (bool, error) {
	var count int64
	result := h.conn.Model(&models.User{}).
		Where("username = ? AND id != ?", username, excludeID).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// ========== Item 相关操作 ==========

// CreateOrUpdateItem 创建或更新物品
// Upserts on (token_id, nft_contract_address) to allow re-listing the same token.
func (h *DBHandle) CreateOrUpdateItem(item *models.Item) error {
	jsonData, _ := json.Marshal(item)
	logger.LogInfof("Creating or updating item: %s", string(jsonData))
	return h.conn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "token_id"}, {Name: "nft_contract_address"}},
		DoUpdates: clause.AssignmentColumns([]string{"owner_address", "name", "description", "image_url", "token_uri", "amount", "is_listed", "list_price", "listed_at", "updated_at"}),
	}).Create(item).Error
}

// GetListedItems 获取在售物品列表
func (h *DBHandle) GetListedItems(page, size int) ([]*models.Item, int64, error) {
	var items []*models.Item
	var total int64

	offset := (page - 1) * size
	h.conn.Model(&models.Item{}).Where("is_listed = ?", true).Count(&total)
	result := h.conn.Where("is_listed = ?", true).
		Order("listed_at DESC").
		Offset(offset).Limit(size).
		Find(&items)

	if result.Error != nil {
		return nil, 0, result.Error
	}
	return items, total, nil
}

// GetUserItemsCount 获取用户物品数量
func (h *DBHandle) GetUserItemsCount(address string) (int64, error) {
	var count int64
	result := h.conn.Model(&models.Item{}).
		Where("owner_address = ?", address).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// GetUserListedItemsCount 获取用户在售物品数量
func (h *DBHandle) GetUserListedItemsCount(address string) (int64, error) {
	var count int64
	result := h.conn.Model(&models.Item{}).
		Where("owner_address = ? AND is_listed = ?", address, true).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// UpdateItemStatus 更新物品状态
func (h *DBHandle) UpdateItemStatus(itemID uint, isListed bool) error {
	return h.conn.Model(&models.Item{}).
		Where("id = ?", itemID).
		Update("is_listed", isListed).Error
}

// ========== Order 相关操作 ==========

// CreateOrder 创建订单
func (h *DBHandle) CreateOrder(order *models.Order) error {
	return h.conn.Create(order).Error
}

// GetOrderByID 根据ID获取订单
func (h *DBHandle) GetOrderByID(orderID string) (*models.Order, error) {
	var order models.Order
	result := h.conn.Where("order_id = ?", orderID).First(&order)
	if result.Error != nil {
		return nil, result.Error
	}
	return &order, nil
}

// UpdateOrder 更新订单
func (h *DBHandle) UpdateOrder(order *models.Order) error {
	return h.conn.Save(order).Error
}

// GetUserBuyOrdersCount 获取用户购买订单数量
func (h *DBHandle) GetUserBuyOrdersCount(address string, status string) (int64, error) {
	var count int64
	result := h.conn.Model(&models.Order{}).
		Where("buyer_address = ? AND status = ?", address, status).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// GetUserSellOrdersCount 获取用户销售订单数量
func (h *DBHandle) GetUserSellOrdersCount(address string, status string) (int64, error) {
	var count int64
	result := h.conn.Model(&models.Order{}).
		Where("seller_address = ? AND status = ?", address, status).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// GetUserBuyVolume 获取用户购买总额
func (h *DBHandle) GetUserBuyVolume(address string) (float64, error) {
	var buyTotal float64
	result := h.conn.Model(&models.Order{}).
		Select("COALESCE(SUM(CAST(price AS REAL)), 0) as buy_total").
		Where("buyer_address = ? AND status = ?", address, "completed").
		Scan(&buyTotal)
	if result.Error != nil {
		return 0, result.Error
	}
	return buyTotal, nil
}

// GetUserSellVolume 获取用户销售总额
func (h *DBHandle) GetUserSellVolume(address string) (float64, error) {
	var sellTotal float64
	result := h.conn.Model(&models.Order{}).
		Select("COALESCE(SUM(CAST(price AS REAL)), 0) as sell_total").
		Where("seller_address = ? AND status = ?", address, "completed").
		Scan(&sellTotal)
	if result.Error != nil {
		return 0, result.Error
	}
	return sellTotal, nil
}

// ========== Transaction 相关操作 ==========

// CreateTransaction 创建交易记录
func (h *DBHandle) CreateTransaction(tx *models.Transaction) error {
	return h.conn.Create(tx).Error
}

// GetTransactionByHash 根据哈希获取交易
func (h *DBHandle) GetTransactionByHash(txHash string) (*models.Transaction, error) {
	var tx models.Transaction
	result := h.conn.Where("tx_hash = ?", txHash).First(&tx)
	if result.Error != nil {
		return nil, result.Error
	}
	return &tx, nil
}

// GetItemByTokenID 返回给定 token id 的物品（取第一个匹配）
func (h *DBHandle) GetItemByTokenID(tokenID int64) (*models.Item, error) {
	var item models.Item
	result := h.conn.Where("token_id = ?", tokenID).First(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	return &item, nil
}

// GetItemByTokenAndContract returns an item by token id and contract address.
func (h *DBHandle) GetItemByTokenAndContract(tokenID int64, contract string) (*models.Item, error) {
	var item models.Item
	result := h.conn.Where("token_id = ? AND nft_contract_address = ?", tokenID, strings.ToLower(contract)).First(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	return &item, nil
}

// GetItemsByTokenID returns all items with a given token ID (for debugging).
func (h *DBHandle) GetItemsByTokenID(tokenID int64) ([]*models.Item, error) {
	var items []*models.Item
	result := h.conn.Where("token_id = ?", tokenID).Order("id ASC").Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}
	return items, nil
}

// GetItemByID returns item by primary ID
func (h *DBHandle) GetItemByID(id uint) (*models.Item, error) {
	var item models.Item
	result := h.conn.First(&item, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &item, nil
}

// GetItemsByOwner returns items owned by the given address.
func (h *DBHandle) GetItemsByOwner(address string) ([]*models.Item, error) {
	var items []*models.Item
	result := h.conn.Where("owner_address = ?", strings.ToLower(address)).Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}
	return items, nil
}

// GetOrdersByUser returns orders where the user is buyer or seller.
func (h *DBHandle) GetOrdersByUser(address string) ([]*models.Order, error) {
	var orders []*models.Order
	result := h.conn.Where("buyer_address = ? OR seller_address = ?", address, address).Find(&orders)
	if result.Error != nil {
		return nil, result.Error
	}
	return orders, nil
}

// ========== PlayerItem 相关操作 ==========

// UpsertPlayerItem creates or increments a player item quantity.
func (h *DBHandle) UpsertPlayerItem(address string, itemType string) error {
	addr := strings.ToLower(address)
	logger.LogInfof("UpsertPlayerItem: upserting user=%s item=%s", addr, itemType)
	result := h.conn.Where("user_address = ? AND item_type = ?", addr, itemType).
		FirstOrCreate(&models.PlayerItem{
			UserAddress: addr,
			ItemType:    itemType,
			Quantity:    0,
		})
	if result.Error != nil {
		logger.LogErrorf("UpsertPlayerItem: FirstOrCreate failed user=%s item=%s err=%v", addr, itemType, result.Error)
		return result.Error
	}
	if err := h.conn.Model(&models.PlayerItem{}).
		Where("user_address = ? AND item_type = ?", addr, itemType).
		Update("quantity", gorm.Expr("quantity + 1")).Error; err != nil {
		logger.LogErrorf("UpsertPlayerItem: update failed user=%s item=%s err=%v", addr, itemType, err)
		return err
	}
	logger.LogInfof("UpsertPlayerItem: done user=%s item=%s", addr, itemType)
	return nil
}

// GetPlayerItemsByUser returns all reward items for a user (quantity > 0).
func (h *DBHandle) GetPlayerItemsByUser(address string) ([]*models.PlayerItem, error) {
	var items []*models.PlayerItem
	result := h.conn.Where("user_address = ? AND quantity > 0", strings.ToLower(address)).
		Order("item_type ASC").
		Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}
	return items, nil
}

// ConsumePlayerItem decrements the quantity of a player item by 1.
func (h *DBHandle) ConsumePlayerItem(address string, itemType string) (int, error) {
	return h.ConsumePlayerItemN(address, itemType, 1)
}

// ConsumePlayerItemN decrements the quantity of a player item by n.
func (h *DBHandle) ConsumePlayerItemN(address string, itemType string, n int) (int, error) {
	if n <= 0 {
		return 0, errors.New("n must be > 0")
	}
	addr := strings.ToLower(address)
	var item models.PlayerItem
	result := h.conn.Where("user_address = ? AND item_type = ?", addr, itemType).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return 0, errors.New("item not found")
		}
		return 0, result.Error
	}
	if item.Quantity < n {
		return 0, errors.New("insufficient items available")
	}
	if err := h.conn.Model(&models.PlayerItem{}).
		Where("user_address = ? AND item_type = ? AND quantity >= ?", addr, itemType, n).
		Update("quantity", gorm.Expr("quantity - ?", n)).Error; err != nil {
		return 0, err
	}
	return item.Quantity - n, nil
}

// ========== TokenPurchase 相关操作 ==========

// CreateTokenPurchase persists a token purchase record.
func (h *DBHandle) CreateTokenPurchase(p *models.TokenPurchase) error {
	return h.conn.Create(p).Error
}

// GetTokenPurchasesByUser returns all purchases for a user.
func (h *DBHandle) GetTokenPurchasesByUser(address string) ([]*models.TokenPurchase, error) {
	var purchases []*models.TokenPurchase
	result := h.conn.Where("user_address = ?", strings.ToLower(address)).
		Order("created_at DESC").Find(&purchases)
	if result.Error != nil {
		return nil, result.Error
	}
	return purchases, nil
}

// ========== Task 相关操作 ==========

// CreateTask inserts a new task.
func (h *DBHandle) CreateTask(task *models.Task) error {
	return h.conn.Create(task).Error
}

// GetTaskByTaskID returns a task by its UUID task_id.
func (h *DBHandle) GetTaskByTaskID(taskID string) (*models.Task, error) {
	var task models.Task
	result := h.conn.Where("task_id = ?", taskID).First(&task)
	if result.Error != nil {
		return nil, result.Error
	}
	return &task, nil
}

// GetTaskByTxHash returns a task by its on-chain tx hash.
func (h *DBHandle) GetTaskByTxHash(txHash string) (*models.Task, error) {
	var task models.Task
	result := h.conn.Where("tx_hash = ?", txHash).First(&task)
	if result.Error != nil {
		return nil, result.Error
	}
	return &task, nil
}

// UpdateTaskStatus updates a task's status and optional fields.
func (h *DBHandle) UpdateTaskStatus(taskID string, status string, txHash string, errMsg string, retryCount int) error {
	updates := map[string]interface{}{"status": status}
	if txHash != "" {
		updates["tx_hash"] = txHash
	}
	if errMsg != "" {
		updates["error_msg"] = errMsg
	}
	updates["retry_count"] = retryCount
	return h.conn.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(updates).Error
}

// GetPendingTasks returns tasks that haven't reached a terminal state.
func (h *DBHandle) GetPendingTasks() ([]*models.Task, error) {
	var tasks []*models.Task
	result := h.conn.Where("status IN ?", []string{"pending", "processing", "tx_sent"}).
		Order("created_at ASC").Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}
	return tasks, nil
}

// HasPendingTask checks if user already has an in-flight task for the same item type.
func (h *DBHandle) HasPendingTask(userAddress string, itemType string) (bool, error) {
	var count int64
	result := h.conn.Model(&models.Task{}).
		Where("user_address = ? AND item_type = ? AND status IN ?",
			strings.ToLower(userAddress), itemType,
			[]string{"pending", "processing", "tx_sent"}).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// MarkEventProcessed inserts a processed event record.
// Returns true if this is the first time the tx_hash is seen (new event).
// Returns false if the tx_hash was already processed (duplicate, should skip).
func (h *DBHandle) MarkEventProcessed(txHash string) (bool, error) {
	result := h.conn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tx_hash"}},
		DoNothing: true,
	}).Create(&models.ProcessedEvent{TxHash: txHash})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
