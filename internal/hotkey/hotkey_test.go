package hotkey

import "testing"

func TestParse(t *testing.T) {
	code, mods, err := Parse("cmd+shift+j")
	if err != nil || code != 38 || mods != Cmd|Shift {
		t.Fatalf("cmd+shift+j = %d %d %v", code, mods, err)
	}
	if code, mods, err = Parse("Ctrl + Option + Space"); err != nil || code != 49 || mods != Control|Option {
		t.Fatalf("ctrl+option+space = %d %d %v", code, mods, err)
	}
	for _, bad := range []string{"", "j", "shift+j", "cmd+", "hyper+j", "cmd+enter"} {
		if _, _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}
