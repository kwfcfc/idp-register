// SPDX-License-Identifier: GPL-3.0-or-later

// Command server is the idp-register entrypoint: it loads config, opens the
// database, wires the provisioner, admin OIDC RP, and services, then serves the
// embedded SPA + JSON API. See AGENTS.md for the architecture.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/application"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/audit"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/oidcauth"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/provisioner"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/provisioner/rauthy"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/token"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()
	log.Printf("database ready (driver=%s)", cfg.DBDriver)

	// Provisioning runs synchronously inside a request; a row still in
	// 'provisioning' after a restart was interrupted and can never leave that
	// state on its own. Recover it into the normal retry/reject path.
	if n, err := st.RecoverStaleProvisioning(ctx,
		"provisioning was interrupted by a service restart; retry or reject"); err != nil {
		return err
	} else if n > 0 {
		log.Printf("recovered %d application(s) stuck in provisioning", n)
	}

	prov, err := buildProvisioner(cfg)
	if err != nil {
		return err
	}
	log.Printf("provisioner ready (%s)", prov.Name())

	auth, err := oidcauth.New(ctx, cfg, st)
	if err != nil {
		return err
	}
	log.Printf("admin OIDC RP ready (issuer=%s)", cfg.OIDCIssuer)

	auditLog := audit.New(st)
	tokenSvc := token.New(st, auditLog)
	appSvc := application.New(st, prov, auditLog, cfg.ProfileGroupDenylist)
	srv := web.New(cfg, auth, appSvc, tokenSvc, st)

	go purgeSessions(ctx, st)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("listening on %s (origin=%s)", cfg.Addr, cfg.Origin)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

func buildProvisioner(cfg *config.Config) (provisioner.Provisioner, error) {
	switch cfg.ProvisionerKind {
	case "rauthy":
		return rauthy.New(rauthy.Config{
			APIBase:    cfg.RauthyAPIBase,
			APIKeyName: cfg.RauthyAPIKeyName,
			APIKey:     cfg.RauthyAPIKey,
			Language:   cfg.RauthyLanguage,
			Timezone:   cfg.RauthyTimezone,
		}), nil
	default:
		return nil, errors.New("unknown provisioner: " + cfg.ProvisionerKind)
	}
}

// purgeSessions periodically deletes expired admin sessions.
func purgeSessions(ctx context.Context, st *store.Store) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := st.PurgeExpiredSessions(ctx, time.Now().UnixMilli()); err != nil {
				log.Printf("session purge: %v", err)
			}
		}
	}
}
