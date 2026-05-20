package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

func setupTestDBForService(t *testing.T) (*gorm.DB, *db.DBHandle) {
	logger.InitLogger(logger.Config{Level: "info", Filename: "test.log", MaxSize: 10, MaxBackups: 1, MaxAge: 1, Compress: false})

	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	if err := gdb.AutoMigrate(&models.User{}, &models.Item{}, &models.Order{}, &models.Transaction{}, &models.PlayerItem{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	handle := db.NewDBHandle(gdb)
	return gdb, handle
}

func TestGetOrCreateUser(t *testing.T) {
	_, handle := setupTestDBForService(t)
	svc := NewUserService(handle, &JWTSettings{JWTSecret: "s", JWTExpiry: time.Hour})

	addr := "0xAbCdEf1234567890abcdef1234567890abcdef12"
	u, err := svc.GetOrCreateUser(addr)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}
	if u == nil {
		t.Fatalf("expected user, got nil")
	}
	if u.EthAddress != "0xabcdef1234567890abcdef1234567890abcdef12" {
		t.Fatalf("expected normalized address lowercase, got %s", u.EthAddress)
	}
	if u.Nonce == "" {
		t.Fatalf("expected nonce to be set")
	}

	// calling again should return existing and update last login
	u2, err := svc.GetOrCreateUser(addr)
	if err != nil {
		t.Fatalf("second GetOrCreateUser failed: %v", err)
	}
	if u2.ID != u.ID {
		t.Fatalf("expected same user id on second call")
	}
}

func TestGenerateLoginChallenge_Verify_Authenticate_Validate(t *testing.T) {
	_, handle := setupTestDBForService(t)
	svc := NewUserService(handle, &JWTSettings{JWTSecret: "secret123", JWTExpiry: time.Hour})

	// generate a random key and address
	pk, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	addr := crypto.PubkeyToAddress(pk.PublicKey).Hex()

	// create challenge (this will create user if not exists)
	msg, err := svc.GenerateLoginChallenge(context.Background(), addr)
	if err != nil {
		t.Fatalf("GenerateLoginChallenge failed: %v", err)
	}
	if msg == "" {
		t.Fatalf("expected non-empty challenge message")
	}

	// sign message
	sig, err := crypto.Sign(signHash([]byte(msg)), pk)
	if err != nil {
		t.Fatalf("failed to sign message: %v", err)
	}
	// adjust V to 27/28 as expected by SigToPub
	sig[64] += 27
	sigHex := fmt.Sprintf("0x%x", sig)

	verifiedUser, err := svc.VerifySignature(addr, msg, sigHex)
	if err != nil {
		t.Fatalf("VerifySignature returned error: %v", err)
	}
	if verifiedUser == nil {
		t.Fatalf("expected user from VerifySignature")
	}

	user, token, err := svc.Authenticate(addr, msg, sigHex)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if user == nil || token == "" {
		t.Fatalf("expected user and token from Authenticate")
	}

	// Validate token
	validatedUser, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if validatedUser == nil || validatedUser.EthAddress != user.EthAddress {
		t.Fatalf("ValidateToken returned unexpected user")
	}
}

func TestUpdateProfile(t *testing.T) {
	_, handle := setupTestDBForService(t)
	svc := NewUserService(handle, &JWTSettings{JWTSecret: "s", JWTExpiry: time.Hour})

	addr := "0xabcupdate1234567890abcdef1234567890abcdef12"
	user, err := svc.GetOrCreateUser(addr)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// update allowed fields
	updated, err := svc.UpdateProfile(user.EthAddress, map[string]interface{}{"username": "newname", "bio": "hello"})
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if updated.Username != "newname" {
		t.Fatalf("expected username updated to newname, got %s", updated.Username)
	}

	// create another user with username 'taken'
	other := &models.User{EthAddress: "0xothertaken1234567890abcdef1234567890abcd", Username: "taken", IsActive: true}
	if err := handle.CreateUser(other); err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	// attempt to set username to taken -> expect error
	_, err = svc.UpdateProfile(user.EthAddress, map[string]interface{}{"username": "taken"})
	if err == nil {
		t.Fatalf("expected error when updating username to an existing one")
	}
}

func TestGetUserProfile(t *testing.T) {
	_, handle := setupTestDBForService(t)
	svc := NewUserService(handle, &JWTSettings{JWTSecret: "s", JWTExpiry: time.Hour})

	addr := "0xstatuser1234567890abcdef1234567890abcdef12"
	if _, err := svc.GetOrCreateUser(addr); err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	user, allItems, listedItems, playerItems, err := svc.GetUserProfile(addr)
	if err != nil {
		t.Fatalf("GetUserProfile(%s) failed: %v", addr, err)
	}
	if user == nil {
		t.Fatalf("expected user, got nil")
	}
	if allItems == nil {
		t.Fatalf("expected allItems, got nil")
	}
	if listedItems == nil {
		t.Fatalf("expected listedItems, got nil")
	}
	if playerItems == nil {
		t.Fatalf("expected playerItems, got nil")
	}
}
