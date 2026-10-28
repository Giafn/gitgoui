package main

import (
	"log"
	"net/http"
	"time"

	"github.com/example/gitui/internal/config"
)

func main() {
	c, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	s := &http.Server{Addr: c.Addr, Handler: http.NotFoundHandler(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("gitui listening on %s", c.Addr)
	log.Fatal(s.ListenAndServe())
}
