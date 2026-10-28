package github

import (
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
