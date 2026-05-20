// services/user_service.go
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v4"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

type UserService struct {
	db     *db.DBHandle
	config *JWTSettings
	jwtKey []byte
	cache  *CacheService
}

type JWTSettings struct {
	JWTSecret string
	JWTExpiry time.Duration
}

// 初始化UserService
func NewUserService(db *db.DBHandle, config *JWTSettings) *UserService {
	return &UserService{
		db:     db,
		config: config,
		jwtKey: []byte(config.JWTSecret),
	}
}

// SetCache sets an optional CacheService used to accelerate reads.
func (s *UserService) SetCache(c *CacheService) {
	s.cache = c
}

// GetOrCreateUser 获取或创建用户
func (s *UserService) GetOrCreateUser(address string) (*models.User, error) {
	// 标准化地址格式
	normalizedAddr := strings.ToLower(address)

	logger.LogInfof("get or create user for address: %s", normalizedAddr)
	// 检查用户是否存在
	user, err := s.db.GetUserByAddress(normalizedAddr)
	if err == nil {
		// 用户已存在，更新最后登录时间
		now := time.Now()
		s.db.UpdateUserLastLogin(normalizedAddr)
		user.LastLogin = now
		logger.LogInfof("User found for address: %s, updated last login time", normalizedAddr)
		return user, nil
	} else {
		// If the error is a permission issue, return early with a helpful hint
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "SQLSTATE 42501") {
			logger.LogErrorf("DB permission error when querying users for address %s: %v", normalizedAddr, err)
			logger.LogErrorf("Hint: run GRANTs as the DB owner or superuser. Example commands:\n  -- As postgres or the schema owner:\n  GRANT USAGE ON SCHEMA public TO admin1;\n  GRANT CREATE ON SCHEMA public TO admin1;\n  GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO admin1;\n  ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO admin1;\n  -- Or run migrations as the current table owner or a superuser.")
			return nil, fmt.Errorf("db permission denied when reading users table: %w", err)
		}

		logger.LogInfof("No existing user found for address: %s, creating new user", normalizedAddr)
	}

	// 用户不存在，创建新用户
	nonce, err := generateNonce()
	if err != nil {
		logger.LogErrorf("Failed to generate nonce for address: %s, error: %v", normalizedAddr, err)
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	user = &models.User{
		EthAddress: normalizedAddr,
		Username:   generateUsername(normalizedAddr),
		Nonce:      nonce,
		IsActive:   true,
		LastLogin:  time.Now(),
	}

	logger.LogInfof("CreateUser: address=%s username=%s nonce=%s", normalizedAddr, user.Username, nonce)
	if err := s.db.CreateUser(user); err != nil {
		// Detect permission denied and give actionable guidance
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "SQLSTATE 42501") {
			logger.LogErrorf("Failed to create user for address: %s, error: %v", normalizedAddr, err)
			logger.LogErrorf("DB permission denied while inserting into users. Suggest running as DB owner or executing these commands as postgres/superuser:\n  GRANT USAGE ON SCHEMA public TO admin1;\n  GRANT CREATE ON SCHEMA public TO admin1;\n  GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO admin1;\n  ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO admin1;")
			return nil, fmt.Errorf("permission denied creating user: %w", err)
		}

		logger.LogErrorf("Failed to create user for address: %s, error: %v", normalizedAddr, err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GenerateLoginChallenge 生成登录挑战
func (s *UserService) GenerateLoginChallenge(ctx context.Context, address string) (string, error) {
	// 获取或创建用户
	user, err := s.GetOrCreateUser(address)
	if err != nil {
		logger.LogErrorf("Failed to generate login challenge for address: %s, error: %v", address, err)
		return "", fmt.Errorf("failed to get/create user: %w", err)
	}

	// 生成新的nonce
	newNonce, err := generateNonce()
	if err != nil {
		logger.LogErrorf("Failed to generate nonce for address: %s, error: %v", address, err)
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 更新用户的nonce（使用标准化地址）
	if err := s.db.UpdateUserNonce(user.EthAddress, newNonce); err != nil {
		logger.LogErrorf("Failed to update user nonce for address: %s, error: %v", address, err)
		return "", fmt.Errorf("failed to update user nonce: %w", err)
	}

	// 返回签名消息
	message := fmt.Sprintf(
		"Welcome to GameNFT Marketplace!\n\n"+
			"Please sign this message to authenticate.\n\n"+
			"Address: %s\n"+
			"Nonce: %s\n"+
			"Issued at: %s",
		address,
		newNonce,
		time.Now().UTC().Format(time.RFC3339),
	)

	return message, nil
}

// VerifySignature 验证以太坊签名，验证通过时返回对应的 user 对象
func (s *UserService) VerifySignature(address, message, signature string) (*models.User, error) {
	// 获取用户当前的nonce
	user, err := s.db.GetUserByAddress(address)
	if err != nil {
		logger.LogErrorf("Failed to get user by address: %s, error: %v", address, err)
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 验证消息中是否包含正确的nonce
	if !strings.Contains(message, user.Nonce) {
		logger.LogErrorf("Invalid nonce in message for address: %s, nonce: %s, message: %s", address, user.Nonce, message)
		return nil, errors.New("invalid nonce in message")
	}

	// 解码签名
	sigBytes := common.FromHex(signature)
	if len(sigBytes) != 65 {
		logger.LogErrorf("Invalid signature length for address: %s, expected 65 bytes, got %d bytes", address, len(sigBytes))
		return nil, errors.New("invalid signature length")
	}

	// normalize recovery id: accept both {0,1} and {27,28}
	if sigBytes[64] >= 27 {
		sigBytes[64] = sigBytes[64] - 27
	}

	// 验证签名
	sigPublicKey, err := crypto.SigToPub(signHash([]byte(message)), sigBytes)
	if err != nil {
		logger.LogErrorf("Signature verification failed for address: %s, error: %v", address, err)
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// 恢复地址
	recoveredAddr := crypto.PubkeyToAddress(*sigPublicKey)

	if !strings.EqualFold(recoveredAddr.Hex(), address) {
		logger.LogErrorf("Signature verification failed for address: %s, recovered address: %s, expected address: %s", address, recoveredAddr.Hex(), address)
		return nil, errors.New("invalid signature")
	}

	logger.LogInfof("Signature verified successfully for address: %s", address)
	return user, nil
}

// Authenticate 用户认证
func (s *UserService) Authenticate(address, message, signature string) (*models.User, string, error) {
	// 验证签名（签名有效时返回 user，避免后续再查数据库）
	user, err := s.VerifySignature(address, message, signature)
	if err != nil {
		logger.LogErrorf("Signature verification failed for address: %s, error: %v", address, err)
		return nil, "", fmt.Errorf("signature verification failed: %w", err)
	}

	// 生成新的nonce用于下次登录
	newNonce, err := generateNonce()
	if err != nil {
		logger.LogErrorf("Failed to generate new nonce for address: %s, error: %v", address, err)
		return nil, "", fmt.Errorf("failed to generate new nonce: %w", err)
	}

	// 更新用户nonce和登录时间
	user.Nonce = newNonce
	user.LastLogin = time.Now()
	if err := s.db.UpdateUser(user); err != nil {
		logger.LogErrorf("Failed to update user after authentication for address: %s, error: %v", address, err)
		return nil, "", fmt.Errorf("failed to update user: %w", err)
	}
	if s.cache != nil {
		_ = s.cache.InvalidateUser(context.Background(), address)
	}
	logger.LogInfof("User %s logged in successfully", address)

	// 生成JWT token
	token, err := s.generateJWTToken(user)
	if err != nil {
		logger.LogErrorf("Failed to generate JWT token for address: %s, error: %v", address, err)
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	logger.LogInfof("Generated JWT token for user %s, token: %s", address, token)

	return user, token, nil
}

// generateJWTToken 生成JWT Token
func (s *UserService) generateJWTToken(user *models.User) (string, error) {
	expiry := time.Now().Add(s.config.JWTExpiry)

	claims := jwt.MapClaims{
		"sub":   user.EthAddress,
		"exp":   expiry.Unix(),
		"iat":   time.Now().Unix(),
		"type":  "access",
		"roles": []string{"user"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtKey)
}

// ValidateToken 验证JWT Token
func (s *UserService) ValidateToken(tokenString string) (*models.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.LogErrorf("unexpected signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtKey, nil
	})

	if err != nil {
		logger.LogErrorf("Token parsing error: %v", err)
		return nil, fmt.Errorf("token parsing error: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		address, ok := claims["sub"].(string)
		if !ok {
			logger.LogErrorf("Invalid token claims: missing 'sub' field")
			return nil, errors.New("invalid token claims")
		}

		return s.GetUserByAddress(address)
	}

	return nil, errors.New("invalid token")
}

// GetUserByAddress 通过地址获取用户
func (s *UserService) GetUserByAddress(address string) (*models.User, error) {
	if s.cache != nil {
		user, err := s.cache.GetUserWithListedItems(context.Background(), address)
		if err != nil {
			logger.LogErrorf("Failed to get user by address: %s, error: %v", address, err)
			return nil, err
		}
		return user, nil
	}
	user, err := s.db.GetUserWithListedItems(address)
	if err != nil {
		logger.LogErrorf("Failed to get user by address: %s, error: %v", address, err)
		return nil, err
	}
	logger.LogInfof("GetUserByAddress: address=%s username=%s", address, user.Username)
	return user, nil
}

// UpdateProfile 更新用户资料
func (s *UserService) UpdateProfile(address string, updateData map[string]interface{}) (*models.User, error) {
	user, err := s.GetUserByAddress(address)
	if err != nil {
		logger.LogErrorf("Failed to get user by address: %s, error: %v", address, err)
		return nil, err
	}

	// 允许更新的字段
	allowedFields := map[string]bool{
		"username": true,
		"avatar":   true,
		"bio":      true,
		"email":    true,
	}

	// 过滤更新数据
	updates := make(map[string]interface{})
	for key, value := range updateData {
		if allowedFields[key] {
			updates[key] = value
		}
	}

	if len(updates) == 0 {
		logger.LogInfof("No updates provided for address: %s", address)
		return user, nil
	}

	// 如果更新用户名，检查是否重复
	if username, ok := updates["username"].(string); ok && username != user.Username {
		exists, err := s.db.CheckUsernameExists(username, user.ID)
		if err != nil {
			logger.LogErrorf("Failed to check username existence for username: %s, error: %v", username, err)
			return nil, err
		}
		if exists {
			logger.LogErrorf("Username %s already taken", username)
			return nil, errors.New("username already taken")
		}
	}

	// 执行更新
	if err := s.db.UpdateUserProfile(user, updates); err != nil {
		logger.LogErrorf("Failed to update user profile for address: %s, error: %v", address, err)
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.InvalidateUser(context.Background(), address)
	}

	logger.LogInfof("User profile for address: %s updated successfully", address)
	// 重新加载用户数据
	return s.GetUserByAddress(address)
}

// GetUserProfile returns user info, all owned items, listed items, and game rewards for any address.
func (s *UserService) GetUserProfile(address string) (*models.User, []*models.Item, []*models.Item, []*models.PlayerItem, error) {
	normalized := strings.ToLower(address)
	user, err := s.GetUserByAddress(normalized)
	if err != nil {
		logger.LogErrorf("Failed to get user by address: %s, error: %v", normalized, err)
		return nil, nil, nil, nil, err
	}
	allItems, err := s.db.GetItemsByOwner(normalized)
	if err != nil {
		logger.LogErrorf("Failed to get items by owner for address: %s, error: %v", normalized, err)
		return nil, nil, nil, nil, err
	}
	var listedItems []*models.Item
	for _, it := range allItems {
		if it.IsListed {
			listedItems = append(listedItems, it)
		}
	}
	if listedItems == nil {
		listedItems = []*models.Item{}
	}
	if allItems == nil {
		allItems = []*models.Item{}
	}
	playerItems, err := s.db.GetPlayerItemsByUser(normalized)
	if err != nil {
		logger.LogErrorf("Failed to get player items by user for address: %s, error: %v", normalized, err)
		return nil, nil, nil, nil, err
	}
	if playerItems == nil {
		playerItems = []*models.PlayerItem{}
	}
	logger.LogInfof("GetUserProfile: address=%s total_items=%d listed_items=%d player_items=%d", normalized, len(allItems), len(listedItems), len(playerItems))
	return user, allItems, listedItems, playerItems, nil
}

// 辅助函数
func generateNonce() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateUsername(address string) string {
	// 使用地址后6位作为默认用户名
	if len(address) >= 6 {
		return "user_" + address[len(address)-6:]
	}
	return "user_" + address
}

func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
