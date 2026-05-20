// order_service.go
package services

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

type OrderService struct {
	db *db.DBHandle
}

func NewOrderService(dbh *db.DBHandle) *OrderService {
	return &OrderService{db: dbh}
}

// RecordBuyOrder creates a completed order record for an on-chain purchase
// that has already been executed by the client. The ItemBought event listener
// handles DB state updates (is_listed, owner_address).
func (s *OrderService) RecordBuyOrder(ctx context.Context,
	userAddress string,
	itemID uint,
	txHash string) (*models.Order, error) {

	item, err := s.db.GetItemByID(itemID)
	if err != nil {
		logger.LogErrorf("RecordBuyOrder: item not found itemID=%d err=%v", itemID, err)
		return nil, errors.New("item not found")
	}

	logger.LogInfof("RecordBuyOrder: itemID=%d tokenID=%d owner=%s isListed=%v", item.ID, item.TokenID, item.OwnerAddress, item.IsListed)

	now := time.Now()
	order := &models.Order{
		OrderID:       uuid.New().String(),
		ItemID:        item.ID,
		SellerAddress: item.OwnerAddress,
		BuyerAddress:  userAddress,
		Price:         item.ListPrice,
		Status:        "completed",
		TxHash:        txHash,
		PaidAt:        &now,
		CompletedAt:   &now,
		CreatedAt:     now,
	}

	if err := s.db.CreateOrder(order); err != nil {
		logger.LogErrorf("RecordBuyOrder: create order failed itemID=%d err=%v", itemID, err)
		return nil, err
	}

	logger.LogInfof("RecordBuyOrder: success orderID=%s itemID=%d buyer=%s tx=%s", order.OrderID, itemID, userAddress, txHash)
	return order, nil
}

// GetOrdersByUser returns orders where the user is buyer or seller.
func (s *OrderService) GetOrdersByUser(ctx context.Context, address string) ([]*models.Order, error) {
	ret, err := s.db.GetOrdersByUser(address)
	if err != nil {
		logger.LogErrorf("GetOrdersByUser: failed to get orders for user=%s: %v", address, err)
		return nil, err
	}
	logger.LogInfof("GetOrdersByUser: user=%s orders=%v", address, ret)
	return ret, nil
}

// parseBigInt parses a decimal string into a *big.Int, handling the case
// where PostgreSQL numeric columns include a decimal point.
func parseBigInt(s string) (*big.Int, bool) {
	if idx := strings.IndexByte(s, '.'); idx != -1 {
		s = s[:idx]
	}
	bigInt, ok := new(big.Int).SetString(s, 10)
	if !ok {
		logger.LogErrorf("parseBigInt: failed to parse string=%s", s)
		return nil, false
	}
	return bigInt, true
}
