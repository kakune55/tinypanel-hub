package weather

import (
	"context"
	"testing"
	"time"

	"tinypanel-hub/internal/domain"
)

type countingProvider struct {
	count int
}

func (p *countingProvider) Current(context.Context) (domain.Weather, error) {
	p.count++
	return domain.Weather{
		Location:    "101020100",
		Condition:   "晴",
		Temperature: float64(p.count),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func TestCacheCurrentCachesUntilTTLExpires(t *testing.T) {
	provider := &countingProvider{}
	cache := NewCache(provider, 10*time.Minute)

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	cache.clock = func() time.Time { return now }

	first, err := cache.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if provider.count != 1 {
		t.Fatalf("provider count = %d, want 1", provider.count)
	}
	if second.Temperature != first.Temperature {
		t.Fatalf("cached temperature = %v, want %v", second.Temperature, first.Temperature)
	}

	now = now.Add(10*time.Minute + time.Second)
	third, err := cache.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if provider.count != 2 {
		t.Fatalf("provider count = %d, want 2", provider.count)
	}
	if third.Temperature == first.Temperature {
		t.Fatalf("temperature was not refreshed: %v", third.Temperature)
	}
}
