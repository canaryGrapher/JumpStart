package github

import (
	"context"
	"fmt"
	"strings"
)

type projectNode struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	ShortDesc string `json:"shortDescription"`
	Closed    bool   `json:"closed"`
	Public    bool   `json:"public"`
	UpdatedAt string `json:"updatedAt"`
}

type projectConn struct {
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Nodes []projectNode `json:"nodes"`
}

func (n projectNode) toProject(owner, ownerType string) Project {
	return Project{
		ID: n.ID, Number: n.Number, Title: n.Title, URL: n.URL,
		ShortDesc: n.ShortDesc, Closed: n.Closed, Public: n.Public,
		Owner: owner, OwnerType: ownerType, UpdatedAt: n.UpdatedAt,
	}
}

// ListViewerProjects returns the boards owned by the signed-in account.
func (c *Client) ListViewerProjects(ctx context.Context) ([]Project, error) {
	var out []Project
	cursor := ""
	for {
		var resp struct {
			Viewer struct {
				Login      string      `json:"login"`
				ProjectsV2 projectConn `json:"projectsV2"`
			} `json:"viewer"`
		}
		vars := map[string]any{"cursor": nullable(cursor)}
		if err := c.Query(ctx, queryViewerProjects, vars, &resp); err != nil {
			return nil, err
		}
		for _, n := range resp.Viewer.ProjectsV2.Nodes {
			out = append(out, n.toProject(resp.Viewer.Login, "user"))
		}
		if !resp.Viewer.ProjectsV2.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Viewer.ProjectsV2.PageInfo.EndCursor
	}
}

// ListOwnerProjects returns the boards owned by a specific login. The
// query asks for both a user and an organization by that name; only one
// of the two comes back non-null.
func (c *Client) ListOwnerProjects(ctx context.Context, login string) ([]Project, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return c.ListViewerProjects(ctx)
	}
	var out []Project
	cursor := ""
	for {
		var resp struct {
			User *struct {
				ProjectsV2 projectConn `json:"projectsV2"`
			} `json:"user"`
			Organization *struct {
				ProjectsV2 projectConn `json:"projectsV2"`
			} `json:"organization"`
		}
		vars := map[string]any{"login": login, "cursor": nullable(cursor)}
		if err := c.Query(ctx, queryOwnerProjects, vars, &resp); err != nil {
			return nil, err
		}
		conn := projectConn{}
		ownerType := ""
		switch {
		case resp.Organization != nil:
			conn, ownerType = resp.Organization.ProjectsV2, "organization"
		case resp.User != nil:
			conn, ownerType = resp.User.ProjectsV2, "user"
		default:
			return out, nil
		}
		for _, n := range conn.Nodes {
			out = append(out, n.toProject(login, ownerType))
		}
		if !conn.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = conn.PageInfo.EndCursor
	}
}

// GetProject fetches one board with its full field definitions.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	var resp struct {
		Node struct {
			projectNode
			Owner struct {
				Typename string `json:"__typename"`
				Login    string `json:"login"`
			} `json:"owner"`
			Fields struct {
				Nodes []fieldNode `json:"nodes"`
			} `json:"fields"`
		} `json:"node"`
	}
	if err := c.Query(ctx, queryProjectFields, map[string]any{"id": projectID}, &resp); err != nil {
		return nil, err
	}
	if resp.Node.ID == "" {
		return nil, fmt.Errorf("project not found, or the token cannot see it")
	}
	ownerType := "user"
	if strings.EqualFold(resp.Node.Owner.Typename, "Organization") {
		ownerType = "organization"
	}
	p := resp.Node.projectNode.toProject(resp.Node.Owner.Login, ownerType)
	for _, f := range resp.Node.Fields.Nodes {
		if f.ID == "" {
			continue
		}
		p.Fields = append(p.Fields, f.toField())
	}
	return &p, nil
}

type fieldNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"dataType"`
	Options  []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	} `json:"options"`
	Configuration struct {
		Iterations          []iterationNode `json:"iterations"`
		CompletedIterations []iterationNode `json:"completedIterations"`
	} `json:"configuration"`
}

type iterationNode struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
	Duration  int    `json:"duration"`
}

func (f fieldNode) toField() Field {
	out := Field{ID: f.ID, Name: f.Name, DataType: f.DataType, Writable: Writable(f.DataType)}
	for _, o := range f.Options {
		out.Options = append(out.Options, SelectOption{
			ID: o.ID, Name: o.Name, Color: o.Color, Description: o.Description,
		})
	}
	for _, it := range f.Configuration.Iterations {
		out.Iterations = append(out.Iterations, Iteration{
			ID: it.ID, Title: it.Title, StartDate: it.StartDate, Duration: it.Duration,
		})
	}
	for _, it := range f.Configuration.CompletedIterations {
		out.Iterations = append(out.Iterations, Iteration{
			ID: it.ID, Title: it.Title, StartDate: it.StartDate,
			Duration: it.Duration, Completed: true,
		})
	}
	return out
}

// nullable turns an empty cursor into a JSON null, which GraphQL needs
// for the first page of a connection.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
