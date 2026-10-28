# GitHub Pull Request and Actions UI

## Goal

Build a lightweight single-user web UI in Go for browsing repositories across all accessible organizations, reviewing and merging pull requests, and monitoring/managing GitHub Actions workflows.

## Scope

### Included

- Server-rendered HTML using Go and templ.
- GitHub classic PAT loaded from `.env` on server startup.
- Repository listing across the authenticated user's repositories, collaborations, and organization memberships.
- Pagination until all accessible repositories are collected.
- Organization/name grouping and simple repository selection.
- Pull request list and detail view.
- Pull request review actions: approve and request changes.
- Pull request merge action with selectable merge method where GitHub permits it.
- Workflow list and run detail/status.
- Workflow dispatch, cancel, and rerun actions where GitHub permits them.
- Clear display of GitHub API errors without exposing the token.

### Excluded from MVP

- Multi-user authentication.
- Storing PATs in a database or browser storage.
- Creating or editing pull requests.
- Full diff editor, inline review comments, notifications, webhooks, and background synchronization.
- SPA frontend or client-side state management.

## Authentication and Security

- Read `GITHUB_TOKEN` from `.env` using environment configuration at startup.
- Never place the token in HTML, JavaScript, URL parameters, cookies, logs, or API responses.
- Require `.env` in `.gitignore`; document `chmod 600 .env` for local deployments.
- Use a classic PAT with the minimum required scopes documented in the README: `repo`, `workflow`, and `read:org` for private repositories, Actions, and organization visibility.
- Use GitHub REST API requests only from the Go server.
- Validate route parameters and restrict mutations to the selected repository/PR/workflow resources returned by GitHub.

## Architecture

The application is a single Go process using `net/http`. A small GitHub client wraps REST requests and pagination. HTTP handlers load data and invoke mutations. templ components render complete pages and small HTMX-compatible response fragments; HTMX is optional and used only for action forms so the application remains usable without a SPA.

Suggested boundaries:

- `internal/github`: API client, GitHub DTOs, pagination, and error mapping.
- `internal/config`: environment loading and validation.
- `internal/web`: handlers, route wiring, view models, and templ components.
- `templates`: templ source files.
- `static`: minimal CSS.

## Routes

- `GET /` — repository list.
- `GET /repos/{owner}/{repo}/pulls` — open pull requests.
- `GET /repos/{owner}/{repo}/pulls/{number}` — pull request detail and available actions.
- `POST /repos/{owner}/{repo}/pulls/{number}/review` — approve or request changes.
- `POST /repos/{owner}/{repo}/pulls/{number}/merge` — merge pull request.
- `GET /repos/{owner}/{repo}/actions` — workflows and recent runs.
- `POST /repos/{owner}/{repo}/actions/workflows/{workflow}/dispatch` — dispatch workflow.
- `POST /repos/{owner}/{repo}/actions/runs/{run}/cancel` — cancel run.
- `POST /repos/{owner}/{repo}/actions/runs/{run}/rerun` — rerun run.

Mutation forms use POST and include a same-origin CSRF token stored in a server-side session or short-lived signed cookie. No mutation uses GET.

## GitHub API Behavior

- Repository discovery starts with `GET /user/repos?visibility=all&affiliation=owner,collaborator,organization_member&per_page=100` and follows `Link` pagination until exhausted.
- If the API response does not expose a repository through that aggregate endpoint, the client may supplement it with `GET /user/orgs` followed by `GET /orgs/{org}/repos`; duplicates are removed by repository ID.
- All list endpoints follow GitHub pagination.
- The client sends `Accept: application/vnd.github+json`, `X-GitHub-Api-Version`, and bearer authorization.
- GitHub API errors preserve HTTP status and safe response message, but never include authorization headers or token values.

## Testing and Verification

- Unit tests use `httptest.Server` to verify authorization headers, pagination, repository deduplication, request payloads, and error handling.
- Handler tests verify successful HTML responses and that mutation requests use POST with expected GitHub API calls.
- `go test ./...` is the required baseline check.
- Manual verification uses a real classic PAT in a local `.env`; credentials must not be committed.

## Acceptance Criteria

1. Starting the server with a valid `.env` displays repositories from the user account and all accessible organizations, across multiple API pages.
2. A user can open a repository, inspect open PRs, approve/request changes, and merge a PR if GitHub permits it.
3. A user can list workflow runs and dispatch, cancel, or rerun them when permitted.
4. Unauthorized or forbidden GitHub responses are shown as actionable errors, not server panics or blank pages.
5. The PAT is never present in rendered output, browser storage, logs, or committed files.
