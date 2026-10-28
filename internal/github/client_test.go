package github

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRepositoriesUsesAuthAndPagination(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing auth")
		}
		if r.URL.Query().Get("page") == "1" {
			w.Header().Set("Link", "<"+"http://"+r.Host+"/user/repos?page=2>; rel=\"next\"")
			rows := strings.Repeat(`{"id":1,"name":"a"},`, 99) + `{"id":2,"name":"b"}`
			_, _ = w.Write([]byte("[" + rows + "]"))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer s.Close()
	v, err := NewClient(nil, s.URL, "secret").ListRepositories(context.Background())
	if err != nil || len(v) != 2 {
		t.Fatalf("%v %#v", err, v)
	}
}

func TestCreatePullRequestPayload(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/repos/acme/app/pulls" {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"number":12,"title":"Add feature"}`))
	}))
	defer s.Close()
	pr, err := NewClient(nil, s.URL, "secret").CreatePullRequest(context.Background(), "acme", "app", PullRequestInput{Title: "Add feature", Head: "feature", Base: "main"})
	if err != nil || pr.Number != 12 {
		t.Fatalf("pr=%#v err=%v", pr, err)
	}
}

func TestEmptyJSONResponseIsAccepted(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(" \n\t")) }))
	defer s.Close()
	var out struct {
		Runs []WorkflowRun `json:"workflow_runs"`
	}
	if err := NewClient(nil, s.URL, "secret").request(context.Background(), http.MethodGet, "/actions/runs", nil, &out); err != nil {
		t.Fatalf("empty response should not be a json error: %v", err)
	}
}

func TestLargeJSONResponseIsNotTruncated(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := bytes.Repeat([]byte("x"), 2<<20)
		_, _ = w.Write([]byte(`{"workflow_runs":[{"name":"`))
		_, _ = w.Write(body)
		_, _ = w.Write([]byte(`"}]}`))
	}))
	defer s.Close()
	var out struct {
		WorkflowRuns []WorkflowRun `json:"workflow_runs"`
	}
	if err := NewClient(nil, s.URL, "secret").request(context.Background(), http.MethodGet, "/actions/runs", nil, &out); err != nil {
		t.Fatalf("large response should decode: %v", err)
	}
}

func TestCompareBranchesDecodesFiles(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ahead_by":1,"additions":3,"deletions":1,"files":[{"filename":"main.go","status":"modified","patch":"@@ -1 +1 @@"}]}`))
	}))
	defer s.Close()
	_, additions, _, files, err := NewClient(nil, s.URL, "secret").CompareBranches(context.Background(), "org", "repo", "main", "feature")
	if err != nil || additions != 3 || len(files) != 1 || files[0].Filename != "main.go" {
		t.Fatalf("files=%#v additions=%d err=%v", files, additions, err)
	}
}
func TestInvalidMutationRejected(t *testing.T) {
	if err := NewClient(nil, "http://example", "x").CreateReview(context.Background(), "a", "b", 1, "BAD"); err == nil {
		t.Fatal("expected validation")
	}
}

func TestListRepositoriesDecodesOwnerObject(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"name":"repo","full_name":"acme/repo","owner":{"login":"acme"}}]`))
	}))
	defer s.Close()
	repos, err := NewClient(nil, s.URL, "secret").ListRepositories(context.Background())
	if err != nil || len(repos) != 1 || repos[0].Owner.Login != "acme" {
		t.Fatalf("repos=%#v err=%v", repos, err)
	}
}
