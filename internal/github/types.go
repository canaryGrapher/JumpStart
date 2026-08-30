package github

// FieldType enumerates every Projects v2 field data type. The writable
// ones can be set from JumpStart; the rest are read-only rollups GitHub
// computes and we mirror onto the card.
const (
	FieldText         = "TEXT"
	FieldNumber       = "NUMBER"
	FieldDate         = "DATE"
	FieldSingleSelect = "SINGLE_SELECT"
	FieldIteration    = "ITERATION"

	FieldTitle       = "TITLE"
	FieldAssignees   = "ASSIGNEES"
	FieldLabels      = "LABELS"
	FieldMilestone   = "MILESTONE"
	FieldRepository  = "REPOSITORY"
	FieldReviewers   = "REVIEWERS"
	FieldLinkedPRs   = "LINKED_PULL_REQUESTS"
	FieldParentIssue = "PARENT_ISSUE"
	FieldSubIssues   = "SUB_ISSUES_PROGRESS"
	FieldIssueType   = "ISSUE_TYPE"
	FieldTrackedBy   = "TRACKED_BY"
)

// Writable reports whether a field type accepts updateProjectV2ItemFieldValue.
func Writable(dataType string) bool {
	switch dataType {
	case FieldText, FieldNumber, FieldDate, FieldSingleSelect, FieldIteration, FieldTitle:
		return true
	}
	return false
}

// SelectOption is one choice on a single-select field, with the colour
// and description GitHub shows next to it.
type SelectOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

// Iteration is one cycle of an iteration field.
type Iteration struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartDate string `json:"startDate,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Completed bool   `json:"completed,omitempty"`
}

// Field is one column definition on the board, with everything the UI
// needs to render an editor for it.
type Field struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	DataType   string         `json:"dataType"`
	Options    []SelectOption `json:"options,omitempty"`
	Iterations []Iteration    `json:"iterations,omitempty"`
	Writable   bool           `json:"writable"`
}

// Project is a Projects v2 board.
type Project struct {
	ID        string  `json:"id"`
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	URL       string  `json:"url"`
	ShortDesc string  `json:"shortDescription,omitempty"`
	Closed    bool    `json:"closed"`
	Public    bool    `json:"public"`
	Owner     string  `json:"owner,omitempty"`
	OwnerType string  `json:"ownerType,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
	UpdatedAt string  `json:"updatedAt,omitempty"`
}

// Item is one row on the board: a draft, an issue, or a pull request,
// with the values of every field on it.
type Item struct {
	ID          string   `json:"id"`
	ContentID   string   `json:"contentId,omitempty"`
	ContentType string   `json:"contentType"` // DraftIssue | Issue | PullRequest
	Number      int      `json:"number,omitempty"`
	Title       string   `json:"title"`
	Body        string   `json:"body,omitempty"`
	URL         string   `json:"url,omitempty"`
	State       string   `json:"state,omitempty"`
	Repo        string   `json:"repo,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Reviewers   []string `json:"reviewers,omitempty"`
	Milestone   string   `json:"milestone,omitempty"`
	IssueType   string   `json:"issueType,omitempty"`
	ParentTitle string   `json:"parentTitle,omitempty"`
	LinkedPRs   []string `json:"linkedPrs,omitempty"`
	SubIssues   string   `json:"subIssues,omitempty"`
	IsArchived  bool     `json:"isArchived,omitempty"`
	UpdatedAt   string   `json:"updatedAt,omitempty"`

	// Values holds every field value on the item, keyed by field id.
	Values map[string]ItemFieldValue `json:"values,omitempty"`
}

// ItemFieldValue is one field's value on one item, in the shape the
// sync layer copies into model.FieldValue.
type ItemFieldValue struct {
	FieldID     string   `json:"fieldId"`
	FieldName   string   `json:"fieldName"`
	DataType    string   `json:"dataType"`
	Text        string   `json:"text,omitempty"`
	Number      *float64 `json:"number,omitempty"`
	Date        string   `json:"date,omitempty"`
	OptionID    string   `json:"optionId,omitempty"`
	OptionName  string   `json:"optionName,omitempty"`
	IterationID string   `json:"iterationId,omitempty"`
	Display     string   `json:"display,omitempty"`
}

// Repository is a repo the token can open issues in.
type Repository struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	FullName string `json:"fullName"`
	URL      string `json:"url"`
}
