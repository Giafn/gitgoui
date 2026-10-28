package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFValidation(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	token := ensureCSRF(w, r)
	r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	r2.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
	if !validateCSRF(r2, token) {
		t.Fatal("valid token rejected")
	}
	if validateCSRF(r2, "wrong") {
		t.Fatal("invalid token accepted")
	}
}
