package services

import (
	"context"
	"time"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

// TokenService handles GMTK token info and purchase recording.
type TokenService struct {
	db           *db.DBHandle
	tokenAddress string
}

func NewTokenService(dbh *db.DBHandle, addr string) *TokenService {
	return &TokenService{db: dbh, tokenAddress: addr}
}

// TokenInfoResponse is returned by GetTokenInfo.
type TokenInfoResponse struct {
	ContractAddress string `json:"contract_address"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
}

// GetTokenInfo returns GMTK token metadata. Rate is read on-chain by the frontend.
func (s *TokenService) GetTokenInfo(ctx context.Context) (*TokenInfoResponse, error) {
	return &TokenInfoResponse{
		ContractAddress: s.tokenAddress,
		Symbol:          "GMTK",
		Decimals:        18,
	}, nil
}

// RecordPurchase saves a completed on-chain purchase to the database.
func (s *TokenService) RecordPurchase(ctx context.Context, userAddress, txHash, ethAmount, tokenAmount, rate string) (*models.TokenPurchase, error) {
	purchase := &models.TokenPurchase{
		UserAddress: userAddress,
		EthAmount:   ethAmount,
		TokenAmount: tokenAmount,
		Rate:        rate,
		TxHash:      txHash,
		Status:      "completed",
		CreatedAt:   time.Now(),
	}
	logger.LogInfof("try to save token purchase to db: %v", purchase)
	if err := s.db.CreateTokenPurchase(purchase); err != nil {
		return nil, err
	}
	logger.LogInfof("Recorded token purchase: user=%s tx=%s eth=%s gmtk=%s", userAddress, txHash, ethAmount, tokenAmount)
	return purchase, nil
}
