package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"devdeck/internal/model"
)

// extendedFile shadows the fields older JumpStart builds do not know about
// (due dates, links, attachments, project quarters). An older build reads
// config.json into structs without these fields and writes it back without
// them. Only a build that knows the fields can clear them, and it rewrites
// this shadow on every save, so a field present here but missing from
// config.json was dropped by an older writer and is restored on load.
const extendedFile = "extended-fields.json"

type taskExt struct {
	DueDate     string             `json:"dueDate,omitempty"`
	Links       []model.TaskLink   `json:"links,omitempty"`
	Attachments []model.Attachment `json:"attachments,omitempty"`
}

type projectExt struct {
	Quarters []model.QuarterRange `json:"quarters,omitempty"`
}

type extended struct {
	Version  int                   `json:"version"`
	Projects map[string]projectExt `json:"projects"`
	Tasks    map[string]taskExt    `json:"tasks"` // key: projectID + "/" + taskID
}

func (s *Store) extendedPath() string {
	return filepath.Join(filepath.Dir(s.path), extendedFile)
}

func buildExtended(projects []model.Project) extended {
	ext := extended{Version: 1, Projects: map[string]projectExt{}, Tasks: map[string]taskExt{}}
	for _, p := range projects {
		if len(p.Quarters) > 0 {
			ext.Projects[p.ID] = projectExt{Quarters: p.Quarters}
		}
		for _, t := range p.Tasks {
			if t.DueDate == "" && len(t.Links) == 0 && len(t.Attachments) == 0 {
				continue
			}
			ext.Tasks[p.ID+"/"+t.ID] = taskExt{DueDate: t.DueDate, Links: t.Links, Attachments: t.Attachments}
		}
	}
	return ext
}

func (s *Store) writeExtended(projects []model.Project) error {
	data, err := json.MarshalIndent(buildExtended(projects), "", "  ")
	if err != nil {
		return err
	}
	tmp := s.extendedPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.extendedPath())
}

// restoreExtended puts back fields an older writer dropped. It returns how
// many fields were restored. A missing or unreadable shadow restores nothing.
func (s *Store) restoreExtended(projects []model.Project) int {
	data, err := os.ReadFile(s.extendedPath())
	if err != nil {
		return 0
	}
	var ext extended
	if json.Unmarshal(data, &ext) != nil {
		return 0
	}
	restored := 0
	for i := range projects {
		p := &projects[i]
		if pe, ok := ext.Projects[p.ID]; ok && len(p.Quarters) == 0 && len(pe.Quarters) > 0 {
			p.Quarters = pe.Quarters
			restored++
		}
		for j := range p.Tasks {
			t := &p.Tasks[j]
			te, ok := ext.Tasks[p.ID+"/"+t.ID]
			if !ok {
				continue
			}
			if t.DueDate == "" && te.DueDate != "" {
				t.DueDate = te.DueDate
				restored++
			}
			if len(t.Links) == 0 && len(te.Links) > 0 {
				t.Links = te.Links
				restored++
			}
			if len(t.Attachments) == 0 && len(te.Attachments) > 0 {
				t.Attachments = te.Attachments
				restored++
			}
		}
	}
	return restored
}
