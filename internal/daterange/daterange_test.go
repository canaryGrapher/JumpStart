package daterange

import (
	"testing"
	"time"

	"devdeck/internal/model"
)

func d(s string) time.Time {
	t, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestResolveWeeksAndMonths(t *testing.T) {
	// Wednesday 2026-10-07.
	today := d("2026-10-07")
	cases := map[string]Range{
		LastWeek:  {"2026-09-28", "2026-10-04"},
		ThisWeek:  {"2026-10-05", "2026-10-11"},
		NextWeek:  {"2026-10-12", "2026-10-18"},
		ThisMonth: {"2026-10-01", "2026-10-31"},
		NextMonth: {"2026-11-01", "2026-11-30"},
		ThisYear:  {"2026-01-01", "2026-12-31"},
	}
	for preset, want := range cases {
		got, err := Resolve(preset, today, nil)
		if err != nil {
			t.Fatalf("%s: %v", preset, err)
		}
		if got != want {
			t.Errorf("%s = %+v, want %+v", preset, got, want)
		}
	}
}

func TestResolveSundayBelongsToItsWeek(t *testing.T) {
	got, _ := Resolve(ThisWeek, d("2026-10-11"), nil)
	if got != (Range{"2026-10-05", "2026-10-11"}) {
		t.Errorf("sunday week = %+v", got)
	}
}

func TestResolveNextMonthAcrossYearEnd(t *testing.T) {
	got, _ := Resolve(NextMonth, d("2026-12-15"), nil)
	if got != (Range{"2027-01-01", "2027-01-31"}) {
		t.Errorf("december next month = %+v", got)
	}
}

func TestResolveCalendarQuarters(t *testing.T) {
	qs := DefaultQuarters()
	today := d("2026-10-07")
	want := map[string]Range{
		Q1: {"2026-01-01", "2026-03-31"},
		Q2: {"2026-04-01", "2026-06-30"},
		Q3: {"2026-07-01", "2026-09-30"},
		Q4: {"2026-10-01", "2026-12-31"},
	}
	for preset, w := range want {
		got, err := Resolve(preset, today, qs)
		if err != nil || got != w {
			t.Errorf("%s = %+v (%v), want %+v", preset, got, err, w)
		}
	}
}

func TestResolveFiscalQuartersJulyStart(t *testing.T) {
	qs := []model.QuarterRange{
		{Start: "07-01", End: "09-30"},
		{Start: "10-01", End: "12-31"},
		{Start: "01-01", End: "03-31"},
		{Start: "04-01", End: "06-30"},
	}
	// March 2027 is still in the fiscal year that began July 2026.
	today := d("2027-03-10")
	want := map[string]Range{
		Q1: {"2026-07-01", "2026-09-30"},
		Q2: {"2026-10-01", "2026-12-31"},
		Q3: {"2027-01-01", "2027-03-31"},
		Q4: {"2027-04-01", "2027-06-30"},
	}
	for preset, w := range want {
		got, err := Resolve(preset, today, qs)
		if err != nil || got != w {
			t.Errorf("%s = %+v (%v), want %+v", preset, got, err, w)
		}
	}
	// August 2026 starts the same fiscal year.
	got, _ := Resolve(Q4, d("2026-08-01"), qs)
	if got != (Range{"2027-04-01", "2027-06-30"}) {
		t.Errorf("august Q4 = %+v", got)
	}
}

func TestResolveQuarterWrappingYearEnd(t *testing.T) {
	qs := []model.QuarterRange{
		{Start: "02-01", End: "04-30"},
		{Start: "05-01", End: "07-31"},
		{Start: "08-01", End: "10-31"},
		{Start: "11-01", End: "01-31"},
	}
	got, err := Resolve(Q4, d("2026-12-01"), qs)
	if err != nil || got != (Range{"2026-11-01", "2027-01-31"}) {
		t.Errorf("wrapping Q4 = %+v (%v)", got, err)
	}
	// January 2027 is before Q1's start, so it belongs to the year that began Feb 2026.
	got, _ = Resolve(Q4, d("2027-01-15"), qs)
	if got != (Range{"2026-11-01", "2027-01-31"}) {
		t.Errorf("january Q4 = %+v", got)
	}
}

func TestResolveErrors(t *testing.T) {
	if _, err := Resolve("someday", d("2026-10-07"), nil); err == nil {
		t.Error("unknown preset should fail")
	}
	bad := []model.QuarterRange{{Start: "13-01", End: "03-31"}, {}, {}, {}}
	if _, err := Resolve(Q1, d("2026-10-07"), bad); err == nil {
		t.Error("invalid quarters should fail")
	}
}

func TestEffectivePrefersProjectThenGlobalThenDefault(t *testing.T) {
	project := []model.QuarterRange{
		{Start: "02-01", End: "04-30"}, {Start: "05-01", End: "07-31"},
		{Start: "08-01", End: "10-31"}, {Start: "11-01", End: "01-31"},
	}
	global := []model.QuarterRange{
		{Start: "07-01", End: "09-30"}, {Start: "10-01", End: "12-31"},
		{Start: "01-01", End: "03-31"}, {Start: "04-01", End: "06-30"},
	}
	if got := Effective(project, global); got[0].Start != "02-01" {
		t.Errorf("project not preferred: %+v", got[0])
	}
	if got := Effective(nil, global); got[0].Start != "07-01" {
		t.Errorf("global not used: %+v", got[0])
	}
	if got := Effective(nil, nil); got[0].Start != "01-01" {
		t.Errorf("default not used: %+v", got[0])
	}
	if got := Effective([]model.QuarterRange{{Start: "bad"}}, global); got[0].Start != "07-01" {
		t.Errorf("invalid project override should fall back: %+v", got[0])
	}
}

func TestContains(t *testing.T) {
	r := Range{"2026-10-05", "2026-10-11"}
	for due, want := range map[string]bool{
		"2026-10-05": true, "2026-10-11": true, "2026-10-04": false,
		"2026-10-12": false, "": false, "garbage": false,
	} {
		if got := r.Contains(due); got != want {
			t.Errorf("Contains(%q) = %v, want %v", due, got, want)
		}
	}
}

func TestIsPastDue(t *testing.T) {
	today := d("2026-10-07")
	cases := []struct {
		due  string
		done bool
		want bool
	}{
		{"2026-10-06", false, true},
		{"2026-10-07", false, false}, // due today is not past due
		{"2026-10-08", false, false},
		{"2026-10-06", true, false}, // finished work is never past due
		{"", false, false},
		{"nonsense", false, false},
	}
	for _, c := range cases {
		if got := IsPastDue(c.due, c.done, today); got != c.want {
			t.Errorf("IsPastDue(%q, done=%v) = %v, want %v", c.due, c.done, got, c.want)
		}
	}
}

func TestGlobalStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := LoadGlobal(dir); got != nil {
		t.Fatalf("empty dir should yield nil, got %+v", got)
	}
	qs := []model.QuarterRange{
		{Name: "Q1", Start: "07-01", End: "09-30"}, {Name: "Q2", Start: "10-01", End: "12-31"},
		{Name: "Q3", Start: "01-01", End: "03-31"}, {Name: "Q4", Start: "04-01", End: "06-30"},
	}
	if err := SaveGlobal(dir, qs); err != nil {
		t.Fatal(err)
	}
	if got := LoadGlobal(dir); len(got) != 4 || got[0].Start != "07-01" {
		t.Fatalf("round trip = %+v", got)
	}
	if err := SaveGlobal(dir, qs[:2]); err == nil {
		t.Error("saving two quarters should fail")
	}
	if err := SaveGlobal(dir, nil); err != nil {
		t.Fatal(err)
	}
	if got := LoadGlobal(dir); got != nil {
		t.Fatalf("cleared config should yield nil, got %+v", got)
	}
}

