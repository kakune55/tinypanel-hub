package weather

import (
	"context"
	"sync"
	"time"

	"tinypanel-hub/internal/domain"
)

type Provider interface {
	Current(context.Context) (domain.Weather, error)
}

type Cache struct {
	provider Provider
	ttl      time.Duration
	clock    func() time.Time

	mu        sync.Mutex
	value     domain.Weather
	expiresAt time.Time
}

func NewCache(provider Provider, ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Cache{
		provider: provider,
		ttl:      ttl,
		clock:    time.Now,
	}
}

func (c *Cache) Current(ctx context.Context) (domain.Weather, error) {
	now := c.clock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.expiresAt.IsZero() && now.Before(c.expiresAt) {
		return c.value, nil
	}

	value, err := c.provider.Current(ctx)
	if err != nil {
		return domain.Weather{}, err
	}
	c.value = value
	c.expiresAt = now.Add(c.ttl)
	return value, nil
}
