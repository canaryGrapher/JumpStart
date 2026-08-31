package github

import "context"

// RemoteBranch is one branch as GitHub currently sees it, live — this is
// what tells the pull request form whether a branch has actually been
// pushed, as opposed to what the local clone last fetched.
type RemoteBranch struct {
	Name          string `json:"name"`
	OID           string `json:"oid"`
	CommittedDate string `json:"committedDate,omitempty"`
}

// ListBranches lists a repository's branches on GitHub.
func (c *Client) ListBranches(ctx context.Context, owner, name string) ([]RemoteBranch, error) {
	var out []RemoteBranch
	cursor := ""
	for {
		var resp struct {
			Repository struct {
				Refs struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						Name   string `json:"name"`
						Target struct {
							OID           string `json:"oid"`
							CommittedDate string `json:"committedDate"`
						} `json:"target"`
					} `json:"nodes"`
				} `json:"refs"`
			} `json:"repository"`
		}
		vars := map[string]any{"owner": owner, "name": name, "cursor": nullable(cursor)}
		if err := c.Query(ctx, queryRepoBranches, vars, &resp); err != nil {
			return nil, err
		}
		for _, n := range resp.Repository.Refs.Nodes {
			out = append(out, RemoteBranch{Name: n.Name, OID: n.Target.OID, CommittedDate: n.Target.CommittedDate})
		}
		if !resp.Repository.Refs.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Repository.Refs.PageInfo.EndCursor
	}
}
