// Package daterange resolves named due-date ranges ("this week", "Q2",
// "next month") to concrete inclusive YYYY-MM-DD bounds. Quarters come
// from a configurable set of month/day spans so fiscal calendars work.
package daterange

import (
	"fmt"
	"strings"
	"time"

	"devdeck/internal/model"
)

// DateLayout is the on-disk format for Task.DueDate.
const DateLayout = "2006-01-02"

// Preset names accepted by Resolve.
const (
	LastWeek    = "last_week"
	ThisWeek    = "this_week"
	NextWeek    = "next_week"
	ThisMonth   = "this_month"
	NextMonth   = "next_month"
	Q1          = "q1"
	Q2          = "q2"
	Q3          = "q3"
	Q4          = "q4"
	ThisYear    = "this_year"
	Today       = "today"
	Tomorrow    = "tomorrow"
	ThisQuarter = "this_quarter"
	NextQuarter = "next_quarter"
	LastQuarter = "last_quarter"
)

// Presets lists every preset in display order.
var Presets = []string{
	Today, Tomorrow, LastWeek, ThisWeek, NextWeek, ThisMonth, NextMonth,
	LastQuarter, ThisQuarter, NextQuarter, Q1, Q2, Q3, Q4, ThisYear,
}

// Range is an inclusive pair of YYYY-MM-DD dates.
type Range struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// DefaultQuarters is the calendar-year layout used when nothing is configured.
func DefaultQuarters() []model.QuarterRange {
	return []model.QuarterRange{
		{Name: "Q1", Start: "01-01", End: "03-31"},
		{Name: "Q2", Start: "04-01", End: "06-30"},
		{Name: "Q3", Start: "07-01", End: "09-30"},
		{Name: "Q4", Start: "10-01", End: "12-31"},
	}
}

// ParseDate parses a YYYY-MM-DD due date.
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation(DateLayout, strings.TrimSpace(s), time.Local)
}

// parseMD parses "MM-DD". The year 2000 is a leap year so 02-29 is valid.
func parseMD(s string) (month time.Month, day int, err error) {
	t, err := time.Parse("2006-01-02", "2000-"+strings.TrimSpace(s))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid MM-DD %q", s)
	}
	return t.Month(), t.Day(), nil
}

// ValidateQuarters checks that qs holds exactly four well-formed MM-DD spans.
func ValidateQuarters(qs []model.QuarterRange) error {
	if len(qs) != 4 {
		return fmt.Errorf("quarters must have exactly 4 entries, got %d", len(qs))
	}
	for i, q := range qs {
		if _, _, err := parseMD(q.Start); err != nil {
			return fmt.Errorf("Q%d start: %w", i+1, err)
		}
		if _, _, err := parseMD(q.End); err != nil {
			return fmt.Errorf("Q%d end: %w", i+1, err)
		}
	}
	return nil
}

// Effective returns the first valid quarter set from project, global, then
// the calendar default.
func Effective(project, global []model.QuarterRange) []model.QuarterRange {
	if ValidateQuarters(project) == nil {
		return project
	}
	if ValidateQuarters(global) == nil {
		return global
	}
	return DefaultQuarters()
}

func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func format(from, to time.Time) Range {
	return Range{From: from.Format(DateLayout), To: to.Format(DateLayout)}
}

// weekStart returns the Monday on or before t.
func weekStart(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7
	return day(t).AddDate(0, 0, -offset)
}

// quarterRange places quarter idx (0-3) in the reporting year that contains
// today. The reporting year begins on Q1's start date, so a July-June fiscal
// year puts Q1 in July even when today is in March.
func quarterRange(qs []model.QuarterRange, idx int, today time.Time) (Range, error) {
	q1m, q1d, err := parseMD(qs[0].Start)
	if err != nil {
		return Range{}, err
	}
	fy := today.Year()
	if day(today).Before(time.Date(fy, q1m, q1d, 0, 0, 0, 0, time.Local)) {
		fy--
	}
	sm, sd, err := parseMD(qs[idx].Start)
	if err != nil {
		return Range{}, err
	}
	em, ed, err := parseMD(qs[idx].End)
	if err != nil {
		return Range{}, err
	}
	startYear := fy
	if sm < q1m || (sm == q1m && sd < q1d) {
		startYear++
	}
	endYear := startYear
	if em < sm || (em == sm && ed < sd) {
		endYear++
	}
	start := time.Date(startYear, sm, sd, 0, 0, 0, 0, time.Local)
	end := time.Date(endYear, em, ed, 0, 0, 0, 0, time.Local)
	return format(start, end), nil
}

