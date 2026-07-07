// SPDX-License-Identifier: GPL-3.0-or-later

package rauthy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/provisioner"
)

func TestClientListGroupsAndFindUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAPIKey(t, r)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/groups":
			writeTestJSON(t, w, []provisioner.Group{{ID: "g1", Name: "svc:matrix:user"}})
		case r.Method == http.MethodGet && requestEmailPath(t, r) == "user@example.test":
			writeTestJSON(t, w, rauthyUser{
				ID:     "user-1",
				Email:  "user@example.test",
				Groups: []string{"svc:matrix:user"},
				Roles:  []string{"user"},
			})
		case r.Method == http.MethodGet && requestEmailPath(t, r) == "missing@example.test":
			http.NotFound(w, r)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	groups, err := c.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if !reflect.DeepEqual(groups, []provisioner.Group{{ID: "g1", Name: "svc:matrix:user"}}) {
		t.Fatalf("unexpected groups: %+v", groups)
	}

	u, ok, err := c.FindUserByEmail(context.Background(), "user@example.test")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if !ok || u.ID != "user-1" || u.Email != "user@example.test" ||
		!reflect.DeepEqual(u.Groups, []string{"svc:matrix:user"}) {
		t.Fatalf("unexpected found user: ok=%v user=%+v", ok, u)
	}

	if u, ok, err := c.FindUserByEmail(context.Background(), "missing@example.test"); err != nil || ok || u != nil {
		t.Fatalf("missing user should return nil,false,nil; got user=%+v ok=%v err=%v", u, ok, err)
	}
}

func TestClientCreateUserSendsExpectedPayload(t *testing.T) {
	var sawPost, sawUsername bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAPIKey(t, r)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/users":
			sawPost = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if body["email"] != "new@example.test" {
				t.Fatalf("email not sent correctly: %+v", body)
			}
			if body["language"] != "de" || body["tz"] != "Europe/Berlin" {
				t.Fatalf("language/timezone not sent correctly: %+v", body)
			}
			if body["family_name"] != nil || body["given_name"] != nil || body["user_expires"] != nil {
				t.Fatalf("nullable Rauthy fields should be explicit nils: %+v", body)
			}
			if !stringSliceEqual(body["groups"], []string{"svc:matrix:user"}) {
				t.Fatalf("groups not sent correctly: %+v", body["groups"])
			}
			if !stringSliceEqual(body["roles"], nil) {
				t.Fatalf("roles should default to an empty list: %+v", body["roles"])
			}
			writeTestJSON(t, w, rauthyUser{ID: "created-1", Email: "new@example.test"})
		case r.Method == http.MethodPut && r.URL.EscapedPath() == "/users/created-1/self/preferred_username":
			sawUsername = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode username body: %v", err)
			}
			if body["preferred_username"] != "newuser" || body["force_overwrite"] != false {
				t.Fatalf("unexpected username body: %+v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	id, err := c.CreateUser(context.Background(), provisioner.NewUser{
		Email:    "new@example.test",
		Username: "newuser",
		Groups:   []string{"svc:matrix:user"},
		Language: "de",
		Timezone: "Europe/Berlin",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if id != "created-1" || !sawPost || !sawUsername {
		t.Fatalf("create flow incomplete: id=%q sawPost=%v sawUsername=%v", id, sawPost, sawUsername)
	}
}

func TestClientCreateUserMapsExistsErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{name: "conflict", status: http.StatusConflict, body: "already exists"},
		{name: "rauthy unique email", status: http.StatusNotAcceptable, body: "UNIQUE constraint failed: users.email"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requireAPIKey(t, r)
				if r.Method != http.MethodPost || r.URL.Path != "/users" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				http.Error(w, tc.body, tc.status)
			}))
			defer srv.Close()

			_, err := testClient(srv.URL).CreateUser(context.Background(), provisioner.NewUser{Email: "exists@example.test"})
			if !errors.Is(err, provisioner.ErrUserExists) {
				t.Fatalf("want ErrUserExists, got %v", err)
			}
		})
	}
}

func TestClientCreateUserUsernameFailureIsPartialSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAPIKey(t, r)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/users":
			writeTestJSON(t, w, rauthyUser{ID: "created-1", Email: "partial@example.test"})
		case r.Method == http.MethodPut && r.URL.EscapedPath() == "/users/created-1/self/preferred_username":
			http.Error(w, "username unavailable", http.StatusConflict)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	id, err := testClient(srv.URL).CreateUser(context.Background(), provisioner.NewUser{
		Email:    "partial@example.test",
		Username: "partial",
	})
	if id != "created-1" {
		t.Fatalf("partial success should return created id, got %q", id)
	}
	if !errors.Is(err, provisioner.ErrUsernameNotSet) {
		t.Fatalf("want ErrUsernameNotSet wrapper, got %v", err)
	}
}

func TestIntegrationAgainstRauthy(t *testing.T) {
	if os.Getenv("RAUTHY_INTEGRATION") != "1" {
		t.Skip("set RAUTHY_INTEGRATION=1 and Rauthy env vars to run")
	}
	base := os.Getenv("RAUTHY_API_BASE")
	keyName := os.Getenv("RAUTHY_API_KEY_NAME")
	key := os.Getenv("RAUTHY_API_KEY_SECRET")
	if base == "" || keyName == "" || key == "" {
		t.Fatal("RAUTHY_API_BASE, RAUTHY_API_KEY_NAME, and RAUTHY_API_KEY_SECRET are required")
	}

	c := New(Config{
		APIBase:    strings.TrimRight(base, "/"),
		APIKeyName: keyName,
		APIKey:     key,
		Language:   "en",
		Timezone:   "UTC",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := c.ListGroups(ctx); err != nil {
		t.Fatalf("list groups against Rauthy: %v", err)
	}

	email := fmt.Sprintf("rauthy-it-%d@example.test", time.Now().UnixNano())
	id, err := c.CreateUser(ctx, provisioner.NewUser{Email: email})
	if err != nil {
		t.Fatalf("create user against Rauthy: %v", err)
	}
	if id == "" {
		t.Fatal("Rauthy returned an empty user id")
	}

	u, ok, err := c.FindUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("find created user against Rauthy: %v", err)
	}
	if !ok || u == nil || u.ID != id || !strings.EqualFold(u.Email, email) {
		t.Fatalf("created user lookup mismatch: id=%q user=%+v ok=%v", id, u, ok)
	}
}

func testClient(base string) *Client {
	return New(Config{
		APIBase:    base,
		APIKeyName: "idp-register",
		APIKey:     "secret",
		Language:   "en",
		Timezone:   "UTC",
	})
}

func requireAPIKey(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "API-Key idp-register$secret" {
		t.Fatalf("unexpected Authorization header: %q", got)
	}
	if got := r.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("unexpected Accept header: %q", got)
	}
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func stringSliceEqual(v any, want []string) bool {
	raw, ok := v.([]any)
	if !ok {
		return false
	}
	if len(raw) != len(want) {
		return false
	}
	for i, item := range raw {
		if item != want[i] {
			return false
		}
	}
	return true
}

func requestEmailPath(t *testing.T, r *http.Request) string {
	t.Helper()
	escaped := strings.TrimPrefix(r.URL.EscapedPath(), "/users/email/")
	email, err := url.PathUnescape(escaped)
	if err != nil {
		t.Fatalf("unescape email path: %v", err)
	}
	return email
}
