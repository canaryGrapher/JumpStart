package search

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"devdeck/internal/daterange"
	"devdeck/internal/model"
)

// DateContext carries what date parsing needs: today, how to read 10/9,
// and the quarter layout.
type DateContext struct {
	Today    time.Time
	Order    string // "mdy" (US, default) or "dmy"
	Quarters []model.QuarterRange
}

var months = map[string]time.Month{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3,
	"apr": 4, "april": 4, "may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7,
	"aug": 8, "august": 8, "sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10,
	"nov": 11, "november": 11, "dec": 12, "december": 12,
}

var weekdays = map[string]time.Weekday{
	"sun": 0, "sunday": 0, "mon": 1, "monday": 1, "tue": 2, "tues": 2, "tuesday": 2,
	"wed": 3, "wednesday": 3, "thu": 4, "thur": 4, "thurs": 4, "thursday": 4,
	"fri": 5, "friday": 5, "sat": 6, "saturday": 6,
}

var (
	reISO      = regexp.MustCompile(`^(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})$`)
	reCompact  = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)
	reNumeric  = regexp.MustCompile(`^(\d{1,2})[/.-](\d{1,2})(?:[/.-](\d{2}|\d{4}))?$`)
	reOrdinal  = regexp.MustCompile(`^(\d{1,2})(?:st|nd|rd|th)?$`)
	reYear     = regexp.MustCompile(`^(\d{4})$`)
	reInN      = regexp.MustCompile(`^in (\d{1,3}) (day|days|week|weeks|month|months)$`)
	reNextN    = regexp.MustCompile(`^(next|last|past) (\d{1,3}) (day|days|week|weeks)$`)
	reAgo      = regexp.MustCompile(`^(\d{1,3}) (day|days|week|weeks) ago$`)
	spaceComma = strings.NewReplacer(",", " ")
)

func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func one(t time.Time) daterange.Range {
	s := t.Format(daterange.DateLayout)
	return daterange.Range{From: s, To: s}
}

func span(from, to time.Time) daterange.Range {
	return daterange.Range{From: from.Format(daterange.DateLayout), To: to.Format(daterange.DateLayout)}
}

// mkDate builds a date and rejects overflow such as 2/30.
func mkDate(y int, m time.Month, d int) (time.Time, bool) {
	if m < 1 || m > 12 || d < 1 || d > 31 || y < 1900 || y > 2200 {
		return time.Time{}, false
	}
	t := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	return t, t.Month() == m && t.Day() == d
}

func year2(s string, today time.Time) int {
	y, _ := strconv.Atoi(s)
	if len(s) == 2 {
		y += 2000
	}
	if s == "" {
		y = today.Year()
	}
	return y
}

