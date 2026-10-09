package appsettings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsWhenMissingOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	if got := Load(dir); got != Defaults() {
		t.Fatalf("missing file: %+v", got)
	}
	os.WriteFile(filepath.Join(dir, fileName), []byte("{nope"), 0o644)
	if got := Load(dir); got != Defaults() {
		t.Fatalf("corrupt file: %+v", got)
	}
	os.WriteFile(filepath.Join(dir, fileName), []byte(`{"thinkLevel":"extreme"}`), 0o644)
	if got := Load(dir); got != Defaults() {
		t.Fatalf("invalid value should fall back to defaults: %+v", got)
	}
}

func TestSaveRoundTripAndPartialFill(t *testing.T) {
	dir := t.TempDir()
	saved, err := Save(dir, Settings{Autosave: true, ThinkLevel: ThinkHigh})
	if err != nil {
		t.Fatal(err)
	}
	if saved.HotkeyMode != HotkeyApp || saved.OCREngine != OCRVision || saved.DateOrder != "mdy" {
		t.Errorf("blanks not defaulted: %+v", saved)
	}
	if got := Load(dir); got != saved || !got.Autosave || got.ThinkLevel != ThinkHigh {
		t.Errorf("round trip = %+v, want %+v", got, saved)
	}
}

func TestSaveRejectsUnknownValues(t *testing.T) {
	for _, s := range []Settings{
		{ThinkLevel: "max"}, {HotkeyMode: "global"}, {OCREngine: "magic"}, {DateOrder: "ymd"},
	} {
		if _, err := Save(t.TempDir(), s); err == nil {
			t.Errorf("%+v should be rejected", s)
		}
	}
}
