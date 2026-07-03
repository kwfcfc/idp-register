// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"strings"
	"testing"
)

// setRequiredEnv fills the minimum environment Load needs to succeed.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"ORIGIN":                "https://register.example.com/",
		"OIDC_ISSUER":           "https://auth.example.com/auth/v1/",
		"OIDC_CLIENT_ID":        "idp-register",
		"OIDC_CLIENT_SECRET":    "secret",
		"OIDC_REDIRECT_URI":     "https://register.example.com/auth/callback",
		"RAUTHY_API_BASE":       "https://auth.example.com/auth/v1",
		"RAUTHY_API_KEY_NAME":   "idp-register",
		"RAUTHY_API_KEY_SECRET": "key-secret",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadNormalizesOrigin(t *testing.T) {
	setRequiredEnv(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// The CSRF check compares against browser Origin headers, which never have
	// a trailing slash.
	if c.Origin != "https://register.example.com" {
		t.Fatalf("origin not normalized: %q", c.Origin)
	}
}

// TestLoadTurnstilePairing locks the both-or-neither rule: a half-configured
// challenge is always a broken deployment (ADR-0016).
func TestLoadTurnstilePairing(t *testing.T) {
	cases := []struct {
		name, siteKey, secret string
		wantErr               bool
	}{
		{"both unset", "", "", false},
		{"both set", "0xSITE", "0xSECRET", false},
		{"secret only", "", "0xSECRET", true},
		{"site key only", "0xSITE", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("TURNSTILE_SITE_KEY", tc.siteKey)
			t.Setenv("TURNSTILE_SECRET", tc.secret)
			c, err := Load()
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "TURNSTILE") {
					t.Fatalf("want pairing error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if c.TurnstileSiteKey != tc.siteKey || c.TurnstileSecret != tc.secret {
				t.Fatalf("turnstile values not loaded: %+v", c)
			}
		})
	}
}
