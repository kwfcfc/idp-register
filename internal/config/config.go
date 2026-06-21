// SPDX-License-Identifier: GPL-3.0-or-later

// Package config loads typed configuration from the environment. It replaces
// the draft's src/lib/server/env.ts. Unlike the draft it is dual-DB aware
// (SQLite or PostgreSQL) and carries no INVITE_HMAC_KEY — invite codes are
// stored plaintext now (see docs/DECISIONS.md ADR-0004).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Driver identifies which database engine the store talks to.
type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
)

// Config is the fully-resolved, validated runtime configuration.
type Config struct {
	// HTTP
	Addr      string // listen address, e.g. ":8080"
	Origin    string // public origin, e.g. https://register.example.com
	TrustedCFIP bool // restore client IP from CF-Connecting-IP

	// Database
	DBDriver Driver
	DBDSN    string // pgx DSN, or path/DSN for sqlite

	// Admin-auth OIDC Relying Party (generic; never assume provider APIs).
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCScopes       []string
	// Admin authorization: a user is admin if their groups/roles claim
	// contains AdminGroup, OR their sub/email is in the allow-lists.
	OIDCGroupsClaim string
	AdminGroup      string
	AdminSubs       []string
	AdminEmails     []string

	// Provisioning target IdP (Rauthy today; behind the Provisioner interface).
	ProvisionerKind  string // "rauthy"
	RauthyAPIBase    string
	RauthyAPIKeyName string
	RauthyAPIKey     string
	RauthyLanguage   string
	RauthyTimezone   string

	// Sessions / cookies
	SessionTTL    time.Duration
	SecureCookies bool

	// Anti-abuse
	TurnstileSecret string // empty disables CAPTCHA verification
}

// Load reads and validates configuration from the process environment.
func Load() (*Config, error) {
	c := &Config{
		Addr:            envOr("ADDR", ":8080"),
		OIDCGroupsClaim: envOr("OIDC_GROUPS_CLAIM", "groups"),
		AdminGroup:      envOr("OIDC_ADMIN_GROUP", "svc:idp-register:admin"),
		ProvisionerKind: envOr("PROVISIONER", "rauthy"),
		RauthyLanguage:  envOr("RAUTHY_DEFAULT_LANGUAGE", "en"),
		RauthyTimezone:  envOr("RAUTHY_DEFAULT_TIMEZONE", "UTC"),
		TurnstileSecret: os.Getenv("TURNSTILE_SECRET"),
		SecureCookies:   envBool("SECURE_COOKIES", strings.EqualFold(os.Getenv("APP_ENV"), "production")),
	}

	var err error
	if c.Origin, err = required("ORIGIN"); err != nil {
		return nil, err
	}

	// Database: prefer DATABASE_URL (postgres); else SQLITE_PATH.
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		c.DBDriver = DriverPostgres
		c.DBDSN = dsn
	} else {
		c.DBDriver = DriverSQLite
		c.DBDSN = envOr("SQLITE_PATH", "idp-register.db")
	}

	// Admin OIDC RP.
	for _, f := range []struct {
		dst  *string
		name string
	}{
		{&c.OIDCIssuer, "OIDC_ISSUER"},
		{&c.OIDCClientID, "OIDC_CLIENT_ID"},
		{&c.OIDCClientSecret, "OIDC_CLIENT_SECRET"},
		{&c.OIDCRedirectURL, "OIDC_REDIRECT_URI"},
	} {
		if *f.dst, err = required(f.name); err != nil {
			return nil, err
		}
	}
	c.OIDCScopes = splitList(envOr("OIDC_SCOPES", "openid email profile groups"))
	c.AdminSubs = splitList(os.Getenv("OIDC_ADMIN_SUBS"))
	c.AdminEmails = lowerAll(splitList(os.Getenv("OIDC_ADMIN_EMAILS")))

	// Provisioner (Rauthy).
	if c.ProvisionerKind == "rauthy" {
		if c.RauthyAPIBase, err = required("RAUTHY_API_BASE"); err != nil {
			return nil, err
		}
		c.RauthyAPIBase = strings.TrimRight(c.RauthyAPIBase, "/")
		if c.RauthyAPIKeyName, err = required("RAUTHY_API_KEY_NAME"); err != nil {
			return nil, err
		}
		if c.RauthyAPIKey, err = required("RAUTHY_API_KEY_SECRET"); err != nil {
			return nil, err
		}
	}

	hours := envInt("SESSION_TTL_HOURS", 12)
	if hours <= 0 {
		hours = 12
	}
	c.SessionTTL = time.Duration(hours) * time.Hour
	c.TrustedCFIP = envBool("TRUST_CF_CONNECTING_IP", true)

	return c, nil
}

func required(name string) (string, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	return v, nil
}

func envOr(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func envInt(name string, def int) int {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(name string, def bool) bool {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

// splitList splits on commas and whitespace, dropping empties.
func splitList(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func lowerAll(in []string) []string {
	for i := range in {
		in[i] = strings.ToLower(in[i])
	}
	return in
}
