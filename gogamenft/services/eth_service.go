package services

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/gamenft/gogamenft/bindings"
	"github.com/gamenft/gogamenft/logger"
)

// EthService handles on-chain interactions via owner wallet.
type EthService struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	chainID    *big.Int
	ownerAddr  common.Address
	gameItems  *bindings.GameItems
}

func NewEthService(rpcURL, privateKeyHex string, chainID int64, gameItemsAddr string) (*EthService, error) {
	logger.LogInfof("EthService: connecting to RPC %s", rpcURL)

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		logger.LogErrorf("EthService: ethclient dial failed rpc=%s err=%v", rpcURL, err)
		return nil, fmt.Errorf("ethclient dial: %w", err)
	}

	chainIDBig := big.NewInt(chainID)
	actualChainID, err := client.ChainID(context.Background())
	if err != nil {
		logger.LogWarnf("EthService: failed to fetch chain ID from RPC: %v, using config chainID=%d", err, chainID)
	} else if actualChainID.Cmp(chainIDBig) != 0 {
		logger.LogWarnf("EthService: RPC chainID=%s differs from config chainID=%d, using config", actualChainID.String(), chainID)
	}
	logger.LogInfof("EthService: connected to RPC %s chainID=%d", rpcURL, chainID)

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		logger.LogErrorf("EthService: invalid private key err=%v", err)
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		logger.LogErrorf("EthService: failed to cast public key to ECDSA")
		return nil, errors.New("failed to cast public key to ECDSA")
	}
	ownerAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	logger.LogInfof("EthService: owner address derived %s", ownerAddr.Hex())

	gameItems, err := bindings.NewGameItems(common.HexToAddress(gameItemsAddr), client)
	if err != nil {
		logger.LogErrorf("EthService: bind GameItems failed addr=%s err=%v", gameItemsAddr, err)
		return nil, fmt.Errorf("bind GameItems: %w", err)
	}

	logger.LogInfof("EthService: initialized owner=%s gameItems=%s chainID=%d", ownerAddr.Hex(), gameItemsAddr, chainID)
	return &EthService{
		client:     client,
		privateKey: privateKey,
		chainID:    chainIDBig,
		ownerAddr:  ownerAddr,
		gameItems:  gameItems,
	}, nil
}

// MintGameItem mints an ERC1155 token to the player.
// Returns the transaction hash.
func (s *EthService) MintGameItem(ctx context.Context, to common.Address, tokenID int64, amount int64) (string, error) {
	logger.LogInfof("EthService: MintGameItem starting to=%s tokenID=%d amount=%d owner=%s",
		to.Hex(), tokenID, amount, s.ownerAddr.Hex())

	nonce, err := s.client.PendingNonceAt(ctx, s.ownerAddr)
	if err != nil {
		logger.LogErrorf("EthService: MintGameItem nonce fetch failed owner=%s err=%v", s.ownerAddr.Hex(), err)
		return "", fmt.Errorf("nonce fetch: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(s.privateKey, s.chainID)
	if err != nil {
		logger.LogErrorf("EthService: MintGameItem create transactor failed err=%v", err)
		return "", fmt.Errorf("create transactor: %w", err)
	}
	auth.Context = ctx
	auth.Nonce = big.NewInt(int64(nonce))

	tx, err := s.gameItems.Mint(auth, to, big.NewInt(tokenID), big.NewInt(amount))
	if err != nil {
		logger.LogErrorf("EthService: MintGameItem tx failed to=%s tokenID=%d amount=%d nonce=%d err=%v",
			to.Hex(), tokenID, amount, nonce, err)
		return "", fmt.Errorf("mint tx: %w", err)
	}

	logger.LogInfof("EthService: MintGameItem tx sent tx=%s to=%s tokenID=%d amount=%d nonce=%d",
		tx.Hash().Hex(), to.Hex(), tokenID, amount, nonce)
	return tx.Hash().Hex(), nil
}

// GameItemsContract returns the bound GameItems instance for event subscriptions.
func (s *EthService) GameItemsContract() *bindings.GameItems {
	return s.gameItems
}

// Client returns the underlying ethclient.
func (s *EthService) Client() *ethclient.Client {
	return s.client
}
