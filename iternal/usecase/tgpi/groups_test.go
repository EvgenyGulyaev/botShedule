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
