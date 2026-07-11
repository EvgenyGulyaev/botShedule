package tgpi

import (
	"fmt"
	"runtime"
	"strings"
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

func BenchmarkFilterGroups(b *testing.B) {
	items := make([]El, 1000)
	for i := range items {
		name := fmt.Sprintf("Группа %d", i)
		items[i] = El{ID: i, Name: name, searchName: strings.ToLower(name)}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterGroups("группа", items)
	}
}
