package cache

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

type Cache struct {
	c   *ristretto.Cache
	ttl time.Duration
}

func New(maxCost int64, ttl time.Duration) (*Cache, error) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e5,
		MaxCost:     maxCost,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}
	return &Cache{c: c, ttl: ttl}, nil
}

func (c *Cache) Get(key string) (any, bool) { return c.c.Get(key) }

func (c *Cache) Set(key string, val any) { c.c.SetWithTTL(key, val, 1, c.ttl) }

func (c *Cache) Del(key string) { c.c.Del(key) }
