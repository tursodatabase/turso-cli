package cmd

import (
	"strings"
	"testing"

	"github.com/tursodatabase/turso-cli/internal/turso"
)

func TestDbListModelViewIncludesTypeColumn(t *testing.T) {
	model := dbListModel{
		databases: []turso.Database{
			{
				Name:     "sqlite-db",
				ID:       "00000000-1100-0000-0000-000000000000",
				Hostname: "sqlite-db.example.com",
			},
			{
				Name:     "turso-db",
				ID:       "019db7a2-9210-79e5-afed-0b1755901d50",
				Group:    "default",
				Hostname: "turso-db.example.com",
			},
		},
	}

	view := model.View()
	lines := strings.Split(strings.TrimSpace(view), "\n")
	expected := [][]string{
		{"NAME", "TYPE", "GROUP", "URL"},
		{"sqlite-db", "SQLite", "-", "libsql://sqlite-db.example.com"},
		{"turso-db", "Turso", "default", "turso://turso-db.example.com"},
	}

	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d:\n%s", len(expected), len(lines), view)
	}

	for i, want := range expected {
		got := strings.Fields(lines[i])
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Fatalf("line %d fields = %q, want %q:\n%s", i, got, want, view)
		}
	}
}
