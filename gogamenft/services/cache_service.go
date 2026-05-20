package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/gamenft/gogamenft/db"
	"github.com/gamenft/gogamenft/logger"
	"github.com/gamenft/gogamenft/models"
)

// CacheService provides a small caching layer backed by Redis for
// frequently-read DB results (listed items, user profile with listed items).
type CacheService struct {
	redis *redis.Client
	db    *db.DBHandle
	ttl   time.Duration
}

// NewCacheService constructs a CacheService. ttl = 0 defaults to 30s.
func NewCacheService(r *redis.Client, dbh *db.DBHandle, ttl time.Duration) *CacheService {
	if ttl == 0 {
		ttl = 30 * time.Second
	}
	logger.LogInfof("CacheService: initialized ttl=%v", ttl)
	return &CacheService{redis: r, db: dbh, ttl: ttl}
}

// GetListedItems returns cached listed items (and total). It falls back to DB on miss.
func (c *CacheService) GetListedItems(ctx context.Context, page, size int) ([]*models.Item, int64, error) {
	key := fmt.Sprintf("listed_items:page:%d:size:%d", page, size)

	// try cache
	if data, err := c.redis.Get(ctx, key).Bytes(); err == nil {
		var payload struct {
			Items []*models.Item `json:"items"`
			Total int64          `json:"total"`
		}
		if json.Unmarshal(data, &payload) == nil {
			logger.LogInfof("CacheService: hit key=%s items=%d total=%d", key, len(payload.Items), payload.Total)
			return payload.Items, payload.Total, nil
		}
		logger.LogWarnf("CacheService: unmarshal failed key=%s err=%v, falling back to DB", key, err)
	}
	logger.LogInfof("CacheService: miss key=%s, reading from DB", key)

	// cache miss -> read from DB
	items, total, err := c.db.GetListedItems(page, size)
	if err != nil {
		logger.LogErrorf("CacheService: DB fallback failed key=%s err=%v", key, err)
		return nil, 0, err
	}

	// cache the result
	payload := struct {
		Items []*models.Item `json:"items"`
		Total int64          `json:"total"`
	}{Items: items, Total: total}
	if b, err := json.Marshal(payload); err == nil {
		if err := c.redis.Set(ctx, key, b, c.ttl).Err(); err != nil {
			logger.LogWarnf("CacheService: set failed key=%s err=%v", key, err)
		} else {
			logger.LogInfof("CacheService: cached key=%s items=%d total=%d ttl=%v", key, len(items), total, c.ttl)
		}
	} else {
		logger.LogWarnf("CacheService: marshal failed key=%s err=%v", key, err)
	}

	return items, total, nil
}

// GetUserWithListedItems returns cached user profile with listed items.
func (c *CacheService) GetUserWithListedItems(ctx context.Context, address string) (*models.User, error) {
	norm := strings.ToLower(address)
	key := fmt.Sprintf("user:%s", norm)

	if data, err := c.redis.Get(ctx, key).Bytes(); err == nil {
		var u models.User
		if json.Unmarshal(data, &u) == nil {
			logger.LogInfof("CacheService: hit key=%s user=%s", key, norm)
			return &u, nil
		}
		logger.LogWarnf("CacheService: unmarshal failed key=%s err=%v, falling back to DB", key, err)
	}
	logger.LogInfof("CacheService: miss key=%s, reading from DB", key)

	u, err := c.db.GetUserWithListedItems(norm)
	if err != nil {
		logger.LogErrorf("CacheService: DB fallback failed key=%s user=%s err=%v", key, norm, err)
		return nil, err
	}

	if b, err := json.Marshal(u); err == nil {
		if err := c.redis.Set(ctx, key, b, c.ttl).Err(); err != nil {
			logger.LogWarnf("CacheService: set failed key=%s err=%v", key, err)
		} else {
			logger.LogInfof("CacheService: cached key=%s user=%s ttl=%v", key, norm, c.ttl)
		}
	} else {
		logger.LogWarnf("CacheService: marshal failed key=%s err=%v", key, err)
	}

	return u, nil
}

// InvalidateListedItems invalidates cached listed items for the given page/size.
func (c *CacheService) InvalidateListedItems(ctx context.Context, page, size int) error {
	key := fmt.Sprintf("listed_items:page:%d:size:%d", page, size)
	if err := c.redis.Del(ctx, key).Err(); err != nil {
		logger.LogErrorf("CacheService: invalidate listed items failed key=%s err=%v", key, err)
		return err
	}
	logger.LogInfof("CacheService: invalidated key=%s", key)
	return nil
}

// InvalidateUser invalidates cached user profile.
func (c *CacheService) InvalidateUser(ctx context.Context, address string) error {
	key := fmt.Sprintf("user:%s", strings.ToLower(address))
	if err := c.redis.Del(ctx, key).Err(); err != nil {
		logger.LogErrorf("CacheService: invalidate user failed key=%s err=%v", key, err)
		return err
	}
	logger.LogInfof("CacheService: invalidated key=%s", key)
	return nil
}

// InvalidateAllListedItems removes any cached listed_items keys.
func (c *CacheService) InvalidateAllListedItems(ctx context.Context) error {
	iter := c.redis.Scan(ctx, 0, "listed_items:*", 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		logger.LogErrorf("CacheService: scan listed_items keys failed err=%v", err)
		return err
	}
	if len(keys) == 0 {
		logger.LogInfof("CacheService: invalidate all listed_items — no keys found")
		return nil
	}
	if err := c.redis.Del(ctx, keys...).Err(); err != nil {
		logger.LogErrorf("CacheService: invalidate all listed_items failed keys=%d err=%v", len(keys), err)
		return err
	}
	logger.LogInfof("CacheService: invalidated listed_items keys=%d", len(keys))
	return nil
}
