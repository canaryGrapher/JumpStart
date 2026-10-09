package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EnrichPayload is the structured fill for one task/story.
type EnrichPayload struct {
	Description string
	Acceptance  []string
	Subtasks    []string
	Priority    string
	Labels      []string
	StoryPoints int
	DueDate     string // YYYY-MM-DD, or "" when the model gave none or an invalid one
}

// StoryPayload is one user story proposed by the chat assistant.
type StoryPayload struct {
	Title       string
	Description string
	Acceptance  []string
	Priority    string
	Labels      []string
	StoryPoints int
	DueDate     string
	Tasks       []TaskPayload
}

// TaskPayload is a child task under a generated story.
type TaskPayload struct {
	Title string
}

// ChatPayload is the assistant's reply plus any drafted stories.
type ChatPayload struct {
	Reply   string
	Stories []StoryPayload
}

// ParseEnrich decodes a model JSON blob into EnrichPayload, tolerating
// common shape drift (string instead of array, mixed types, code fences).
func ParseEnrich(raw string) (EnrichPayload, error) {
	var out EnrichPayload
	obj, err := decodeObject(raw)
	if err != nil {
		return out, err
	}
	out.Description = asString(obj["description"])
	out.Acceptance = asStringSlice(obj["acceptance"])
	out.Subtasks = asStringSlice(obj["subtasks"])
	out.Priority = normPriority(asString(obj["priority"]))
	out.Labels = asStringSlice(obj["labels"])
	out.StoryPoints = asInt(obj["storyPoints"])
	if out.StoryPoints == 0 {
		out.StoryPoints = asInt(obj["story_points"])
	}
	out.DueDate = normDueDate(firstNonEmpty(asString(obj["dueDate"]), asString(obj["due_date"])))
	if out.Description == "" && len(out.Acceptance) == 0 && len(out.Subtasks) == 0 {
		return out, fmt.Errorf("model returned no usable fields")
	}
	return out, nil
}

// ParseChat decodes a chat turn. On total failure it returns the raw text
// as Reply so the UI still shows something.
func ParseChat(raw string) ChatPayload {
	obj, err := decodeObject(raw)
	if err != nil {
		return ChatPayload{Reply: strings.TrimSpace(raw)}
	}
	out := ChatPayload{Reply: asString(obj["reply"])}
	for _, item := range asObjectSlice(obj["stories"]) {
		out.Stories = append(out.Stories, parseStory(item))
	}
	if out.Reply == "" && len(out.Stories) == 0 {
		out.Reply = strings.TrimSpace(raw)
	}
	return out
}

func parseStory(obj map[string]any) StoryPayload {
	s := StoryPayload{
		Title:       firstNonEmpty(asString(obj["title"]), asString(obj["name"])),
		Description: asString(obj["description"]),
		Acceptance:  asStringSlice(obj["acceptance"]),
		Priority:    normPriority(asString(obj["priority"])),
		Labels:      asStringSlice(obj["labels"]),
		StoryPoints: asInt(obj["storyPoints"]),
		DueDate:     normDueDate(firstNonEmpty(asString(obj["dueDate"]), asString(obj["due_date"]))),
	}
	if s.StoryPoints == 0 {
		s.StoryPoints = asInt(obj["story_points"])
	}
	if rawTasks := obj["tasks"]; rawTasks != nil {
		if objs := asObjectSlice(rawTasks); len(objs) > 0 {
			for _, t := range objs {
				title := firstNonEmpty(
					asString(t["title"]),
					asString(t["name"]),
					asString(t["description"]),
				)
				if title != "" {
					s.Tasks = append(s.Tasks, TaskPayload{Title: title})
				}
			}
		} else {
			// Bare string list: ["Do A", "Do B"]
			for _, title := range asStringSlice(rawTasks) {
				if title != "" {
					s.Tasks = append(s.Tasks, TaskPayload{Title: title})
				}
			}
		}
	}
	return s
}

// normDueDate keeps only a real YYYY-MM-DD date; models often return prose
// such as "next Friday" or an impossible date, which must not reach the board.
func normDueDate(s string) string {
	s = strings.TrimSpace(s)
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ""
	}
	return s
}

func decodeObject(raw string) (map[string]any, error) {
	s := ExtractJSON(raw)
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("empty model output")
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil, fmt.Errorf("could not parse model output: %w", err)
	}
	return obj, nil
}

// ExtractJSON pulls the first {...} block out of a model response, in
// case the model wrapped it in code fences or stray prose.
func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	case bool:
		if t {
			return "true"
		}
		return "false"
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		var n int
		fmt.Sscanf(strings.TrimSpace(t), "%d", &n)
		return n
	default:
		return 0
	}
}

func asStringSlice(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "true" || s == "false" {
			return nil
		}
		// "a, b, c" or single criterion
		if strings.Contains(s, "\n") {
			var out []string
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
				line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
				if line != "" {
					out = append(out, line)
				}
			}
			return out
		}
		if strings.Contains(s, ",") {
			parts := strings.Split(s, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				if p = strings.TrimSpace(p); p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return []string{s}
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			switch x := item.(type) {
			case string:
				if s := strings.TrimSpace(x); s != "" {
					out = append(out, s)
				}
			case map[string]any:
				if s := firstNonEmpty(asString(x["title"]), asString(x["text"]), asString(x["description"])); s != "" {
					out = append(out, s)
				}
			default:
				if s := asString(item); s != "" && s != "true" && s != "false" {
					out = append(out, s)
				}
			}
		}
		return out
	case bool:
		return nil
	default:
		return nil
	}
}

func asObjectSlice(v any) []map[string]any {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func normPriority(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "low", "l", "p3":
		return "low"
	case "medium", "med", "m", "normal", "p2":
		return "medium"
	case "high", "h", "urgent", "critical", "p1", "p0":
		return "high"
	default:
		return ""
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
