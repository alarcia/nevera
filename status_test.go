package main

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

func TestItemStatus(t *testing.T) {
	// 23:30 UTC on 13 Nov is already 14 Nov in Andorra: days are counted there.
	now := time.Date(2026, 11, 13, 23, 30, 0, 0, time.UTC)
	cases := []struct {
		name            string
		expires, opened time.Time
		maxOpen         int
		level           Level
		chip            string
	}{
		{"no dates", time.Time{}, time.Time{}, 0, None, "Sin fecha"},
		{"expires today (Andorra)", d(2026, 11, 14), time.Time{}, 0, Urgent, "Caduca hoy"},
		{"expired", d(2026, 11, 13), time.Time{}, 0, Urgent, "Caducado"},
		{"tomorrow", d(2026, 11, 15), time.Time{}, 0, Urgent, "Caduca mañana"},
		{"in 4 days", d(2026, 11, 18), time.Time{}, 0, Soon, "Caduca 18 nov"},
		{"in 7 days", d(2026, 11, 21), time.Time{}, 0, Watch, "Caduca 21 nov"},
		{"in 8 days", d(2026, 11, 22), time.Time{}, 0, None, "Caduca 22 nov"},
		{"open too long wins", d(2026, 12, 30), d(2026, 11, 4), 4, Urgent, "Abierto 10 días"},
		{"open over limit", d(2026, 12, 30), d(2026, 11, 9), 4, Soon, "Abierto 5 días"},
		{"open near limit", d(2026, 12, 30), d(2026, 11, 10), 4, Watch, "Abierto 4 días"},
		{"open at 75% is fine", d(2026, 12, 30), d(2026, 11, 11), 4, None, "Caduca 30 dic"},
		{"open, no expiry, no limit", time.Time{}, d(2026, 11, 12), 0, None, "Abierto 2 días"},
	}
	for _, c := range cases {
		got := itemStatus(now, c.expires, c.opened, c.maxOpen)
		if got.Level != c.level || got.Chip != c.chip {
			t.Errorf("%s: got level %d chip %q, want %d %q", c.name, got.Level, got.Chip, c.level, c.chip)
		}
	}
}

func TestFillPercent(t *testing.T) {
	if got := fillPercent(24, 48); got != 50 {
		t.Errorf("got %d, want 50", got)
	}
	if got := fillPercent(5, 0); got != 0 {
		t.Errorf("no capacity must disable the index, got %d", got)
	}
}
