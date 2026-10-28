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
			w.Write([]byte(`[{"id":1,"name":"repo","full_name":"org/repo","owner":"org"}]`))
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
	rw2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/repos/org/repo/pulls", nil)
	h.ServeHTTP(rw2, req2)
	if rw2.Code != 200 || !strings.Contains(rw2.Body.String(), "Fix") {
		t.Fatalf("pulls page: %d %s", rw2.Code, rw2.Body.String())
	}
}
