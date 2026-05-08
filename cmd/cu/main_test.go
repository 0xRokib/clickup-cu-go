package main

import (
	"strings"
	"testing"
)

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

func TestStylePlainMode(t *testing.T) {
	style := newStyle(false)
	if got := style.purple("cu"); got != "cu" {
		t.Fatalf("plain purple=%q", got)
	}
	if got := style.statusPill("IN PROGRESS"); got != "[IN PROGRESS]" {
		t.Fatalf("plain pill=%q", got)
	}
}

func TestStyleColorMode(t *testing.T) {
	style := newStyle(true)
	got := style.purple("cu")
	if got == "cu" || got[:2] != "\x1b[" {
		t.Fatalf("expected ansi color, got %q", got)
	}
	pill := style.statusPill("BLOCKED")
	if !strings.Contains(pill, "BLOCKED") || !strings.Contains(pill, "\x1b[") {
		t.Fatalf("expected colored status pill, got %q", pill)
	}
}

func TestColorEnabledFromEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("CU_PLAIN", "")
	t.Setenv("TERM", "xterm-256color")
	if colorEnabledFromEnv(true) {
		t.Fatal("NO_COLOR should disable color")
	}

	t.Setenv("NO_COLOR", "")
	t.Setenv("CU_PLAIN", "1")
	if colorEnabledFromEnv(true) {
		t.Fatal("CU_PLAIN should disable color")
	}

	t.Setenv("CU_PLAIN", "")
	t.Setenv("TERM", "dumb")
	if colorEnabledFromEnv(true) {
		t.Fatal("TERM=dumb should disable color")
	}

	t.Setenv("TERM", "xterm-256color")
	if !colorEnabledFromEnv(true) {
		t.Fatal("tty with normal env should enable color")
	}
}
