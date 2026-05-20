package services

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

// interfaces are defined in services/interfaces.go

// ItemService handles item related operations.
type ItemService struct {
	db        *db.DBHandle
	ethClient *ethclient.Client
	market    MarketI
	cache     sync.Map
	httpCli   *http.Client
	cacheSvc  *CacheService
}

// NewItemService constructs an ItemService.
func NewItemService(dbh *db.DBHandle, eth *ethclient.Client, market MarketI) *ItemService {
	return &ItemService{
		db:        dbh,
		ethClient: eth,
		market:    market,
		httpCli:   &http.Client{Timeout: 5 * time.Second},
	}
}

// SetCache sets an optional CacheService used to accelerate DB reads.
func (s *ItemService) SetCache(c *CacheService) {
	s.cacheSvc = c
}

// GetListedItems returns paged listed items and verifies on-chain state.
func (s *ItemService) GetListedItems(ctx context.Context, page, size int) ([]*models.Item, int64, error) {
	var items []*models.Item
	var total int64
	var err error

	if s.cacheSvc != nil {
		items, total, err = s.cacheSvc.GetListedItems(ctx, page, size)
		if err != nil {
			logger.LogErrorf("GetListedItems: failed to get listed items from cache: %v", err)
			return nil, 0, err
		}
		logger.LogInfof("GetListedItems: cached %d items", len(items))
	} else {
		items, total, err = s.db.GetListedItems(page, size)
		if err != nil {
			logger.LogErrorf("GetListedItems: failed to get listed items from db: %v", err)
			return nil, 0, err
		}
		logger.LogInfof("GetListedItems: db returned %d items", len(items))
	}

	// verify against chain
	for _, item := range items {
		if s.market == nil {
			continue
		}

		listing, err := s.market.GetListing(&bind.CallOpts{Context: ctx}, common.HexToAddress(item.NFTContractAddress), big.NewInt(item.TokenID))
		if err == nil && listing.Active {
			if listing.Price != nil {
				item.ListPrice = listing.Price.String()
			}
		} else {
			item.IsListed = false
			_ = s.db.CreateOrUpdateItem(item)
			if s.cacheSvc != nil {
				_ = s.cacheSvc.InvalidateAllListedItems(context.Background())
				_ = s.cacheSvc.InvalidateUser(context.Background(), item.OwnerAddress)
			}
		}
	}
	logger.LogInfof("GetListedItems: verified %d items: %v", len(items), items)
	return items, total, nil
}

// fetchTokenMetadata tries to GET the tokenURI and decode JSON into Metadata.
func (s *ItemService) fetchTokenMetadata(ctx context.Context, tokenURI string) (*models.Metadata, error) {
	if tokenURI == "" {
		logger.LogInfof("fetchTokenMetadata: empty tokenURI")
		return &models.Metadata{}, nil
	}

	// handle ipfs:// and data: URIs naive replacements
	if strings.HasPrefix(tokenURI, "ipfs://") {
		tokenURI = strings.Replace(tokenURI, "ipfs://", "https://ipfs.io/ipfs/", 1)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURI, nil)
	if err != nil {
		logger.LogErrorf("fetchTokenMetadata: failed to create request: %v", err)
		return &models.Metadata{}, err
	}

	resp, err := s.httpCli.Do(req)
	if err != nil {
		logger.LogErrorf("fetchTokenMetadata: failed to fetch tokenURI=%s err=%v", tokenURI, err)
		return &models.Metadata{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogErrorf("fetchTokenMetadata: failed to read response body for tokenURI=%s err=%v", tokenURI, err)
		return &models.Metadata{}, err
	}

	var md models.Metadata
	if err := json.Unmarshal(data, &md); err != nil {
		logger.LogErrorf("fetchTokenMetadata: failed to decode JSON for tokenURI=%s err=%v", tokenURI, err)
		return &models.Metadata{}, err
	}
	logger.LogInfof("fetchTokenMetadata: fetched metadata for tokenURI=%s metadata=%+v", tokenURI, md)
	return &md, nil
}

// GetItemByID returns an item by primary key.
func (s *ItemService) GetItemByID(ctx context.Context, id uint) (*models.Item, error) {
	return s.db.GetItemByID(id)
}

// ListItem marks an item as listed with the given price (string decimal in smallest unit).
func (s *ItemService) ListItem(ctx context.Context, itemID uint, price *big.Int) error {
	item, err := s.db.GetItemByID(itemID)
	if err != nil {
		return err
	}
	item.IsListed = true
	item.ListPrice = price.String()
	now := time.Now()
	item.ListedAt = &now
	logger.LogInfof("ListItem: marked item %d as listed with price %s", itemID, price.String())
	return s.db.CreateOrUpdateItem(item)
}

// GetMyItems returns items for a given owner address.
func (s *ItemService) GetMyItems(ctx context.Context, owner string) ([]*models.Item, error) {
	return s.db.GetItemsByOwner(owner)
}

// CreateOrImportItem creates an item record if it doesn't exist, or updates
// the existing record. nftContract must be a hex address (normalized expected).
func (s *ItemService) CreateOrImportItem(ctx context.Context, nftContract string, tokenID int64, tokenURI string, owner string) (*models.Item, error) {
	// normalize contract and owner
	contract := strings.ToLower(nftContract)
	ownerAddr := strings.ToLower(owner)

	// check existing by token+contract
	var existing *models.Item
	if it, err := s.db.GetItemByTokenAndContract(tokenID, contract); err == nil {
		existing = it
	}

	metadata, _ := s.fetchTokenMetadata(ctx, tokenURI)

	if existing == nil {
		item := &models.Item{
			TokenID:            tokenID,
			NFTContractAddress: contract,
			OwnerAddress:       ownerAddr,
			TokenURI:           tokenURI,
			Name:               metadata.Name,
			Description:        metadata.Description,
			ImageURL:           metadata.Image,
			IsListed:           false,
		}
		itemJsonData, _ := json.Marshal(item)
		logger.LogInfof("CreateOrImportItem: creating new item with data %s", string(itemJsonData))
		if err := s.db.CreateOrUpdateItem(item); err != nil {
			return nil, err
		}
		if s.cacheSvc != nil {
			_ = s.cacheSvc.InvalidateAllListedItems(context.Background())
			logger.LogInfof("CreateOrImportItem: invalidated all listed items")
			_ = s.cacheSvc.InvalidateUser(context.Background(), ownerAddr)
			logger.LogInfof("CreateOrImportItem: invalidated cache for user %s", ownerAddr)
		}
		logger.LogInfof("CreateOrImportItem: created new item %d for owner %s", item.TokenID, ownerAddr)
		return item, nil
	}

	// update fields if changed
	existing.TokenURI = tokenURI
	existing.Name = metadata.Name
	existing.Description = metadata.Description
	existing.ImageURL = metadata.Image
	existing.OwnerAddress = ownerAddr
	itemJsonData, _ := json.Marshal(existing)
	logger.LogInfof("CreateOrImportItem: updating existing item %d with data %s", existing.ID, string(itemJsonData))
	if err := s.db.CreateOrUpdateItem(existing); err != nil {
		return nil, err
	}
	if s.cacheSvc != nil {
		_ = s.cacheSvc.InvalidateAllListedItems(context.Background())
		logger.LogInfof("CreateOrImportItem: invalidated all listed items")
		_ = s.cacheSvc.InvalidateUser(context.Background(), ownerAddr)
		logger.LogInfof("CreateOrImportItem: invalidated cache for user %s", ownerAddr)
	}
	return existing, nil
}
