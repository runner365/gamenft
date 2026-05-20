// db/db_test.go
package db

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

func initLogger() {
	// 初始化日志记录器，输出到test.log文件，日志级别为info
	logger.InitLogger(logger.Config{
		Level:      "info",
		Filename:   "test.log",
		MaxSize:    1000, // 1GB
		MaxBackups: 3,
		MaxAge:     7,    // 7天
		Compress:   true, // 压缩旧日志
	})
}

// setupTestDB 创建测试数据库连接
func setupTestDB(t *testing.T) *gorm.DB {
	initLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// 自动迁移模型
	err = db.AutoMigrate(
		&models.User{},
		&models.Item{},
		&models.Order{},
		&models.Transaction{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

// TestDBHandle_CreateUser 测试创建用户
func TestDBHandle_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	user := &models.User{
		EthAddress: "0x1234567890abcdef1234567890abcdef12345678",
		Username:   "testuser",
		Nonce:      "random_nonce",
		IsActive:   true,
		LastLogin:  time.Now(),
	}

	err := handle.CreateUser(user)
	if err != nil {
		t.Errorf("CreateUser failed: %v", err)
	}

	// 验证用户已创建
	foundUser, err := handle.GetUserByAddress(user.EthAddress)
	if err != nil {
		t.Errorf("GetUserByAddress failed: %v", err)
	}
	t.Logf("found user ok by EthAddress:%v", user.EthAddress)

	if foundUser.Username != user.Username {
		t.Errorf("expected username %s, got %s", user.Username, foundUser.Username)
	}

	if !strings.EqualFold(foundUser.EthAddress, user.EthAddress) {
		t.Errorf("expected eth_address %s, got %s", user.EthAddress, foundUser.EthAddress)
	}
	logger.LogInfof("TestDBHandle_CreateUser done,found user ok by EthAddress:%v", user.EthAddress)
}

// TestDBHandle_GetUserByAddress 测试根据地址获取用户
func TestDBHandle_GetUserByAddress(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	// 创建测试用户
	user := &models.User{
		EthAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
		Username:   "testuser2",
		Nonce:      "nonce123",
		IsActive:   true,
	}

	err := handle.CreateUser(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 测试正常获取
	foundUser, err := handle.GetUserByAddress(user.EthAddress)
	if err != nil {
		t.Errorf("GetUserByAddress failed: %v", err)
	}
	if foundUser.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, foundUser.ID)
	}

	// 测试地址大小写不敏感
	foundUser, err = handle.GetUserByAddress(strings.ToUpper(user.EthAddress))
	if err != nil {
		t.Errorf("GetUserByAddress with uppercase address failed: %v", err)
	}
	if foundUser.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, foundUser.ID)
	}

	// 测试获取不存在的用户
	_, err = handle.GetUserByAddress("0x0000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
	logger.LogInfof("TestDBHandle_GetUserByAddress done,found user ok by EthAddress:%v", user.EthAddress)
}

