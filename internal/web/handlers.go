package web

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/gitui/internal/github"
)

type Handler struct {
	client *github.Client
	secret string
}

func NewHandler(client *github.Client) http.Handler {
	return &Handler{client: client, secret: "gitui-dev-secret"}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Trim(strings.TrimPrefix(r.URL.Path, "/"), "/")
	segs := []string{}
	if p != "" {
		segs = strings.Split(p, "/")
	}
	token := ensureCSRF(w, r)
	// strip leading "repos"
	if len(segs) > 0 && segs[0] == "repos" {
		segs = segs[1:]
	}
	switch {
	case len(segs) == 0:
		h.repos(w, r)
	case len(segs) == 3 && segs[2] == "pulls":
		h.pulls(w, r, segs[0], segs[1])
	case len(segs) == 4 && segs[2] == "pulls":
		n, _ := strconv.Atoi(segs[3])
		if r.Method == http.MethodPost {
			h.reviewOrMerge(w, r, segs[0], segs[1], n, token)
			return
		}
		h.pullDetail(w, r, segs[0], segs[1], n)
	case len(segs) == 3 && segs[2] == "actions":
		h.actions(w, r, segs[0], segs[1])
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) repos(w http.ResponseWriter, r *http.Request) {
	repos, err := h.client.ListRepositories(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<!doctype html><html><body><h1>Repositories</h1><ul>")
	for _, repo := range repos {
		fmt.Fprintf(w, "<li><a href=\"/repos/%s/%s/pulls\">%s</a></li>", repo.Owner, repo.Name, template.HTMLEscapeString(repo.FullName))
	}
	fmt.Fprint(w, "</ul></body></html>")
}

func (h *Handler) pulls(w http.ResponseWriter, r *http.Request, owner, repo string) {
	prs, err := h.client.ListPullRequests(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><html><body><h1>Pull Requests for %s/%s</h1><ul>", owner, repo)
	for _, pr := range prs {
		fmt.Fprintf(w, "<li><a href=\"/repos/%s/%s/pulls/%d\">%s</a></li>", owner, repo, pr.Number, template.HTMLEscapeString(pr.Title))
	}
	fmt.Fprint(w, "</ul></body></html>")
}

func (h *Handler) pullDetail(w http.ResponseWriter, r *http.Request, owner, repo string, n int) {
	pr, err := h.client.GetPullRequest(r.Context(), owner, repo, n)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	csrfToken := ensureCSRF(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><html><body><h1>#%d %s</h1><form method=\"post\" action=\"/repos/%s/%s/pulls/%d/review\"><input type=\"hidden\" name=\"csrf\" value=\"%s\"><button name=\"event\" value=\"APPROVE\">Approve</button></form></body></html>", pr.Number, template.HTMLEscapeString(pr.Title), owner, repo, n, csrfToken)
}

func (h *Handler) reviewOrMerge(w http.ResponseWriter, r *http.Request, owner, repo string, n int, token string) {
	if !validateCSRF(r, r.FormValue("csrf")) {
		http.Error(w, "invalid csrf", http.StatusForbidden)
		return
	}
	if r.FormValue("merge") == "1" {
		if err := h.client.MergePullRequest(r.Context(), owner, repo, n, "merge"); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	} else {
		event := r.FormValue("event")
		if err := h.client.CreateReview(r.Context(), owner, repo, n, event); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	http.Redirect(w, r, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, n), http.StatusSeeOther)
}

func (h *Handler) actions(w http.ResponseWriter, r *http.Request, owner, repo string) {
	runs, _ := h.client.ListWorkflowRuns(r.Context(), owner, repo)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><html><body><h1>Actions for %s/%s</h1><ul>", owner, repo)
	for _, run := range runs {
		fmt.Fprintf(w, "<li>%s - %s</li>", template.HTMLEscapeString(run.Name), template.HTMLEscapeString(run.Status))
	}
	fmt.Fprint(w, "</ul></body></html>")
}