// Resolve turns a preset into inclusive bounds relative to today. Weeks run
// Monday to Sunday; "this_year" is the calendar year; q1..q4 use qs.
func Resolve(preset string, today time.Time, qs []model.QuarterRange) (Range, error) {
	today = day(today)
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case LastWeek, ThisWeek, NextWeek:
		start := weekStart(today)
		switch strings.ToLower(strings.TrimSpace(preset)) {
		case LastWeek:
			start = start.AddDate(0, 0, -7)
		case NextWeek:
			start = start.AddDate(0, 0, 7)
		}
		return format(start, start.AddDate(0, 0, 6)), nil
	case ThisMonth, NextMonth:
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.Local)
		if strings.EqualFold(strings.TrimSpace(preset), NextMonth) {
			first = first.AddDate(0, 1, 0)
		}
		return format(first, first.AddDate(0, 1, -1)), nil
	case Today:
		return format(today, today), nil
	case Tomorrow:
		t := today.AddDate(0, 0, 1)
		return format(t, t), nil
	case ThisQuarter, NextQuarter, LastQuarter:
		if err := ValidateQuarters(qs); err != nil {
			return Range{}, err
		}
		cur, err := quarterContaining(qs, today)
		if err != nil {
			return Range{}, err
		}
		switch strings.ToLower(strings.TrimSpace(preset)) {
		case NextQuarter:
			end, _ := ParseDate(cur.To)
			return quarterContaining(qs, end.AddDate(0, 0, 1))
		case LastQuarter:
			start, _ := ParseDate(cur.From)
			return quarterContaining(qs, start.AddDate(0, 0, -1))
		}
		return cur, nil
	case ThisYear:
		return format(
			time.Date(today.Year(), 1, 1, 0, 0, 0, 0, time.Local),
			time.Date(today.Year(), 12, 31, 0, 0, 0, 0, time.Local),
		), nil
	case Q1, Q2, Q3, Q4:
		if err := ValidateQuarters(qs); err != nil {
			return Range{}, err
		}
		idx := int(strings.ToLower(strings.TrimSpace(preset))[1] - '1')
		return quarterRange(qs, idx, today)
	}
	return Range{}, fmt.Errorf("unknown due-date preset %q (want one of %s)", preset, strings.Join(Presets, ", "))
}

// quarterContaining returns the configured quarter that includes day. With
// gaps between configured quarters, it falls back to the next one that
// starts after day.
func quarterContaining(qs []model.QuarterRange, day time.Time) (Range, error) {
	key := day.Format(DateLayout)
	var next *Range
	for _, ref := range []time.Time{day, day.AddDate(-1, 0, 0), day.AddDate(1, 0, 0)} {
		for i := range qs {
			r, err := quarterRange(qs, i, ref)
			if err != nil {
				return Range{}, err
			}
			if r.Contains(key) {
				return r, nil
			}
			if r.From > key && (next == nil || r.From < next.From) {
				rr := r
				next = &rr
			}
		}
	}
	if next != nil {
		return *next, nil
	}
	return Range{}, fmt.Errorf("no quarter contains %s", key)
}

// Contains reports whether the YYYY-MM-DD due date falls inside r. An empty
// or malformed due date is never inside a range.
func (r Range) Contains(due string) bool {
	if due == "" {
		return false
	}
	if _, err := ParseDate(due); err != nil {
		return false
	}
	if r.From != "" && due < r.From {
		return false
	}
	if r.To != "" && due > r.To {
		return false
	}
	return true
}

// IsPastDue reports whether a still-open task's due date is before today.
func IsPastDue(due string, done bool, today time.Time) bool {
	if done || due == "" {
		return false
	}
	if _, err := ParseDate(due); err != nil {
		return false
	}
	return due < day(today).Format(DateLayout)
}