// TestDBHandle_GetUserWithListedItems 测试获取用户及其在售物品
func TestDBHandle_GetUserWithListedItems(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	// 创建用户
	user := &models.User{
		EthAddress: "0xuser1234567890abcdef1234567890abcdef12",
		Username:   "userwithitems",
		Nonce:      "nonce456",
		IsActive:   true,
	}
	err := handle.CreateUser(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 创建在售物品
	item := &models.Item{
		TokenID:            1,
		NFTContractAddress: "0xcontract1234567890abcdef1234567890abcdef12",
		OwnerAddress:       user.EthAddress,
		Name:               "Test Item",
		IsListed:           true,
		ListPrice:          "1000000000000000000",
		ListedAt:           &[]time.Time{time.Now()}[0],
	}
	err = handle.CreateOrUpdateItem(item)
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	// 获取用户及其在售物品
	foundUser, err := handle.GetUserWithListedItems(user.EthAddress)
	if err != nil {
		t.Errorf("GetUserWithListedItems failed: %v", err)
	}

	if len(foundUser.Items) != 1 {
		t.Errorf("expected 1 listed item, got %d", len(foundUser.Items))
	}

	if foundUser.Items[0].Name != "Test Item" {
		t.Errorf("expected item name 'Test Item', got '%s'", foundUser.Items[0].Name)
	}
	logger.LogInfof("TestDBHandle_GetUserWithListedItems done,found user ok by EthAddress:%v", user.EthAddress)
}

// TestDBHandle_UpdateUserNonce 测试更新用户nonce
func TestDBHandle_UpdateUserNonce(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	user := &models.User{
		EthAddress: "0xupdate1234567890abcdef1234567890abcdef12",
		Username:   "updateuser",
		Nonce:      "old_nonce",
		IsActive:   true,
	}
	err := handle.CreateUser(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	newNonce := "new_nonce_123"
	err = handle.UpdateUserNonce(user.EthAddress, newNonce)
	if err != nil {
		t.Errorf("UpdateUserNonce failed: %v", err)
	}

	foundUser, err := handle.GetUserByAddress(user.EthAddress)
	if err != nil {
		t.Errorf("GetUserByAddress failed: %v", err)
	}

	if foundUser.Nonce != newNonce {
		t.Errorf("expected nonce %s, got %s", newNonce, foundUser.Nonce)
	}
	logger.LogInfof("TestDBHandle_UpdateUserNonce done,updated nonce for EthAddress:%v", user.EthAddress)
}

// TestDBHandle_CheckUsernameExists 测试检查用户名是否存在
func TestDBHandle_CheckUsernameExists(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	user1 := &models.User{
		EthAddress: "0xuser11234567890abcdef1234567890abcdef12",
		Username:   "uniqueuser",
		Nonce:      "nonce1",
		IsActive:   true,
	}
	err := handle.CreateUser(user1)
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	user2 := &models.User{
		EthAddress: "0xuser21234567890abcdef1234567890abcdef12",
		Username:   "anotheruser",
		Nonce:      "nonce2",
		IsActive:   true,
	}
	err = handle.CreateUser(user2)
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	// 测试用户名已存在
	exists, err := handle.CheckUsernameExists("uniqueuser", 0)
	if err != nil {
		t.Errorf("CheckUsernameExists failed: %v", err)
	}
	if !exists {
		t.Error("expected username to exist, got false")
	}

	// 测试用户名不存在
	exists, err = handle.CheckUsernameExists("nonexistent", 0)
	if err != nil {
		t.Errorf("CheckUsernameExists failed: %v", err)
	}
	if exists {
		t.Error("expected username to not exist, got true")
	}

	// 测试排除当前用户
	exists, err = handle.CheckUsernameExists("uniqueuser", user1.ID)
	if err != nil {
		t.Errorf("CheckUsernameExists with excludeID failed: %v", err)
	}
	if exists {
		t.Error("expected username to not exist when excluding current user, got true")
	}
	logger.LogInfof("TestDBHandle_CheckUsernameExists done,checked username exists for %s", user1.Username)
}

// TestDBHandle_CreateOrUpdateItem 测试创建或更新物品
func TestDBHandle_CreateOrUpdateItem(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	item := &models.Item{
		TokenID:            100,
		NFTContractAddress: "0xcontractitem1234567890abcdef1234567890abcdef",
		OwnerAddress:       "0xowner1234567890abcdef1234567890abcdef12",
		Name:               "Original Item",
		IsListed:           false,
	}

	// 创建物品
	err := handle.CreateOrUpdateItem(item)
	if err != nil {
		t.Errorf("CreateOrUpdateItem failed: %v", err)
	}

	// 更新物品
	item.Name = "Updated Item"
	item.IsListed = true
	err = handle.CreateOrUpdateItem(item)
	if err != nil {
		t.Errorf("CreateOrUpdateItem (update) failed: %v", err)
	}

	// 验证更新
	var foundItem models.Item
	result := db.First(&foundItem, item.ID)
	if result.Error != nil {
		t.Fatalf("failed to find item: %v", result.Error)
	}

	if foundItem.Name != "Updated Item" {
		t.Errorf("expected item name 'Updated Item', got '%s'", foundItem.Name)
	}

	if !foundItem.IsListed {
		t.Error("expected item to be listed")
	}
	logger.LogInfof("TestDBHandle_CreateOrUpdateItem done,created or updated item with ID:%v", item.ID)
}

// TestDBHandle_GetListedItems 测试获取在售物品列表
func TestDBHandle_GetListedItems(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	// 创建多个物品
	for i := 0; i < 15; i++ {
		isListed := i%2 == 0 // 偶数索引的物品在售
		listedAt := time.Now().Add(time.Duration(-i) * time.Hour)

		item := &models.Item{
			TokenID:            int64(i + 1),
			NFTContractAddress: "0xcontractlist1234567890abcdef1234567890abcdef",
			OwnerAddress:       "0xownerlist1234567890abcdef1234567890abcdef12",
			Name:               "Item " + string(rune('A'+i)),
			IsListed:           isListed,
			ListPrice:          "1000000000000000000",
		}
		if isListed {
			item.ListedAt = &listedAt
		}

		err := handle.CreateOrUpdateItem(item)
		if err != nil {
			t.Fatalf("failed to create item %d: %v", i, err)
		}
	}

	// 测试分页获取
	items, total, err := handle.GetListedItems(1, 5)
	if err != nil {
		t.Errorf("GetListedItems failed: %v", err)
	}

	if total != 8 { // 15个物品中，偶数索引的8个在售
		t.Errorf("expected total 8, got %d", total)
	}

	if len(items) != 5 {
		t.Errorf("expected 5 items per page, got %d", len(items))
	}

	// 测试第二页
	items, _, err = handle.GetListedItems(2, 5)
	if err != nil {
		t.Errorf("GetListedItems page 2 failed: %v", err)
	}

	if len(items) != 3 { // 剩余3个在售物品
		t.Errorf("expected 3 items on page 2, got %d", len(items))
	}
	logger.LogInfof("TestDBHandle_GetListedItems done,got %d listed items with total %d", len(items), total)
}

// TestDBHandle_CreateOrder 测试创建订单
func TestDBHandle_CreateOrder(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	order := &models.Order{
		OrderID:       "test-order-001",
		SellerAddress: "0xseller1234567890abcdef1234567890abcdef12",
		BuyerAddress:  "0xbuyer1234567890abcdef1234567890abcdef12",
		Price:         "2000000000000000000",
		Status:        "pending",
	}

	err := handle.CreateOrder(order)
	if err != nil {
		t.Errorf("CreateOrder failed: %v", err)
	}

	foundOrder, err := handle.GetOrderByID(order.OrderID)
	if err != nil {
		t.Errorf("GetOrderByID failed: %v", err)
	}

	if foundOrder.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", foundOrder.Status)
	}

	if foundOrder.Price != "2000000000000000000" {
		t.Errorf("expected price '2000000000000000000', got '%s'", foundOrder.Price)
	}
	logger.LogInfof("TestDBHandle_CreateOrder done,created order with ID:%v", order.OrderID)
}

// TestDBHandle_UpdateOrder 测试更新订单
func TestDBHandle_UpdateOrder(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	order := &models.Order{
		OrderID:       "test-order-002",
		SellerAddress: "0xseller21234567890abcdef1234567890abcdef12",
		Price:         "3000000000000000000",
		Status:        "pending",
	}

	err := handle.CreateOrder(order)
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// 更新订单状态
	order.Status = "completed"
	now := time.Now()
	order.CompletedAt = &now

	err = handle.UpdateOrder(order)
	if err != nil {
		t.Errorf("UpdateOrder failed: %v", err)
	}

	foundOrder, err := handle.GetOrderByID(order.OrderID)
	if err != nil {
		t.Errorf("GetOrderByID failed: %v", err)
	}

	if foundOrder.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", foundOrder.Status)
	}

	if foundOrder.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
	logger.LogInfof("TestDBHandle_UpdateOrder done,updated order with ID:%v", order.OrderID)
}

