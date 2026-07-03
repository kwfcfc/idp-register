// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"
)

// TestRegisterRequiresConsent: with rules/terms configured, a submission
// without termsAccepted must be rejected before it reaches the service layer
// (the Server here has no application service — reaching it would panic).
func TestRegisterRequiresConsent(t *testing.T) {
	s := &Server{cfg: &config.Config{FormTermsURL: "https://example.com/tos"}}

	r := httptest.NewRequest(http.MethodPost, "/api/register",
		strings.NewReader(`{"email":"user@example.test","termsAccepted":false}`))
	w := httptest.NewRecorder()
	s.handleRegister(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unconsented submission should be 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "terms") {
		t.Fatalf("error should mention the terms, got %s", w.Body.String())
	}
}

// TestCheckCSRF locks the exact-match origin comparison: a prefix match would
// accept attacker origins like https://register.example.com.evil.com.
func TestCheckCSRF(t *testing.T) {
	s := &Server{cfg: &config.Config{Origin: "https://register.example.com"}}

	cases := []struct {
		name    string
		origin  string
		referer string
		want    bool
	}{
		{"exact origin", "https://register.example.com", "", true},
		{"origin case-insensitive", "https://Register.Example.com", "", true},
		{"prefix-extended host", "https://register.example.com.evil.com", "", false},
		{"different host", "https://evil.com", "", false},
		{"null origin", "null", "", false},
		{"no origin, same-origin referer with path", "", "https://register.example.com/admin/tokens", true},
		{"no origin, prefix-extended referer host", "", "https://register.example.community.evil.com/x", false},
		{"no origin, malformed referer", "", "::not-a-url", false},
		{"nothing at all", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/admin/tokens", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				r.Header.Set("Referer", tc.referer)
			}
			w := httptest.NewRecorder()
			if got := s.checkCSRF(w, r); got != tc.want {
				t.Fatalf("checkCSRF = %v, want %v (status %d)", got, tc.want, w.Code)
			}
			if !tc.want && w.Code != http.StatusForbidden {
				t.Fatalf("blocked request should write 403, got %d", w.Code)
			}
		})
	}
}
