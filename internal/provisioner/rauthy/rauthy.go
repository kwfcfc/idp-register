// SPDX-License-Identifier: GPL-3.0-or-later

// Package rauthy implements provisioner.Provisioner against a Rauthy instance.
// Ported from the draft's src/lib/server/rauthy.ts. Auth header form is
// "API-Key <name>$<secret>". Confirmed facts (see docs/ARCHITECTURE.md):
//   - POST /users creates the user and, when SMTP is configured, sends the
//     initial "new password" activation email.
//   - POST /users/request_reset is the public password-reset path and requires
//     a PoW payload in Rauthy 0.35.2, even with an API key.
package rauthy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/provisioner"
)

// Client talks to the Rauthy admin API.
type Client struct {
	base       string // e.g. https://auth.example.com/auth/v1 (no trailing slash)
	apiKeyName string
	apiKey     string
	language   string
	timezone   string
	http       *http.Client
}

// Config carries the Rauthy connection settings.
type Config struct {
	APIBase    string
	APIKeyName string
	APIKey     string
	Language   string
	Timezone   string
}

// New constructs a Rauthy client.
func New(cfg Config) *Client {
	return &Client{
		base:       cfg.APIBase,
		apiKeyName: cfg.APIKeyName,
		apiKey:     cfg.APIKey,
		language:   orDefault(cfg.Language, "en"),
		timezone:   orDefault(cfg.Timezone, "UTC"),
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

var _ provisioner.Provisioner = (*Client)(nil)

// Name identifies this provisioner.
func (c *Client) Name() string { return "rauthy" }

func (c *Client) authorization() string {
	return fmt.Sprintf("API-Key %s$%s", c.apiKeyName, c.apiKey)
}

// do performs a JSON request. out may be nil. Returns the HTTP status so
// callers can distinguish 404 etc.
func (c *Client) do(ctx context.Context, method, path string, body, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", c.authorization())

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out != nil && resp.StatusCode != http.StatusNoContent {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil && !errors.Is(err, io.EOF) {
				return resp.StatusCode, fmt.Errorf("decode response: %w", err)
			}
		}
		return resp.StatusCode, nil
	}

	// Non-2xx: include a bounded snippet but never the request body (which may
	// carry PII); Rauthy error bodies are safe to surface.
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 800))
	return resp.StatusCode, fmt.Errorf("rauthy %s %s failed (%d): %s", method, path, resp.StatusCode, snippet)
}

type rauthyUser struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
	Roles  []string `json:"roles"`
}

// ListGroups returns the groups Rauthy has defined (GET /groups). Requires the
// API key to carry Groups:read access.
func (c *Client) ListGroups(ctx context.Context) ([]provisioner.Group, error) {
	var groups []provisioner.Group
	if _, err := c.do(ctx, http.MethodGet, "/groups", nil, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// FindUserByEmail looks up a user by email; (nil, false, nil) when absent.
func (c *Client) FindUserByEmail(ctx context.Context, email string) (*provisioner.User, bool, error) {
	var u rauthyUser
	status, err := c.do(ctx, http.MethodGet, "/users/email/"+url.PathEscape(email), nil, &u)
	if status == http.StatusNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &provisioner.User{ID: u.ID, Email: u.Email, Groups: u.Groups}, true, nil
}

// CreateUser creates a user. Rauthy 0.35.2 sends the initial set-password
// email as part of this request when SMTP is configured.
func (c *Client) CreateUser(ctx context.Context, in provisioner.NewUser) (string, error) {
	lang := orDefault(in.Language, c.language)
	tz := orDefault(in.Timezone, c.timezone)
	body := map[string]any{
		"email":        in.Email,
		"family_name":  nil,
		"given_name":   nil,
		"language":     lang,
		"groups":       in.Groups,
		"roles":        []string{},
		"user_expires": nil,
		"tz":           tz,
	}
	var u rauthyUser
	status, err := c.do(ctx, http.MethodPost, "/users", body, &u)
	if isUserExistsError(status, err) {
		return "", provisioner.ErrUserExists
	}
	if err != nil {
		return "", err
	}
	if in.Username != "" {
		if err := c.setPreferredUsername(ctx, u.ID, in.Username); err != nil {
			// Non-fatal: the account exists; username can be set later.
			return u.ID, fmt.Errorf("set preferred username: %w", err)
		}
	}
	return u.ID, nil
}

func isUserExistsError(status int, err error) bool {
	if status == http.StatusConflict {
		return true
	}
	if status != http.StatusNotAcceptable || err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE") && strings.Contains(msg, "email")
}

func (c *Client) setPreferredUsername(ctx context.Context, userID, username string) error {
	body := map[string]any{"preferred_username": username, "force_overwrite": false}
	_, err := c.do(ctx, http.MethodPut,
		"/users/"+url.PathEscape(userID)+"/self/preferred_username", body, nil)
	return err
}

// InitCredentials is intentionally a no-op for Rauthy 0.35.2. POST /users
// already sends the initial set-password email for newly created users when SMTP
// is configured.
//
// The public POST /users/request_reset path requires a PoW payload and is not an
// admin API shortcut, even when called with an API key.
func (c *Client) InitCredentials(ctx context.Context, userID, email string) (*string, error) {
	return nil, nil
}