// TestDBHandle_GetUserBuyOrdersCount 测试获取用户购买订单数量
func TestDBHandle_GetUserBuyOrdersCount(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	buyerAddr := "0xbuyercount1234567890abcdef1234567890abcdef"

	// 创建测试订单
	for i := 0; i < 5; i++ {
		status := "completed"
		if i%2 == 0 {
			status = "pending"
		}

		order := &models.Order{
			OrderID:       "order-count-" + string(rune('0'+i)),
			SellerAddress: "0xsellercount1234567890abcdef1234567890abcdef",
			BuyerAddress:  buyerAddr,
			Price:         "1000000000000000000",
			Status:        status,
		}

		err := handle.CreateOrder(order)
		if err != nil {
			t.Fatalf("failed to create order %d: %v", i, err)
		}
	}

	// 测试获取已完成订单数量
	count, err := handle.GetUserBuyOrdersCount(buyerAddr, "completed")
	if err != nil {
		t.Errorf("GetUserBuyOrdersCount failed: %v", err)
	}

	if count != 2 { // 5个订单中，奇数索引的2个已完成
		t.Errorf("expected count 2, got %d", count)
	}

	// 测试获取待处理订单数量
	count, err = handle.GetUserBuyOrdersCount(buyerAddr, "pending")
	if err != nil {
		t.Errorf("GetUserBuyOrdersCount (pending) failed: %v", err)
	}

	if count != 3 { // 5个订单中，偶数索引的3个待处理
		t.Errorf("expected count 3, got %d", count)
	}
	logger.LogInfof("TestDBHandle_GetUserBuyOrdersCount done,checked user buy orders count for %s", buyerAddr)
}

