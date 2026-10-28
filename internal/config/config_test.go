package config

import "testing"

func TestLoadRequiresToken(t *testing.T) {

	t.Setenv("GITHUB_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestLoadDefaultsAddress(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret")
	t.Setenv("GITUI_ADDR", "")
	c, err := Load()
	if err != nil || c.Addr != "127.0.0.1:8080" {
		t.Fatalf("config = %#v, err = %v", c, err)
	}
}
