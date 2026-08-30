package github

import "testing"

func TestParseRemote(t *testing.T) {
	cases := []struct {
		remote string
		want   string
	}{
		{"https://github.com/canaryGrapher/JumpStart.git", "canaryGrapher/JumpStart"},
		{"https://github.com/canaryGrapher/JumpStart", "canaryGrapher/JumpStart"},
		{"http://github.com/o/r.git", "o/r"},
		{"git@github.com:canaryGrapher/JumpStart.git", "canaryGrapher/JumpStart"},
		{"ssh://git@github.com/o/r.git", "o/r"},
		{"  https://github.com/o/r.git  ", "o/r"},
		// Anything that is not GitHub returns empty rather than a guess,
		// so the link panel asks instead of prefilling something wrong.
		{"https://gitlab.com/o/r.git", ""},
		{"git@bitbucket.org:o/r.git", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := ParseRemote(tc.remote); got != tc.want {
			t.Errorf("ParseRemote(%q) = %q, want %q", tc.remote, got, tc.want)
		}
	}
}

func TestWritable(t *testing.T) {
	writable := []string{
		FieldText, FieldNumber, FieldDate, FieldSingleSelect, FieldIteration, FieldTitle,
	}
	for _, dt := range writable {
		if !Writable(dt) {
			t.Errorf("%s should be writable", dt)
		}
	}

	// These are rollups GitHub computes. Writing to one is rejected by
	// the API, so the UI must render them read-only.
	readOnly := []string{
		FieldAssignees, FieldLabels, FieldMilestone, FieldRepository,
		FieldReviewers, FieldLinkedPRs, FieldParentIssue, FieldSubIssues,
		FieldIssueType, FieldTrackedBy, "SOMETHING_NEW",
	}
	for _, dt := range readOnly {
		if Writable(dt) {
			t.Errorf("%s should not be writable", dt)
		}
	}
}

func TestNormalizeType(t *testing.T) {
	// The items query returns the enum form (DRAFT_ISSUE) while the
	// content union reports the type name; both must land on one spelling
	// or UpdateContent picks the wrong mutation.
	cases := map[string]string{
		"DRAFT_ISSUE":  "DraftIssue",
		"DraftIssue":   "DraftIssue",
		"ISSUE":        "Issue",
		"Issue":        "Issue",
		"PULL_REQUEST": "PullRequest",
		"PullRequest":  "PullRequest",
	}
	for in, want := range cases {
		if got := normalizeType(in); got != want {
			t.Errorf("normalizeType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitDisplay(t *testing.T) {
	got := splitDisplay("https://a/1, https://a/2")
	if len(got) != 2 || got[0] != "https://a/1" || got[1] != "https://a/2" {
		t.Errorf("splitDisplay = %v", got)
	}
	if got := splitDisplay(""); got != nil {
		t.Errorf("empty should be nil, got %v", got)
	}
}

func TestTrimFloat(t *testing.T) {
	cases := map[float64]string{5: "5", 5.5: "5.5", 5.25: "5.25", 0: "0"}
	for in, want := range cases {
		if got := trimFloat(in); got != want {
			t.Errorf("trimFloat(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFieldValueDecoding(t *testing.T) {
	// One node per writable type, checking the union member the field's
	// dataType selects is the one that gets read.
	mk := func(dataType string, n fieldValueNode) fieldValueNode {
		n.Field.ID = "f1"
		n.Field.Name = "Field"
		n.Field.DataType = dataType
		return n
	}

	text, _ := mk(FieldText, fieldValueNode{Text: "hello"}).toValue()
	if text.Text != "hello" || text.Display != "hello" {
		t.Errorf("text value = %+v", text)
	}

	n := 3.5
	num, _ := mk(FieldNumber, fieldValueNode{Number: &n}).toValue()
	if num.Number == nil || *num.Number != 3.5 || num.Display != "3.5" {
		t.Errorf("number value = %+v", num)
	}

	sel, _ := mk(FieldSingleSelect, fieldValueNode{OptionID: "o1", Name: "Doing"}).toValue()
	if sel.OptionID != "o1" || sel.OptionName != "Doing" {
		t.Errorf("select value = %+v", sel)
	}

	iter, _ := mk(FieldIteration, fieldValueNode{IterationID: "it1", Title: "Sprint 4"}).toValue()
	if iter.IterationID != "it1" || iter.Display != "Sprint 4" {
		t.Errorf("iteration value = %+v", iter)
	}

	date, _ := mk(FieldDate, fieldValueNode{Date: "2026-08-30"}).toValue()
	if date.Date != "2026-08-30" {
		t.Errorf("date value = %+v", date)
	}

	// A value whose field the query did not select is dropped rather than
	// stored under an empty id, which would collide across fields.
	if _, ok := (fieldValueNode{Text: "orphan"}).toValue(); ok {
		t.Error("a value with no field should be skipped")
	}
}

func TestFieldValueDecodingReadOnly(t *testing.T) {
	var n fieldValueNode
	n.Field.ID = "f1"
	n.Field.DataType = FieldLabels
	n.Labels.Nodes = []struct {
		Name string `json:"name"`
	}{{Name: "bug"}, {Name: "p1"}}

	v, ok := n.toValue()
	if !ok || v.Display != "bug, p1" {
		t.Errorf("labels display = %q (ok=%v)", v.Display, ok)
	}
}
