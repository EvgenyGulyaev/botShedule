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
