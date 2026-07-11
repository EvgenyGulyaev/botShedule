# VK and Schedule Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cache VK display names for one hour and schedules for five minutes while bounding each cache to 1000 entries and retaining stale values during upstream failures.

**Architecture:** Add two private, package-local cache types with the same small policy but different value types. Each cache owns a mutex, map, TTL, stale fallback, request coalescing, and oldest-entry eviction; loaders remain in the existing VK and TGPI clients.

**Tech Stack:** Go standard library, existing VK SDK and goquery, `httptest`, Go race detector.

## Global Constraints

- VK TTL is exactly one hour.
- Schedule TTL is exactly five minutes.
- Each cache contains at most 1000 entries.
- No background goroutines, new dependencies, or distributed state.
- Preserve `getUserName(int) string` and `GetSchedule(*El) []Schedule`.
- Preserve the directory cache, 15-second HTTP timeout, and user-facing output.

---

### Task 1: Schedule cache and safe loader

**Files:**
- Modify: `iternal/usecase/tgpi/groups.go`
- Modify: `iternal/usecase/tgpi/schedule.go`
- Modify: `iternal/usecase/tgpi/scheduleFilter.go`
- Create: `iternal/usecase/tgpi/schedule_test.go`

**Interfaces:**
- Produces: `scheduleKey`, `scheduleEntry`, `scheduleCache`, `(*scheduleCache).get(scheduleKey, func() ([]Schedule, error)) []Schedule`.
- Preserves: `(*Client).GetSchedule(*El) []Schedule`.
- Changes private parser to `getSchedule(*goquery.Document) ([]Schedule, error)`.

- [ ] **Step 1: Add failing cache tests**

Create `schedule_test.go` with tests that use a loader counter:

```go
package tgpi

import (
	"errors"
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

func TestScheduleCacheUsesStaleValueAfterFailure(t *testing.T) {
	cache := scheduleCache{}
	key := scheduleKey{Type: Group, ID: 1}
	cache.get(key, func() ([]Schedule, error) { return []Schedule{{Day: "old"}}, nil })
	entry := cache.entries[key]
	entry.checkedAt = time.Now().Add(-scheduleCacheTTL)
	cache.entries[key] = entry

	got := cache.get(key, func() ([]Schedule, error) { return nil, errors.New("offline") })
	if len(got) != 1 || got[0].Day != "old" {
		t.Fatalf("result=%#v", got)
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
		go func() { defer wg.Done(); cache.get(key, load) }()
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

func TestGetScheduleRejectsShortScript(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body><script>short</script></body>"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := getSchedule(doc); err == nil {
		t.Fatal("expected malformed schedule error")
	}
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./iternal/usecase/tgpi -run 'Test(ScheduleCache|GetScheduleRejects)' -count=1`

Expected: build FAIL because `scheduleCache`, `scheduleKey`, and the error-returning parser do not exist.

- [ ] **Step 3: Implement the bounded schedule cache**

Add to `schedule.go`:

```go
const (
	scheduleCacheTTL   = 5 * time.Minute
	scheduleCacheLimit = 1000
)

type scheduleKey struct {
	Type TypeEl
	ID   int
}

type scheduleEntry struct {
	value     []Schedule
	checkedAt time.Time
}

type scheduleCache struct {
	mu      sync.Mutex
	entries map[scheduleKey]scheduleEntry
}

func (c *scheduleCache) get(key scheduleKey, load func() ([]Schedule, error)) []Schedule {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[scheduleKey]scheduleEntry)
	}
	entry, ok := c.entries[key]
	if ok && time.Since(entry.checkedAt) < scheduleCacheTTL {
		return entry.value
	}
	value, err := load()
	if err != nil {
		if ok {
			entry.checkedAt = time.Now()
			c.entries[key] = entry
			return entry.value
		}
		return nil
	}
	if !ok && len(c.entries) >= scheduleCacheLimit {
		c.evictOldest()
	}
	c.entries[key] = scheduleEntry{value: value, checkedAt: time.Now()}
	return value
}

func (c *scheduleCache) evictOldest() {
	var oldestKey scheduleKey
	var oldestTime time.Time
	for key, entry := range c.entries {
		if oldestTime.IsZero() || entry.checkedAt.Before(oldestTime) {
			oldestKey, oldestTime = key, entry.checkedAt
		}
	}
	delete(c.entries, oldestKey)
}
```

