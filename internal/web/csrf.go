package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

const csrfCookie = "gitui_csrf"

var csrfSecret = newToken()

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func sign(secret, token string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func ensureCSRF(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(csrfCookie); err == nil && c.Value != "" {
		parts := strings.SplitN(c.Value, ".", 2)
		if len(parts) == 2 && hmac.Equal([]byte(sign(csrfSecret, parts[0])), []byte(parts[1])) {
			return parts[0]
		}
	}
	token := newToken()
	http.SetCookie(w, &http.Cookie{Name: csrfCookie, Value: token + "." + sign(csrfSecret, token), Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(24 * time.Hour)})
	return token
}

func validateCSRF(r *http.Request, token string) bool {
	c, err := r.Cookie(csrfCookie)
	if err != nil {
		return false
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	if !hmac.Equal([]byte(parts[1]), []byte(sign(csrfSecret, parts[0]))) {
		return false
	}
	return hmac.Equal([]byte(token), []byte(parts[0]))
}