// TestDBHandle_CreateTransaction 测试创建交易记录
func TestDBHandle_CreateTransaction(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	tx := &models.Transaction{
		TxHash:             "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678",
		BlockNumber:        12345678,
		BlockTimestamp:     time.Now().Unix(),
		NFTContractAddress: "0xcontracttx1234567890abcdef1234567890abcdef12",
		TokenID:            123,
		FromAddress:        "0xfromtx1234567890abcdef1234567890abcdef12",
		ToAddress:          "0xtotx1234567890abcdef1234567890abcdef12",
		Price:              "5000000000000000000",
		TxType:             "transfer",
		Status:             "confirmed",
	}

	err := handle.CreateTransaction(tx)
	if err != nil {
		t.Errorf("CreateTransaction failed: %v", err)
	}

	foundTx, err := handle.GetTransactionByHash(tx.TxHash)
	if err != nil {
		t.Errorf("GetTransactionByHash failed: %v", err)
	}

	if foundTx.TxType != "transfer" {
		t.Errorf("expected tx_type 'transfer', got '%s'", foundTx.TxType)
	}

	if foundTx.Status != "confirmed" {
		t.Errorf("expected status 'confirmed', got '%s'", foundTx.Status)
	}
	logger.LogInfof("TestDBHandle_CreateTransaction done,created transaction with hash:%v", tx.TxHash)
}

// TestDBHandle_UpdateUserLastLogin 测试更新用户最后登录时间
func TestDBHandle_UpdateUserLastLogin(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	user := &models.User{
		EthAddress: "0xlastlogin1234567890abcdef1234567890abcdef",
		Username:   "lastloginuser",
		Nonce:      "nonce789",
		IsActive:   true,
		LastLogin:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	err := handle.CreateUser(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 更新最后登录时间
	err = handle.UpdateUserLastLogin(user.EthAddress)
	if err != nil {
		t.Errorf("UpdateUserLastLogin failed: %v", err)
	}

	foundUser, err := handle.GetUserByAddress(user.EthAddress)
	if err != nil {
		t.Errorf("GetUserByAddress failed: %v", err)
	}

	// 验证登录时间已更新（应该接近当前时间）
	if foundUser.LastLogin.Before(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Error("last_login was not updated")
	}
	logger.LogInfof("TestDBHandle_UpdateUserLastLogin done,updated last login for EthAddress:%v", user.EthAddress)
}

// TestDBHandle_UpdateItemStatus 测试更新物品状态
func TestDBHandle_UpdateItemStatus(t *testing.T) {
	db := setupTestDB(t)
	handle := NewDBHandle(db)

	item := &models.Item{
		TokenID:            456,
		NFTContractAddress: "0xcontractstatus1234567890abcdef1234567890abcdef",
		OwnerAddress:       "0xownerstatus1234567890abcdef1234567890abcdef",
		Name:               "Status Test Item",
		IsListed:           false,
	}

	err := handle.CreateOrUpdateItem(item)
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	// 更新为在售
	err = handle.UpdateItemStatus(item.ID, true)
	if err != nil {
		t.Errorf("UpdateItemStatus failed: %v", err)
	}

	var foundItem models.Item
	result := db.First(&foundItem, item.ID)
	if result.Error != nil {
		t.Fatalf("failed to find item: %v", result.Error)
	}

	if !foundItem.IsListed {
		t.Error("expected item to be listed after update")
	}

	// 更新为不在售
	err = handle.UpdateItemStatus(item.ID, false)
	if err != nil {
		t.Errorf("UpdateItemStatus (unlist) failed: %v", err)
	}

	result = db.First(&foundItem, item.ID)
	if result.Error != nil {
		t.Fatalf("failed to find item: %v", result.Error)
	}

	if foundItem.IsListed {
		t.Error("expected item to not be listed after update")
	}
	logger.LogInfof("TestDBHandle_UpdateItemStatus done,updated item status for item ID:%v", item.ID)
}
