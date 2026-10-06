package analytics

import "strings"

// Category is a user-toggleable analytics bucket. Events map to exactly one
// category so Preferences can drop whole feature areas without random sampling.
type Category string

const (
	CatLifecycle  Category = "lifecycle"
	CatOnboarding Category = "onboarding"
	CatProcesses  Category = "processes"
	CatGitDocker  Category = "git_docker"
	CatKanban     Category = "kanban"
	CatAI         Category = "ai"
	CatUpdates    Category = "updates"
	CatUIPanels   Category = "ui_panels"
)

// AllCategories is the stable order shown in Preferences.
var AllCategories = []Category{
	CatLifecycle,
	CatOnboarding,
	CatProcesses,
	CatGitDocker,
	CatKanban,
	CatAI,
	CatUpdates,
	CatUIPanels,
}

// Detail level presets. "custom" means the user overrode individual toggles.
const (
	LevelFull     = "full"
	LevelBalanced = "balanced"
	LevelMinimal  = "minimal"
	LevelNone     = "none"
	LevelCustom   = "custom"
)

// eventCategory is the explicit event → category map. Unknown names fall
// through EventCategory's prefix rules, then to ui_panels so new UI events
// cannot silently bypass a Minimal install.
var eventCategory = map[string]Category{
	"app_launched":    CatLifecycle,
	"app_closed":      CatLifecycle,
	"app_crashed":     CatLifecycle,
	"consent_decided": CatLifecycle,
	"session_start":   CatLifecycle,

	"project_created":    CatOnboarding,
	"project_deleted":    CatOnboarding,
	"project_favorited":  CatOnboarding,
	"process_added":      CatOnboarding,
	"processes_detected": CatOnboarding,
	"processes_accepted": CatOnboarding,
	"config_imported":    CatOnboarding,

	"process_started":         CatProcesses,
	"process_stopped":         CatProcesses,
	"process_crashed":         CatProcesses,
	"deps_installed":          CatProcesses,
	"logs_opened":             CatProcesses,
	"ports_viewed":            CatProcesses,
	"env_file_edited":         CatProcesses,
	"script_run":              CatProcesses,
	"script_stopped_early":    CatProcesses,
	"tests_run":               CatProcesses,
	"external_open_performed": CatProcesses,

	"git_action_performed":    CatGitDocker,
	"git_token_saved":         CatGitDocker,
	"docker_action_performed": CatGitDocker,
	"release_created":         CatGitDocker,
	"issue_submitted":         CatGitDocker,

	"task_created":   CatKanban,
	"task_moved":     CatKanban,
	"sprint_created": CatKanban,
	"roadmap_opened": CatKanban,

	"ollama_detected":             CatAI,
	"ai_model_selected":           CatAI,
	"ai_task_enriched":            CatAI,
	"ai_chat_message_sent":        CatAI,
	"ai_description_generated":    CatAI,
	"ai_commit_message_generated": CatAI,
	"ai_suggestion_accepted":      CatAI,
	"code_context_built":          CatAI,
	"code_context_searched":       CatAI,

	"update_checked":      CatUpdates,
	"update_installed":    CatUpdates,
	"update_dismissed":    CatUpdates,
	"update_notes_opened": CatUpdates,
	"banner_shown":        CatUpdates,
	"banner_clicked":      CatUpdates,
	"banner_dismissed":    CatUpdates,

	"panel_opened":   CatUIPanels,
	"theme_changed":  CatUIPanels,
	"accent_changed": CatUIPanels,
}

// EventCategory resolves which preference bucket gates an event.
func EventCategory(name string) Category {
	if c, ok := eventCategory[name]; ok {
		return c
	}
	switch {
	case strings.HasPrefix(name, "ai_"), strings.HasPrefix(name, "code_context_"), strings.HasPrefix(name, "ollama_"):
		return CatAI
	case strings.HasPrefix(name, "update_"), strings.HasPrefix(name, "banner_"):
		return CatUpdates
	case strings.HasPrefix(name, "git_"), strings.HasPrefix(name, "docker_"), strings.HasPrefix(name, "release_"):
		return CatGitDocker
	case strings.HasPrefix(name, "process_"), strings.HasPrefix(name, "script_"), strings.HasPrefix(name, "tests_"), strings.HasPrefix(name, "deps_"), strings.HasPrefix(name, "logs_"), strings.HasPrefix(name, "ports_"), strings.HasPrefix(name, "env_"):
		return CatProcesses
	case strings.HasPrefix(name, "task_"), strings.HasPrefix(name, "sprint_"), strings.HasPrefix(name, "roadmap_"):
		return CatKanban
	case strings.HasPrefix(name, "project_"), strings.HasPrefix(name, "config_"), strings.HasPrefix(name, "processes_"):
		return CatOnboarding
	case strings.HasPrefix(name, "app_"), name == "consent_decided":
		return CatLifecycle
	}
	return CatUIPanels
}

// DefaultCategories returns Full — every category enabled.
func DefaultCategories() map[string]bool {
	out := make(map[string]bool, len(AllCategories))
	for _, c := range AllCategories {
		out[string(c)] = true
	}
	return out
}

// CategoriesForLevel expands a named preset. Unknown levels behave like Full.
// LevelCustom is not a preset; callers should keep the existing map.
func CategoriesForLevel(level string) map[string]bool {
	cats := DefaultCategories()
	switch NormalizeDetailLevel(level) {
	case LevelBalanced:
		cats[string(CatUIPanels)] = false
	case LevelMinimal:
		for _, c := range AllCategories {
			cats[string(c)] = c == CatLifecycle || c == CatUpdates
		}
	case LevelNone:
		for _, c := range AllCategories {
			cats[string(c)] = false
		}
	}
	return cats
}

// NormalizeDetailLevel coerces an arbitrary string to a known level.
func NormalizeDetailLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case LevelBalanced:
		return LevelBalanced
	case LevelMinimal:
		return LevelMinimal
	case LevelNone:
		return LevelNone
	case LevelCustom:
		return LevelCustom
	default:
		return LevelFull
	}
}

// InferDetailLevel returns the preset that matches cats, or custom.
func InferDetailLevel(cats map[string]bool) string {
	for _, level := range []string{LevelFull, LevelBalanced, LevelMinimal, LevelNone} {
		want := CategoriesForLevel(level)
		if categoriesEqual(want, cats) {
			return level
		}
	}
	return LevelCustom
}

func categoriesEqual(a, b map[string]bool) bool {
	for _, c := range AllCategories {
		key := string(c)
		if boolOr(a, key) != boolOr(b, key) {
			return false
		}
	}
	return true
}

func boolOr(m map[string]bool, key string) bool {
	if m == nil {
		return true // missing key defaults on
	}
	v, ok := m[key]
	if !ok {
		return true
	}
	return v
}

// MergeCategories fills missing keys with true (default on) and drops unknowns.
func MergeCategories(in map[string]bool) map[string]bool {
	out := DefaultCategories()
	if in == nil {
		return out
	}
	for _, c := range AllCategories {
		key := string(c)
		if v, ok := in[key]; ok {
			out[key] = v
		}
	}
	return out
}

// EnvKeyDiff reports how many keys were added/removed between two env maps.
// Values are ignored — only key sets matter for analytics.
func EnvKeyDiff(before, after map[string]string) (varCount, added, removed int) {
	varCount = len(after)
	for k := range after {
		if _, ok := before[k]; !ok {
			added++
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			removed++
		}
	}
	return varCount, added, removed
}
