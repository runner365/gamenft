package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
	"github.com/gamenft/gogamenft/services"
)

func setupAPIEnv(t *testing.T) (*APIHandler, *db.DBHandle) {
	logger.InitLogger(logger.Config{Level: "info", Filename: "test.log", MaxSize: 10, MaxBackups: 1, MaxAge: 1, Compress: false})

	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := gdb.AutoMigrate(&models.User{}, &models.Item{}, &models.Order{}, &models.Transaction{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	handle := db.NewDBHandle(gdb)

	userSvc := services.NewUserService(handle, &services.JWTSettings{JWTSecret: "s", JWTExpiry: 3600})
	// ethclient and contract bindings not needed for import tests
	itemSvc := services.NewItemService(handle, (*ethclient.Client)(nil), nil)
	orderSvc := services.NewOrderService(handle)

	h := NewAPIHandler(userSvc, itemSvc, orderSvc, nil, nil, nil, nil, nil, nil)
	return h, handle
}

func setupTestCORS() cors.Config {
	return cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}
}

func TestCreateItem_ImportOnly(t *testing.T) {
	h, _ := setupAPIEnv(t)
	router := gin.New()
	h.SetupRoutes(router, setupTestCORS())

	// generate an address
	pk, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(pk.PublicKey).Hex()

	reqBody := map[string]interface{}{
		"nft_contract": addr,
		"token_id":     12345,
		"token_uri":    "",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["item"] == nil {
		t.Fatalf("expected item in response")
	}
}

func TestCreateItem_ImportAndList(t *testing.T) {
	h, _ := setupAPIEnv(t)
	router := gin.New()
	h.SetupRoutes(router, setupTestCORS())

	// use a valid address
	pk, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(pk.PublicKey).Hex()

	reqBody := map[string]interface{}{
		"nft_contract":  addr,
		"token_id":      "54321",
		"token_uri":     "",
		"owner_address": addr,
		"list_price":    "1000000000000000000",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	item, ok := resp["item"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected item object in response")
	}
	// verify list price set
	if item["list_price"] == nil {
		t.Fatalf("expected list_price to be set when list_price provided")
	}
}
