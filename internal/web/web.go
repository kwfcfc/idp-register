// SPDX-License-Identifier: GPL-3.0-or-later

// Package web is the HTTP layer: it serves the embedded SPA and a JSON API. The
// public surface is anti-enumeration (uniform responses); the admin surface is
// gated by an OIDC session with an Origin-based CSRF check on mutations.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/application"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/oidcauth"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/token"
)

//go:embed assets
var assets embed.FS

// Server wires the HTTP handlers to the service layer.
type Server struct {
	cfg    *config.Config
	auth   *oidcauth.Authenticator
	apps   *application.Service
	tokens *token.Service
	store  *store.Store
	http   *http.Client
}

// New builds a Server.
func New(cfg *config.Config, auth *oidcauth.Authenticator, apps *application.Service, tokens *token.Service, s *store.Store) *Server {
	return &Server{
		cfg: cfg, auth: auth, apps: apps, tokens: tokens, store: s,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// Handler returns the root http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public registration.
	mux.HandleFunc("POST /api/register", s.handleRegister)

	// Admin OIDC login.
	mux.HandleFunc("GET /auth/login", s.handleLogin)
	mux.HandleFunc("GET /auth/callback", s.handleCallback)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)

	// Admin API (session-gated).
	mux.HandleFunc("GET /api/admin/me", s.admin(s.handleMe))
	mux.HandleFunc("GET /api/admin/profiles", s.admin(s.handleProfiles))
	mux.HandleFunc("GET /api/admin/applications", s.admin(s.handleListApplications))
	mux.HandleFunc("GET /api/admin/applications/{id}", s.admin(s.handleGetApplication))
	mux.HandleFunc("POST /api/admin/applications/{id}/approve", s.admin(s.handleApprove))
	mux.HandleFunc("POST /api/admin/applications/{id}/decide", s.admin(s.handleDecide))
	mux.HandleFunc("GET /api/admin/tokens", s.admin(s.handleListTokens))
	mux.HandleFunc("POST /api/admin/tokens", s.admin(s.handleCreateToken))
	mux.HandleFunc("POST /api/admin/tokens/{id}/active", s.admin(s.handleSetTokenActive))
	mux.HandleFunc("DELETE /api/admin/tokens/{id}", s.admin(s.handleDeleteToken))
	mux.HandleFunc("GET /api/admin/audit", s.admin(s.handleAudit))

	// Static SPA (catch-all) — must be registered last via "/".
	mux.Handle("/", s.spaHandler())

	return mux
}

// ----- public -----

