package github

import (
	"context"
	"fmt"
	"strings"
)

// fieldValueNode is the union GitHub returns for one field value on one
// item. Only the members matching the concrete type are populated, so
// toValue picks by the field's dataType rather than by guessing.
type fieldValueNode struct {
	Field struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		DataType string `json:"dataType"`
	} `json:"field"`

	Text        string   `json:"text"`
	Number      *float64 `json:"number"`
	Date        string   `json:"date"`
	OptionID    string   `json:"optionId"`
	Name        string   `json:"name"`
	IterationID string   `json:"iterationId"`
	Title       string   `json:"title"`

	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	Repository *struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Users struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"users"`
	PullRequests struct {
		Nodes []struct {
			URL string `json:"url"`
		} `json:"nodes"`
	} `json:"pullRequests"`
	Reviewers struct {
		Nodes []struct {
			Login string `json:"login"`
			Name  string `json:"name"`
		} `json:"nodes"`
	} `json:"reviewers"`
}

// toValue flattens one union member into an ItemFieldValue. It reports
// false for values whose field could not be identified, which happens
// for union members the query does not select.
func (n fieldValueNode) toValue() (ItemFieldValue, bool) {
	if n.Field.ID == "" {
		return ItemFieldValue{}, false
	}
	v := ItemFieldValue{
		FieldID:   n.Field.ID,
		FieldName: n.Field.Name,
		DataType:  n.Field.DataType,
	}
	switch n.Field.DataType {
	case FieldText, FieldTitle:
		v.Text = n.Text
		v.Display = n.Text
	case FieldNumber:
		v.Number = n.Number
		if n.Number != nil {
			v.Display = trimFloat(*n.Number)
		}
	case FieldDate:
		v.Date = n.Date
		v.Display = n.Date
	case FieldSingleSelect:
		v.OptionID = n.OptionID
		v.OptionName = n.Name
		v.Display = n.Name
	case FieldIteration:
		v.IterationID = n.IterationID
		v.Display = n.Title
	case FieldMilestone:
		if n.Milestone != nil {
			v.Display = n.Milestone.Title
		}
	case FieldRepository:
		if n.Repository != nil {
			v.Display = n.Repository.NameWithOwner
		}
	case FieldLabels:
		names := make([]string, 0, len(n.Labels.Nodes))
		for _, l := range n.Labels.Nodes {
			names = append(names, l.Name)
		}
		v.Display = strings.Join(names, ", ")
	case FieldAssignees:
		logins := make([]string, 0, len(n.Users.Nodes))
		for _, u := range n.Users.Nodes {
			logins = append(logins, u.Login)
		}
		v.Display = strings.Join(logins, ", ")
	case FieldLinkedPRs, FieldTrackedBy:
		urls := make([]string, 0, len(n.PullRequests.Nodes))
		for _, p := range n.PullRequests.Nodes {
			urls = append(urls, p.URL)
		}
		v.Display = strings.Join(urls, ", ")
	case FieldReviewers:
		names := make([]string, 0, len(n.Reviewers.Nodes))
		for _, r := range n.Reviewers.Nodes {
			names = append(names, firstNonEmpty(r.Login, r.Name))
		}
		v.Display = strings.Join(names, ", ")
	default:
		// Unknown or future field types still carry whatever scalar came
		// back, so the card can show something rather than nothing.
		v.Display = firstNonEmpty(n.Text, n.Name, n.Title, n.Date)
	}
	return v, true
}

// SetText sets a TEXT field.
func (c *Client) SetText(ctx context.Context, projectID, itemID, fieldID, text string) error {
	return c.setValue(ctx, projectID, itemID, fieldID, map[string]any{"text": text})
}

// SetNumber sets a NUMBER field.
func (c *Client) SetNumber(ctx context.Context, projectID, itemID, fieldID string, n float64) error {
	return c.setValue(ctx, projectID, itemID, fieldID, map[string]any{"number": n})
}

// SetDate sets a DATE field. date is YYYY-MM-DD.
func (c *Client) SetDate(ctx context.Context, projectID, itemID, fieldID, date string) error {
	return c.setValue(ctx, projectID, itemID, fieldID, map[string]any{"date": date})
}

// SetSingleSelect sets a SINGLE_SELECT field to one of its options. This
// is what moving a card between Kanban columns calls.
func (c *Client) SetSingleSelect(ctx context.Context, projectID, itemID, fieldID, optionID string) error {
	return c.setValue(ctx, projectID, itemID, fieldID, map[string]any{"singleSelectOptionId": optionID})
}

// SetIteration assigns the item to a cycle of an ITERATION field.
func (c *Client) SetIteration(ctx context.Context, projectID, itemID, fieldID, iterationID string) error {
	return c.setValue(ctx, projectID, itemID, fieldID, map[string]any{"iterationId": iterationID})
}

// ClearField removes any value from a field on an item.
func (c *Client) ClearField(ctx context.Context, projectID, itemID, fieldID string) error {
	vars := map[string]any{"projectId": projectID, "itemId": itemID, "fieldId": fieldID}
	return c.Query(ctx, mutationClearFieldValue, vars, nil)
}

// SetValue dispatches to the right mutation for a field's data type. An
// empty value clears the field instead of writing a blank one.
func (c *Client) SetValue(ctx context.Context, projectID, itemID string, f Field, v ItemFieldValue) error {
	if !Writable(f.DataType) {
		return fmt.Errorf("%q is read-only on GitHub", f.Name)
	}
	switch f.DataType {
	case FieldText, FieldTitle:
		if v.Text == "" {
			return c.ClearField(ctx, projectID, itemID, f.ID)
		}
		return c.SetText(ctx, projectID, itemID, f.ID, v.Text)
	case FieldNumber:
		if v.Number == nil {
			return c.ClearField(ctx, projectID, itemID, f.ID)
		}
		return c.SetNumber(ctx, projectID, itemID, f.ID, *v.Number)
	case FieldDate:
		if v.Date == "" {
			return c.ClearField(ctx, projectID, itemID, f.ID)
		}
		return c.SetDate(ctx, projectID, itemID, f.ID, v.Date)
	case FieldSingleSelect:
		if v.OptionID == "" {
			return c.ClearField(ctx, projectID, itemID, f.ID)
		}
		return c.SetSingleSelect(ctx, projectID, itemID, f.ID, v.OptionID)
	case FieldIteration:
		if v.IterationID == "" {
			return c.ClearField(ctx, projectID, itemID, f.ID)
		}
		return c.SetIteration(ctx, projectID, itemID, f.ID, v.IterationID)
	}
	return fmt.Errorf("unsupported field type %q", f.DataType)
}

func (c *Client) setValue(ctx context.Context, projectID, itemID, fieldID string, value map[string]any) error {
	vars := map[string]any{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"value":     value,
	}
	return c.Query(ctx, mutationSetFieldValue, vars, nil)
}

func trimFloat(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