Add `sync` to the imports and `schedules scheduleCache` to `Client` in `groups.go`.

- [ ] **Step 4: Split cache lookup from schedule loading**

Implement:

```go
func (t *Client) GetSchedule(el *El) []Schedule {
	key := scheduleKey{Type: el.Type, ID: el.ID}
	return t.schedules.get(key, func() ([]Schedule, error) {
		return t.fetchSchedule(el)
	})
}

func (t *Client) fetchSchedule(el *El) ([]Schedule, error) {
	req, err := t.getReqSchedule(el)
	if err != nil { return nil, err }
	res, err := t.client.Do(req)
	if err != nil { return nil, err }
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("TGPI schedule: unexpected HTTP status %s", res.Status)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil { return nil, err }
	return getSchedule(doc)
}
```

Change `getSchedule` to return `([]Schedule, error)`, add `fmt` to its imports, and use this guarded prefix:

```go
func getSchedule(doc *goquery.Document) (result []Schedule, err error) {
	d := doc.Find("body script").Eq(0).Text()
	const prefix, suffix = 15, 18
	if len(d) < prefix+suffix {
		return nil, fmt.Errorf("invalid schedule payload: script is too short")
	}
	var s preload
	if err = json.Unmarshal([]byte(d[prefix:len(d)-suffix]), &s); err != nil {
		return nil, err
	}
	for _, day := range s.Schedule.Day {
		lessons := []Lesson{}
		for _, lesson := range day.Rec {
			teachers := make([]string, len(lesson.Teacher))
			for i, teacher := range lesson.Teacher {
				teachers[i] = teacher.Name
			}
			for _, number := range lesson.Lesson {
				lessons = append(lessons, Lesson{
					Time: number, Name: lesson.Subject, Place: lesson.Aud,
					Teacher: strings.Join(teachers, ","), Type: lesson.Type,
				})
			}
		}
		result = append(result, Schedule{Day: day.Date, Lessons: lessons})
	}
	return result, nil
}
```

- [ ] **Step 5: Verify schedule GREEN**

Run: `go test -race ./iternal/usecase/tgpi -run 'Test(ScheduleCache|GetScheduleRejects)' -count=10`

Expected: PASS, 50 test executions, no races.

- [ ] **Step 6: Commit schedule cache**

```sh
git add iternal/usecase/tgpi/groups.go iternal/usecase/tgpi/schedule.go iternal/usecase/tgpi/scheduleFilter.go iternal/usecase/tgpi/schedule_test.go
git commit -m "perf: cache schedules for five minutes"
```

---

### Task 2: VK display-name cache

**Files:**
- Modify: `iternal/adapters/vk/bot.go`
- Modify: `iternal/adapters/vk/user.go`
- Create: `iternal/adapters/vk/user_test.go`

**Interfaces:**
- Produces: private `userNameCache`, `userNameEntry`, and `(*userNameCache).get(int, func() (string, error)) string`.
- Preserves: `(*Bot).getUserName(int) string`.

- [ ] **Step 1: Add failing VK cache tests**

Create `user_test.go` with the same five behaviors as the schedule cache, using string values:

```go
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
	load := func() (string, error) { calls.Add(1); return "Ivan Ivanov", nil }
	cache.get(1, load)
	got := cache.get(1, load)
	if calls.Load() != 1 || got != "Ivan Ivanov" { t.Fatalf("calls=%d value=%q", calls.Load(), got) }
}

func TestUserNameCacheUsesStaleValueAfterFailure(t *testing.T) {
	cache := userNameCache{}
	cache.get(1, func() (string, error) { return "old", nil })
	entry := cache.entries[1]
	entry.checkedAt = time.Now().Add(-userNameCacheTTL)
	cache.entries[1] = entry
	got := cache.get(1, func() (string, error) { return "", errors.New("offline") })
	if got != "old" { t.Fatalf("value=%q", got) }
}

func TestUserNameCacheCoalescesConcurrentLoad(t *testing.T) {
	var calls atomic.Int32
	cache := userNameCache{}
	load := func() (string, error) { calls.Add(1); time.Sleep(time.Millisecond); return "Ivan", nil }
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ { wg.Add(1); go func() { defer wg.Done(); cache.get(1, load) }() }
	wg.Wait()
	if calls.Load() != 1 { t.Fatalf("calls=%d", calls.Load()) }
}

func TestUserNameCacheEvictsOldestEntry(t *testing.T) {
	cache := userNameCache{entries: make(map[int]userNameEntry)}
	now := time.Now()
	for i := 0; i < userNameCacheLimit; i++ {
		cache.entries[i] = userNameEntry{checkedAt: now.Add(time.Duration(i) * time.Second)}
	}
	cache.get(userNameCacheLimit, func() (string, error) { return "new", nil })
	if len(cache.entries) != userNameCacheLimit { t.Fatalf("size=%d", len(cache.entries)) }
	if _, ok := cache.entries[0]; ok { t.Fatal("oldest entry was not evicted") }
}
```