type registerRequest struct {
	Email          string `json:"email"`
	Username       string `json:"username"`
	ReviewText     string `json:"reviewText"`
	InviteCode     string `json:"inviteCode"`
	TurnstileToken string `json:"turnstileToken"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Email) == "" || !strings.Contains(req.Email, "@") {
		// A clearly invalid email is the one thing we can reject without
		// leaking account state.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a valid email is required"})
		return
	}

	if s.cfg.TurnstileSecret != "" {
		if !s.verifyTurnstile(r.Context(), req.TurnstileToken, clientIP(r, s.cfg.TrustedCFIP)) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "captcha verification failed"})
			return
		}
	}

	// Best-effort submit; the response is uniform regardless of outcome so the
	// form cannot be used to enumerate emails or probe invite codes.
	_, _ = s.apps.Submit(r.Context(), application.SubmitInput{
		Email:           req.Email,
		Username:        req.Username,
		ReviewText:      req.ReviewText,
		InviteCode:      req.InviteCode,
		CaptchaProvider: captchaProvider(s.cfg),
		SubmittedIP:     clientIP(r, s.cfg.TrustedCFIP),
	})

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "received",
		"message": "Your application has been received. If approved, you'll get an email to set up your account.",
	})
}

// ----- auth -----

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	url, err := s.auth.BeginLogin(w)
	if err != nil {
		http.Error(w, "login init failed", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.CompleteLogin(r.Context(), w, r)
	if err != nil {
		// Redirect back to a login page with a generic error.
		http.Redirect(w, r, "/login?error=auth", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	s.auth.Logout(r.Context(), w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// ----- admin middleware -----

type adminHandler func(http.ResponseWriter, *http.Request, store.AdminUser)

// admin wraps a handler with session auth and (for mutations) a CSRF origin check.
func (s *Server) admin(h adminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := s.auth.Authenticate(r.Context(), r)
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if r.Method != http.MethodGet && !s.checkCSRF(w, r) {
			return
		}
		h(w, r, *user)
	}
}

// checkCSRF enforces a same-origin check on state-changing requests, since the
// session cookie is SameSite=Lax. Returns false (and writes 403) on mismatch.
func (s *Server) checkCSRF(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Fall back to Referer when Origin is absent.
		origin = r.Header.Get("Referer")
	}
	if origin == "" || !strings.HasPrefix(origin, s.cfg.Origin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request blocked"})
		return false
	}
	return true
}

// ----- admin handlers -----

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, u store.AdminUser) {
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request, _ store.AdminUser) {
	profiles, err := s.store.ListProfiles(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *Server) handleListApplications(w http.ResponseWriter, r *http.Request, _ store.AdminUser) {
	apps, err := s.apps.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *Server) handleGetApplication(w http.ResponseWriter, r *http.Request, _ store.AdminUser) {
	app, err := s.apps.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request, u store.AdminUser) {
	var body struct {
		ProfileID string `json:"profileId"`
		Note      string `json:"note"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := s.apps.Approve(r.Context(), r.PathValue("id"), body.ProfileID, body.Note, u); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleDecide(w http.ResponseWriter, r *http.Request, u store.AdminUser) {
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	ok, err := s.apps.Decide(r.Context(), r.PathValue("id"), body.Status, body.Note, u)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "application is not in a decidable state"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": body.Status})
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request, _ store.AdminUser) {
	validOnly := r.URL.Query().Get("valid") == "true"
	tokens, err := s.tokens.List(r.Context(), validOnly, time.Now().UnixMilli())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request, u store.AdminUser) {
	var body struct {
		Token           string  `json:"token"`
		UsesAllowed     *int64  `json:"usesAllowed"`
		ExpiryTime      *int64  `json:"expiryTime"`
		EmailConstraint *string `json:"emailConstraint"`
		ProfileID       *string `json:"profileId"`
		Note            string  `json:"note"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	t, err := s.tokens.Mint(r.Context(), token.MintParams{
		Token:           body.Token,
		UsesAllowed:     body.UsesAllowed,
		ExpiryTime:      body.ExpiryTime,
		EmailConstraint: body.EmailConstraint,
		ProfileID:       body.ProfileID,
		Note:            body.Note,
	}, u)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleSetTokenActive(w http.ResponseWriter, r *http.Request, u store.AdminUser) {
	var body struct {
		Active bool `json:"active"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	ok, err := s.tokens.SetActive(r.Context(), r.PathValue("id"), body.Active, u)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"active": body.Active})
}

func (s *Server) handleDeleteToken(w http.ResponseWriter, r *http.Request, u store.AdminUser) {
	ok, err := s.tokens.Delete(r.Context(), r.PathValue("id"), u)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request, _ store.AdminUser) {
	entries, err := s.store.ListAudit(r.Context(), 200)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// ----- static SPA -----

func (s *Server) spaHandler() http.Handler {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err) // embedded path is a compile-time constant; cannot fail
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never serve the SPA for unmatched API paths — return JSON 404.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		// SPA fallback: serve index.html for paths without a file extension.
		if p := strings.TrimPrefix(r.URL.Path, "/"); p == "" || !strings.Contains(pathBase(p), ".") {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

func pathBase(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// ----- helpers -----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return false
	}
	return true
}

func captchaProvider(cfg *config.Config) string {
	if cfg.TurnstileSecret != "" {
		return "turnstile"
	}
	return ""
}

// clientIP returns the real client IP, restoring it from CF-Connecting-IP when
// the deployment trusts Cloudflare.
func clientIP(r *http.Request, trustCF bool) string {
	if trustCF {
		if v := r.Header.Get("CF-Connecting-IP"); v != "" {
			return v
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// verifyTurnstile validates a Cloudflare Turnstile token server-side.
func (s *Server) verifyTurnstile(ctx context.Context, token, ip string) bool {
	if token == "" {
		return false
	}
	form := "secret=" + s.cfg.TurnstileSecret + "&response=" + token
	if ip != "" {
		form += "&remoteip=" + ip
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var out struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false
	}
	return out.Success
}
