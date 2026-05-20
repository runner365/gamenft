// config/config.go
package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Ethereum EthereumConfig `yaml:"ethereum"`
	JWT      JWTConfig      `yaml:"jwt"`
	Log      LogConfig      `yaml:"log"`
	Cache    CacheConfig    `yaml:"cache"`
	Security SecurityConfig `yaml:"security"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host                    string          `yaml:"host"`
	Port                    int             `yaml:"port"`
	Env                     string          `yaml:"env"`
	ReadTimeout             time.Duration   `yaml:"read_timeout"`
	WriteTimeout            time.Duration   `yaml:"write_timeout"`
	IdleTimeout             time.Duration   `yaml:"idle_timeout"`
	GracefulShutdownTimeout time.Duration   `yaml:"graceful_shutdown_timeout"`
	CORS                    CORSConfig      `yaml:"cors"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`

	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`

	LogLevel      string        `yaml:"log_level"` // silent/error/warn/info
	SlowThreshold time.Duration `yaml:"slow_threshold"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`

	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`

	// 缓存配置
	CacheTTL   time.Duration `yaml:"cache_ttl"`
	SessionTTL time.Duration `yaml:"session_ttl"`
	LockTTL    time.Duration `yaml:"lock_ttl"`
}

// EthereumConfig 以太坊配置
type EthereumConfig struct {
	RPCEndpoint string `yaml:"rpc_endpoint"`
	WSEndpoint  string `yaml:"ws_endpoint"`
	ChainID     int64  `yaml:"chain_id"`
	Network     string `yaml:"network"` // mainnet/goerli/sepolia/local

	// 合约地址
	ContractAddresses ContractAddresses `yaml:"contract_addresses"`

	// 私钥（仅用于服务账户操作，如mint NFT等）
	PrivateKey    string  `yaml:"private_key"`
	GasLimit      uint64  `yaml:"gas_limit"`
	GasMultiplier float64 `yaml:"gas_multiplier"`
	MaxGasPrice   string  `yaml:"max_gas_price"`
	MinGasPrice   string  `yaml:"min_gas_price"`

	// 重试配置
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`
	Timeout    time.Duration `yaml:"timeout"`
}

// ContractAddresses 合约地址配置
type ContractAddresses struct {
	GameToken             string `yaml:"game_token"`
	GameItems             string `yaml:"game_items"`
		Marketplace           string `yaml:"marketplace"`
	UniswapV2Factory      string `yaml:"uniswap_v2_factory"`
	UniswapV2Router       string `yaml:"uniswap_v2_router"`
	WETH                  string `yaml:"weth"`
	UniswapLiquiditySetup string `yaml:"uniswap_liquidity_setup"`
}

// GetPrivateKey returns the private key, falling back to PRIVATE_KEY env var.
func (c *EthereumConfig) GetPrivateKey() string {
	if c.PrivateKey != "" {
		return c.PrivateKey
	}
	return os.Getenv("PRIVATE_KEY")
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret           string        `yaml:"secret"`
	AccessExpiry     time.Duration `yaml:"access_expiry"`
	RefreshExpiry    time.Duration `yaml:"refresh_expiry"`
	Issuer           string        `yaml:"issuer"`
	Audience         string        `yaml:"audience"`
	RefreshTokenSize int           `yaml:"refresh_token_size"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`       // debug/info/warn/error
	Format     string `yaml:"format"`      // json/text
	Output     string `yaml:"output"`      // stdout/file
	File       string `yaml:"file"`        // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 文件大小限制(MB)
	MaxBackups int    `yaml:"max_backups"` // 最大备份数
	MaxAge     int    `yaml:"max_age"`     // 保存天数
	Compress   bool   `yaml:"compress"`    // 是否压缩
}

