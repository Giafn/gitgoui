package main

import (
	"log"
	"net/http"
	"time"

	"github.com/example/gitui/internal/config"
	"github.com/example/gitui/internal/github"
	"github.com/example/gitui/internal/web"
)

func main() {
	c, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	client := github.NewClient(nil, c.GitHubAPIBaseURL, c.GitHubToken)
	s := &http.Server{Addr: c.Addr, Handler: web.NewHandler(client), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("gitui listening on %s", c.Addr)
	log.Fatal(s.ListenAndServe())
}
