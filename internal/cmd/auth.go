package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

//go:embed login.html
var LOGIN_HTML string

var headlessFlag bool

var authCmd = &cobra.Command{
	Use:               "auth",
	Short:             "Authenticate with Turso",
	ValidArgsFunction: noSpaceArg,
}

var loginCmd = &cobra.Command{
	Use:               "login",
	Short:             "Login to the platform.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE:              login,
}

var logoutCmd = &cobra.Command{
	Use:               "logout",
	Short:             "Log out currently logged in user.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE:              logout,
}

var tokenCmd = &cobra.Command{
	Use:               "token",
	Short:             "Show token used for authorization.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		settings, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("could not retrieve local config: %w", err)
		}
		token := settings.GetToken()
		if !isJwtTokenValid(token) {
			return fmt.Errorf("no user logged in. Run `turso auth login` to log in and get a token")
		}
		fmt.Println(token)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(tokenCmd)
	loginCmd.Flags().BoolVar(&headlessFlag, "headless", false, "Show access token on the website instead of updating the CLI.")
}

func isJwtTokenValid(token string) bool {
	if len(token) == 0 {
		return false
	}
	resp, err := createTursoClient().Get("/v2/validate/token", nil)
	return err == nil && resp.StatusCode == http.StatusOK
}

func login(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	settings, err := settings.ReadSettings()
	if err != nil {
		return fmt.Errorf("could not retrieve local config: %w", err)
	}
	if isJwtTokenValid(settings.GetToken()) {
		username := settings.GetUsername()
		if len(username) > 0 {
			fmt.Printf("Already signed in as %s. Use %s to log out of this account\n", username, turso.Emph("turso auth logout"))
		} else {
			fmt.Println("✔  Success! Existing JWT still valid")
		}
		return nil
	}
	versionChannel := make(chan string, 1)

	go func() {
		latestVersion, err := fetchLatestVersion()
		if err != nil {
			// On error we just behave as the version check has never happend
			versionChannel <- version
			return
		}
		versionChannel <- latestVersion
	}()

	if headlessFlag {
		url, err := beginAuth(0, headlessFlag)
		if err != nil {
			return fmt.Errorf("internal error. Cannot initiate auth flow: %w", err)
		}
		fmt.Println("Visit this URL on this device to log in:")
		fmt.Println(url)
	} else {
		ch := make(chan string, 1)
		server, err := createCallbackServer(ch)
		if err != nil {
			return fmt.Errorf("internal error. Cannot create callback: %w", err)
		}

		port, err := runServer(server)
		if err != nil {
			return fmt.Errorf("internal error. Cannot run authentication server: %w", err)
		}

		url, err := beginAuth(port, headlessFlag)
		if err != nil {
			return fmt.Errorf("internal error. Cannot initiate auth flow: %w", err)
		}
		fmt.Println("Visit this URL on this device to log in:")
		fmt.Println(url)
		fmt.Println("Waiting for authentication...")

		jwt := <-ch
		username := <-ch

		server.Shutdown(context.Background())

		err = settings.SetToken(jwt)
		if err != nil {
			return fmt.Errorf("error persisting token on local config: %w", err)
		}

		err = settings.SetUsername(username)
		if err != nil {
			return fmt.Errorf("error persisting username on local config: %w", err)
		}
		fmt.Printf("✔  Success! Logged in as %s\n", username)
	}

	latestVersion := <-versionChannel

	if version != latestVersion {

		fmt.Printf("\nFriendly reminder that there's a newer version of %s available.\n", turso.Emph("Turso CLI"))
		fmt.Printf("You're currently using version %s while latest available version is %s.\n", turso.Emph(version), turso.Emph(latestVersion))
		fmt.Printf("Please consider updating to get new features and more stable experience.\n\n")
	}

	return nil
}

func fetchLatestVersion() (string, error) {
	resp, err := createUnauthenticatedTursoClient().Get("/releases/latest", nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting latest release: %s", resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var versionResp struct {
		Version string `json:"latest"`
	}
	if err := json.Unmarshal(body, &versionResp); err != nil {
		return "", err
	}
	if len(versionResp.Version) == 0 {
		return "", fmt.Errorf("got empty version for latest release")
	}
	return versionResp.Version, nil
}

func beginAuth(port int, headless bool) (string, error) {
	authUrl, err := url.Parse(getHost())
	if err != nil {
		return "", fmt.Errorf("error parsing auth URL: %w", err)
	}
	if !headless {
		authUrl.RawQuery = url.Values{
			"port":     {strconv.Itoa(port)},
			"redirect": {"true"},
		}.Encode()
	} else {
		authUrl.RawQuery = url.Values{
			"redirect": {"false"},
		}.Encode()
	}

	err = browser.OpenURL(authUrl.String())
	if err != nil {
		fmt.Println("error: Unable to open browser.")
	}

	return authUrl.String(), nil
}

func createCallbackServer(ch chan string) (*http.Server, error) {
	tmpl, err := template.New("login.html").Parse(LOGIN_HTML)
	if err != nil {
		return nil, fmt.Errorf("could not parse login callback template: %w", err)
	}
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		ch <- q.Get("jwt")
		ch <- q.Get("username")

		w.WriteHeader(200)
		tmpl.Execute(w, q.Get("username"))
	})

	return &http.Server{Handler: handler}, nil
}

func runServer(server *http.Server) (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("could not allocate port for http server: %w", err)
	}

	go func() {
		server.Serve(listener)
	}()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

func logout(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	settings, err := settings.ReadSettings()
	if err != nil {
		return fmt.Errorf("could not retrieve local config: %w", err)
	}

	token := settings.GetToken()
	if len(token) == 0 {
		fmt.Println("No user logged in.")
	} else {
		settings.SetToken("")
		settings.SetUsername("")
		fmt.Println("Logged out.")
	}

	return nil
}
