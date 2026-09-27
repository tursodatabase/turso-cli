package cmd

import (
	"encoding/json"
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
	previousOrg, previousURL, previousFlag := config.Organization(), viper.GetString("baseURL"), mintOrgFlag
	t.Cleanup(func() {
		config.SetOrganization(previousOrg)
		viper.Set("baseURL", previousURL)
		mintOrgFlag = previousFlag
	})
	config.SetOrganization("selected-org")
	t.Setenv(ENV_ACCESS_TOKEN, "test-token")

	for _, tc := range []struct {
		name, flag, want string
	}{
		{"selected organization", "", "selected-org"},
		{"explicit organization", "other-org", "other-org"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mintOrgFlag = tc.flag
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
					var body struct {
						Organization string `json:"organization"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("missing or invalid request body: %v", err)
					} else if body.Organization != tc.want {
						t.Errorf("organization = %q, want %q", body.Organization, tc.want)
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
