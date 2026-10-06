package model

// GitHubLink ties one local Task to its counterpart on a GitHub
// Projects v2 board. A task is linked once and keeps the link for life:
// ItemID is the board row, ContentID is the issue or draft behind it.
type GitHubLink struct {
	ItemID      string `json:"itemId,omitempty"`      // PVTI_… project item node id
	ContentID   string `json:"contentId,omitempty"`   // issue / PR / draft node id
	ContentType string `json:"contentType,omitempty"` // DraftIssue | Issue | PullRequest
	Number      int    `json:"number,omitempty"`      // issue number, 0 for drafts
	URL         string `json:"url,omitempty"`
	Repo        string `json:"repo,omitempty"`  // owner/name
	State       string `json:"state,omitempty"` // OPEN | CLOSED | MERGED

	// RemoteUpdatedAt is the item's updatedAt as GitHub last reported it,
	// in unix ms. Used with SyncedAt to see whether each side moved.
	RemoteUpdatedAt int64 `json:"remoteUpdatedAt,omitempty"`
	// SyncedAt is when this task last reconciled cleanly, unix ms.
	SyncedAt int64 `json:"syncedAt,omitempty"`
	// Conflict marks a task where local pending edits and GitHub both
	// changed to different field values. The UI shows per-field reasons.
	Conflict bool `json:"conflict,omitempty"`
	// ConflictFields lists each diverged field (local vs GitHub) so the
	// user can see exactly why the badge is up. Cleared with Conflict.
	ConflictFields []ConflictField `json:"conflictFields,omitempty"`
	// Pending marks a local edit that has not reached GitHub yet.
	Pending bool `json:"pending,omitempty"`
	// ForcePush is set by "Overwrite GitHub" so the next pass pushes
	// local even when the remote timestamp looks newer.
	ForcePush bool `json:"forcePush,omitempty"`
}

// ConflictField is one atomic reason a task conflicted with GitHub.
type ConflictField struct {
	Field  string `json:"field"`  // title|status|assignee|labels|body|storyPoints|field:<id>
	Label  string `json:"label"`  // human-readable field name
	Local  string `json:"local"`  // local value as shown to the user
	Remote string `json:"remote"` // GitHub value as shown to the user
}

// FieldValue is one Projects v2 field value on a task. Only the member
// matching the field's data type is populated; the rest stay zero.
//
// This single shape covers every Projects v2 field kind, so adding a
// custom field to the board needs no new Go type: TEXT, NUMBER, DATE,
// SINGLE_SELECT, ITERATION, and the read-only built-ins (ASSIGNEES,
// LABELS, MILESTONE, REPOSITORY, LINKED_PULL_REQUESTS, REVIEWERS,
// PARENT_ISSUE, SUB_ISSUES_PROGRESS, ISSUE_TYPE, TRACKED_BY).
type FieldValue struct {
	FieldID   string   `json:"fieldId"`
	Name      string   `json:"name"`               // field name as shown on the board
	DataType  string   `json:"dataType"`           // see github.FieldType
	Text      string   `json:"text,omitempty"`     // TEXT
	Number    *float64 `json:"number,omitempty"`   // NUMBER
	Date      string   `json:"date,omitempty"`     // DATE, YYYY-MM-DD
	OptionID  string   `json:"optionId,omitempty"` // SINGLE_SELECT
	OptionKey string   `json:"optionKey,omitempty"`
	Iteration string   `json:"iterationId,omitempty"` // ITERATION
	Users     []string `json:"users,omitempty"`       // ASSIGNEES, REVIEWERS
	Labels    []string `json:"labels,omitempty"`      // LABELS
	Display   string   `json:"display,omitempty"`     // rendered value for read-only fields
}

// GitHubSync is a project's board linkage and sync preferences. It lives
// on model.Project so each JumpStart project binds to its own board.
type GitHubSync struct {
	Enabled bool `json:"enabled"`

	// Owner is the org or user login that owns the board; OwnerType is
	// "organization" or "user".
	Owner     string `json:"owner,omitempty"`
	OwnerType string `json:"ownerType,omitempty"`

	ProjectID     string `json:"projectId,omitempty"` // PVT_… node id
	ProjectNumber int    `json:"projectNumber,omitempty"`
	ProjectTitle  string `json:"projectTitle,omitempty"`
	ProjectURL    string `json:"projectUrl,omitempty"`

	// Repo is where new issues are opened when a task is promoted from a
	// draft to a real issue. Format "owner/name".
	Repo string `json:"repo,omitempty"`
	// RepoID is that repository's node id, cached to save a lookup.
	RepoID string `json:"repoId,omitempty"`

	// StatusFieldID is the single-select field the Kanban columns map to,
	// and StatusMap maps a local column id to a select option id.
	StatusFieldID string            `json:"statusFieldId,omitempty"`
	StatusMap     map[string]string `json:"statusMap,omitempty"`

	// Direction is "both", "pull", or "push". Defaults to "both".
	Direction string `json:"direction,omitempty"`
	// PollSeconds overrides the focused polling interval when non-zero.
	PollSeconds int `json:"pollSeconds,omitempty"`
	// CreateAsIssue opens a real issue instead of a draft for new tasks.
	CreateAsIssue bool `json:"createAsIssue,omitempty"`

	LastSyncAt    int64  `json:"lastSyncAt,omitempty"`
	LastSyncError string `json:"lastSyncError,omitempty"`

	// PendingDeletes are Projects v2 item IDs that were removed locally
	// and still need deleteProjectV2Item on GitHub. Kept across passes
	// so a failed delete cannot resurrect the card on the next pull.
	PendingDeletes []string `json:"pendingDeletes,omitempty"`
}
