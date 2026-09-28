package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/tursodatabase/turso-cli/internal/settings"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func TestInvalidateGroupedDatabaseTokensCommand(t *testing.T) {
	config, err := settings.ReadSettings()
	if err != nil {
		t.Fatal(err)
	}
	previousOrg, previousURL, previousYes := config.Organization(), viper.GetString("baseURL"), yesFlag
	t.Cleanup(func() {
		config.SetOrganization(previousOrg)
		viper.Set("baseURL", previousURL)
		yesFlag = previousYes
	})
	config.SetOrganization("org")
	yesFlag = true
	t.Setenv(ENV_ACCESS_TOKEN, "test-token")

	rotations := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/org/databases":
			w.Write([]byte(`{"databases":[{"name":"db","group":"group","version":"v1"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/organizations/org/databases/db/auth/rotate":
			rotations++
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	viper.Set("baseURL", server.URL)

	if err := dbInvalidateTokensCmd.RunE(dbInvalidateTokensCmd, []string{"db"}); err != nil {
		t.Fatal(err)
	}
	if rotations != 1 {
		t.Errorf("database rotations = %d, want 1", rotations)
	}
}

func TestRotateDatabaseTokens(t *testing.T) {
	for _, tc := range []struct {
		name     string
		database turso.Database
		status   int
		wantErr  string
	}{
		{"grouped database", turso.Database{Name: "db", Group: "group", Version: "v1"}, http.StatusOK, ""},
		{"tech-preview grouped database", turso.Database{Name: "db", Group: "group", Version: "tech-preview"}, http.StatusOK, ""},
		{"ungrouped database", turso.Database{Name: "db"}, http.StatusOK, ""},
		{"legacy dedicated grouped database", turso.Database{Name: "db", Group: "group", Version: "v1"}, http.StatusBadRequest, "cannot rotate credentials for a single database in a group"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodPost || r.URL.Path != "/v1/organizations/org/databases/db/auth/rotate" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
					return
				}
				w.WriteHeader(tc.status)
				if tc.status != http.StatusOK {
					w.Write([]byte(`{"error":"cannot rotate credentials for a single database in a group"}`))
				}
			}))
			defer server.Close()
			base, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}

			err = rotate(turso.New(base, "token", "dev", "org"), tc.database)
			if requests != 1 {
				t.Errorf("requests = %d, want 1", requests)
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("rotate: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("rotate error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}
