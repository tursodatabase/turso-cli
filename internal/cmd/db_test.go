package cmd

import (
	"testing"

	"github.com/tursodatabase/turso-cli/internal/turso"
)

func TestSuggestDatabaseNameFromHostnamePrefix(t *testing.T) {
	databases := []turso.Database{
		{
			Name:     "hello",
			Hostname: "hello-penberg.turso.io",
		},
		{
			Name:     "other",
			Hostname: "other-penberg.turso.io",
		},
	}

	got := suggestDatabaseName("hello-penberg", databases)
	if got != "hello" {
		t.Fatalf("suggestDatabaseName() = %q, want %q", got, "hello")
	}
}

func TestSuggestDatabaseNameIgnoresUnmatchedName(t *testing.T) {
	databases := []turso.Database{
		{
			Name:     "hello",
			Hostname: "hello-penberg.turso.io",
		},
	}

	got := suggestDatabaseName("missing", databases)
	if got != "" {
		t.Fatalf("suggestDatabaseName() = %q, want empty string", got)
	}
}
