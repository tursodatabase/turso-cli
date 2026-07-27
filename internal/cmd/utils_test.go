package cmd

import (
	"testing"

	"github.com/tursodatabase/turso-cli/internal/turso"
)

func TestGetDatabaseUrl(t *testing.T) {
	tests := []struct {
		name string
		db   turso.Database
		want string
	}{
		{
			name: "turso database",
			db:   turso.Database{ID: "019db7a2-9210-79e5-afed-0b1755901d50", Hostname: "turso-db.example.com"},
			want: "turso://turso-db.example.com",
		},
		{
			name: "sqlite database",
			db:   turso.Database{ID: "00000000-1100-0000-0000-000000000000", Hostname: "sqlite-db.example.com"},
			want: "libsql://sqlite-db.example.com",
		},
		{
			name: "no id",
			db:   turso.Database{Hostname: "db.example.com"},
			want: "libsql://db.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDatabaseUrl(&tt.db); got != tt.want {
				t.Fatalf("getDatabaseUrl() = %q, want %q", got, tt.want)
			}
		})
	}
}
