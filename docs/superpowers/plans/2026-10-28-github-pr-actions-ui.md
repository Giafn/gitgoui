# GitHub Pull Request and Actions UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a lightweight Go/templ server-rendered UI for repositories across all accessible organizations, pull-request reviews/merges, and GitHub Actions management using a classic PAT from `.env`.

**Architecture:** One `net/http` process with a small REST client in `internal/github`, validated environment configuration in `internal/config`, and HTML handlers/components in `internal/web` plus `templates`. The browser only receives rendered data and CSRF-protected forms; the token remains server-side.

**Tech Stack:** Go, `net/http`, `encoding/json`, `net/http/httptest`, templ, HTMX for optional progressive enhancement, plain CSS.

**Spec:** `docs/superpowers/specs/2026-10-28-github-pr-actions-ui-design.md`

## Global Constraints

- Use a classic PAT from `GITHUB_TOKEN`; never expose it in HTML, JavaScript, URL parameters, cookies, logs, or API responses.
- Use GitHub REST API only from the Go server.
- Repository discovery must cover owner, collaborator, and organization-member repositories and follow pagination.
- All mutations use POST and require same-origin CSRF protection.
- Keep the MVP server-rendered; no SPA, database, background sync, or SDK dependency.
- Required baseline verification is `go test ./...`.

## Review Focus

- Multiple GitHub pagination pages and duplicate repositories: all accessible repositories appear once.
- Organization API omission or failure: the app reports a safe actionable error and does not silently lose all results.
- GitHub 401/403/422/rate-limit responses: users see a useful safe message and the server does not panic or leak the token.
- Invalid owner/repository/number/workflow/run route values and mutation methods: requests are rejected before an unsafe API call.
- Missing/weak CSRF token and failed GitHub mutations: no mutation occurs and the rendered response remains usable.

---

### Task 1: Bootstrap Go Application and Configuration

