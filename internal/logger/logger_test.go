package logger

import "testing"

func TestNewLevels(t *testing.T) {
	for _, lvl := range []string{"debug", "info", "warn", "warning", "error", "unknown", ""} {
		if l := New(lvl); l == nil {
			t.Fatalf("nil logger for %q", lvl)
		}
	}
}
