package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GitLab reads merge requests from a GitLab instance via the REST API.
type GitLab struct {
	BaseURL  string
	Username string
	Token    string
	Client   *http.Client
}

// NewGitLab builds a GitLab client.
func NewGitLab(baseURL, username, token string) *GitLab {
	return &GitLab{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Username: username,
		Token:    token,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

type glMR struct {
	IID       int    `json:"iid"`
	ProjectID int    `json:"project_id"`
	Title     string `json:"title"`
	WebURL    string `json:"web_url"`
	Draft     bool   `json:"draft"`
}

type glApprovals struct {
	ApprovedBy []struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	} `json:"approved_by"`
}

// AwaitingReview returns the user's open, non-draft, unapproved merge requests.
func (g *GitLab) AwaitingReview(ctx context.Context) ([]PullRequest, error) {
	q := url.Values{}
	q.Set("author_username", g.Username)
	q.Set("state", "opened")
	q.Set("scope", "all")
	q.Set("per_page", "100")
	endpoint := g.BaseURL + "/api/v4/merge_requests?" + q.Encode()

	var mrs []glMR
	if err := g.getJSON(ctx, endpoint, &mrs); err != nil {
		return nil, err
	}

	var out []PullRequest
	for _, mr := range mrs {
		if mr.Draft {
			continue
		}
		approved, err := g.hasApproval(ctx, mr.ProjectID, mr.IID)
		if err != nil {
			return nil, err
		}
		if approved {
			continue
		}
		out = append(out, PullRequest{URL: mr.WebURL, Title: mr.Title})
	}
	return out, nil
}

func (g *GitLab) hasApproval(ctx context.Context, projectID, iid int) (bool, error) {
	endpoint := g.BaseURL + "/api/v4/projects/" + strconv.Itoa(projectID) +
		"/merge_requests/" + strconv.Itoa(iid) + "/approvals"
	var a glApprovals
	if err := g.getJSON(ctx, endpoint, &a); err != nil {
		return false, err
	}
	return len(a.ApprovedBy) > 0, nil
}

func (g *GitLab) getJSON(ctx context.Context, endpoint string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", g.Token)
	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gitlab GET %s: %s", endpoint, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
