package vk

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUserNameCacheHit(t *testing.T) {
	var calls atomic.Int32
	cache := userNameCache{}
	load := func() (string, error) {
		calls.Add(1)
		return "Ivan Ivanov", nil
	}

	cache.get(1, load)
	got := cache.get(1, load)

	if calls.Load() != 1 || got != "Ivan Ivanov" {
		t.Fatalf("calls=%d value=%q", calls.Load(), got)
	}
}

func TestUserNameCacheRefreshesExpiredValue(t *testing.T) {
	var calls atomic.Int32
	cache := userNameCache{}
	load := func() (string, error) {
		if calls.Add(1) == 2 {
			return "new", nil
		}
		return "old", nil
	}

	cache.get(1, load)
	entry := cache.entries[1]
	entry.checkedAt = time.Now().Add(-userNameCacheTTL)
	cache.entries[1] = entry
	got := cache.get(1, load)

	if calls.Load() != 2 || got != "new" {
		t.Fatalf("calls=%d value=%q", calls.Load(), got)
	}
}

func TestUserNameCacheUsesStaleValueAfterFailure(t *testing.T) {
	cache := userNameCache{}
	cache.get(1, func() (string, error) { return "old", nil })
	entry := cache.entries[1]
	entry.checkedAt = time.Now().Add(-userNameCacheTTL)
	cache.entries[1] = entry

	var calls atomic.Int32
	load := func() (string, error) {
		calls.Add(1)
		return "", errors.New("offline")
	}
	first := cache.get(1, load)
	second := cache.get(1, load)

	if calls.Load() != 1 || first != "old" || second != "old" {
		t.Fatalf("calls=%d first=%q second=%q", calls.Load(), first, second)
	}
}

func TestUserNameCacheCoalescesConcurrentLoad(t *testing.T) {
	var calls atomic.Int32
	cache := userNameCache{}
	load := func() (string, error) {
		calls.Add(1)
		time.Sleep(time.Millisecond)
		return "Ivan", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.get(1, load)
		}()
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("calls=%d, want 1", calls.Load())
	}
}

func TestUserNameCacheEvictsOldestEntry(t *testing.T) {
	cache := userNameCache{entries: make(map[int]userNameEntry)}
	now := time.Now()
	for i := 0; i < userNameCacheLimit; i++ {
		cache.entries[i] = userNameEntry{checkedAt: now.Add(time.Duration(i) * time.Second)}
	}

	cache.get(userNameCacheLimit, func() (string, error) { return "new", nil })

	if len(cache.entries) != userNameCacheLimit {
		t.Fatalf("size=%d", len(cache.entries))
	}
	if _, ok := cache.entries[0]; ok {
		t.Fatal("oldest entry was not evicted")
	}
}
