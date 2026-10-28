package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	http        *http.Client
	base, token string
}

func NewClient(h *http.Client, base, token string) *Client {
	if h == nil {
		h = http.DefaultClient
	}
	return &Client{h, strings.TrimRight(base, "/"), token}
}
func (c *Client) request(ctx context.Context, method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 10<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &e)
		if e.Message == "" {
			e.Message = http.StatusText(res.StatusCode)
		}
		return fmt.Errorf("github: %s: %s", res.Status, e.Message)
	}
	if out != nil && len(bytes.TrimSpace(data)) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
func (c *Client) paged(ctx context.Context, path string, out any) error {
	return c.request(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	var all []Repository
	for page := 1; ; page++ {
		var rows []Repository
		err := c.paged(ctx, "/user/repos?visibility=all&affiliation=owner,collaborator,organization_member&per_page=100&page="+strconv.Itoa(page), &rows)
		if err != nil {
			return nil, err
		}
		all = append(all, rows...)
		if len(rows) < 100 {
			break
		}
	}
	seen := map[int64]bool{}
	unique := all[:0]
	for _, r := range all {
		if !seen[r.ID] {
			seen[r.ID] = true
			unique = append(unique, r)
		}
	}
	return unique, nil
}
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string) ([]PullRequest, error) {
	var v []PullRequest
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=100", url.PathEscape(owner), url.PathEscape(repo)), &v)
	return v, err
}
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, n int) (PullRequest, error) {
	var v PullRequest
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/pulls/%d", url.PathEscape(owner), url.PathEscape(repo), n), &v)
	return v, err
}
func (c *Client) ListBranches(ctx context.Context, owner, repo string) ([]Branch, error) {
	var v []Branch
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/branches?per_page=100", owner, repo), &v)
	return v, err
}
func (c *Client) ListCollaborators(ctx context.Context, owner, repo string) ([]User, error) {
	var v []User
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/collaborators?affiliation=all&per_page=100", owner, repo), &v)
	return v, err
}
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo string, in PullRequestInput) (PullRequest, error) {
	var v PullRequest
	err := c.request(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), in, &v)
	return v, err
}
func (c *Client) RequestReviewers(ctx context.Context, owner, repo string, n int, reviewers []string) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/pulls/%d/requested_reviewers", owner, repo, n), map[string][]string{"reviewers": reviewers}, nil)
}
func (c *Client) CreateReview(ctx context.Context, owner, repo string, n int, event string) error {
	if event != "APPROVE" && event != "REQUEST_CHANGES" {
		return fmt.Errorf("invalid review event")
	}
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, n), map[string]string{"event": event}, nil)
}
func (c *Client) MergePullRequest(ctx context.Context, owner, repo string, n int, method string) error {
	if method != "merge" && method != "squash" && method != "rebase" {
		return fmt.Errorf("invalid merge method")
	}
	return c.request(ctx, http.MethodPut, fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, n), map[string]string{"merge_method": method}, nil)
}
func (c *Client) ListWorkflows(ctx context.Context, owner, repo string) ([]Workflow, error) {
	var v struct {
		Workflows []Workflow `json:"workflows"`
	}
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/actions/workflows?per_page=100", owner, repo), &v)
	return v.Workflows, err
}
func (c *Client) ListWorkflowRuns(ctx context.Context, owner, repo string) ([]WorkflowRun, error) {
	var v struct {
		WorkflowRuns []WorkflowRun `json:"workflow_runs"`
	}
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/actions/runs?per_page=100", owner, repo), &v)
	return v.WorkflowRuns, err
}
func (c *Client) GetWorkflowRun(ctx context.Context, owner, repo string, id int64) (WorkflowRun, error) {
	var v WorkflowRun
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/actions/runs/%d", owner, repo, id), &v)
	return v, err
}
func (c *Client) ListWorkflowJobs(ctx context.Context, owner, repo string, id int64) ([]Job, error) {
	var v struct {
		Jobs []Job `json:"jobs"`
	}
	err := c.paged(ctx, fmt.Sprintf("/repos/%s/%s/actions/runs/%d/jobs?per_page=100", owner, repo, id), &v)
	return v.Jobs, err
}
func (c *Client) DispatchWorkflow(ctx context.Context, owner, repo, workflow, ref string) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/dispatches", owner, repo, workflow), map[string]string{"ref": ref}, nil)
}
func (c *Client) CancelWorkflowRun(ctx context.Context, owner, repo string, id int64) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/actions/runs/%d/cancel", owner, repo, id), nil, nil)
}
func (c *Client) RerunWorkflowRun(ctx context.Context, owner, repo string, id int64) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/actions/runs/%d/rerun", owner, repo, id), nil, nil)
}