func TestRelativeDayAndQuarterPresets(t *testing.T) {
	today := d("2026-10-07")
	cal := DefaultQuarters()
	fiscal := []model.QuarterRange{
		{Start: "07-01", End: "09-30"}, {Start: "10-01", End: "12-31"},
		{Start: "01-01", End: "03-31"}, {Start: "04-01", End: "06-30"},
	}
	cases := []struct {
		preset string
		qs     []model.QuarterRange
		want   Range
	}{
		{Today, nil, Range{"2026-10-07", "2026-10-07"}},
		{Tomorrow, nil, Range{"2026-10-08", "2026-10-08"}},
		{ThisQuarter, cal, Range{"2026-10-01", "2026-12-31"}},
		{NextQuarter, cal, Range{"2027-01-01", "2027-03-31"}},
		{LastQuarter, cal, Range{"2026-07-01", "2026-09-30"}},
		{ThisQuarter, fiscal, Range{"2026-10-01", "2026-12-31"}},
		{NextQuarter, fiscal, Range{"2027-01-01", "2027-03-31"}},
		{LastQuarter, fiscal, Range{"2026-07-01", "2026-09-30"}},
	}
	for _, c := range cases {
		got, err := Resolve(c.preset, today, c.qs)
		if err != nil || got != c.want {
			t.Errorf("%s = %+v (%v), want %+v", c.preset, got, err, c.want)
		}
	}
	// Year boundaries: Dec 31 -> next quarter is Q1 next year; Jan 1 -> last quarter is Q4 last year.
	if got, _ := Resolve(NextQuarter, d("2026-12-31"), cal); got != (Range{"2027-01-01", "2027-03-31"}) {
		t.Errorf("Dec 31 next quarter = %+v", got)
	}
	if got, _ := Resolve(LastQuarter, d("2027-01-01"), cal); got != (Range{"2026-10-01", "2026-12-31"}) {
		t.Errorf("Jan 1 last quarter = %+v", got)
	}
	wrap := []model.QuarterRange{
		{Start: "02-01", End: "04-30"}, {Start: "05-01", End: "07-31"},
		{Start: "08-01", End: "10-31"}, {Start: "11-01", End: "01-31"},
	}
	if got, _ := Resolve(ThisQuarter, d("2027-01-15"), wrap); got != (Range{"2026-11-01", "2027-01-31"}) {
		t.Errorf("wrapping this quarter = %+v", got)
	}
	if got, _ := Resolve(NextQuarter, d("2027-01-15"), wrap); got != (Range{"2027-02-01", "2027-04-30"}) {
		t.Errorf("wrapping next quarter = %+v", got)
	}
}
