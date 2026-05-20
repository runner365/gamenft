// handler.go
package api

import (
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/services"
)

type APIHandler struct {
	userService       *services.UserService
	itemService       *services.ItemService
	orderService      *services.OrderService
	playerItemService *services.PlayerItemService
	tokenService      *services.TokenService
	ethService        *services.EthService
	taskService       *services.TaskService
	wsHub             *services.WsHub
	contractAddresses map[string]string
}

func NewAPIHandler(us *services.UserService, is *services.ItemService, os *services.OrderService, pis *services.PlayerItemService, ts *services.TokenService, es *services.EthService, taskSvc *services.TaskService, hub *services.WsHub, addresses map[string]string) *APIHandler {
	return &APIHandler{userService: us, itemService: is, orderService: os, playerItemService: pis, tokenService: ts, ethService: es, taskService: taskSvc, wsHub: hub, contractAddresses: addresses}
}

// 初始化路由
func (h *APIHandler) SetupRoutes(r *gin.Engine, corsCfg cors.Config) {
	r.Use(cors.New(corsCfg))

	public := r.Group("/api/v1")
	{
		public.POST("/items", h.CreateItem)
		public.GET("/auth/challenge", h.GetLoginChallenge)
		public.POST("/auth/login", h.Login)
		public.GET("/tokens/info", h.GetTokenInfo) //get game token info to public
		public.GET("/users/:address", h.GetUserProfile)
		public.GET("/items", h.GetItems)
		public.GET("/items/:id", h.GetItemDetail)
		public.GET("/contracts", h.GetContracts)
	}

	auth := r.Group("/api/v1")
	auth.Use(h.AuthMiddleware())
	{
		auth.GET("/profile", h.GetProfile)
		auth.GET("/my-items", h.GetMyItems)
		auth.POST("/items/:id/list", h.ListItem)
		auth.POST("/items/:id/buy", h.BuyItem)
		auth.GET("/orders", h.GetMyOrders)
		auth.POST("/tokens/purchase", h.RecordTokenPurchase)
		auth.POST("/rewards", h.EarnReward)
		auth.GET("/rewards", h.GetInventory)
		auth.POST("/rewards/use", h.UseItem)
		auth.POST("/rewards/sell", h.SellItem)
		auth.GET("/tasks/:id", h.GetTaskStatus)
	}
	r.GET("/ws", h.wsHub.HandleUpgrade(func(token string) (string, error) {
		user, err := h.userService.ValidateToken(token)
		if err != nil || user == nil {
			return "", err
		}
		return user.EthAddress, nil
	}))
	logger.LogInfof("API routes set up successfully")
}

