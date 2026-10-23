package cmd

import (
	"testing"
	"time"
)

func TestWindowStartsLate(t *testing.T) {
	now := time.Now().UTC()
	at := func(d time.Duration) map[string]interface{} {
		return map[string]interface{}{"startsAt": now.Add(d).Format(time.RFC3339Nano)}
	}
	if !windowStartsLate(map[string]interface{}{"startsAt": nil}, "week") {
		t.Fatal("no snapshot in the window must fall back")
	}
	if !windowStartsLate(at(-time.Hour), "week") {
		t.Fatal("a window that starts today must fall back")
	}
	if windowStartsLate(at(-6*24*time.Hour-12*time.Hour), "week") {
		t.Fatal("a window covering the week must stand")
	}
	if windowStartsLate(at(-time.Hour), "nonsense") {
		t.Fatal("unknown period never falls back")
	}
}

func TestSnapshotBefore(t *testing.T) {
	cutoff := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	snaps := []interface{}{
		map[string]interface{}{"createdAt": "2026-10-23T20:52:25.376Z"},
		map[string]interface{}{"createdAt": "2026-10-11T11:08:54.120Z"},
		map[string]interface{}{"createdAt": "2026-10-06T04:18:17.294Z"},
	}
	got, ok := snapshotBefore(snaps, cutoff)
	if !ok || got != "2026-10-11T11:08:53Z" {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := snapshotBefore(snaps[:1], cutoff); ok {
		t.Fatal("nothing before the cutoff must report none")
	}
}
