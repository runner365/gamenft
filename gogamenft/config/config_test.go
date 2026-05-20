// config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestReadConfigYAML 测试读取 etc/config.yaml 配置文件
func TestReadConfigYAML(t *testing.T) {
	// 获取项目根目录
	rootDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}

	// 构建配置文件路径
	configPath := filepath.Join(rootDir, "../etc/config.yaml")

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("配置文件不存在: %s", configPath)
	}

	// 读取配置文件内容
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}

	// 解析YAML内容
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("解析YAML配置失败: %v", err)
	}

	// 验证配置内容
	validateServerConfig(t, &cfg.Server)
	validateDatabaseConfig(t, &cfg.Database)
	validateRedisConfig(t, &cfg.Redis)
	validateEthereumConfig(t, &cfg.Ethereum)
	validateJWTConfig(t, &cfg.JWT)
	validateLogConfig(t, &cfg.Log)
	validateCacheConfig(t, &cfg.Cache)
	validateSecurityConfig(t, &cfg.Security)

	t.Log("配置文件读取成功，所有配置项验证通过！")
}

// validateServerConfig 验证服务器配置
func validateServerConfig(t *testing.T, cfg *ServerConfig) {
	t.Helper()

	if cfg.Host == "" {
		t.Error("server.host 不能为空")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		t.Errorf("server.port 无效: %d", cfg.Port)
	}
	if cfg.Env == "" {
		t.Error("server.env 不能为空")
	}
	if cfg.ReadTimeout <= 0 {
		t.Error("server.read_timeout 必须大于0")
	}
	if cfg.WriteTimeout <= 0 {
		t.Error("server.write_timeout 必须大于0")
	}
	if cfg.IdleTimeout <= 0 {
		t.Error("server.idle_timeout 必须大于0")
	}
	if cfg.GracefulShutdownTimeout <= 0 {
		t.Error("server.graceful_shutdown_timeout 必须大于0")
	}

	// 验证CORS配置
	if len(cfg.CORS.AllowedOrigins) == 0 {
		t.Error("server.cors.allowed_origins 不能为空")
	}
	if len(cfg.CORS.AllowedMethods) == 0 {
		t.Error("server.cors.allowed_methods 不能为空")
	}

}

// validateDatabaseConfig 验证数据库配置
func validateDatabaseConfig(t *testing.T, cfg *DatabaseConfig) {
	t.Helper()

	if cfg.Host == "" {
		t.Error("database.host 不能为空")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		t.Errorf("database.port 无效: %d", cfg.Port)
	}
	if cfg.User == "" {
		t.Error("database.user 不能为空")
	}
	if cfg.DBName == "" {
		t.Error("database.name 不能为空")
	}
	if cfg.MaxOpenConns <= 0 {
		t.Error("database.max_open_conns 必须大于0")
	}
	if cfg.MaxIdleConns <= 0 {
		t.Error("database.max_idle_conns 必须大于0")
	}
	if cfg.ConnMaxLifetime <= 0 {
		t.Error("database.conn_max_lifetime 必须大于0")
	}
	if cfg.ConnMaxIdleTime <= 0 {
		t.Error("database.conn_max_idle_time 必须大于0")
	}
}

// validateRedisConfig 验证Redis配置
func validateRedisConfig(t *testing.T, cfg *RedisConfig) {
	t.Helper()

	if cfg.Addr == "" {
		t.Error("redis.addr 不能为空")
	}
	if cfg.DB < 0 || cfg.DB > 15 {
		t.Errorf("redis.db 无效: %d (应该在0-15之间)", cfg.DB)
	}
	if cfg.PoolSize <= 0 {
		t.Error("redis.pool_size 必须大于0")
	}
	if cfg.DialTimeout <= 0 {
		t.Error("redis.dial_timeout 必须大于0")
	}
	if cfg.ReadTimeout <= 0 {
		t.Error("redis.read_timeout 必须大于0")
	}
	if cfg.WriteTimeout <= 0 {
		t.Error("redis.write_timeout 必须大于0")
	}
}

