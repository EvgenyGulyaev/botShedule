package tgpi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTGPIClientsUseBoundedTimeout(t *testing.T) {
	clients := []*Client{InitClientGroup(), InitClientSchedule()}
	for _, client := range clients {
		if client.client.Timeout > 30*time.Second {
			t.Fatalf("HTTP timeout = %s, want at most 30s", client.client.Timeout)
		}
	}
}

func TestGetScheduleRejectsNonSuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := &Client{client: server.Client(), url: server.URL + "/"}
	if got := client.GetSchedule(&El{ID: 1, Type: Group}); len(got) != 0 {
		t.Fatalf("schedule = %#v, want empty result", got)
	}
}
