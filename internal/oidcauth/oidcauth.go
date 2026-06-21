// SPDX-License-Identifier: GPL-3.0-or-later

// Package oidcauth implements admin login as a standards-only OIDC Relying
// Party (invariant #2: no provider-specific APIs here — the admin IdP may be
// Rauthy, Authentik, Dex, ...). Sessions are opaque, server-side, HttpOnly.
package oidcauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	sessionCookie  = "idp_register_session"
	stateCookie    = "idp_register_oidc_state"
	nonceCookie    = "idp_register_oidc_nonce"
	verifierCookie = "idp_register_oidc_verifier"
)

// Authenticator handles the OIDC code flow and session lifecycle.
type Authenticator struct {
	cfg      *config.Config
	store    *store.Store
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
}

// New discovers the issuer and builds the RP. Call once at startup.
func New(ctx context.Context, cfg *config.Config, s *store.Store) (*Authenticator, error) {
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	return &Authenticator{
		cfg:      cfg,
		store:    s,
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		oauth: oauth2.Config{
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			RedirectURL:  cfg.OIDCRedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       cfg.OIDCScopes,
		},
	}, nil
}

// BeginLogin sets short-lived state/nonce/PKCE cookies and returns the IdP auth
// URL. PKCE (S256) is sent unconditionally: it is a standards feature every
// modern OIDC IdP accepts, and some (e.g. Rauthy) require it — see invariant #2,
// this RP stays provider-neutral.
func (a *Authenticator) BeginLogin(w http.ResponseWriter) (string, error) {
	state, err := randomToken()
	if err != nil {
		return "", err
	}
	nonce, err := randomToken()
	if err != nil {
		return "", err
	}
	verifier := oauth2.GenerateVerifier()
	a.setTempCookie(w, stateCookie, state)
	a.setTempCookie(w, nonceCookie, nonce)
	a.setTempCookie(w, verifierCookie, verifier)
	return a.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), nil
}

// ErrUnauthorized indicates a valid login that is not permitted admin access.
var ErrUnauthorized = errors.New("not authorized for admin access")

// CompleteLogin validates the callback, enforces admin authorization, creates a
// session, and sets the session cookie. Returns the authenticated admin.
func (a *Authenticator) CompleteLogin(ctx context.Context, w http.ResponseWriter, r *http.Request) (*store.AdminUser, error) {
	wantState, err := r.Cookie(stateCookie)
	if err != nil {
		return nil, errors.New("missing state cookie")
	}
	if r.URL.Query().Get("state") != wantState.Value {
		return nil, errors.New("state mismatch")
	}
	verifier, err := r.Cookie(verifierCookie)
	if err != nil {
		return nil, errors.New("missing PKCE verifier cookie")
	}
	a.clearTempCookies(w)

	oauth2Token, err := a.oauth.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(verifier.Value))
	if err != nil {
		return nil, fmt.Errorf("code exchange: %w", err)
	}
	rawID, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token in token response")
	}
	idToken, err := a.verifier.Verify(ctx, rawID)
	if err != nil {
		return nil, fmt.Errorf("verify id_token: %w", err)
	}
	if wantNonce, err := r.Cookie(nonceCookie); err == nil && idToken.Nonce != wantNonce.Value {
		return nil, errors.New("nonce mismatch")
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	user := a.userFromClaims(idToken.Subject, claims)
	if !a.isAdmin(user) {
		return nil, ErrUnauthorized
	}
	if err := a.createSession(ctx, w, user); err != nil {
		return nil, err
	}
	return &user, nil
}

// Authenticate resolves the current admin from the session cookie, or nil.
func (a *Authenticator) Authenticate(ctx context.Context, r *http.Request) *store.AdminUser {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	user, err := a.store.LoadSession(ctx, digest(c.Value), time.Now().UnixMilli())
	if err != nil {
		return nil
	}
	return user
}

// Logout destroys the session and clears the cookie.
func (a *Authenticator) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		_ = a.store.DeleteSession(ctx, digest(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.cfg.SecureCookies, SameSite: http.SameSiteLaxMode,
	})
}

func (a *Authenticator) createSession(ctx context.Context, w http.ResponseWriter, user store.AdminUser) error {
	raw, err := randomToken()
	if err != nil {
		return err
	}
	expires := time.Now().Add(a.cfg.SessionTTL)
	if err := a.store.CreateSession(ctx, uuid.NewString(), digest(raw), user, expires.UnixMilli()); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: raw, Path: "/",
		HttpOnly: true, Secure: a.cfg.SecureCookies, SameSite: http.SameSiteLaxMode,
		Expires: expires,
	})
	return nil
}

func (a *Authenticator) userFromClaims(sub string, claims map[string]any) store.AdminUser {
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := claims[k].(string); ok && v != "" {
				return v
			}
		}
		return ""
	}
	return store.AdminUser{
		Sub:         sub,
		Email:       str("email"),
		DisplayName: firstNonEmpty(str("name", "preferred_username"), str("email"), sub),
		Groups:      normalizeGroups(claims[a.cfg.OIDCGroupsClaim]),
	}
}

func (a *Authenticator) isAdmin(u store.AdminUser) bool {
	for _, g := range u.Groups {
		if g == a.cfg.AdminGroup {
			return true
		}
	}
	for _, s := range a.cfg.AdminSubs {
		if s == u.Sub {
			return true
		}
	}
	email := strings.ToLower(u.Email)
	for _, e := range a.cfg.AdminEmails {
		if e == email {
			return true
		}
	}
	return false
}

// normalizeGroups accepts an array claim or a space/comma-separated string.
func normalizeGroups(claim any) []string {
	switch v := claim.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	case string:
		return strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == ',' })
	default:
		return nil
	}
}

func (a *Authenticator) setTempCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/", MaxAge: 600,
		HttpOnly: true, Secure: a.cfg.SecureCookies, SameSite: http.SameSiteLaxMode,
	})
}

func (a *Authenticator) clearTempCookies(w http.ResponseWriter) {
	for _, n := range []string{stateCookie, nonceCookie, verifierCookie} {
		http.SetCookie(w, &http.Cookie{
			Name: n, Value: "", Path: "/", MaxAge: -1,
			HttpOnly: true, Secure: a.cfg.SecureCookies, SameSite: http.SameSiteLaxMode,
		})
	}
}

// SessionCookieName exposes the session cookie name (used for CSRF origin checks).
func SessionCookieName() string { return sessionCookie }

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
