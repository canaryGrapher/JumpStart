package ghsync

import (
	"testing"

	"devdeck/internal/model"
)

func TestEffectiveColumnsDefault(t *testing.T) {
	got := EffectiveColumnIDs(&model.Project{})
	want := LocalColumns()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestEffectiveColumnsCustom(t *testing.T) {
	p := &model.Project{
		Columns: []model.BoardColumn{
			{ID: "backlog", Label: "Backlog"},
			{ID: "review", Label: "Review"},
		},
	}
	got := EffectiveColumnIDs(p)
	if len(got) != 2 || got[0] != "backlog" || got[1] != "review" {
		t.Fatalf("got %v", got)
	}
}

func TestColumnIDFromLabel(t *testing.T) {
	id := ColumnIDFromLabel("Ready for Review", nil)
	if id != "readyforreview" {
		t.Fatalf("got %q", id)
	}
	id2 := ColumnIDFromLabel("Ready for Review", []string{"readyforreview"})
	if id2 != "readyforreview2" {
		t.Fatalf("got %q", id2)
	}
}

func TestKnownColumn(t *testing.T) {
	if !KnownColumn(nil, "testing") {
		t.Fatal("expected default testing column")
	}
	p := &model.Project{Columns: []model.BoardColumn{{ID: "review", Label: "Review"}}}
	if !KnownColumn(p, "review") {
		t.Fatal("expected custom column")
	}
	if KnownColumn(p, "testing") {
		t.Fatal("built-in should not remain when Columns is set")
	}
}