**Files:**
- Create: `go.mod`
- Create: `cmd/gitui/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `.env.example`
- Create: `.gitignore`
- Create: `README.md`

**Interfaces:**
- Produces `config.Config{GitHubToken, Addr, GitHubAPIBaseURL}` and `config.Load() (Config, error)` for the server.

- [ ] **Step 1: Write failing configuration tests**

Test missing `GITHUB_TOKEN` returns an error; test defaults address to `127.0.0.1:8080`; test reads an explicit API base URL. Use isolated environment variables.

- [ ] **Step 2: Run `go test ./internal/config` and verify failure**

- [ ] **Step 3: Implement `config.Load`**

Read process environment plus a minimal `.env` loader using standard library code; do not add a dotenv dependency. Validate a non-empty token and default non-secret settings.

- [ ] **Step 4: Add bootstrap files and documentation**

Initialize the module, add `.env` to `.gitignore`, document `repo`, `workflow`, and `read:org`, and include `chmod 600 .env`. `main.go` should load config and start an `http.Server` with timeouts.

- [ ] **Step 5: Run `go test ./...` and commit**

```bash
git add go.mod cmd internal/config .env.example .gitignore README.md
git commit -m "chore: bootstrap gitui configuration"
```

### Task 2: Implement GitHub REST Client and Repository Discovery

**Files:**
- Create: `internal/github/client.go`
- Create: `internal/github/types.go`
- Create: `internal/github/errors.go`
- Create: `internal/github/client_test.go`

**Interfaces:**
- `NewClient(httpClient *http.Client, baseURL, token string) *Client`
- `Client.ListRepositories(ctx context.Context) ([]Repository, error)`
- `Client.ListPullRequests(ctx context.Context, owner, repo string) ([]PullRequest, error)`
- `Client.GetPullRequest(ctx context.Context, owner, repo string, number int) (PullRequest, error)`
- `Client.CreateReview(ctx context.Context, owner, repo string, number int, event string) error`
- `Client.MergePullRequest(ctx context.Context, owner, repo string, number int, method string) error`
- `Client.ListWorkflows(ctx context.Context, owner, repo string) ([]Workflow, error)`
- `Client.ListWorkflowRuns(ctx context.Context, owner, repo string) ([]WorkflowRun, error)`
- `Client.DispatchWorkflow(ctx context.Context, owner, repo, workflow string, ref string) error`
- `Client.CancelWorkflowRun(ctx context.Context, owner, repo string, runID int64) error`
- `Client.RerunWorkflowRun(ctx context.Context, owner, repo string, runID int64) error`

- [ ] **Step 1: Write `httptest.Server` tests first**

Cover auth and required headers; two-page `Link` pagination; repository deduplication between `/user/repos` and organization fallback; JSON review/merge/dispatch payloads; cancel/rerun paths; and safe 401/403 error text that excludes the token.

- [ ] **Step 2: Run focused tests and verify failure**

```bash
go test ./internal/github -run 'Test' -v
```

- [ ] **Step 3: Implement typed DTOs and request helper**

Use `encoding/json`, bearer auth, `Accept: application/vnd.github+json`, and a fixed GitHub API version header. Follow `Link` pagination. Map API failures to an error type containing status and sanitized message only.

- [ ] **Step 4: Implement repository fallback and deduplication**

Call `/user/repos` with `visibility=all`, `affiliation=owner,collaborator,organization_member`, and `per_page=100`; supplement via `/user/orgs` and `/orgs/{org}/repos` when needed; deduplicate by repository ID.

- [ ] **Step 5: Implement PR and Actions methods**

Validate mutation event/method values in the client (`APPROVE`, `REQUEST_CHANGES`; `merge`, `squash`, `rebase`) and encode only the required JSON fields.

- [ ] **Step 6: Run tests and commit**

```bash
go test ./internal/github -v
git add internal/github
git commit -m "feat: add github api client"
```

### Task 3: Add CSRF Protection, Routing, and Handler View Models

**Files:**
- Create: `internal/web/csrf.go`
- Create: `internal/web/csrf_test.go`
- Create: `internal/web/handlers.go`
- Create: `internal/web/handlers_test.go`
- Modify: `cmd/gitui/main.go`

**Interfaces:**
- `NewHandler(client *github.Client) http.Handler`
- Handlers consume the client interfaces from Task 2 and render view models through templ components from Task 4.

- [ ] **Step 1: Write CSRF and handler tests**

Test token generation/validation, rejected missing or mismatched tokens, GET repository response, POST review dispatch, invalid numeric route rejection, and safe GitHub error response.

- [ ] **Step 2: Run focused tests and verify failure**

```bash
go test ./internal/web -v
```

- [ ] **Step 3: Implement minimal signed-cookie CSRF**

Generate a random token, store only a hash/signature in a short-lived same-origin cookie, require the matching form token on every POST, and use constant-time comparison. Do not use GET for mutations.

- [ ] **Step 4: Implement route matching and validation**

Use `net/http` route handling and strict path parsing for owner/repo/workflow/run/PR values. Reject empty, malformed, or non-positive numeric identifiers before calling GitHub.

- [ ] **Step 5: Implement repository, PR, and Actions handlers**

Load data from the client, map to display models, invoke mutation methods only after CSRF and form validation, then redirect or render a safe status message. Make HTMX requests receive fragments while normal requests receive full pages.

- [ ] **Step 6: Wire handler and run tests**

```bash
go test ./...
git add cmd/gitui/main.go internal/web
git commit -m "feat: add web handlers and csrf protection"
```

### Task 4: Add templ Pages and Minimal Styling

**Files:**
- Create: `templates/layout.templ`
- Create: `templates/repositories.templ`
- Create: `templates/pulls.templ`
- Create: `templates/pull_detail.templ`
- Create: `templates/actions.templ`
- Create: `static/app.css`
- Modify: `internal/web/handlers.go`

**Interfaces:**
- Templ components accept the view models produced by handlers and render complete pages plus optional action fragments.

- [ ] **Step 1: Define page rendering tests**

Assert rendered pages contain repository organization/name, PR title/state, workflow status, POST action forms, and no token value.

- [ ] **Step 2: Generate or run templ tooling and verify initial failure**

Use the project’s pinned templ command/version; if unavailable, document the exact generation command in `README.md` and keep generated output reproducible.

- [ ] **Step 3: Implement focused templ components**

Create a shared layout, repository links, PR detail action forms for approve/request-changes/merge, and Actions forms for dispatch/cancel/rerun. Escape all dynamic values through templ’s normal text rendering.

- [ ] **Step 4: Add plain CSS and optional HTMX attributes**

Keep styles minimal and usable without JavaScript. Add HTMX only to progressively enhance forms; normal POST/redirect behavior remains the fallback.

- [ ] **Step 5: Run tests and commit**

```bash
go test ./...
git add templates static internal/web
git commit -m "feat: add templ github ui"
```

### Task 5: Complete Documentation and End-to-End Verification

**Files:**
- Modify: `README.md`
- Create: `internal/web/integration_test.go` if handler coverage needs a full route test

- [ ] **Step 1: Add configuration and PAT safety documentation**

Document classic PAT scopes, organization SSO authorization, organization PAT policies, `.env` permissions, startup command, templ generation, and local verification without including a real token.

- [ ] **Step 2: Run static and test checks**

```bash
gofmt -w cmd internal
go vet ./...
go test ./...
```

Expected: all commands exit zero; tests verify pagination, deduplication, safe errors, CSRF, and mutation payloads.

- [ ] **Step 3: Run a local manual smoke test**

Create an untracked `.env` with a classic PAT, start the server, confirm repositories from multiple organizations and pages, inspect a PR, and exercise only permitted actions. Remove or retain the untracked `.env`; never stage it.

- [ ] **Step 4: Inspect output for secret leakage and commit**

Search tracked files and rendered test output for token-like values, verify `.env` is ignored, then commit documentation/final fixes:

```bash
git check-ignore .env
git diff --check
git add README.md internal/web/integration_test.go
git commit -m "docs: document setup and verification"
```
