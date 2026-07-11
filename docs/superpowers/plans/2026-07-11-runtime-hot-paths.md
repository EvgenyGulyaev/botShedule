# Runtime Hot-Paths Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce per-message allocations, logging, hung-request lifetime, and short-lived goroutines without changing bot responses.

**Architecture:** Normalize TGPI directory names once when the hourly cache is decoded, then reuse the normalized value for every search. Apply three direct hot-path reductions in existing adapters and clients: production Telegram logging off, 15-second HTTP timeouts, and direct NATS publish calls.

**Tech Stack:** Go standard library, existing Telegram/VK/NATS clients, Go test/race/benchmark tools.

## Global Constraints

- Do not add dependencies.
- Preserve user-facing responses, one-hour directory TTL, and stale fallback.
- Use an exact 15-second timeout for both TGPI clients.
- Do not add VK or schedule caches.
- Do not remove `cmd/scrapper` or unrelated code.

---

### Task 1: Precompute case-insensitive search names

**Files:**
- Modify: `iternal/usecase/tgpi/groupFilter.go`
- Modify: `iternal/usecase/tgpi/groupFilter_test.go`

**Interfaces:**
- Consumes: `getGroups([]byte) ([]El, error)` and `filterGroups(string, []El) []El`.
- Produces: private `El.searchName string`; public fields and function signatures stay unchanged.

- [ ] **Step 1: Record the current benchmark baseline**

Run: `go test ./iternal/usecase/tgpi -run '^$' -bench BenchmarkFilterGroups -benchmem -count=5`

Expected: PASS with approximately `114 KB/op` and `1011 allocs/op`.

- [ ] **Step 2: Add a failing decoding test**

Append to `groupFilter_test.go`:

```go
func TestGetGroupsPrecomputesSearchNames(t *testing.T) {
	body := []byte(`{
		"aud":[{"id":1,"title":"АУД-1"}],
		"teacher":[{"id":2,"name":"ИВАНОВ"}],
		"group":[{"id":3,"title":"ИВТ-1"}]
	}`)

	items, err := getGroups(body)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"ивт-1", "ауд-1", "иванов"}
	for i := range want {
		if items[i].searchName != want[i] {
			t.Fatalf("item %d searchName = %q, want %q", i, items[i].searchName, want[i])
		}
	}
}
```

- [ ] **Step 3: Verify RED**

Run: `go test ./iternal/usecase/tgpi -run TestGetGroupsPrecomputesSearchNames -count=1`

Expected: FAIL to compile because `El.searchName` does not exist.

- [ ] **Step 4: Add the normalized field and populate it once**

Change `El`:

```go
type El struct {
	ID   int    `json:"id"`
	Name string `json:"title"`
	Type TypeEl

	searchName string
}
```

Change `setType`:

```go
func setType(els *[]El, gt TypeEl) {
	for i := range *els {
		(*els)[i].Type = gt
		(*els)[i].searchName = strings.ToLower((*els)[i].Name)
	}
}
```

Change `convert`:

```go
func convert(t *[]ElTeacher) (elements []El) {
	for _, v := range *t {
		elements = append(elements, El{
			ID:         v.ID,
			Name:       v.Name,
			Type:       Teacher,
			searchName: strings.ToLower(v.Name),
		})
	}
	return
}
```

Change the filter loop while preserving support for package-local manually constructed values:

```go
for _, el := range els {
	name := el.searchName
	if name == "" {
		name = strings.ToLower(el.Name)
	}
	if strings.Contains(name, mask) {
		results = append(results, el)
	}
}
```

- [ ] **Step 5: Make the benchmark represent cached production data**

Change benchmark item creation:

```go
for i := range items {
	name := fmt.Sprintf("Группа %d", i)
	items[i] = El{ID: i, Name: name, searchName: strings.ToLower(name)}
}
```

Add `strings` to the test imports.

- [ ] **Step 6: Verify GREEN and benchmark**

Run: `go test ./iternal/usecase/tgpi -run 'Test(GetGroupsPrecomputesSearchNames|FilterGroupsPreservesOrder)' -count=10`

Expected: PASS, 20 test executions.

Run: `go test ./iternal/usecase/tgpi -run '^$' -bench BenchmarkFilterGroups -benchmem -count=5`

Expected: PASS with substantially fewer than `1011 allocs/op`.

- [ ] **Step 7: Commit**

```sh
git add iternal/usecase/tgpi/groupFilter.go iternal/usecase/tgpi/groupFilter_test.go
git commit -m "perf: precompute directory search names"
```

---

### Task 2: Remove remaining per-message overhead

**Files:**
- Modify: `iternal/adapters/tg/bot.go`
- Modify: `iternal/adapters/vk/bot.go`
- Modify: `iternal/usecase/tgpi/groups.go`
- Modify: `iternal/usecase/tgpi/schedule.go`

**Interfaces:**
- Existing adapter and TGPI client signatures remain unchanged.
- Produces no new public interface or configuration.

- [ ] **Step 1: Disable verbose Telegram logging**

Delete these lines from `iternal/adapters/tg/bot.go`:

```go
bot.Debug = true
log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
```

Keep startup and error logs.

- [ ] **Step 2: Use 15-second TGPI timeouts**

In both `InitClientGroup` and `InitClientSchedule`, replace:

```go
&http.Client{Timeout: 1000 * time.Second}
```

with:

```go
&http.Client{Timeout: 15 * time.Second}
```

- [ ] **Step 3: Remove one goroutine per user event**

In Telegram and VK message handlers, replace:

```go
go user.Publish()
```

with:

```go
user.Publish()
```

Do not change the long-lived VK goroutine in `cmd/bot/main.go`.

- [ ] **Step 4: Format changed Go files**

Run: `gofmt -w iternal/adapters/tg/bot.go iternal/adapters/vk/bot.go iternal/usecase/tgpi/groups.go iternal/usecase/tgpi/schedule.go iternal/usecase/tgpi/groupFilter.go iternal/usecase/tgpi/groupFilter_test.go`

Expected: exit 0.

- [ ] **Step 5: Run full verification**

Run: `go test -count=1 ./...`

Expected: PASS with zero failing packages.

Run: `go test -race -count=1 ./...`

Expected: PASS with no race reports.

Run: `go vet ./...`

Expected: exit 0 with no diagnostics.

Run: `go test ./iternal/usecase/tgpi -run '^$' -bench BenchmarkFilterGroups -benchmem -count=5`

Expected: PASS with the Task 1 allocation reduction preserved.

- [ ] **Step 6: Check scope and commit**

Run: `git diff --check && git diff --stat && git status --short`

Expected: only the four planned production files plus TGPI tests and documentation are changed; no dependency files change.

```sh
git add iternal/adapters/tg/bot.go iternal/adapters/vk/bot.go iternal/usecase/tgpi/groups.go iternal/usecase/tgpi/schedule.go
git commit -m "perf: reduce per-message runtime overhead"
```