// validateEthereumConfig 验证以太坊配置
func validateEthereumConfig(t *testing.T, cfg *EthereumConfig) {
	t.Helper()

	if cfg.RPCEndpoint == "" {
		t.Error("ethereum.rpc_endpoint 不能为空")
	}
	if cfg.WSEndpoint == "" {
		t.Error("ethereum.ws_endpoint 不能为空")
	}
	if cfg.ChainID <= 0 {
		t.Errorf("ethereum.chain_id 无效: %d", cfg.ChainID)
	}
	if cfg.Network == "" {
		t.Error("ethereum.network 不能为空")
	}
	if cfg.GasLimit <= 0 {
		t.Error("ethereum.gas_limit 必须大于0")
	}
	if cfg.GasMultiplier <= 0 {
		t.Error("ethereum.gas_multiplier 必须大于0")
	}
	if cfg.MaxRetries < 0 {
		t.Error("ethereum.max_retries 不能为负数")
	}
	if cfg.Timeout <= 0 {
		t.Error("ethereum.timeout 必须大于0")
	}
}

// validateJWTConfig 验证JWT配置
func validateJWTConfig(t *testing.T, cfg *JWTConfig) {
	t.Helper()

	if cfg.Secret == "" {
		t.Error("jwt.secret 不能为空")
	}
	if cfg.AccessExpiry <= 0 {
		t.Error("jwt.access_expiry 必须大于0")
	}
	if cfg.RefreshExpiry <= 0 {
		t.Error("jwt.refresh_expiry 必须大于0")
	}
	if cfg.Issuer == "" {
		t.Error("jwt.issuer 不能为空")
	}
	if cfg.Audience == "" {
		t.Error("jwt.audience 不能为空")
	}
	if cfg.RefreshTokenSize <= 0 {
		t.Error("jwt.refresh_token_size 必须大于0")
	}
}

// validateLogConfig 验证日志配置
func validateLogConfig(t *testing.T, cfg *LogConfig) {
	t.Helper()

	if cfg.Level == "" {
		t.Error("log.level 不能为空")
	}
	if cfg.Format == "" {
		t.Error("log.format 不能为空")
	}
	if cfg.Output == "" {
		t.Error("log.output 不能为空")
	}
	if cfg.MaxSize <= 0 {
		t.Error("log.max_size 必须大于0")
	}
	if cfg.MaxBackups < 0 {
		t.Error("log.max_backups 不能为负数")
	}
	if cfg.MaxAge < 0 {
		t.Error("log.max_age 不能为负数")
	}
}

// validateCacheConfig 验证缓存配置
func validateCacheConfig(t *testing.T, cfg *CacheConfig) {
	t.Helper()

	if cfg.DefaultTTL <= 0 {
		t.Error("cache.default_ttl 必须大于0")
	}
	if cfg.UserTTL <= 0 {
		t.Error("cache.user_ttl 必须大于0")
	}
	if cfg.ItemsTTL <= 0 {
		t.Error("cache.items_ttl 必须大于0")
	}
	if cfg.Prefix == "" {
		t.Error("cache.prefix 不能为空")
	}
}

// validateSecurityConfig 验证安全配置
func validateSecurityConfig(t *testing.T, cfg *SecurityConfig) {
	t.Helper()

	if cfg.ContentSecurityPolicy == "" {
		t.Error("security.content_security_policy 不能为空")
	}
	if len(cfg.TrustedProxies) == 0 {
		t.Error("security.trusted_proxies 不能为空")
	}
}

// TestConfigPath 测试配置文件路径是否正确
func TestConfigPath(t *testing.T) {
	// 获取项目根目录
	rootDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}

	// 测试不同的路径组合
	testCases := []struct {
		name    string
		relPath string
		wantErr bool
	}{
		{"相对路径", "../etc/config.yaml", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(rootDir, tc.relPath)
			_, err := os.Stat(path)
			if tc.wantErr && err == nil {
				t.Errorf("期望文件不存在，但文件存在: %s", path)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("期望文件存在，但文件不存在: %s, 错误: %v", path, err)
			}
		})
	}
}

// TestConfigUnmarshal 测试YAML解析是否正确
func TestConfigUnmarshal(t *testing.T) {
	testYAML := `
server:
  host: "localhost"
  port: 8080
  env: "test"
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 60s
  graceful_shutdown_timeout: 30s
`

	var cfg Config
	err := yaml.Unmarshal([]byte(testYAML), &cfg)
	if err != nil {
		t.Fatalf("解析YAML失败: %v", err)
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("期望 server.host = 'localhost', 实际 = '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("期望 server.port = 8080, 实际 = %d", cfg.Server.Port)
	}
	if cfg.Server.Env != "test" {
		t.Errorf("期望 server.env = 'test', 实际 = '%s'", cfg.Server.Env)
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Errorf("期望 server.read_timeout = 10s, 实际 = %v", cfg.Server.ReadTimeout)
	}
}