// CacheConfig 缓存配置
type CacheConfig struct {
	DefaultTTL      time.Duration `yaml:"default_ttl"`
	UserTTL         time.Duration `yaml:"user_ttl"`
	ItemsTTL        time.Duration `yaml:"items_ttl"`
	MarketStatsTTL  time.Duration `yaml:"market_stats_ttl"`
	PopularItemsTTL time.Duration `yaml:"popular_items_ttl"`
	Enabled         bool          `yaml:"enabled"`
	Prefix          string        `yaml:"prefix"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	HTTPSOnly             bool              `yaml:"https_only"`
	TrustedProxies        []string          `yaml:"trusted_proxies"`
	ContentSecurityPolicy string            `yaml:"content_security_policy"`
	HSTS                  HSTSConfig        `yaml:"hsts"`
	CSPNonce              bool              `yaml:"csp_nonce"`
	JWTBlacklistEnabled   bool              `yaml:"jwt_blacklist_enabled"`
}

// HSTSConfig HSTS配置
type HSTSConfig struct {
	Enabled           bool `yaml:"enabled"`
	MaxAge            int  `yaml:"max_age"`
	IncludeSubdomains bool `yaml:"include_subdomains"`
	Preload           bool `yaml:"preload"`
}

// LoadConfig 从YAML文件加载配置
func LoadConfig(path string) (*Config, error) {
	// 如果路径为空，尝试从多个默认位置查找
	if path == "" {
		possiblePaths := []string{
			"config.yaml",
			"config/config.yaml",
			"/etc/gamenft/config.yaml",
			"./config.yaml",
		}

		for _, p := range possiblePaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}

		if path == "" {
			return nil, fmt.Errorf("config file not found in default locations")
		}
	}

	// 确保路径是绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 读取文件
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	config.SetDefaults()

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// SetDefaults 设置配置默认值
func (c *Config) SetDefaults() {
	// 服务器配置默认值
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Env == "" {
		c.Server.Env = "development"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 10 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 10 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}
	if c.Server.GracefulShutdownTimeout == 0 {
		c.Server.GracefulShutdownTimeout = 30 * time.Second
	}

	// CORS默认值
	if len(c.Server.CORS.AllowedMethods) == 0 {
		c.Server.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.Server.CORS.AllowedHeaders) == 0 {
		c.Server.CORS.AllowedHeaders = []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}
	}
	if c.Server.CORS.MaxAge == 0 {
		c.Server.CORS.MaxAge = 300
	}

	// 数据库默认值
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.DBName == "" {
		c.Database.DBName = "gamenft"
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}
	if c.Database.ConnMaxLifetime == 0 {
		c.Database.ConnMaxLifetime = 5 * time.Minute
	}
	if c.Database.ConnMaxIdleTime == 0 {
		c.Database.ConnMaxIdleTime = 5 * time.Minute
	}
	if c.Database.LogLevel == "" {
		c.Database.LogLevel = "warn"
	}
	if c.Database.SlowThreshold == 0 {
		c.Database.SlowThreshold = 200 * time.Millisecond
	}

	// Redis默认值
	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}
	if c.Redis.PoolSize == 0 {
		c.Redis.PoolSize = 10
	}
	if c.Redis.MinIdleConns == 0 {
		c.Redis.MinIdleConns = 2
	}
	if c.Redis.MaxRetries == 0 {
		c.Redis.MaxRetries = 3
	}
	if c.Redis.DialTimeout == 0 {
		c.Redis.DialTimeout = 5 * time.Second
	}
	if c.Redis.ReadTimeout == 0 {
		c.Redis.ReadTimeout = 3 * time.Second
	}
	if c.Redis.WriteTimeout == 0 {
		c.Redis.WriteTimeout = 3 * time.Second
	}
	if c.Redis.CacheTTL == 0 {
		c.Redis.CacheTTL = 5 * time.Minute
	}
	if c.Redis.SessionTTL == 0 {
		c.Redis.SessionTTL = 24 * time.Hour
	}
	if c.Redis.LockTTL == 0 {
		c.Redis.LockTTL = 10 * time.Second
	}

	// 以太坊默认值
	if c.Ethereum.ChainID == 0 {
		c.Ethereum.ChainID = 1 // Mainnet
	}
	if c.Ethereum.Network == "" {
		c.Ethereum.Network = "mainnet"
	}
	if c.Ethereum.GasLimit == 0 {
		c.Ethereum.GasLimit = 300000
	}
	if c.Ethereum.GasMultiplier == 0 {
		c.Ethereum.GasMultiplier = 1.2
	}
	if c.Ethereum.MaxGasPrice == "" {
		c.Ethereum.MaxGasPrice = "100000000000" // 100 Gwei
	}
	if c.Ethereum.MinGasPrice == "" {
		c.Ethereum.MinGasPrice = "1000000000" // 1 Gwei
	}
	if c.Ethereum.MaxRetries == 0 {
		c.Ethereum.MaxRetries = 3
	}
	if c.Ethereum.RetryDelay == 0 {
		c.Ethereum.RetryDelay = 1 * time.Second
	}
	if c.Ethereum.Timeout == 0 {
		c.Ethereum.Timeout = 30 * time.Second
	}

	// JWT默认值
	if c.JWT.Secret == "" {
		c.JWT.Secret = "your-secret-key-change-in-production"
	}
	if c.JWT.AccessExpiry == 0 {
		c.JWT.AccessExpiry = 24 * time.Hour
	}
	if c.JWT.RefreshExpiry == 0 {
		c.JWT.RefreshExpiry = 7 * 24 * time.Hour
	}
	if c.JWT.Issuer == "" {
		c.JWT.Issuer = "gamenft"
	}
	if c.JWT.Audience == "" {
		c.JWT.Audience = "gamenft-api"
	}
	if c.JWT.RefreshTokenSize == 0 {
		c.JWT.RefreshTokenSize = 32
	}

	// 日志默认值
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}
	if c.Log.Output == "" {
		c.Log.Output = "stdout"
	}
	if c.Log.File == "" {
		c.Log.File = "./logs/gamenft.log"
	}
	if c.Log.MaxSize == 0 {
		c.Log.MaxSize = 100 // 100MB
	}
	if c.Log.MaxBackups == 0 {
		c.Log.MaxBackups = 10
	}
	if c.Log.MaxAge == 0 {
		c.Log.MaxAge = 30 // 30天
	}

	// 缓存默认值
	if c.Cache.DefaultTTL == 0 {
		c.Cache.DefaultTTL = 5 * time.Minute
	}
	if c.Cache.UserTTL == 0 {
		c.Cache.UserTTL = 10 * time.Minute
	}
	if c.Cache.ItemsTTL == 0 {
		c.Cache.ItemsTTL = 1 * time.Minute
	}
	if c.Cache.MarketStatsTTL == 0 {
		c.Cache.MarketStatsTTL = 2 * time.Minute
	}
	if c.Cache.PopularItemsTTL == 0 {
		c.Cache.PopularItemsTTL = 30 * time.Second
	}
	if c.Cache.Prefix == "" {
		c.Cache.Prefix = "gamenft:"
	}

	// 安全默认值
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// 验证数据库配置
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// 验证以太坊配置
	if c.Ethereum.RPCEndpoint == "" {
		return fmt.Errorf("ethereum RPC endpoint is required")
	}
	if c.Ethereum.ChainID == 0 {
		return fmt.Errorf("ethereum chain ID is required")
	}

	// 验证合约地址
	if c.Ethereum.ContractAddresses.GameItems == "" {
		return fmt.Errorf("game items contract address is required")
	}

	// 验证JWT配置
	if c.JWT.Secret == "" || c.JWT.Secret == "your-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret must be set and secure in production")
	}

	return nil
}

// GetDSN 获取数据库连接DSN
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// GetRedisURL 获取Redis连接URL
func (c *RedisConfig) GetRedisURL() string {
	if c.Password != "" {
		return fmt.Sprintf("redis://:%s@%s/%d", c.Password, c.Addr, c.DB)
	}
	return fmt.Sprintf("redis://%s/%d", c.Addr, c.DB)
}

// IsProduction 判断是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// IsDevelopment 判断是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsTest 判断是否为测试环境
func (c *Config) IsTest() bool {
	return c.Server.Env == "test"
}
