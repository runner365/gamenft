package services

import (
	"context"
	"errors"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

// PlayerItemService handles game reward items (knife/pistol/bomb).
type PlayerItemService struct {
	db            *db.DBHandle
	gameItemsAddr string
}

func NewPlayerItemService(dbh *db.DBHandle, gameItemsAddr string) *PlayerItemService {
	return &PlayerItemService{db: dbh, gameItemsAddr: gameItemsAddr}
}

var validItemTypes = map[string]bool{"knife": true, "pistol": true, "bomb": true}

// Matches GameItems.sol: KNIFE=1, PISTOL=2, BOMB=3
var itemTypeToTokenID = map[string]int64{"knife": 1, "pistol": 2, "bomb": 3}

func (s *PlayerItemService) EarnReward(ctx context.Context, address string, itemType string) error {
	if _, err := s.db.GetUserByAddress(address); err != nil {
		return errors.New("user not found")
	}
	if !validItemTypes[itemType] {
		return errors.New("invalid item type: " + itemType)
	}
	logger.LogInfof("EarnReward: user=%s item=%s", address, itemType)
	return s.db.UpsertPlayerItem(address, itemType)
}

// ValidateReward checks that the user exists and the item type is valid,
// without writing to the database. The DB write is handled by the event listener.
func (s *PlayerItemService) ValidateReward(ctx context.Context, address string, itemType string) error {
	logger.LogInfof("ValidateReward: checking user=%s item=%s", address, itemType)
	if _, err := s.db.GetUserByAddress(address); err != nil {
		logger.LogErrorf("ValidateReward: user not found address=%s", address)
		return errors.New("user not found")
	}
	if !validItemTypes[itemType] {
		logger.LogErrorf("ValidateReward: invalid item type address=%s item=%s", address, itemType)
		return errors.New("invalid item type: " + itemType)
	}
	logger.LogInfof("ValidateReward: passed user=%s item=%s", address, itemType)
	return nil
}

func (s *PlayerItemService) GetInventory(ctx context.Context, address string) ([]*models.PlayerItem, error) {
	ret, err := s.db.GetPlayerItemsByUser(address)
	if err != nil {
		logger.LogErrorf("GetInventory: failed to get inventory for user=%s: %v", address, err)
		return nil, err
	}
	logger.LogInfof("GetInventory: user=%s inventory=%v", address, ret)
	return ret, nil
}

func (s *PlayerItemService) UseItem(ctx context.Context, address string, itemType string) error {
	if _, err := s.db.GetUserByAddress(address); err != nil {
		logger.LogErrorf("UseItem: user not found address=%s", address)
		return errors.New("user not found")
	}
	if !validItemTypes[itemType] {
		logger.LogErrorf("UseItem: invalid item type address=%s item=%s", address, itemType)
		return errors.New("invalid item type: " + itemType)
	}
	logger.LogInfof("UseItem: user=%s item=%s", address, itemType)
	remaining, err := s.db.ConsumePlayerItem(address, itemType)
	if err != nil {
		logger.LogErrorf("UseItem: failed to consume item for user=%s item=%s: %v", address, itemType, err)
		return err
	}
	logger.LogInfof("UseItem: user=%s item=%s remaining=%d", address, itemType, remaining)
	return nil
}

// ValidateSell checks preconditions for listing items on-chain.
// The actual DB update (consume + create listing) is handled by the
// ItemListed event listener after the chain transaction confirms.
func (s *PlayerItemService) ValidateSell(ctx context.Context, address string, itemType string, amount int) error {
	if _, err := s.db.GetUserByAddress(address); err != nil {
		logger.LogErrorf("ValidateSell: user not found address=%s", address)
		return errors.New("user not found")
	}
	if !validItemTypes[itemType] {
		logger.LogErrorf("ValidateSell: invalid item type address=%s item=%s", address, itemType)
		return errors.New("invalid item type: " + itemType)
	}
	if amount <= 0 {
		logger.LogErrorf("ValidateSell: invalid amount address=%s item=%s amount=%d", address, itemType, amount)
		return errors.New("amount must be > 0")
	}
	items, err := s.db.GetPlayerItemsByUser(address)
	if err != nil {
		logger.LogErrorf("ValidateSell: failed to get player items for user=%s: %v", address, err)
		return err
	}
	for _, it := range items {
		if it.ItemType == itemType && it.Quantity >= amount {
			logger.LogInfof("ValidateSell: passed user=%s item=%s amount=%d", address, itemType, amount)
			return nil
		}
	}
	logger.LogErrorf("ValidateSell: insufficient balance for user=%s item=%s amount=%d", address, itemType, amount)
	return errors.New("insufficient balance")
}