// ParseDate reads one date phrase. Strict mode (used for words inside
// free text) only accepts phrases that are unmistakably dates, so a search
// for "may" or "march" or "friday" still searches text; those forms are
// accepted after a qualifier such as due:.
func ParseDate(phrase string, c DateContext, strict bool) (daterange.Range, bool) {
	s := strings.Join(strings.Fields(strings.ToLower(spaceComma.Replace(phrase))), " ")
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		return daterange.Range{}, false
	}
	today := day(c.Today)
	if c.Today.IsZero() {
		today = day(time.Now())
	}

	switch s {
	case "yesterday":
		return one(today.AddDate(0, 0, -1)), true
	case "last month":
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -1, 0)
		return span(first, first.AddDate(0, 1, -1)), true
	case "last year", "next year":
		y := today.Year() - 1
		if s == "next year" {
			y = today.Year() + 1
		}
		return span(time.Date(y, 1, 1, 0, 0, 0, 0, time.Local), time.Date(y, 12, 31, 0, 0, 0, 0, time.Local)), true
	}
	// "Q3" alone is as often a project name as a date; free text needs due:q3.
	if preset := strings.ReplaceAll(s, " ", "_"); isPreset(preset) && !(strict && len(preset) == 2 && preset[0] == 'q') {
		if r, err := daterange.Resolve(preset, today, daterange.Effective(c.Quarters, nil)); err == nil {
			return r, true
		}
	}

	if m := reInN.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		switch m[2][:3] {
		case "day":
			return one(today.AddDate(0, 0, n)), true
		case "wee":
			return one(today.AddDate(0, 0, 7*n)), true
		default:
			return one(today.AddDate(0, n, 0)), true
		}
	}
	if m := reNextN.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[2])
		if strings.HasPrefix(m[3], "week") {
			n *= 7
		}
		if m[1] == "next" {
			return span(today, today.AddDate(0, 0, n)), true
		}
		return span(today.AddDate(0, 0, -n), today), true
	}
	if m := reAgo.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		if strings.HasPrefix(m[2], "week") {
			n *= 7
		}
		return one(today.AddDate(0, 0, -n)), true
	}

	// Weekdays: "friday" is the next one (today counts), "this friday" is in
	// this Monday-Sunday week, "next friday" in next week, "last friday" the
	// most recent one before today.
	words := strings.Fields(s)
	if len(words) <= 2 {
		lead, name := "", words[len(words)-1]
		if len(words) == 2 {
			lead = words[0]
		}
		if wd, ok := weekdays[name]; ok && (lead == "" || lead == "this" || lead == "next" || lead == "last" || lead == "on") {
			if strict && (lead == "" || lead == "on") {
				return daterange.Range{}, false
			}
			monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
			offset := (int(wd) + 6) % 7 // days after Monday
			switch lead {
			case "this":
				return one(monday.AddDate(0, 0, offset)), true
			case "next":
				return one(monday.AddDate(0, 0, 7+offset)), true
			case "last":
				d := (int(today.Weekday()) - int(wd) + 7) % 7
				if d == 0 {
					d = 7
				}
				return one(today.AddDate(0, 0, -d)), true
			default:
				d := (int(wd) - int(today.Weekday()) + 7) % 7
				return one(today.AddDate(0, 0, d)), true
			}
		}
	}

	if m := reISO.FindStringSubmatch(s); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if t, ok := mkDate(y, time.Month(mo), d); ok {
			return one(t), true
		}
		return daterange.Range{}, false
	}
	if m := reCompact.FindStringSubmatch(s); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if t, ok := mkDate(y, time.Month(mo), d); ok {
			return one(t), true
		}
		return daterange.Range{}, false
	}
	if m := reNumeric.FindStringSubmatch(s); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		mo, d := a, b
		if c.Order == "dmy" {
			mo, d = b, a
		}
		// An impossible month means the other order was meant (13/10).
		if mo > 12 && d <= 12 {
			mo, d = d, mo
		}
		// "1-2" without a year is too likely to be something else (a range,
		// a version) unless it uses a slash.
		if m[3] == "" && strict && !strings.Contains(s, "/") {
			return daterange.Range{}, false
		}
		if t, ok := mkDate(year2(m[3], today), time.Month(mo), d); ok {
			return one(t), true
		}
		return daterange.Range{}, false
	}

	// Month names: "oct 9", "october 9 2026", "9 oct", "9th of october",
	// "oct 2026", and (not strict) a lone "october".
	words = strings.Fields(strings.ReplaceAll(s, " of ", " "))
	monthAt := -1
	for i, w := range words {
		if _, ok := months[w]; ok {
			monthAt = i
			break
		}
	}
	if monthAt >= 0 && len(words) <= 3 {
		mo := months[words[monthAt]]
		rest := append(append([]string{}, words[:monthAt]...), words[monthAt+1:]...)
		dayN, yearN := 0, 0
		for _, w := range rest {
			switch {
			case reYear.MatchString(w) && yearN == 0:
				yearN, _ = strconv.Atoi(w)
			case reOrdinal.MatchString(w) && dayN == 0:
				dayN, _ = strconv.Atoi(reOrdinal.FindStringSubmatch(w)[1])
			default:
				return daterange.Range{}, false
			}
		}
		if dayN == 0 {
			// A whole month: "oct 2026", or a lone month name when not strict.
			if yearN == 0 && strict {
				return daterange.Range{}, false
			}
			if yearN == 0 {
				yearN = today.Year()
			}
			first := time.Date(yearN, mo, 1, 0, 0, 0, 0, time.Local)
			return span(first, first.AddDate(0, 1, -1)), true
		}
		if yearN == 0 {
			yearN = today.Year()
		}
		if t, ok := mkDate(yearN, mo, dayN); ok {
			return one(t), true
		}
	}
	return daterange.Range{}, false
}

func isPreset(p string) bool {
	for _, x := range daterange.Presets {
		if x == p {
			return true
		}
	}
	return false
}

// FindDates scans free text for date phrases (longest match first, up to
// four words) and returns the ranges found plus the words left over.
func FindDates(words []string, c DateContext) (ranges []daterange.Range, phrases []string, rest []string) {
	for i := 0; i < len(words); {
		matched := 0
		// After "due"/"by"/"until", a bare "friday" or "october" is a date too.
		strict := true
		if i > 0 {
			switch strings.ToLower(words[i-1]) {
			case "due", "by", "before", "until":
				strict = false
			}
		}
		for n := minInt(4, len(words)-i); n >= 1; n-- {
			phrase := strings.Join(words[i:i+n], " ")
			if r, ok := ParseDate(phrase, c, strict); ok {
				ranges = append(ranges, r)
				phrases = append(phrases, phrase)
				matched = n
				break
			}
		}
		if matched == 0 {
			rest = append(rest, words[i])
			i++
			continue
		}
		i += matched
	}
	return ranges, phrases, rest
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Describe renders a range for the "searching for" line.
func Describe(r daterange.Range) string {
	from, err1 := daterange.ParseDate(r.From)
	to, err2 := daterange.ParseDate(r.To)
	if err1 != nil || err2 != nil {
		return r.From + " – " + r.To
	}
	if r.From == r.To {
		return from.Format("Mon Jan 2, 2006")
	}
	if from.Year() == to.Year() {
		return from.Format("Jan 2") + " – " + to.Format("Jan 2, 2006")
	}
	return from.Format("Jan 2, 2006") + " – " + to.Format("Jan 2, 2006")
}
