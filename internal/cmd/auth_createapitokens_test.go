package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/tursodatabase/turso-cli/internal/settings"
)

func TestMintApiTokenOrganization(t *testing.T) {
	config, err := settings.ReadSettings()
	if err != nil {
		t.Fatal(err)
	}
	previousOrg, previousURL := config.Organization(), viper.GetString("baseURL")
	previousMintOrg, previousMintGroup, previousMintScopes := mintOrgFlag, mintGroupFlag, mintScopeFlags
	previousMintReadOnly, previousMintFullAccess := mintReadOnlyFlag, mintFullAccessFlag
	t.Cleanup(func() {
		config.SetOrganization(previousOrg)
		viper.Set("baseURL", previousURL)
		mintOrgFlag, mintGroupFlag, mintScopeFlags = previousMintOrg, previousMintGroup, previousMintScopes
		mintReadOnlyFlag, mintFullAccessFlag = previousMintReadOnly, previousMintFullAccess
	})
	config.SetOrganization("selected-org")
	mintOrgFlag, mintGroupFlag, mintScopeFlags = "", "", nil
	mintReadOnlyFlag, mintFullAccessFlag = false, false
	t.Setenv(ENV_ACCESS_TOKEN, "test-token")

	for _, tc := range []struct {
		name, flag, want string
		readOnly         bool
	}{
		{"selected organization", "", "selected-org", false},
		{"explicit organization", "other-org", "other-org", false},
		{"read-only organization", "other-org", "other-org", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mintOrgFlag = tc.flag
			mintReadOnlyFlag = tc.readOnly
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/auth/validate":
					w.Write([]byte(`{"exp":4102444800}`))
				case "/v2/organizations":
					w.Write([]byte(`{"organizations":[{"slug":"selected-org"},{"slug":"other-org"}]}`))
				case "/v2/auth/api-tokens/test-token-name":
					if r.Method != http.MethodPost {
						t.Errorf("method = %s, want POST", r.Method)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("missing or invalid request body: %v", err)
					} else if body["organization"] != tc.want {
						t.Errorf("request body = %v, want organization %q", body, tc.want)
					} else if tc.readOnly && fmt.Sprint(body["scopes"]) != "[read-only]" {
						t.Errorf("request body = %v, want scopes [read-only]", body)
					} else if !tc.readOnly && len(body) != 1 {
						t.Errorf("request body = %v, want only organization", body)
					}
					w.Write([]byte(`{"token":{"value":"minted"}}`))
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			viper.Set("baseURL", server.URL)
			if err := createApiTokensCmd.RunE(createApiTokensCmd, []string{"test-token-name"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
