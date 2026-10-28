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
	if strings.HasPrefix(r.URL.Path, "/static/") {
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
		return
	}
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
	case len(segs) == 2:
		h.repo(w, r, segs[0], segs[1])
	case len(segs) == 3 && segs[2] == "pulls":
		if r.Method == http.MethodPost {
			h.createPullRequest(w, r, segs[0], segs[1], token)
			return
		}
		h.pulls(w, r, segs[0], segs[1])
	case len(segs) == 4 && segs[2] == "pulls":
		n, _ := strconv.Atoi(segs[3])
		if r.Method == http.MethodPost {
			h.reviewOrMerge(w, r, segs[0], segs[1], n, token)
			return
		}
		h.pullDetail(w, r, segs[0], segs[1], n)
	case len(segs) == 5 && segs[2] == "pulls" && (segs[4] == "review" || segs[4] == "merge"):
		n, err := strconv.Atoi(segs[3])
		if err != nil || n <= 0 || r.Method != http.MethodPost || !validateCSRF(r, r.FormValue("csrf")) {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if segs[4] == "merge" {
			err = h.client.MergePullRequest(r.Context(), segs[0], segs[1], n, r.FormValue("method"))
		} else {
			err = h.client.CreateReview(r.Context(), segs[0], segs[1], n, r.FormValue("event"))
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/repos/%s/%s/pulls/%d", segs[0], segs[1], n), http.StatusSeeOther)
	case len(segs) == 3 && segs[2] == "actions":
		h.actions(w, r, segs[0], segs[1])
	case len(segs) == 5 && segs[2] == "actions" && segs[3] == "runs":
		id, err := strconv.ParseInt(segs[4], 10, 64)
		if err != nil || id <= 0 {
			http.NotFound(w, r)
			return
		}
		h.actionDetail(w, r, segs[0], segs[1], id)
	case len(segs) == 6 && segs[2] == "actions" && segs[3] == "workflows" && segs[5] == "dispatch":
		if r.Method != http.MethodPost || !validateCSRF(r, r.FormValue("csrf")) {
			http.Error(w, "invalid request", http.StatusForbidden)
			return
		}
		if err := h.client.DispatchWorkflow(r.Context(), segs[0], segs[1], segs[4], r.FormValue("ref")); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/repos/"+segs[0]+"/"+segs[1]+"/actions", http.StatusSeeOther)
	case len(segs) == 6 && segs[2] == "actions" && segs[3] == "runs" && (segs[5] == "cancel" || segs[5] == "rerun"):
		if r.Method != http.MethodPost || !validateCSRF(r, r.FormValue("csrf")) {
			http.Error(w, "invalid request", http.StatusForbidden)
			return
		}
		id, err := strconv.ParseInt(segs[4], 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "invalid run", 400)
			return
		}
		if segs[5] == "cancel" {
			err = h.client.CancelWorkflowRun(r.Context(), segs[0], segs[1], id)
		} else {
			err = h.client.RerunWorkflowRun(r.Context(), segs[0], segs[1], id)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/repos/"+segs[0]+"/"+segs[1]+"/actions", http.StatusSeeOther)
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
	pageStart(w, "Repositories", "Repositories", "Browse every repository available to your GitHub token.")
	fmt.Fprint(w, `<section class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">`)
	for _, repo := range repos {
		visibility := "public"
		if repo.Private {
			visibility = "private"
		}
		fmt.Fprintf(w, `<a class="group rounded-2xl border border-slate-800 bg-slate-900 p-5 shadow-xl transition hover:-translate-y-0.5 hover:border-indigo-500" href="/repos/%s/%s"><div class="mb-4 flex items-center justify-between"><span class="text-xs font-medium uppercase tracking-wider text-indigo-300">%s</span><span class="rounded-full bg-slate-800 px-2 py-1 text-xs text-slate-400">%s</span></div><h2 class="font-semibold text-white group-hover:text-indigo-300">%s</h2><p class="mt-2 text-sm text-slate-400">%s · Open pull requests and review activity</p></a>`, repo.Owner.Login, repo.Name, template.HTMLEscapeString(repo.Owner.Login), visibility, template.HTMLEscapeString(repo.Name), template.HTMLEscapeString(repo.FullName))
	}
	pageEnd(w)
}

func (h *Handler) pulls(w http.ResponseWriter, r *http.Request, owner, repo string) {
	prs, err := h.client.ListPullRequests(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pageStart(w, "Pull requests", repo, fmt.Sprintf("Review activity in %s/%s.", owner, repo), owner, repo, "Pull requests")
	csrf := ensureCSRF(w, r)
	branches, branchErr := h.client.ListBranches(r.Context(), owner, repo)
	collaborators, collaboratorErr := h.client.ListCollaborators(r.Context(), owner, repo)
	fmt.Fprintf(w, `<div class="mb-8 rounded-2xl border border-slate-800 bg-slate-900 p-6"><div class="mb-5"><h2 class="text-lg font-semibold text-white">Create pull request</h2><p class="mt-1 text-sm text-slate-400">Compare branches and request a review in one step.</p></div><form class="grid gap-4 md:grid-cols-2" method="post" action="/repos/%s/%s/pulls"><input type="hidden" name="csrf" value="%s"><label class="text-sm text-slate-300">From branch<select class="mt-2 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2" name="head">`, owner, repo, csrf)
	if branchErr == nil {
		for _, b := range branches {
			fmt.Fprintf(w, `<option value="%s">%s</option>`, template.HTMLEscapeString(b.Name), template.HTMLEscapeString(b.Name))
		}
	}
	fmt.Fprint(w, `</select></label><label class="text-sm text-slate-300">To branch<select class="mt-2 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2" name="base">`)
	if branchErr == nil {
		for _, b := range branches {
			fmt.Fprintf(w, `<option value="%s">%s</option>`, template.HTMLEscapeString(b.Name), template.HTMLEscapeString(b.Name))
		}
	}
	fmt.Fprint(w, `</select></label><label class="text-sm text-slate-300 md:col-span-2">Title<input required class="mt-2 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2" name="title"></label><label class="text-sm text-slate-300 md:col-span-2">Description<textarea class="mt-2 min-h-28 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2" name="body"></textarea></label><label class="text-sm text-slate-300">Request review from<select class="mt-2 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2" name="reviewer"><option value="">No reviewer</option>`)
	if collaboratorErr == nil {
		for _, user := range collaborators {
			fmt.Fprintf(w, `<option value="%s">@%s</option>`, template.HTMLEscapeString(user.Login), template.HTMLEscapeString(user.Login))
		}
	}
	fmt.Fprint(w, `</select></label><div class="flex items-end"><button class="rounded-lg bg-indigo-500 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-400" type="submit">Create pull request</button></div></form></div>`)
	fmt.Fprint(w, `<div class="space-y-3">`)
	for _, pr := range prs {
		fmt.Fprintf(w, `<a class="block rounded-2xl border border-slate-800 bg-slate-900 p-5 transition hover:border-indigo-500" href="/repos/%s/%s/pulls/%d"><div class="flex items-center gap-3"><span class="text-sm text-slate-500">#%d</span><h2 class="font-medium text-white">%s</h2><span class="ml-auto rounded-full bg-emerald-500/10 px-2.5 py-1 text-xs text-emerald-300">open</span></div></a>`, owner, repo, pr.Number, pr.Number, template.HTMLEscapeString(pr.Title))
	}
	fmt.Fprint(w, `</div>`)
	pageEnd(w)
}

func (h *Handler) repo(w http.ResponseWriter, r *http.Request, owner, repo string) {
	pageStart(w, "Repository", repo, fmt.Sprintf("Choose a workspace for %s/%s.", owner, repo), owner, repo)
	fmt.Fprintf(w, `<div class="grid gap-4 sm:grid-cols-2"><a class="rounded-2xl border border-indigo-500/50 bg-indigo-500/10 p-6 hover:bg-indigo-500/20" href="/repos/%s/%s/pulls"><h2 class="text-lg font-semibold text-white">Pull requests</h2><p class="mt-2 text-sm text-slate-400">Create, review, approve, and merge pull requests.</p></a><a class="rounded-2xl border border-slate-800 bg-slate-900 p-6 hover:border-indigo-500" href="/repos/%s/%s/actions"><h2 class="text-lg font-semibold text-white">Actions</h2><p class="mt-2 text-sm text-slate-400">Monitor workflow runs and manage deployments.</p></a></div>`, owner, repo, owner, repo)
	pageEnd(w)
}

func (h *Handler) createPullRequest(w http.ResponseWriter, r *http.Request, owner, repo, csrf string) {
	if !validateCSRF(r, r.FormValue("csrf")) {
		http.Error(w, "invalid csrf", http.StatusForbidden)
		return
	}
	if strings.TrimSpace(r.FormValue("title")) == "" || r.FormValue("head") == "" || r.FormValue("base") == "" {
		http.Error(w, "title and branches are required", http.StatusBadRequest)
		return
	}
	pr, err := h.client.CreatePullRequest(r.Context(), owner, repo, github.PullRequestInput{Title: r.FormValue("title"), Body: r.FormValue("body"), Head: r.FormValue("head"), Base: r.FormValue("base")})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if reviewer := r.FormValue("reviewer"); reviewer != "" {
		if err := h.client.RequestReviewers(r.Context(), owner, repo, pr.Number, []string{reviewer}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	http.Redirect(w, r, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, pr.Number), http.StatusSeeOther)
}

func (h *Handler) pullDetail(w http.ResponseWriter, r *http.Request, owner, repo string, n int) {
	pr, err := h.client.GetPullRequest(r.Context(), owner, repo, n)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	csrfToken := ensureCSRF(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pageStart(w, fmt.Sprintf("Pull request #%d", n), pr.Title, fmt.Sprintf("%s/%s", owner, repo), owner, repo, "Pull requests", fmt.Sprintf("Pull request #%d", n))
	fmt.Fprintf(w, `<article class="rounded-2xl border border-slate-800 bg-slate-900 p-6"><div class="mb-6 flex flex-wrap gap-3"><span class="rounded-full bg-emerald-500/10 px-3 py-1 text-sm text-emerald-300">%s</span><a class="text-sm text-indigo-300 hover:text-indigo-200" href="%s" target="_blank" rel="noreferrer">Open on GitHub ↗</a></div><div class="flex flex-wrap gap-3"><form method="post" action="/repos/%s/%s/pulls/%d/review"><input type="hidden" name="csrf" value="%s"><button class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-400" name="event" value="APPROVE">Approve</button><button class="rounded-lg bg-amber-500/15 px-4 py-2 text-sm font-semibold text-amber-300 hover:bg-amber-500/25" name="event" value="REQUEST_CHANGES">Request changes</button></form><form method="post" action="/repos/%s/%s/pulls/%d/merge"><input type="hidden" name="csrf" value="%s"><input type="hidden" name="method" value="merge"><button class="rounded-lg bg-indigo-500 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-400" type="submit">Merge pull request</button></form></div></article>`, template.HTMLEscapeString(pr.State), template.HTMLEscapeString(pr.HTMLURL), owner, repo, n, csrfToken, owner, repo, n, csrfToken)
	pageEnd(w)
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
	runs, err := h.client.ListWorkflowRuns(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pageStart(w, "GitHub Actions", repo, fmt.Sprintf("Workflow runs in %s/%s.", owner, repo), owner, repo, "Actions")
	fmt.Fprint(w, `<div class="space-y-3">`)
	for _, run := range runs {
		badge := "bg-slate-800 text-slate-300"
		label := run.Status
		if run.Status == "in_progress" || run.Status == "queued" {
			badge = "bg-amber-500/10 text-amber-300"
		}
		if run.Status == "completed" && run.Conclusion == "success" {
			badge = "bg-emerald-500/10 text-emerald-300"
		}
		if run.Status == "completed" && run.Conclusion == "failure" {
			badge = "bg-rose-500/10 text-rose-300"
			label = "completed-failed"
		}
		fmt.Fprintf(w, `<a href="/repos/%s/%s/actions/runs/%d" class="flex flex-wrap items-center gap-4 rounded-2xl border border-slate-800 bg-slate-900 p-5 hover:border-indigo-500"><div class="min-w-0 flex-1"><h2 class="font-medium text-white">%s</h2><p class="mt-1 text-sm text-slate-400">%s</p></div><span class="rounded-full px-3 py-1 text-xs %s">%s</span></a>`, owner, repo, run.ID, template.HTMLEscapeString(run.Name), template.HTMLEscapeString(run.HTMLURL), badge, template.HTMLEscapeString(label))
	}
	fmt.Fprint(w, `</div>`)
	pageEnd(w)
}

func (h *Handler) actionDetail(w http.ResponseWriter, r *http.Request, owner, repo string, id int64) {
	run, err := h.client.GetWorkflowRun(r.Context(), owner, repo, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	jobs, err := h.client.ListWorkflowJobs(r.Context(), owner, repo, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	pageStart(w, "Action run", run.Name, fmt.Sprintf("%s/%s · run #%d", owner, repo, id), owner, repo, "Actions", fmt.Sprintf("Run #%d", id))
	fmt.Fprint(w, `<div class="space-y-4">`)
	for _, job := range jobs {
		fmt.Fprintf(w, `<article class="rounded-2xl border border-slate-800 bg-slate-900 p-5"><div class="flex items-center justify-between"><h2 class="font-semibold text-white">%s</h2><span class="text-sm text-emerald-300">%s</span></div><div class="mt-4 border-l border-indigo-500/40 pl-5">`, template.HTMLEscapeString(job.Name), template.HTMLEscapeString(job.Conclusion))
		for _, step := range job.Steps {
			fmt.Fprintf(w, `<div class="relative py-2 text-sm text-slate-300"><span class="absolute -left-[25px] mt-1 h-2 w-2 rounded-full bg-emerald-400"></span>%d. %s <span class="ml-2 text-xs text-slate-500">%s</span></div>`, step.Number, template.HTMLEscapeString(step.Name), template.HTMLEscapeString(step.Conclusion))
		}
		fmt.Fprint(w, `</div></article>`)
	}
	fmt.Fprint(w, `</div>`)
	pageEnd(w)
}

func pageStart(w http.ResponseWriter, eyebrow, title, subtitle string, crumbs ...string) {
	fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s · gitui</title><script src="https://cdn.tailwindcss.com"></script></head><body class="min-h-screen bg-slate-950 text-slate-100"><header class="border-b border-slate-800 bg-slate-950/80 backdrop-blur"><div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-5"><a class="text-lg font-bold tracking-tight text-white" href="/">git<span class="text-indigo-400">ui</span></a><nav class="flex gap-4 text-sm text-slate-400"><a class="hover:text-white" href="/">Repositories</a></nav></div></header><main class="mx-auto max-w-6xl px-6 py-12"><nav class="mb-8 flex flex-wrap items-center gap-2 text-sm text-slate-500"><a class="hover:text-white" href="/">Repositories</a>`, template.HTMLEscapeString(title))
	for i := 0; i < len(crumbs); i++ {
		label := crumbs[i]
		if i == len(crumbs)-1 {
			fmt.Fprintf(w, `<span>/</span><span class="text-slate-200">%s</span>`, template.HTMLEscapeString(label))
			continue
		}
		if i == 0 {
			fmt.Fprintf(w, `<span>/</span><span>%s</span>`, template.HTMLEscapeString(label))
			continue
		}
		if i == 1 {
			fmt.Fprintf(w, `<span>/</span><a class="hover:text-white" href="/repos/%s/%s">%s</a>`, crumbs[0], crumbs[1], template.HTMLEscapeString(label))
			continue
		}
		fmt.Fprintf(w, `<span>/</span><span>%s</span>`, template.HTMLEscapeString(label))
	}
	fmt.Fprintf(w, `</nav><p class="text-sm font-medium uppercase tracking-widest text-indigo-400">%s</p><h1 class="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">%s</h1><p class="mt-3 max-w-2xl text-slate-400">%s</p><div class="mt-8">`, template.HTMLEscapeString(eyebrow), template.HTMLEscapeString(title), template.HTMLEscapeString(subtitle))
}

func pageEnd(w http.ResponseWriter) {
	fmt.Fprint(w, `</div></main><footer class="mx-auto max-w-6xl px-6 pb-10 text-xs text-slate-600">Powered by Go, templ and GitHub API</footer></body></html>`)
}
