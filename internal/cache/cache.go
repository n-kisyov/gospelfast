package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(addr string) (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Cache{rdb: rdb, ttl: 24 * time.Hour}, nil
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}

func (c *Cache) Get(ctx context.Context, key string, dest any) (bool, error) {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, dest)
}

func (c *Cache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.ttl
	}
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Cache) DelPattern(ctx context.Context, pattern string) error {
	iter := c.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (c *Cache) ChapterKey(transShort string, bookOrder, chapter int) string {
	return fmt.Sprintf("chapter:%s:%d:%d", transShort, bookOrder, chapter)
}

func (c *Cache) SearchKey(query, transShort string, page int) string {
	return fmt.Sprintf("search:%x:%s:%d", md5Hash(query), transShort, page)
}

func (c *Cache) BooksKey(transShort string) string {
	return fmt.Sprintf("books:%s", transShort)
}

func (c *Cache) TranslationsKey() string {
	return "translations"
}

func md5Hash(s string) string {
	// Simple djb2 hash for cache key
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = ((h << 5) + h) + uint64(s[i])
	}
	return fmt.Sprintf("%x", h)
}
