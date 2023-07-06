package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"text/template"

	"github.com/chiselstrike/turso-cli/internal"
	"github.com/chiselstrike/turso-cli/internal/settings"
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
	PersistentPreRunE: verifyIfTokenIsSetInEnv,
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
	Short: "Shows the token used to authenticate you to Turso platform API.",
	Long: "" +
		"Shows the token used to authenticate you to Turso platform API.\n" +
		"To authenticate to your databases, use " + internal.Emph("turso db tokens create"),
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
		fmt.Fprintf(os.Stderr, "%s %s %s\n", internal.Warn("Use"), internal.Emph("turso db tokens create"), internal.Warn("to create a database token."))

		fmt.Println(token)
		return nil
	},
}

var apiTokensCmd = &cobra.Command{
	Use:   "api-tokens",
	Short: "Manage your API tokens",
	Long: "" +
		"API tokens are revocable non-expiring tokens that authenticate holders as the user who created them.\n" +
		"They can be used to implement automations with the " + internal.Emph("turso") + " CLI or the platform API.",
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(signupCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(tokenCmd)
	authCmd.AddCommand(apiTokensCmd)
	loginCmd.Flags().BoolVar(&headlessFlag, "headless", false, "Show access token on the website instead of updating the CLI.")
}

func isJwtTokenValid(token string) bool {
	if len(token) == 0 {
		return false
	}
	if tokenValidCache(token) {
		return true
	}
	client, err := tursoClient(token)
	if err != nil {
		return false
	}
	exp, err := client.Tokens.Validate(token)
	if err != nil {
		return false
	}
	setTokenValidCache(token, exp)
	return true
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

		settings.SetToken(jwt)
		settings.SetUsername(username)

		fmt.Printf("✔  Success! Logged in as %s\n", username)

		firstTime := settings.RegisterUse("auth_login")
		client, err := createTursoClientFromAccessToken(false)
		if err != nil {
			return err
		}
		dbs, err := client.Databases.List()
		if firstTime && err == nil && len(dbs) == 0 {
			fmt.Printf("✏️  We are so happy you are here! Now that you are authenticated, it is time to create a database:\n\t%s\n", internal.Emph("turso db create"))
		}
	}

	latestVersion := <-versionChannel

	if version != "dev" && version != latestVersion {

		fmt.Printf("\nFriendly reminder that there's a newer version of %s available.\n", internal.Emph("Turso CLI"))
		fmt.Printf("You're currently using version %s while latest available version is %s.\n", internal.Emph(version), internal.Emph(latestVersion))
		fmt.Printf("Please consider updating to get new features and more stable experience. To update:\n\n")
		fmt.Printf("\n\t%s\n", internal.Emph("turso update"))
	}

	return nil
}

func beginAuth(port int, headless bool, path string) (string, error) {
	authUrl, err := url.Parse(fmt.Sprintf("%s%s", getTursoUrl(), path))
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

func verifyIfTokenIsSetInEnv(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	envToken := os.Getenv(ENV_ACCESS_TOKEN)
	if envToken != "" {
		return fmt.Errorf("auth commands aren't effective when a token is set in the environment variable")
	}

	return nil
}
