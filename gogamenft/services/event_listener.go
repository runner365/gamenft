package services

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

// TransferSingle(address,address,address,uint256,uint256)
var transferSingleTopic = common.HexToHash("0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62")
var itemListedTopic = common.HexToHash("0xbd5d1836d4114b1a1785c873a084824203ab9c294b3694e6a9fa69234a9113c6")
var itemBoughtTopic = common.HexToHash("0xfd71e10b7e3ddfdd2ccab44f5cb0855cda40e42f1bfcab4cf1339f2bd5a92c2a")

const reconnectDelay = 5 * time.Second

func ListenMintEvents(ctx context.Context, client *ethclient.Client, gameItemsAddr string, dbh *db.DBHandle, taskSvc *TaskService) {
	contractAddr := common.HexToAddress(gameItemsAddr)
	zeroAddr := common.HexToAddress("0x0000000000000000000000000000000000000000")
	zeroTopic := common.BytesToHash(zeroAddr.Bytes())

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
		Topics: [][]common.Hash{
			{transferSingleTopic},
			{},
			{zeroTopic},
			{},
		},
	}

	logger.LogInfof("Mint event listener: initializing contract=%s", gameItemsAddr)

	for {
		if ctx.Err() != nil {
			logger.LogInfof("Mint event listener: stopped contract=%s err=%v", gameItemsAddr, ctx.Err())
			return
		}

		logs := make(chan types.Log)
		sub, err := client.SubscribeFilterLogs(ctx, query, logs)
		if err != nil {
			logger.LogErrorf("Mint event listener: subscribe failed contract=%s err=%v retry_in=%v",
				gameItemsAddr, err, reconnectDelay)
			select {
			case <-ctx.Done():
				logger.LogInfof("Mint event listener: context done during retry wait contract=%s", gameItemsAddr)
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		logger.LogInfof("Mint event listener: subscribed contract=%s", gameItemsAddr)
		runMintLoop(ctx, sub, logs, dbh, taskSvc)
		sub.Unsubscribe()

		if ctx.Err() != nil {
			logger.LogInfof("Mint event listener: stopped after loop contract=%s err=%v", gameItemsAddr, ctx.Err())
			return
		}
		logger.LogInfof("Mint event listener: reconnecting in %v contract=%s", reconnectDelay, gameItemsAddr)
		select {
		case <-ctx.Done():
			logger.LogInfof("Mint event listener: context done during reconnect contract=%s", gameItemsAddr)
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func runMintLoop(ctx context.Context, sub ethereum.Subscription, logs <-chan types.Log, dbh *db.DBHandle, taskSvc *TaskService) {
	for {
		select {
		case <-ctx.Done():
			logger.LogInfof("Mint loop: context done err=%v", ctx.Err())
			return
		case err := <-sub.Err():
			logger.LogErrorf("Mint loop: subscription error err=%v", err)
			return
		case vLog := <-logs:
			logger.LogInfof("Mint loop: received log block=%d tx=%s", vLog.BlockNumber, vLog.TxHash.Hex())
			handleTransferSingle(vLog, dbh, taskSvc)
		}
	}
}

func handleTransferSingle(vLog types.Log, dbh *db.DBHandle, taskSvc *TaskService) {
	txHash := vLog.TxHash.Hex()

	if len(vLog.Data) < 64 {
		logger.LogWarnf("Mint handler: malformed event data_len=%d tx=%s block=%d",
			len(vLog.Data), txHash, vLog.BlockNumber)
		return
	}

	isNew, err := dbh.MarkEventProcessed(txHash)
	if err != nil {
		logger.LogErrorf("Mint handler: dedup check failed tx=%s err=%v", txHash, err)
		return
	}
	if !isNew {
		logger.LogInfof("Mint handler: duplicate event skipped tx=%s", txHash)
		return
	}

	tokenID := new(big.Int).SetBytes(vLog.Data[:32]).Int64()
	amount := new(big.Int).SetBytes(vLog.Data[32:64]).Int64()

	var toAddr common.Address
	if len(vLog.Topics) > 3 {
		toAddr = common.BytesToAddress(vLog.Topics[3].Bytes())
	}

	logger.LogInfof("Mint handler: to=%s tokenID=%d amount=%d tx=%s block=%d",
		toAddr.Hex(), tokenID, amount, txHash, vLog.BlockNumber)

	itemType := tokenIDToItemType(tokenID)
	if itemType == "" {
		logger.LogWarnf("Mint handler: unknown tokenID=%d tx=%s", tokenID, txHash)
		return
	}

	logger.LogInfof("Mint handler: resolved tokenID=%d -> item=%s", tokenID, itemType)

	if err := dbh.UpsertPlayerItem(toAddr.Hex(), itemType); err != nil {
		logger.LogErrorf("Mint handler: upsert failed to=%s item=%s amount=%d tx=%s err=%v",
			toAddr.Hex(), itemType, amount, txHash, err)
		return
	}
	logger.LogInfof("Mint handler: player item upserted to=%s item=%s amount=%d tx=%s",
		toAddr.Hex(), itemType, amount, txHash)

	if taskSvc != nil {
		taskSvc.ConfirmTask(txHash)
	} else {
		logger.LogWarnf("Mint handler: taskSvc is nil, skipping ConfirmTask tx=%s", txHash)
	}
}

// ListenMarketplaceEvents subscribes to ItemListed events on the NFTMarketplace contract.
func ListenMarketplaceEvents(ctx context.Context, client *ethclient.Client, marketplaceAddr string, gameItemsAddr string, dbh *db.DBHandle, hub *WsHub) {
	contractAddr := common.HexToAddress(marketplaceAddr)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
		Topics: [][]common.Hash{
			{itemListedTopic},
		},
	}

	logger.LogInfof("ItemListed listener: initializing marketplace=%s gameItems=%s", marketplaceAddr, gameItemsAddr)

	for {
		if ctx.Err() != nil {
			logger.LogInfof("ItemListed listener: stopped marketplace=%s err=%v", marketplaceAddr, ctx.Err())
			return
		}

		logs := make(chan types.Log)
		sub, err := client.SubscribeFilterLogs(ctx, query, logs)
		if err != nil {
			logger.LogErrorf("ItemListed listener: subscribe failed marketplace=%s err=%v retry_in=%v",
				marketplaceAddr, err, reconnectDelay)
			select {
			case <-ctx.Done():
				logger.LogInfof("ItemListed listener: context done during retry wait marketplace=%s", marketplaceAddr)
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		logger.LogInfof("ItemListed listener: subscribed marketplace=%s", marketplaceAddr)
		runMarketplaceLoop(ctx, sub, logs, dbh, gameItemsAddr, hub)
		sub.Unsubscribe()

		if ctx.Err() != nil {
			logger.LogInfof("ItemListed listener: stopped after loop marketplace=%s err=%v", marketplaceAddr, ctx.Err())
			return
		}
		logger.LogInfof("ItemListed listener: reconnecting in %v marketplace=%s", reconnectDelay, marketplaceAddr)
		select {
		case <-ctx.Done():
			logger.LogInfof("ItemListed listener: context done during reconnect marketplace=%s", marketplaceAddr)
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func runMarketplaceLoop(ctx context.Context, sub ethereum.Subscription, logs <-chan types.Log, dbh *db.DBHandle, gameItemsAddr string, hub *WsHub) {
	for {
		select {
		case <-ctx.Done():
			logger.LogInfof("Marketplace loop: context done err=%v", ctx.Err())
			return
		case err := <-sub.Err():
			logger.LogErrorf("Marketplace loop: subscription error err=%v", err)
			return
		case vLog := <-logs:
			logger.LogInfof("Marketplace loop: received log block=%d tx=%s", vLog.BlockNumber, vLog.TxHash.Hex())
			handleItemListed(vLog, dbh, gameItemsAddr, hub)
		}
	}
}

func handleItemListed(vLog types.Log, dbh *db.DBHandle, gameItemsAddr string, hub *WsHub) {
	// Event: ItemListed(address indexed nftContract, uint256 indexed tokenId, address seller, uint256 amount, uint256 price)
	// topic[1] = nftContract, topic[2] = tokenId
	// data = ABI-encoded (address seller, uint256 amount, uint256 price) = 96 bytes
	txHash := vLog.TxHash.Hex()

	if len(vLog.Topics) < 3 || len(vLog.Data) < 96 {
		logger.LogWarnf("ItemListed handler: malformed event topics=%d data_len=%d tx=%s block=%d",
			len(vLog.Topics), len(vLog.Data), txHash, vLog.BlockNumber)
		return
	}

	isNew, err := dbh.MarkEventProcessed(txHash)
	if err != nil {
		logger.LogErrorf("ItemListed handler: dedup check failed tx=%s err=%v", txHash, err)
		return
	}
	if !isNew {
		logger.LogInfof("ItemListed handler: duplicate event skipped tx=%s", txHash)
		return
	}

	nftContract := common.BytesToAddress(vLog.Topics[1].Bytes())
	tokenID := new(big.Int).SetBytes(vLog.Topics[2].Bytes()).Int64()
	seller := common.BytesToAddress(vLog.Data[12:32])
	amount := new(big.Int).SetBytes(vLog.Data[32:64]).Int64()
	price := new(big.Int).SetBytes(vLog.Data[64:96])

	logger.LogInfof("ItemListed handler: contract=%s tokenID=%d seller=%s amount=%d price=%s tx=%s block=%d",
		nftContract.Hex(), tokenID, seller.Hex(), amount, price.String(), txHash, vLog.BlockNumber)

	if !strings.EqualFold(nftContract.Hex(), gameItemsAddr) {
		logger.LogInfof("ItemListed handler: skipping non-GameItems contract=%s expected=%s tx=%s",
			nftContract.Hex(), gameItemsAddr, txHash)
		return
	}

	itemType := tokenIDToItemType(tokenID)
	if itemType == "" {
		logger.LogWarnf("ItemListed handler: unknown tokenID=%d tx=%s", tokenID, txHash)
		return
	}

	sellerAddr := strings.ToLower(seller.Hex())
	logger.LogInfof("ItemListed handler: resolved tokenID=%d -> item=%s seller=%s", tokenID, itemType, sellerAddr)

	remaining, err := dbh.ConsumePlayerItemN(sellerAddr, itemType, int(amount))
	if err != nil {
		logger.LogErrorf("ItemListed handler: consume failed seller=%s item=%s amount=%d tx=%s err=%v",
			sellerAddr, itemType, amount, txHash, err)
		return
	}
	logger.LogInfof("ItemListed handler: consumed player item seller=%s item=%s amount=%d remaining=%d",
		sellerAddr, itemType, amount, remaining)

	now := time.Now()
	item := &models.Item{
		TokenID:            tokenID,
		NFTContractAddress: strings.ToLower(nftContract.Hex()),
		OwnerAddress:       sellerAddr,
		CreatorAddress:     sellerAddr,
		Name:               itemType,
		Description:        "Game reward item: " + itemType,
		TokenStandard:      "ERC1155",
		Amount:             int(amount),
		IsListed:           true,
		ListPrice:          price.String(),
		ListedAt:           &now,
	}
	if err := dbh.CreateOrUpdateItem(item); err != nil {
		logger.LogErrorf("ItemListed handler: create item failed seller=%s item=%s tokenID=%d amount=%d price=%s tx=%s err=%v",
			sellerAddr, itemType, tokenID, amount, price.String(), txHash, err)
		return
	}
	logger.LogInfof("ItemListed handler: marketplace item upserted seller=%s item=%s tokenID=%d amount=%d price=%s tx=%s",
		sellerAddr, itemType, tokenID, amount, price.String(), txHash)

	if hub != nil {
		hub.Publish(sellerAddr, WsMessage{
			Type:     "item_listed",
			ItemType: itemType,
			Quantity: int(amount),
			TxHash:   txHash,
		})
		logger.LogInfof("ItemListed handler: WS notified seller=%s type=item_listed item=%s", sellerAddr, itemType)
	} else {
		logger.LogWarnf("ItemListed handler: hub is nil, WS notification skipped seller=%s tx=%s", sellerAddr, txHash)
	}
}

// ListenItemBoughtEvents subscribes to ItemBought events and updates DB state.
func ListenItemBoughtEvents(ctx context.Context, client *ethclient.Client, marketplaceAddr string, gameItemsAddr string, dbh *db.DBHandle, hub *WsHub) {
	contractAddr := common.HexToAddress(marketplaceAddr)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
		Topics: [][]common.Hash{
			{itemBoughtTopic},
		},
	}

	logger.LogInfof("ItemBought listener: initializing marketplace=%s gameItems=%s", marketplaceAddr, gameItemsAddr)

	for {
		if ctx.Err() != nil {
			logger.LogInfof("ItemBought listener: stopped marketplace=%s err=%v", marketplaceAddr, ctx.Err())
			return
		}

		logs := make(chan types.Log)
		sub, err := client.SubscribeFilterLogs(ctx, query, logs)
		if err != nil {
			logger.LogErrorf("ItemBought listener: subscribe failed marketplace=%s err=%v retry_in=%v",
				marketplaceAddr, err, reconnectDelay)
			select {
			case <-ctx.Done():
				logger.LogInfof("ItemBought listener: context done during retry wait marketplace=%s", marketplaceAddr)
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		logger.LogInfof("ItemBought listener: subscribed marketplace=%s", marketplaceAddr)
		runItemBoughtLoop(ctx, sub, logs, dbh, gameItemsAddr, hub)
		sub.Unsubscribe()

		if ctx.Err() != nil {
			logger.LogInfof("ItemBought listener: stopped after loop marketplace=%s err=%v", marketplaceAddr, ctx.Err())
			return
		}
		logger.LogInfof("ItemBought listener: reconnecting in %v marketplace=%s", reconnectDelay, marketplaceAddr)
		select {
		case <-ctx.Done():
			logger.LogInfof("ItemBought listener: context done during reconnect marketplace=%s", marketplaceAddr)
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func runItemBoughtLoop(ctx context.Context, sub ethereum.Subscription, logs <-chan types.Log, dbh *db.DBHandle, gameItemsAddr string, hub *WsHub) {
	for {
		select {
		case <-ctx.Done():
			logger.LogInfof("ItemBought loop: context done err=%v", ctx.Err())
			return
		case err := <-sub.Err():
			logger.LogErrorf("ItemBought loop: subscription error err=%v", err)
			return
		case vLog := <-logs:
			logger.LogInfof("ItemBought loop: received log block=%d tx=%s", vLog.BlockNumber, vLog.TxHash.Hex())
			handleItemBought(vLog, dbh, gameItemsAddr, hub)
		}
	}
}

func handleItemBought(vLog types.Log, dbh *db.DBHandle, gameItemsAddr string, hub *WsHub) {
	// Event: ItemBought(address indexed nftContract, uint256 indexed tokenId, address buyer, uint256 amount, uint256 totalPrice)
	// topic[1] = nftContract, topic[2] = tokenId
	// data = ABI-encoded (address buyer, uint256 amount, uint256 totalPrice) = 96 bytes
	txHash := vLog.TxHash.Hex()

	if len(vLog.Topics) < 3 || len(vLog.Data) < 96 {
		logger.LogWarnf("ItemBought handler: malformed event topics=%d data_len=%d tx=%s block=%d",
			len(vLog.Topics), len(vLog.Data), txHash, vLog.BlockNumber)
		return
	}

	isNew, err := dbh.MarkEventProcessed(txHash)
	if err != nil {
		logger.LogErrorf("ItemBought handler: dedup check failed tx=%s err=%v", txHash, err)
		return
	}
	if !isNew {
		logger.LogInfof("ItemBought handler: duplicate event skipped tx=%s", txHash)
		return
	}

	nftContract := common.BytesToAddress(vLog.Topics[1].Bytes())
	tokenID := new(big.Int).SetBytes(vLog.Topics[2].Bytes()).Int64()
	buyer := common.BytesToAddress(vLog.Data[12:32])
	amount := new(big.Int).SetBytes(vLog.Data[32:64]).Int64()
	totalPrice := new(big.Int).SetBytes(vLog.Data[64:96])

	logger.LogInfof("ItemBought handler: contract=%s tokenID=%d buyer=%s amount=%d totalPrice=%s tx=%s block=%d",
		nftContract.Hex(), tokenID, buyer.Hex(), amount, totalPrice.String(), txHash, vLog.BlockNumber)

	if !strings.EqualFold(nftContract.Hex(), gameItemsAddr) {
		logger.LogInfof("ItemBought handler: skipping non-GameItems contract=%s expected=%s tx=%s",
			nftContract.Hex(), gameItemsAddr, txHash)
		return
	}

	item, err := dbh.GetItemByTokenAndContract(tokenID, nftContract.Hex())
	if err != nil {
		logger.LogErrorf("ItemBought handler: item not found tokenID=%d contract=%s tx=%s err=%v",
			tokenID, nftContract.Hex(), txHash, err)
		return
	}

	sellerAddr := item.OwnerAddress
	buyerAddr := strings.ToLower(buyer.Hex())
	oldAmount := item.Amount
	wasListed := item.IsListed

	logger.LogInfof("ItemBought handler: found item id=%d seller=%s oldAmount=%d wasListed=%v",
		item.ID, sellerAddr, oldAmount, wasListed)

	item.OwnerAddress = buyerAddr
	if int(amount) >= item.Amount {
		item.IsListed = false
		item.Amount = 0
		logger.LogInfof("ItemBought handler: fully sold item id=%d amount=%d >= remaining=%d",
			item.ID, amount, oldAmount)
	} else {
		item.Amount -= int(amount)
		logger.LogInfof("ItemBought handler: partial fill item id=%d sold=%d remaining=%d",
			item.ID, amount, item.Amount)
	}

	if err := dbh.CreateOrUpdateItem(item); err != nil {
		logger.LogErrorf("ItemBought handler: update item failed id=%d seller=%s buyer=%s amount=%d tx=%s err=%v",
			item.ID, sellerAddr, buyerAddr, amount, txHash, err)
		return
	}
	logger.LogInfof("ItemBought handler: item updated id=%d seller=%s buyer=%s isListed=%v remaining=%d tx=%s",
		item.ID, sellerAddr, buyerAddr, item.IsListed, item.Amount, txHash)

	itemType := tokenIDToItemType(tokenID)
	if err := dbh.UpsertPlayerItem(buyerAddr, itemType); err != nil {
		logger.LogErrorf("ItemBought handler: upsert player item failed buyer=%s item=%s tx=%s err=%v",
			buyerAddr, itemType, txHash, err)
	} else {
		logger.LogInfof("ItemBought handler: player item upserted buyer=%s item=%s tx=%s",
			buyerAddr, itemType, txHash)
	}

	if hub != nil {
		hub.Publish(buyerAddr, WsMessage{
			Type:     "item_bought",
			ItemType: tokenIDToItemType(tokenID),
			Quantity: int(amount),
			TxHash:   txHash,
		})
		logger.LogInfof("ItemBought handler: WS notified buyer=%s type=item_bought item=%s amount=%d",
			buyerAddr, tokenIDToItemType(tokenID), amount)
		hub.Publish(sellerAddr, WsMessage{
			Type:     "item_sold",
			ItemType: tokenIDToItemType(tokenID),
			Quantity: int(amount),
			TxHash:   txHash,
		})
		logger.LogInfof("ItemBought handler: WS notified seller=%s type=item_sold item=%s amount=%d",
			sellerAddr, tokenIDToItemType(tokenID), amount)
	} else {
		logger.LogWarnf("ItemBought handler: hub is nil, WS notifications skipped buyer=%s seller=%s tx=%s",
			buyerAddr, sellerAddr, txHash)
	}
}

func tokenIDToItemType(tokenID int64) string {
	switch tokenID {
	case 1:
		return "knife"
	case 2:
		return "pistol"
	case 3:
		return "bomb"
	default:
		return ""
	}
}
