package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/gamenft/gogamenft/api"
	"github.com/gamenft/gogamenft/config"
	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
	"github.com/gamenft/gogamenft/services"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	cfgPath = flag.String("c", "", "path to config yaml")
)

func init() {
	flag.Parse()
}

func main() {
	// load config (will search default locations when cfgPath is empty)
	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// init logger
	logger.InitLogger(logger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.File,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	})
	logger.LogInfof("config loaded successfully: %+v", cfg)
	// open database - use Postgres as configured
	dsn := cfg.Database.GetDSN()
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	if err != nil {
		logger.LogErrorf("failed to open db: %v", err)
		fmt.Fprintf(os.Stderr, "failed to open db: %v\n", err)
		os.Exit(1)
	}

	// AutoMigrate adds missing columns; safe to run on every startup
	// Use ordered slice to respect FK dependency order: users → items → orders → transactions
	models := []interface{}{
			&models.User{},
			&models.Item{},
			&models.Order{},
			&models.Transaction{},
			&models.PlayerItem{},
			&models.TokenPurchase{},
		&models.Task{},
			&models.ProcessedEvent{},
		}
		for _, m := range models {
			if err := gdb.AutoMigrate(m); err != nil {
				logger.LogErrorf("failed to auto-migrate %T: %v", m, err)
				fmt.Fprintf(os.Stderr, "failed to auto-migrate %T: %v\n", m, err)
				os.Exit(1)
			}
			logger.LogInfof("auto-migrated %T", m)
		}

	// create DB handle and services (pass nil for chain integrations for now)
	handle := db.NewDBHandle(gdb)
	userSvc := services.NewUserService(handle, &services.JWTSettings{JWTSecret: cfg.JWT.Secret, JWTExpiry: cfg.JWT.AccessExpiry})
	itemSvc := services.NewItemService(handle, nil, nil)
	orderSvc := services.NewOrderService(handle)
	playerItemSvc := services.NewPlayerItemService(handle, cfg.Ethereum.ContractAddresses.GameItems)
	tokenSvc := services.NewTokenService(handle, cfg.Ethereum.ContractAddresses.GameToken)

	ethService, err := services.NewEthService(
		cfg.Ethereum.RPCEndpoint,
		cfg.Ethereum.GetPrivateKey(),
		cfg.Ethereum.ChainID,
		cfg.Ethereum.ContractAddresses.GameItems,
	)
	if err != nil {
		logger.LogErrorf("failed to init eth service: %v", err)
		fmt.Fprintf(os.Stderr, "failed to init eth service: %v\n", err)
		os.Exit(1)
	}

	// WebSocket hub for client notifications
	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
			logger.LogWarnf("Redis connection failed, WS cross-instance delivery disabled: %v", err)
			redisClient = nil
		} else {
			logger.LogInfof("Redis connected for WS cross-instance delivery: %s", cfg.Redis.Addr)
		}
	}
	wsHub := services.NewWsHub(redisClient)

	// CacheService accelerates repeated DB reads via Redis; skip if Redis is unavailable
	if redisClient != nil && cfg.Cache.Enabled {
		cacheSvc := services.NewCacheService(redisClient, handle, cfg.Cache.DefaultTTL)
		itemSvc.SetCache(cacheSvc)
		userSvc.SetCache(cacheSvc)
		logger.LogInfof("CacheService enabled: default_ttl=%v", cfg.Cache.DefaultTTL)
	} else {
		logger.LogInfof("CacheService disabled")
	}

	// Task queue for async on-chain operations
	taskSvc := services.NewTaskService(handle, ethService, wsHub)

	// Event listener requires WebSocket; connect separately from HTTP ethService.
	if cfg.Ethereum.WSEndpoint != "" {
		wsClient, err := ethclient.Dial(cfg.Ethereum.WSEndpoint)
		if err != nil {
			logger.LogErrorf("failed to dial WS endpoint, event listener disabled: %v", err)
		} else {
			go services.ListenMintEvents(context.Background(), wsClient, cfg.Ethereum.ContractAddresses.GameItems, handle, taskSvc)
			go services.ListenMarketplaceEvents(context.Background(), wsClient, cfg.Ethereum.ContractAddresses.Marketplace, cfg.Ethereum.ContractAddresses.GameItems, handle, wsHub)
			go services.ListenItemBoughtEvents(context.Background(), wsClient, cfg.Ethereum.ContractAddresses.Marketplace, cfg.Ethereum.ContractAddresses.GameItems, handle, wsHub)
		}
	}

	contractAddresses := map[string]string{
		"game_token":             cfg.Ethereum.ContractAddresses.GameToken,
		"game_items":             cfg.Ethereum.ContractAddresses.GameItems,
		"marketplace":            cfg.Ethereum.ContractAddresses.Marketplace,
		"uniswap_v2_factory":     cfg.Ethereum.ContractAddresses.UniswapV2Factory,
		"uniswap_v2_router":      cfg.Ethereum.ContractAddresses.UniswapV2Router,
		"weth":                   cfg.Ethereum.ContractAddresses.WETH,
		"uniswap_liquidity_setup": cfg.Ethereum.ContractAddresses.UniswapLiquiditySetup,
	}
	apiHandler := api.NewAPIHandler(userSvc, itemSvc, orderSvc, playerItemSvc, tokenSvc, ethService, taskSvc, wsHub, contractAddresses)

	// gin engine
	r := gin.New()
	r.Use(gin.Recovery())

		corsCfg := cors.Config{
			AllowOrigins:     cfg.Server.CORS.AllowedOrigins,
			AllowMethods:     cfg.Server.CORS.AllowedMethods,
			AllowHeaders:     cfg.Server.CORS.AllowedHeaders,
			AllowCredentials: cfg.Server.CORS.AllowCredentials,
			MaxAge:           time.Duration(cfg.Server.CORS.MaxAge) * time.Second,
		}

	apiHandler.SetupRoutes(r, corsCfg)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// run server
	go func() {
		logger.LogInfof("starting server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.LogErrorf("server failed: %v", err)
			os.Exit(1)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logger.LogInfof("shutting down server...")
	taskSvc.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.LogErrorf("server forced to shutdown: %v", err)
	}

	logger.Close()
}
