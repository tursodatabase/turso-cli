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
	"os"
	"strconv"
	"text/template"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
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

var signupCmd = &cobra.Command{
	Use:               "signup",
	Short:             "Create a new Turso account.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE:              signup,
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
	Use:   "token",
	Short: "Shows token used to authenticate you to Turso platform API.",
	Long: "" +
		"Shows token used to authenticate you to Turso platform API.\n" +
		"To authenticate to your databases, use " + internal.Emph("turso db token create"),
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
			return fmt.Errorf("no user logged in. Run %s to log in and get a token", internal.Emph("turso auth login"))
		}

		fmt.Fprintln(os.Stderr, internal.Warn("Warning: this token is used to authenticate you to Turso platform API, not your databases."))
		fmt.Fprintf(os.Stderr, "%s %s %s\n", internal.Warn("Use"), internal.Emph("turso db token create"), internal.Warn("to create a database token."))

		fmt.Println(token)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(signupCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(tokenCmd)
	loginCmd.Flags().BoolVar(&headlessFlag, "headless", false, "Show access token on the website instead of updating the CLI.")
}

func isJwtTokenValid(token string) bool {
	if len(token) == 0 {
		return false
	}
	client, err := createTursoClient()
	if err != nil {
		return false
	}
	resp, err := client.Get("/v2/validate/token", nil)
	return err == nil && resp.StatusCode == http.StatusOK
}

func signup(cmd *cobra.Command, args []string) error {
	return auth(cmd, args, "/signup")
}

func login(cmd *cobra.Command, args []string) error {
	return auth(cmd, args, "")
}

func auth(cmd *cobra.Command, args []string, path string) error {
	cmd.SilenceUsage = true
	settings, err := settings.ReadSettings()
	if err != nil {
		return fmt.Errorf("could not retrieve local config: %w", err)
	}
	if isJwtTokenValid(settings.GetToken()) {
		username := settings.GetUsername()
		if len(username) > 0 {
			fmt.Printf("Already signed in as %s. Use %s to log out of this account\n", username, internal.Emph("turso auth logout"))
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
		url, err := beginAuth(0, headlessFlag, path)
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

		url, err := beginAuth(port, headlessFlag, path)
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

	if version != "dev" && version != latestVersion {

		fmt.Printf("\nFriendly reminder that there's a newer version of %s available.\n", internal.Emph("Turso CLI"))
		fmt.Printf("You're currently using version %s while latest available version is %s.\n", internal.Emph(version), internal.Emph(latestVersion))
		fmt.Printf("Please consider updating to get new features and more stable experience.\n\n")
	}

	firstTime := settings.RegisterUse("auth_login")
	client, err := createTursoClient()
	if err != nil {
		return err
	}
	dbs, err := getDatabases(client)
	if firstTime && err == nil && len(dbs) == 0 {
		fmt.Printf("✏️  We are so happy you are here! Now that you are authenticated, it is time to create a database:\n\t%s\n", internal.Emph("turso db create"))
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

func beginAuth(port int, headless bool, path string) (string, error) {
	authUrl, err := url.Parse(fmt.Sprintf("%s%s", getHost(), path))
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
		tmpl.Execute(w, map[string]string{
			"assetsURL": getTursoUrl(),
		})
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