// GetLoginChallenge returns the challenge message to be signed by the client.
func (h *APIHandler) GetLoginChallenge(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}
	logger.LogInfof("Received login challenge request for address: %s", address)
	// validate and normalize
	norm, ok := validateAndNormalizeAddress(address)
	if !ok {
		logger.LogErrorf("Invalid Ethereum address: %s", address)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ethereum address"})
		return
	}
	logger.LogInfof("Generated login challenge for address: %s", norm)
	msg, err := h.userService.GenerateLoginChallenge(c.Request.Context(), norm)
	if err != nil {
		logger.LogErrorf("Failed to generate login challenge for address: %s, error: %v", norm, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("Login challenge message: %s", msg)
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// Login performs signature verification and returns JWT token.
func (h *APIHandler) Login(c *gin.Context) {
	var req struct {
		Address   string `json:"address" binding:"required"`
		Message   string `json:"message" binding:"required"`
		Signature string `json:"signature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("Login: bind JSON failed err=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("Login: address=%s", req.Address)
	// validate and normalize address
	norm, ok := validateAndNormalizeAddress(req.Address)
	if !ok {
		logger.LogErrorf("Login: invalid address address=%s", req.Address)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ethereum address"})
		return
	}
	user, token, err := h.userService.Authenticate(norm, req.Message, req.Signature)
	if err != nil {
		logger.LogErrorf("Login: auth failed address=%s err=%v", norm, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("Login: success address=%s", norm)
	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

// validateAndNormalizeAddress ensures the input is a hex ETH address and
// returns a lower-cased, normalized form (0x....) suitable for DB lookups.
func validateAndNormalizeAddress(addr string) (string, bool) {
	if !common.IsHexAddress(addr) {
		return "", false
	}
	return strings.ToLower(common.HexToAddress(addr).Hex()), true
}

// GetItems returns paged listed items.
func (h *APIHandler) GetItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	logger.LogInfof("GetItems: page=%d size=%d", page, size)
	items, total, err := h.itemService.GetListedItems(c.Request.Context(), page, size)
	if err != nil {
		logger.LogErrorf("GetItems: failed page=%d size=%d err=%v", page, size, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "size": size})
}

// GetItemDetail returns single item by primary ID.
func (h *APIHandler) GetItemDetail(c *gin.Context) {
	idStr := c.Param("id")
	logger.LogInfof("GetItemDetail: id=%s", idStr)
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.LogErrorf("GetItemDetail: invalid id id=%s err=%v", idStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	item, err := h.itemService.GetItemByID(c.Request.Context(), uint(id64))
	if err != nil {
		logger.LogErrorf("GetItemDetail: not found id=%d err=%v", id64, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// AuthMiddleware validates JWT and sets address in context.
func (h *APIHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}
		token := parts[1]
		user, err := h.userService.ValidateToken(token)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("address", user.EthAddress)
		c.Next()
	}
}

// GetProfile returns the authenticated user's profile and stats.
func (h *APIHandler) GetProfile(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)
	logger.LogInfof("GetProfile: user=%s", addr)
	user, allItems, listedItems, playerItems, err := h.userService.GetUserProfile(addr)
	if err != nil {
		logger.LogErrorf("GetProfile: user not found user=%s err=%v", addr, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"owned_items":  allItems,
		"listed_items": listedItems,
		"player_items": playerItems,
		"stats":        gin.H{"total_items": len(allItems), "listed_items": len(listedItems), "player_items": len(playerItems)},
	})
}

// GetUserProfile returns a public user profile by address.
func (h *APIHandler) GetUserProfile(c *gin.Context) {
	address := c.Param("address")
	logger.LogInfof("GetUserProfile: address=%s", address)
	norm, ok := validateAndNormalizeAddress(address)
	if !ok {
		logger.LogErrorf("GetUserProfile: invalid address address=%s", address)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ethereum address"})
		return
	}
	user, allItems, listedItems, playerItems, err := h.userService.GetUserProfile(norm)
	if err != nil {
		logger.LogErrorf("GetUserProfile: user not found address=%s err=%v", norm, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"owned_items":  allItems,
		"listed_items": listedItems,
		"player_items": playerItems,
		"stats":        gin.H{"total_items": len(allItems), "listed_items": len(listedItems), "player_items": len(playerItems)},
	})
}

// GetMyItems returns items owned by authenticated user.
func (h *APIHandler) GetMyItems(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)
	logger.LogInfof("GetMyItems: user=%s", addr)
	items, err := h.itemService.GetMyItems(c.Request.Context(), addr)
	if err != nil {
		logger.LogErrorf("GetMyItems: failed user=%s err=%v", addr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// ListItem marks an item as listed (DB only).
func (h *APIHandler) ListItem(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)
	idStr := c.Param("id")

	var req struct {
		Price string `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("ListItem: bind JSON failed user=%s id=%s err=%v", addr, idStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("ListItem: user=%s id=%s price=%s", addr, idStr, req.Price)

	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.LogErrorf("ListItem: invalid id user=%s id=%s err=%v", addr, idStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	price, ok := new(big.Int).SetString(req.Price, 10)
	if !ok {
		logger.LogErrorf("ListItem: invalid price user=%s id=%d price=%s", addr, id64, req.Price)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid price"})
		return
	}
	item, err := h.itemService.GetItemByID(c.Request.Context(), uint(id64))
	if err != nil {
		logger.LogErrorf("ListItem: item not found user=%s id=%d err=%v", addr, id64, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}
	if !strings.EqualFold(item.OwnerAddress, addr) {
		logger.LogErrorf("ListItem: not owner user=%s id=%d owner=%s", addr, id64, item.OwnerAddress)
		c.JSON(http.StatusForbidden, gin.H{"error": "not owner"})
		return
	}
	if err := h.itemService.ListItem(c.Request.Context(), uint(id64), price); err != nil {
		logger.LogErrorf("ListItem: failed user=%s id=%d price=%s err=%v", addr, id64, price.String(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("ListItem: success user=%s id=%d price=%s", addr, id64, price.String())
	c.JSON(http.StatusOK, gin.H{"status": "listed"})
}

// BuyItem records a completed on-chain purchase.
// The on-chain transaction has already been executed by the client;
// the ItemBought event listener keeps the DB in sync.
// This endpoint records the order for history tracking.
func (h *APIHandler) BuyItem(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)
	idStr := c.Param("id")

	var req struct {
		TxHash string `json:"tx_hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("BuyItem: bind JSON failed buyer=%s id=%s err=%v", addr, idStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.LogInfof("BuyItem: request buyer=%s id=%s tx=%s", addr, idStr, req.TxHash)

	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.LogErrorf("BuyItem: invalid id buyer=%s id=%s err=%v", addr, idStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	item, err := h.itemService.GetItemByID(c.Request.Context(), uint(id64))
	if err != nil {
		logger.LogErrorf("BuyItem: item not found buyer=%s id=%d err=%v", addr, id64, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	order, err := h.orderService.RecordBuyOrder(c.Request.Context(), addr, item.ID, req.TxHash)
	if err != nil {
		logger.LogErrorf("BuyItem: record order failed buyer=%s id=%d tx=%s err=%v", addr, id64, req.TxHash, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.LogInfof("BuyItem: success buyer=%s id=%d orderID=%s tx=%s", addr, id64, order.OrderID, order.TxHash)
	c.JSON(http.StatusOK, gin.H{"order": order})
}

// CreateItem imports or registers an on-chain token into our DB.
// Request JSON:
//
//	{
//	  "nft_contract": "0x...",
//	  "token_id": 12345,
//	  "token_uri": "ipfs://...",
//	  "owner_address": "0x...", // optional
//	  "list_price": "1000000000000000000" // optional
//	}
func (h *APIHandler) CreateItem(c *gin.Context) {
	var req struct {
		NFTContract string      `json:"nft_contract" binding:"required"`
		TokenID     interface{} `json:"token_id" binding:"required"`
		TokenURI    string      `json:"token_uri"`
		Owner       string      `json:"owner_address"`
		ListPrice   string      `json:"list_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("CreateItem: bind JSON failed err=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("CreateItem: contract=%s token_id=%v", req.NFTContract, req.TokenID)

	// validate contract
	contract, ok := validateAndNormalizeAddress(req.NFTContract)
	if !ok {
		logger.LogErrorf("CreateItem: invalid nft_contract contract=%s", req.NFTContract)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nft_contract address"})
		return
	}

	// parse token id (allow number or string)
	var tokenID int64
	switch v := req.TokenID.(type) {
	case float64:
		tokenID = int64(v)
	case string:
		if t, ok := new(big.Int).SetString(v, 10); ok {
			tokenID = t.Int64()
		} else {
			logger.LogErrorf("CreateItem: invalid token_id contract=%s token_id=%v", contract, v)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
	default:
		logger.LogErrorf("CreateItem: invalid token_id type contract=%s token_id=%v", contract, req.TokenID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id type"})
		return
	}

	owner := req.Owner
	if owner == "" {
		owner = ""
	} else {
		if o, ok := validateAndNormalizeAddress(owner); ok {
			owner = o
		} else {
			logger.LogErrorf("CreateItem: invalid owner_address contract=%s tokenID=%d owner=%s", contract, tokenID, req.Owner)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner_address"})
			return
		}
	}

	item, err := h.itemService.CreateOrImportItem(c.Request.Context(), contract, tokenID, req.TokenURI, owner)
	if err != nil {
		logger.LogErrorf("CreateItem: create failed contract=%s tokenID=%d err=%v", contract, tokenID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// optionally list
	if req.ListPrice != "" {
		price, ok := new(big.Int).SetString(req.ListPrice, 10)
		if !ok {
			logger.LogErrorf("CreateItem: invalid list_price contract=%s tokenID=%d price=%s", contract, tokenID, req.ListPrice)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list_price"})
			return
		}
		if err := h.itemService.ListItem(c.Request.Context(), item.ID, price); err != nil {
			logger.LogErrorf("CreateItem: list failed id=%d price=%s err=%v", item.ID, price.String(), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	logger.LogInfof("CreateItem: success id=%d contract=%s tokenID=%d", item.ID, contract, tokenID)
	c.JSON(http.StatusCreated, gin.H{"item": item})
}

// GetMyOrders returns user's orders.
func (h *APIHandler) GetMyOrders(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)
	logger.LogInfof("GetMyOrders: user=%s", addr)
	orders, err := h.orderService.GetOrdersByUser(c.Request.Context(), addr)
	if err != nil {
		logger.LogErrorf("GetMyOrders: failed user=%s err=%v", addr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

// EarnReward enqueues an async mint task and returns immediately.
// The event listener picks up TransferSingle and writes to DB.
func (h *APIHandler) EarnReward(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)

	var req struct {
		ItemType string `json:"item_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("EarnReward: bind JSON failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("EarnReward: request received user=%s item=%s", addr, req.ItemType)

	tokenID, ok := itemTypeToTokenID(req.ItemType)
	if !ok {
		logger.LogErrorf("EarnReward: invalid item type user=%s item=%s", addr, req.ItemType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item type"})
		return
	}

	// Validate user and item type before enqueuing.
	if err := h.playerItemService.ValidateReward(c.Request.Context(), addr, req.ItemType); err != nil {
		logger.LogErrorf("EarnReward: validation failed user=%s item=%s err=%v", addr, req.ItemType, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.EnqueueMint(addr, req.ItemType, tokenID)
	if err != nil {
		logger.LogErrorf("EarnReward: enqueue failed user=%s item=%s err=%v", addr, req.ItemType, err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	logger.LogInfof("EarnReward: task enqueued user=%s item=%s task_id=%s", addr, req.ItemType, task.TaskID)
	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "task_id": task.TaskID})
}

// GetTaskStatus returns the current status of an async task.
func (h *APIHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	logger.LogInfof("GetTaskStatus: task_id=%s", taskID)
	task, err := h.taskService.GetTask(taskID)
	if err != nil {
		logger.LogErrorf("GetTaskStatus: not found task_id=%s err=%v", taskID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// GetInventory returns the authenticated user's reward items.
func (h *APIHandler) GetInventory(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)
	logger.LogInfof("GetInventory: user=%s", addr)

	items, err := h.playerItemService.GetInventory(c.Request.Context(), addr)
	if err != nil {
		logger.LogErrorf("GetInventory: failed user=%s err=%v", addr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	inventory := map[string]int{"knife": 0, "pistol": 0, "bomb": 0}
	for _, item := range items {
		inventory[item.ItemType] = item.Quantity
	}
	c.JSON(http.StatusOK, gin.H{"inventory": inventory})
}

// UseItem consumes one reward item for the authenticated user.
func (h *APIHandler) UseItem(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)

	var req struct {
		ItemType string `json:"item_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("UseItem: bind JSON failed user=%s err=%v", addr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("UseItem: user=%s item=%s", addr, req.ItemType)

	if err := h.playerItemService.UseItem(c.Request.Context(), addr, req.ItemType); err != nil {
		logger.LogErrorf("UseItem: failed user=%s item=%s err=%v", addr, req.ItemType, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("UseItem: success user=%s item=%s", addr, req.ItemType)
	c.JSON(http.StatusOK, gin.H{"status": "consumed"})
}

// SellItem validates the sell request. The actual DB update is handled by the
// ItemListed event listener after the on-chain listERC1155 transaction confirms.
func (h *APIHandler) SellItem(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)

	var req struct {
		ItemType string `json:"item_type" binding:"required"`
		Amount   int    `json:"amount" binding:"required"`
		Price    string `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("SellItem: bind JSON failed user=%s err=%v", addr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("SellItem: user=%s item=%s amount=%d price=%s", addr, req.ItemType, req.Amount, req.Price)

	if req.Amount <= 0 {
		logger.LogErrorf("SellItem: invalid amount user=%s item=%s amount=%d", addr, req.ItemType, req.Amount)
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be > 0"})
		return
	}

	if err := h.playerItemService.ValidateSell(c.Request.Context(), addr, req.ItemType, req.Amount); err != nil {
		logger.LogErrorf("SellItem: validation failed user=%s item=%s amount=%d err=%v", addr, req.ItemType, req.Amount, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("SellItem: validated user=%s item=%s amount=%d price=%s", addr, req.ItemType, req.Amount, req.Price)
	c.JSON(http.StatusOK, gin.H{"status": "validated"})
}

// GetTokenInfo returns GMTK token metadata (contract address, symbol, decimals).
func (h *APIHandler) GetTokenInfo(c *gin.Context) {
	info, err := h.tokenService.GetTokenInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("GetTokenInfo: %v", info)
	c.JSON(http.StatusOK, info)
}

// RecordTokenPurchase records a completed on-chain token purchase.
func (h *APIHandler) RecordTokenPurchase(c *gin.Context) {
	addrI, _ := c.Get("address")
	addr := addrI.(string)

	var req struct {
		TxHash      string `json:"tx_hash" binding:"required"`
		EthAmount   string `json:"eth_amount" binding:"required"`
		TokenAmount string `json:"token_amount" binding:"required"`
		Rate        string `json:"rate" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogErrorf("RecordTokenPurchase: bind JSON failed user=%s err=%v", addr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("RecordTokenPurchase: user=%s tx=%s eth=%s token=%s rate=%s", addr, req.TxHash, req.EthAmount, req.TokenAmount, req.Rate)

	purchase, err := h.tokenService.RecordPurchase(c.Request.Context(), addr, req.TxHash, req.EthAmount, req.TokenAmount, req.Rate)
	if err != nil {
		logger.LogErrorf("RecordTokenPurchase: failed user=%s tx=%s err=%v", addr, req.TxHash, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.LogInfof("RecordTokenPurchase: success user=%s tx=%s", addr, req.TxHash)
	c.JSON(http.StatusCreated, gin.H{"purchase": purchase})
}

// GetContracts returns all configured contract addresses as a JSON map.
func (h *APIHandler) GetContracts(c *gin.Context) {
	c.JSON(http.StatusOK, h.contractAddresses)
}

// Matches GameItems.sol: KNIFE=1, PISTOL=2, BOMB=3
func itemTypeToTokenID(itemType string) (int64, bool) {
	switch itemType {
	case "knife":
		return 1, true
	case "pistol":
		return 2, true
	case "bomb":
		return 3, true
	default:
		return 0, false
	}
}
