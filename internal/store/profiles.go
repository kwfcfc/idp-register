// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// profileColumns is the canonical select order for permission_profiles.
const profileColumns = `id, label, description, groups,
	public_selectable, public_label, sort_order, created_at, updated_at`

// jsonArray marshals a string slice for JSON-in-TEXT storage (never nil JSON).
func jsonArray(in []string) string {
	if in == nil {
		in = []string{}
	}
	b, _ := json.Marshal(in)
	return string(b)
}

// parseJSONArray decodes a JSON-in-TEXT string slice, tolerating empty/NULL.
func parseJSONArray(s string) []string {
	if s == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func scanProfile(sc interface{ Scan(...any) error }) (*PermissionProfile, error) {
	var p PermissionProfile
	var groups string
	var public int
	if err := sc.Scan(&p.ID, &p.Label, &p.Description, &groups,
		&public, &p.PublicLabel, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Groups = parseJSONArray(groups)
	p.PublicSelectable = public != 0
	return &p, nil
}

// PublicService is the anonymous-safe view of a public-selectable profile: the
// public form needs a label and description but must never see the IdP groups.
type PublicService struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// seedProfiles inserts the default permission profiles if absent. Mirrors the
// draft's migration seed but with portable JSON-in-TEXT groups.
func (s *Store) seedProfiles(ctx context.Context) error {
	now := nowMS()
	defaults := []PermissionProfile{
		{ID: "basic", Label: "基础成员", Description: "Matrix 与 GoToSocial",
			Groups: []string{"svc:matrix:user", "svc:gotosocial:user"}},
		{ID: "developer", Label: "开发者", Description: "基础成员权限，加 Forgejo",
			Groups: []string{"svc:matrix:user", "svc:gotosocial:user", "svc:forgejo:user"}},
		{ID: "community-admin", Label: "社区管理员", Description: "社区服务管理权限，不包含基础设施管理",
			Groups: []string{"svc:matrix:user", "svc:gotosocial:admin", "svc:forgejo:user"}},
	}
	for _, p := range defaults {
		_, err := s.exec(ctx,
			`INSERT INTO permission_profiles
			   (id, label, description, groups, public_selectable, public_label, sort_order, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (id) DO NOTHING`,
			p.ID, p.Label, p.Description, jsonArray(p.Groups), 0, "", 0, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListProfiles returns all permission profiles ordered by id.
func (s *Store) ListProfiles(ctx context.Context) ([]PermissionProfile, error) {
	rows, err := s.query(ctx,
		`SELECT `+profileColumns+` FROM permission_profiles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PermissionProfile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ListPublicServices returns the anonymous-safe options for the public form,
// ordered for display. Groups are deliberately omitted.
func (s *Store) ListPublicServices(ctx context.Context) ([]PublicService, error) {
	rows, err := s.query(ctx,
		`SELECT id, label, public_label, description
		   FROM permission_profiles
		  WHERE public_selectable = 1
		  ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PublicService
	for rows.Next() {
		var id, label, publicLabel, desc string
		if err := rows.Scan(&id, &label, &publicLabel, &desc); err != nil {
			return nil, err
		}
		if publicLabel == "" {
			publicLabel = label
		}
		out = append(out, PublicService{ID: id, Label: publicLabel, Description: desc})
	}
	return out, rows.Err()
}

// PublicServiceIDs returns the set of profile ids that are public-selectable, so
// the application service can validate a submitted selection.
func (s *Store) PublicServiceIDs(ctx context.Context) (map[string]bool, error) {
	rows, err := s.query(ctx,
		`SELECT id FROM permission_profiles WHERE public_selectable = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// GetProfile returns a single profile, or ErrNotFound.
func (s *Store) GetProfile(ctx context.Context, id string) (*PermissionProfile, error) {
	p, err := scanProfile(s.queryRow(ctx,
		`SELECT `+profileColumns+` FROM permission_profiles WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// CreateProfile inserts a new profile. Returns ErrConflict if the id is taken.
func (s *Store) CreateProfile(ctx context.Context, p *PermissionProfile) error {
	now := nowMS()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err := s.exec(ctx,
		`INSERT INTO permission_profiles
		   (id, label, description, groups, public_selectable, public_label, sort_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Label, p.Description, jsonArray(p.Groups),
		boolToInt(p.PublicSelectable), p.PublicLabel, p.SortOrder, p.CreatedAt, p.UpdatedAt)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

// UpdateProfile replaces the mutable fields of an existing profile. Returns
// ErrNotFound when the id does not exist.
func (s *Store) UpdateProfile(ctx context.Context, p *PermissionProfile) error {
	p.UpdatedAt = nowMS()
	res, err := s.exec(ctx,
		`UPDATE permission_profiles
		    SET label = ?, description = ?, groups = ?,
		        public_selectable = ?, public_label = ?, sort_order = ?, updated_at = ?
		  WHERE id = ?`,
		p.Label, p.Description, jsonArray(p.Groups),
		boolToInt(p.PublicSelectable), p.PublicLabel, p.SortOrder, p.UpdatedAt, p.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteProfile removes a profile. Returns ErrNotFound when absent, or
// ErrConflict when a token or application still references it (FK violation).
func (s *Store) DeleteProfile(ctx context.Context, id string) error {
	res, err := s.exec(ctx, `DELETE FROM permission_profiles WHERE id = ?`, id)
	if isFKViolation(err) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
