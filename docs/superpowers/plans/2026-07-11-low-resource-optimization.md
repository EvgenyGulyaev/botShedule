# Low-Resource Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce CPU, memory, goroutine, and network usage by caching the TGPI directory for one hour and replacing concurrent string filtering with a deterministic loop.

**Architecture:** Keep the cache inside the existing `tgpi.Client`: one mutex protects the cached full directory and serializes refreshes, while filtering remains outside the critical section. Refresh synchronously on the first request after expiry and retain stale data when the upstream request fails. No background worker or new dependency is introduced.

**Tech Stack:** Go standard library, existing `net/http` client, `httptest`, Go test/race/benchmark tools.

## Global Constraints

- Cache TTL is exactly one hour.
- Preserve `GetGroups(string) []El` and all user-facing output.
- Do not cache schedules.
- Do not add dependencies or background goroutines.
- Preserve the user's untracked `.go-version` file.

---

### Task 1: Deterministic low-overhead filtering

**Files:**
- Modify: `iternal/usecase/tgpi/groupFilter.go`
- Create: `iternal/usecase/tgpi/groupFilter_test.go`

**Interfaces:**
- Consumes: `El`, `filterGroups(string, []El) []El`
- Produces: the same `filterGroups` signature with stable upstream order.

- [ ] **Step 1: Add a failing order and case-insensitivity test**

```go
package tgpi

import (
	"fmt"
	"runtime"
	"testing"
)

func TestFilterGroupsPreservesOrder(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)

	items := make([]El, 100)
	for i := range items {
		items[i] = El{ID: i, Name: fmt.Sprintf("ГРУППА-%03d", i)}
	}

	for attempt := 0; attempt < 20; attempt++ {
		got := filterGroups("группа", items)
		for i := range got {
			if got[i].ID != i {
				t.Fatalf("result order changed at index %d: got ID %d", i, got[i].ID)
			}
		}
	}
}

func BenchmarkFilterGroups(b *testing.B) {
	items := make([]El, 1000)
	for i := range items {
		items[i] = El{ID: i, Name: fmt.Sprintf("Группа %d", i)}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterGroups("группа", items)
	}
}
```

- [ ] **Step 2: Verify the regression test exposes unstable ordering**

Run: `go test ./iternal/usecase/tgpi -run TestFilterGroupsPreservesOrder -count=1`

Expected: FAIL with `result order changed` under the current goroutine implementation. If the scheduler happens to preserve order once, rerun with `-count=10`; do not change production code before observing the failure.

- [ ] **Step 3: Record the baseline benchmark**

Run: `go test ./iternal/usecase/tgpi -run '^$' -bench BenchmarkFilterGroups -benchmem -count=5`

Expected: PASS and five baseline measurements showing goroutine/allocation cost.

- [ ] **Step 4: Replace goroutines with the minimal loop**

```go
func filterGroups(groupName string, els []El) []El {
	if groupName == "" {
		return els
	}

	mask := strings.ToLower(groupName)
	results := make([]El, 0)
	for _, el := range els {
		if strings.Contains(strings.ToLower(el.Name), mask) {
			results = append(results, el)
		}
	}
	return results
}
```

Delete `filterEl` and the unused `sync` import.

- [ ] **Step 5: Verify behavior and benchmark improvement**

Run: `go test ./iternal/usecase/tgpi -run TestFilterGroupsPreservesOrder -count=10`

Expected: PASS.

Run: `go test ./iternal/usecase/tgpi -run '^$' -bench BenchmarkFilterGroups -benchmem -count=5`

Expected: PASS with fewer allocations and lower execution time than Step 3.

---

### Task 2: One-hour directory cache and stale fallback

**Files:**
- Modify: `iternal/usecase/tgpi/groups.go`
- Modify: `iternal/usecase/tgpi/groupFilter.go`
- Create: `iternal/usecase/tgpi/groups_test.go`

**Interfaces:**
- Consumes: `Client`, `getReqGroup() (*http.Request, error)`, decoded `[]El`.
- Produces: `fetchGroups() ([]El, error)`, `cachedGroups() []El`, and `getGroups([]byte) ([]El, error)` while preserving `GetGroups(string) []El`.

- [ ] **Step 1: Add a failing cache-hit test**

```go
package tgpi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const groupsJSON = `{"aud":[],"teacher":[],"group":[{"id":1,"title":"ИВТ-1"}]}`

func TestGetGroupsCachesSuccessfulResponse(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, groupsJSON)
	}))
	defer server.Close()

	client := &Client{client: server.Client(), url: server.URL}
	client.GetGroups("")
	client.GetGroups("")

	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}
```

- [ ] **Step 2: Verify the existing client performs two requests**

Run: `go test ./iternal/usecase/tgpi -run TestGetGroupsCachesSuccessfulResponse -count=1`

Expected: FAIL with `requests = 2, want 1`.

- [ ] **Step 3: Implement the minimum successful-response cache**

Add these fields and constant in `groups.go`:

```go
const groupsCacheTTL = time.Hour

type Client struct {
	client *http.Client
	url    string

	groupsMu        sync.Mutex
	groups          []El
	groupsCheckedAt time.Time
}
```

Change the group flow to:

```go
func (t *Client) GetGroups(groupName string) []El {
	return filterGroups(groupName, t.cachedGroups())
}

func (t *Client) cachedGroups() []El {
	t.groupsMu.Lock()
	defer t.groupsMu.Unlock()

	if !t.groupsCheckedAt.IsZero() && time.Since(t.groupsCheckedAt) < groupsCacheTTL {
		return t.groups
	}

	groups, err := t.fetchGroups()
	if err != nil {
		if !t.groupsCheckedAt.IsZero() {
			t.groupsCheckedAt = time.Now()
		}
		return t.groups
	}
	t.groups = groups
	t.groupsCheckedAt = time.Now()
	return t.groups
}

func (t *Client) fetchGroups() ([]El, error) {
	req, err := t.getReqGroup()
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("TGPI groups: unexpected HTTP status %s", resp.Status)
	}
	reader, err := getReader(resp)
	if err != nil {
		return nil, err
	}
	if reader != resp.Body {
		defer reader.Close()
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return getGroups(body)
}
```

Change `getReader` to return its gzip error instead of panicking or closing the reader before use:

```go
func getReader(resp *http.Response) (io.ReadCloser, error) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		return gzip.NewReader(resp.Body)
	}
	return resp.Body, nil
}
```

Change `getGroups` in `groupFilter.go`:

```go
func getGroups(body []byte) ([]El, error) {
	var group elementGroup
	if err := json.Unmarshal(body, &group); err != nil {
		return nil, err
	}
	teachers := convert(&group.Teacher)
	setType(&group.Group, Group)
	setType(&group.Aud, Aud)
	return append(group.Group, append(group.Aud, teachers...)...), nil
}
```

Remove the unused `fmt` import from `groupFilter.go` and add `fmt` and `sync` to `groups.go`.

- [ ] **Step 4: Verify the cache-hit test passes**

Run: `go test ./iternal/usecase/tgpi -run TestGetGroupsCachesSuccessfulResponse -count=1`

Expected: PASS.

- [ ] **Step 5: Add expiry, concurrent-refresh, stale-fallback, and failed-refresh coalescing tests**

Append to `groups_test.go`:

```go
func TestGetGroupsRefreshesExpiredCache(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, groupsJSON)
	}))
	defer server.Close()
	client := &Client{client: server.Client(), url: server.URL}

	client.GetGroups("")
	client.groupsCheckedAt = time.Now().Add(-groupsCacheTTL)
	client.GetGroups("")

	if got := requests.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}

func TestGetGroupsCoalescesConcurrentRefresh(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		time.Sleep(10 * time.Millisecond)
		fmt.Fprint(w, groupsJSON)
	}))
	defer server.Close()
	client := &Client{client: server.Client(), url: server.URL}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.GetGroups("")
		}()
	}
	wg.Wait()

	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestGetGroupsUsesStaleCacheWhenRefreshFails(t *testing.T) {
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, groupsJSON)
	}))
	defer server.Close()
	client := &Client{client: server.Client(), url: server.URL}

	want := client.GetGroups("")
	fail.Store(true)
	client.groupsCheckedAt = time.Now().Add(-groupsCacheTTL)
	got := client.GetGroups("")

	if len(got) != len(want) || len(got) != 1 || got[0] != want[0] {
		t.Fatalf("stale result = %#v, want %#v", got, want)
	}
}

func TestGetGroupsCoalescesConcurrentFailedRefresh(t *testing.T) {
	var requests atomic.Int32
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		if fail.Load() {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, groupsJSON)
	}))
	defer server.Close()
	client := &Client{client: server.Client(), url: server.URL}

	client.GetGroups("")
	fail.Store(true)
	client.groupsCheckedAt = time.Now().Add(-groupsCacheTTL)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.GetGroups("")
		}()
	}
	wg.Wait()

	if got := requests.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2 (initial load plus one failed refresh)", got)
	}
}
```

- [ ] **Step 6: Run cache tests with the race detector**

Run: `go test -race ./iternal/usecase/tgpi -run 'TestGetGroups' -count=10`

Expected: PASS with no race reports and exactly one refresh attempt in each concurrent test, including upstream failure.

---

### Task 3: Remove duplicate command work and verify the complete change

**Files:**
- Modify: `iternal/formatter/handlerMessage.go`
- Modify: `iternal/usecase/tgpi/groupFilter_test.go` only if formatting is required.

**Interfaces:**
- Consumes and preserves `HandlerCommand() string` and `GetAnswer() (string, []string)`.
- Produces no new interface.

- [ ] **Step 1: Reuse the first command result**

Replace:

```go
c := m.HandlerCommand()
if m.HandlerCommand() != "" {
	return c, nil
}
```

with:

```go
if command := m.HandlerCommand(); command != "" {
	return command, nil
}
```

This is a trivial duplicate-call removal; it introduces no branch or new behavior.

- [ ] **Step 2: Format all changed Go files**

Run: `gofmt -w iternal/usecase/tgpi/groupFilter.go iternal/usecase/tgpi/groupFilter_test.go iternal/usecase/tgpi/groups.go iternal/usecase/tgpi/groups_test.go iternal/formatter/handlerMessage.go`

Expected: exit 0.

- [ ] **Step 3: Run full verification**

Run: `go test ./...`

Expected: PASS, zero failing packages.

Run: `go test -race ./...`

Expected: PASS, no race reports.

Run: `go vet ./...`

Expected: exit 0 with no diagnostics.

Run: `go test ./iternal/usecase/tgpi -run '^$' -bench . -benchmem -count=5`

Expected: PASS and improved filtering time/allocations compared with Task 1 baseline.

- [ ] **Step 4: Review the final diff against the design**

Run: `git diff --check && git diff --stat && git status --short`

Expected: only the planned Go files, tests, and plan are changed; `.go-version` remains untracked and unchanged.
