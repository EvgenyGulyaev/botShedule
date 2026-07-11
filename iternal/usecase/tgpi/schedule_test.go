package tgpi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestScheduleCacheHit(t *testing.T) {
	var calls atomic.Int32
	cache := scheduleCache{}
	key := scheduleKey{Type: Group, ID: 1}
	load := func() ([]Schedule, error) {
		calls.Add(1)
		return []Schedule{{Day: "Monday"}}, nil
	}

	cache.get(key, load)
	got := cache.get(key, load)

	if calls.Load() != 1 || len(got) != 1 || got[0].Day != "Monday" {
		t.Fatalf("calls=%d result=%#v", calls.Load(), got)
	}
}

func TestScheduleCacheRefreshesExpiredValue(t *testing.T) {
	var calls atomic.Int32
	cache := scheduleCache{}
	key := scheduleKey{Type: Group, ID: 1}
	load := func() ([]Schedule, error) {
		day := "old"
		if calls.Add(1) == 2 {
			day = "new"
		}
		return []Schedule{{Day: day}}, nil
	}

	cache.get(key, load)
	entry := cache.entries[key]
	entry.checkedAt = time.Now().Add(-scheduleCacheTTL)
	cache.entries[key] = entry
	got := cache.get(key, load)

	if calls.Load() != 2 || len(got) != 1 || got[0].Day != "new" {
		t.Fatalf("calls=%d result=%#v", calls.Load(), got)
	}
}

func TestScheduleCacheUsesStaleValueAfterFailure(t *testing.T) {
	cache := scheduleCache{}
	key := scheduleKey{Type: Group, ID: 1}
	cache.get(key, func() ([]Schedule, error) { return []Schedule{{Day: "old"}}, nil })
	entry := cache.entries[key]
	entry.checkedAt = time.Now().Add(-scheduleCacheTTL)
	cache.entries[key] = entry

	var calls atomic.Int32
	load := func() ([]Schedule, error) {
		calls.Add(1)
		return nil, errors.New("offline")
	}
	first := cache.get(key, load)
	second := cache.get(key, load)

	if calls.Load() != 1 || len(first) != 1 || first[0].Day != "old" || len(second) != 1 || second[0].Day != "old" {
		t.Fatalf("calls=%d first=%#v second=%#v", calls.Load(), first, second)
	}
}

func TestScheduleCacheCoalescesConcurrentLoad(t *testing.T) {
	var calls atomic.Int32
	cache := scheduleCache{}
	key := scheduleKey{Type: Group, ID: 1}
	load := func() ([]Schedule, error) {
		calls.Add(1)
		time.Sleep(time.Millisecond)
		return []Schedule{{Day: "Monday"}}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.get(key, load)
		}()
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("calls=%d, want 1", calls.Load())
	}
}

func TestScheduleCacheEvictsOldestEntry(t *testing.T) {
	cache := scheduleCache{entries: make(map[scheduleKey]scheduleEntry)}
	now := time.Now()
	for i := 0; i < scheduleCacheLimit; i++ {
		cache.entries[scheduleKey{Type: Group, ID: i}] = scheduleEntry{checkedAt: now.Add(time.Duration(i) * time.Second)}
	}

	cache.get(scheduleKey{Type: Group, ID: scheduleCacheLimit}, func() ([]Schedule, error) { return nil, nil })

	if len(cache.entries) != scheduleCacheLimit {
		t.Fatalf("size=%d", len(cache.entries))
	}
	if _, ok := cache.entries[scheduleKey{Type: Group, ID: 0}]; ok {
		t.Fatal("oldest entry was not evicted")
	}
}

func TestScheduleCacheCachesEmptyResult(t *testing.T) {
	var calls atomic.Int32
	cache := scheduleCache{}
	key := scheduleKey{Type: Group, ID: 1}
	load := func() ([]Schedule, error) {
		calls.Add(1)
		return nil, nil
	}

	cache.get(key, load)
	cache.get(key, load)

	if calls.Load() != 1 {
		t.Fatalf("calls=%d, want 1", calls.Load())
	}
}

func TestGetScheduleRejectsShortScript(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body><script>short</script></body>"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := getSchedule(doc); err == nil {
		t.Fatal("expected malformed schedule error")
	}
}

func TestFetchScheduleRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := &Client{client: server.Client(), url: server.URL + "/"}

	_, err := client.fetchSchedule(&El{Type: Group, ID: 1})
	if err == nil || !strings.Contains(err.Error(), "unexpected HTTP status 503") {
		t.Fatalf("error=%v", err)
	}
}