- [ ] **Step 2: Verify VK RED**

Run: `go test ./iternal/adapters/vk -run TestUserNameCache -count=1`

Expected: build FAIL because `userNameCache` does not exist.

- [ ] **Step 3: Implement VK cache and loader**

Add to `user.go`:

```go
const (
	userNameCacheTTL   = time.Hour
	userNameCacheLimit = 1000
)

type userNameEntry struct {
	value     string
	checkedAt time.Time
}

type userNameCache struct {
	mu      sync.Mutex
	entries map[int]userNameEntry
}

func (c *userNameCache) get(id int, load func() (string, error)) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[int]userNameEntry)
	}
	entry, ok := c.entries[id]
	if ok && time.Since(entry.checkedAt) < userNameCacheTTL {
		return entry.value
	}
	value, err := load()
	if err != nil {
		if ok {
			entry.checkedAt = time.Now()
			c.entries[id] = entry
			return entry.value
		}
		return ""
	}
	if !ok && len(c.entries) >= userNameCacheLimit {
		c.evictOldest()
	}
	c.entries[id] = userNameEntry{value: value, checkedAt: time.Now()}
	return value
}

func (c *userNameCache) evictOldest() {
	oldestID := 0
	var oldestTime time.Time
	for id, entry := range c.entries {
		if oldestTime.IsZero() || entry.checkedAt.Before(oldestTime) {
			oldestID, oldestTime = id, entry.checkedAt
		}
	}
	delete(c.entries, oldestID)
}
```

Add `users userNameCache` to `Bot`. Add `errors`, `sync`, and `time` imports to `user.go`, then refactor the loader:

```go
func (b *Bot) getUserName(id int) string {
	return b.users.get(id, func() (string, error) {
		users, err := b.api.UsersGet(api.Params{
			"user_ids": []int{id},
			"fields":   "first_name,last_name",
		})
		if err != nil {
			return "", err
		}
		if len(users) == 0 {
			return "", errors.New("VK user not found")
		}
		return fmt.Sprintf("%s %s", users[0].FirstName, users[0].LastName), nil
	})
}
```

- [ ] **Step 4: Verify VK GREEN**

Run: `go test -race ./iternal/adapters/vk -run TestUserNameCache -count=10`

Expected: PASS, 40 test executions, no races.

- [ ] **Step 5: Commit VK cache**

```sh
git add iternal/adapters/vk/bot.go iternal/adapters/vk/user.go iternal/adapters/vk/user_test.go
git commit -m "perf: cache VK display names"
```

---

### Task 3: Full verification and scope review

**Files:** No new production changes expected.

- [ ] **Step 1: Format changed files**

Run: `gofmt -w iternal/usecase/tgpi/groups.go iternal/usecase/tgpi/schedule.go iternal/usecase/tgpi/scheduleFilter.go iternal/usecase/tgpi/schedule_test.go iternal/adapters/vk/bot.go iternal/adapters/vk/user.go iternal/adapters/vk/user_test.go`

- [ ] **Step 2: Verify**

Run: `go test -count=1 ./...`

Run: `go test -race -count=1 ./...`

Run: `go vet ./...`

Run: `go build ./cmd/bot`

Expected: all commands exit 0 with no test failures or race reports.

- [ ] **Step 3: Review scope**

Run: `git diff --check && git status --short && git diff --stat origin/optimize/low-resource...HEAD`

Expected: only planned code, tests, and documentation; no dependency changes.
