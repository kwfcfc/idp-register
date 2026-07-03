// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"os"
	"strings"
	"testing"
)

// The operator-facing copies in migrations/ are hand-synced with the embedded
// schemas and must stay SQL-equivalent (comments and whitespace may differ).
func TestMigrationFilesMatchEmbeddedSchemas(t *testing.T) {
	cases := []struct {
		name     string
		embedded string
		path     string
	}{
		{"sqlite", schemaSQLite, "../../migrations/sqlite/001_init.sql"},
		{"postgres", schemaPostgres, "../../migrations/postgres/001_init.sql"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			want := normalizeSQL(tc.embedded)
			got := normalizeSQL(string(b))
			if want != got {
				t.Fatalf("%s has drifted from the embedded schema; re-sync it.\nembedded: %s\nmigration: %s",
					tc.path, want, got)
			}
		})
	}
}

// normalizeSQL strips '--' line comments and collapses all whitespace, so only
// the executable SQL is compared. Our schemas contain no string literals, so a
// line-based scan is sufficient.
func normalizeSQL(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte(' ')
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
