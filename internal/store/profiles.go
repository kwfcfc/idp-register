// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

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
			`INSERT INTO permission_profiles (id, label, description, groups, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (id) DO NOTHING`,
			p.ID, p.Label, p.Description, jsonArray(p.Groups), now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListProfiles returns all permission profiles ordered by id.
func (s *Store) ListProfiles(ctx context.Context) ([]PermissionProfile, error) {
	rows, err := s.query(ctx,
		`SELECT id, label, description, groups, created_at, updated_at
		   FROM permission_profiles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PermissionProfile
	for rows.Next() {
		var p PermissionProfile
		var groups string
		if err := rows.Scan(&p.ID, &p.Label, &p.Description, &groups, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Groups = parseJSONArray(groups)
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProfile returns a single profile, or ErrNotFound.
func (s *Store) GetProfile(ctx context.Context, id string) (*PermissionProfile, error) {
	var p PermissionProfile
	var groups string
	err := s.queryRow(ctx,
		`SELECT id, label, description, groups, created_at, updated_at
		   FROM permission_profiles WHERE id = ?`, id).
		Scan(&p.ID, &p.Label, &p.Description, &groups, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Groups = parseJSONArray(groups)
	return &p, nil
}
