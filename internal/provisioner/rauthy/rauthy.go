// SPDX-License-Identifier: GPL-3.0-or-later

// Package rauthy implements provisioner.Provisioner against a Rauthy instance.
// Ported from the draft's src/lib/server/rauthy.ts. Auth header form is
// "API-Key <name>$<secret>". Confirmed facts (see docs/ARCHITECTURE.md):
//   - POST /users creates a user silently (no email).
//   - The activation/set-password email is sent by a separate request_reset
//     call; Rauthy then delivers its own mail (so InitCredentials returns nil).
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

// CreateUser creates a user silently (POST /users does not send email).
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
	if status == http.StatusConflict {
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

func (c *Client) setPreferredUsername(ctx context.Context, userID, username string) error {
	body := map[string]any{"preferred_username": username, "force_overwrite": false}
	_, err := c.do(ctx, http.MethodPut,
		"/users/"+url.PathEscape(userID)+"/self/preferred_username", body, nil)
	return err
}

// InitCredentials asks Rauthy to send the user its set-password / activation
// email. Rauthy delivers the mail itself, so the returned reset link is nil.
//
// NOTE: the exact admin endpoint and whether a proof-of-work is required varies
// by Rauthy version. The public POST /users/request_reset requires PoW and
// always returns 200 (anti-enumeration). Confirm against the deployed instance
// before relying on this in production; this is the one piece the draft never
// implemented.
func (c *Client) InitCredentials(ctx context.Context, userID, email string) (*string, error) {
	body := map[string]any{"email": email}
	if _, err := c.do(ctx, http.MethodPost, "/users/request_reset", body, nil); err != nil {
		return nil, err
	}
	return nil, nil
}
