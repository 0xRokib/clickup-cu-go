package main

import "testing"

func TestParseDuration(t *testing.T) {
	tests := map[string]int64{
		"2h":        7200000,
		"30m":       1800000,
		"2h 30m":    9000000,
		"45":        2700000,
		"1.5h":      5400000,
		"1 hour 5m": 3900000,
	}
	for input, want := range tests {
		got, err := parseDuration(input)
		if err != nil {
			t.Fatalf("parseDuration(%q) error: %v", input, err)
		}
		if got != want {
			t.Fatalf("parseDuration(%q)=%d want %d", input, got, want)
		}
	}
}

func TestParseDurationRejectsBadInput(t *testing.T) {
	if _, err := parseDuration("later"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSplitDurationAndNote(t *testing.T) {
	dur, note := splitDurationAndNote([]string{"2h", "30m", "backend", "work"})
	if dur != "2h 30m" || note != "backend work" {
		t.Fatalf("got duration %q note %q", dur, note)
	}
}
