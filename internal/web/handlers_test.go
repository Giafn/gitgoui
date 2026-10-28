package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/gitui/internal/github"
)

func TestHandlersBasic(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			w.Write([]byte(`[{"id":1,"name":"repo","full_name":"org/repo","owner":{"login":"org"}}]`))
			return
		}
		if r.URL.Path == "/repos/org/repo/branches" {
			w.Write([]byte(`[{"name":"main"},{"name":"feature"}]`))
			return
		}
		if r.URL.Path == "/repos/org/repo/collaborators" {
			w.Write([]byte(`[{"login":"reviewer"}]`))
			return
		}
		if r.URL.Path == "/repos/org/repo/pulls" {
			w.Write([]byte(`[{"number":7,"title":"Fix"}]`))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()
	client := github.NewClient(nil, s.URL, "secret")
	h := NewHandler(client)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rw, req)
	if rw.Code != 200 || !strings.Contains(rw.Body.String(), "org/repo") {
		t.Fatalf("repos page: %d %s", rw.Code, rw.Body.String())
	}
	if !strings.Contains(rw.Body.String(), "tailwindcss") || !strings.Contains(rw.Body.String(), "bg-slate-950") {
		t.Fatal("repository page is missing modern Tailwind layout")
	}
	rwRepo := httptest.NewRecorder()
	h.ServeHTTP(rwRepo, httptest.NewRequest(http.MethodGet, "/repos/org/repo", nil))
	if !strings.Contains(rwRepo.Body.String(), "Pull requests") || !strings.Contains(rwRepo.Body.String(), "Actions") {
		t.Fatal("repository tabs missing")
	}
	rw2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/repos/org/repo/pulls", nil)
	h.ServeHTTP(rw2, req2)
	if rw2.Code != 200 || !strings.Contains(rw2.Body.String(), "Fix") {
		t.Fatalf("pulls page: %d %s", rw2.Code, rw2.Body.String())
	}
}

func TestActionsShowsGitHubError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Actions access denied"}`))
	}))
	defer s.Close()
	rw := httptest.NewRecorder()
	NewHandler(github.NewClient(nil, s.URL, "secret")).ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/repos/org/repo/actions", nil))
	if rw.Code != http.StatusInternalServerError || !strings.Contains(rw.Body.String(), "Actions access denied") {
		t.Fatalf("expected actionable actions error, got %d %s", rw.Code, rw.Body.String())
	}
}

func TestActionsUsesDistinctStatusBadgeColors(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"workflow_runs":[{"id":1,"name":"Build","status":"in_progress"},{"id":2,"name":"Deploy","status":"completed","conclusion":"success"},{"id":3,"name":"Broken","status":"completed","conclusion":"failure"}]}`))
	}))
	defer s.Close()
	rw := httptest.NewRecorder()
	NewHandler(github.NewClient(nil, s.URL, "secret")).ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/repos/org/repo/actions", nil))
	body := rw.Body.String()
	if !strings.Contains(body, "bg-amber-500/10") || !strings.Contains(body, "bg-emerald-500/10") || !strings.Contains(body, "completed-failed") {
		t.Fatalf("status colors missing: %s", body)
	}
}

func TestActionDetailShowsBreadcrumbAndJobGraph(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/jobs") {
			w.Write([]byte(`{"jobs":[{"id":8,"name":"build","status":"completed","conclusion":"success","steps":[{"number":1,"name":"Test","status":"completed","conclusion":"success"}]}]}`))
			return
		}
		w.Write([]byte(`{"id":7,"name":"CI","status":"completed","conclusion":"success"}`))
	}))
	defer s.Close()
	rw := httptest.NewRecorder()
	NewHandler(github.NewClient(nil, s.URL, "secret")).ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/repos/org/repo/actions/runs/7", nil))
	if !strings.Contains(rw.Body.String(), "Repositories") || !strings.Contains(rw.Body.String(), "build") || !strings.Contains(rw.Body.String(), "Test") {
		t.Fatalf("detail missing: %s", rw.Body.String())
	}
}
