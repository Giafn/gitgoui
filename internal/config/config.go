package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	GitHubToken      string
	Addr             string
	GitHubAPIBaseURL string
}

func Load() (Config, error) {
	loadDotEnv(".env")
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		return Config{}, errors.New("GITHUB_TOKEN is required")
	}
	addr := os.Getenv("GITUI_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	base := os.Getenv("GITHUB_API_BASE_URL")
	if base == "" {
		base = "https://api.github.com/"
	}
	return Config{GitHubToken: token, Addr: addr, GitHubAPIBaseURL: base}, nil
}

func loadDotEnv(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok && os.Getenv(strings.TrimSpace(key)) == "" {
			os.Setenv(strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), "\"'"))
		}
	}
}
