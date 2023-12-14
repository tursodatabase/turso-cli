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
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/settings"
)

//go:embed login.html
var LOGIN_HTML string

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
	PersistentPreRunE: checkEnvAuth,
}

var loginCmd = &cobra.Command{
	Use:               "login",
	Short:             "Login to the platform.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE:              login,
	PersistentPreRunE: checkEnvAuth,
}

var logoutCmd = &cobra.Command{
	Use:               "logout",
	Short:             "Log out currently logged in user.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE:              logout,
	PersistentPreRunE: checkEnvAuth,
}

var whoAmICmd = &cobra.Command{
	Use:               "whoami",
	Short:             "Show the currently logged in user.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE:              whoAmI,
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
	authCmd.AddCommand(whoAmICmd)
	flags.AddHeadless(loginCmd)
	flags.AddHeadless(signupCmd)
	flags.AddAll(logoutCmd, "Invalidate all sessions for the current user")
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
	return auth(cmd, "/signup")
}

func login(cmd *cobra.Command, args []string) error {
	return auth(cmd, "")
}

func exitOnValidAuth(settings *settings.Settings, path string) {
	username := settings.GetUsername()
	if len(username) <= 0 {
		fmt.Println("✔  Success! Existing JWT still valid")
		return
	}
	if path == "/signup" {
		fmt.Printf("Already signed up as %s\n", username)
		return
	}
	fmt.Printf("Already signed in as %s. Use %s to log out of this account\n", username, internal.Emph("turso auth logout"))
}

func auth(cmd *cobra.Command, path string) error {
	cmd.SilenceUsage = true
	settings, err := settings.ReadSettings()
	if err != nil {
		return fmt.Errorf("could not retrieve local config: %w", err)
	}

	if isJwtTokenValid(settings.GetToken()) {
		exitOnValidAuth(settings, path)
		return nil
	}

	if flags.Headless() {
		return printHeadlessLoginInstructions(path)
	}

	state := randString(32)
	callbackServer, err := authCallbackServer(state)
	if err != nil {
		return suggestHeadless(cmd, err)
	}

	url, err := authURL(callbackServer.Port, path, state)
	if err != nil {
		return fmt.Errorf("failed to get auth URL: %w", err)
	}

	if err := browser.OpenURL(url); err != nil {
		err := fmt.Errorf("failed to open auth URL: %w", err)
		return suggestHeadless(cmd, err)
	}

	fmt.Println("Opening your browser at:")
	fmt.Println(url)
	fmt.Println("Waiting for authentication...")

	jwt := callbackServer.Result()
	username, err := validateToken(jwt)
	if err != nil {
		return suggestHeadless(cmd, err)
	}

	settings.SetToken(jwt)
	settings.SetUsername(username)

	fmt.Printf("✔  Success! Logged in as %s\n", username)

	signupHint(settings)

	return nil
}

func validateToken(token string) (string, error) {
	client, err := tursoClient(token)
	if err != nil {
		return "", fmt.Errorf("could not create client to validate token: %w", err)
	}

	user, err := client.Users.GetUser()
	if err != nil {
		return "", fmt.Errorf("could not validate token: %w", err)

	}

	return user.Username, nil
}

func suggestHeadless(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	cmdWithFlag := cmd.CommandPath() + " --headless"
	return fmt.Errorf("%w\nIf the issue persists, try running %s", err, internal.Emph(cmdWithFlag))
}

func printHeadlessLoginInstructions(path string) error {
	url, err := authURL(0, path, "")
	if err != nil {
		return err
	}
	fmt.Println("Visit the following URL to login:")
	fmt.Println(url)
	return nil
}

func signupHint(config *settings.Settings) {
	client, err := authedTursoClient()
	if err != nil {
		return
	}

	firstTime := config.RegisterUse("auth_login")
	if !firstTime {
		return
	}

	dbs, err := client.Databases.List()
	if err != nil || len(dbs) != 0 {
		return
	}

	fmt.Printf("\n✏️  We are so happy you are here!\nNow that you are authenticated, it is time to create a database:\n\t%s\n", internal.Emph("turso db create"))
}

func authURL(port int, path, state string) (string, error) {
	base, err := url.Parse(getTursoUrl())
	if err != nil {
		return "", fmt.Errorf("error parsing auth URL: %w", err)
	}
	authURL := base.JoinPath(path)

	values := url.Values{
		"redirect": {"false"},
	}
	if port != 0 {
		values = url.Values{
			"port":     {strconv.Itoa(port)},
			"redirect": {"true"},
			"type":     {"cli"},
		}
	}
	if state != "" {
		values["state"] = []string{state}
	}
	authURL.RawQuery = values.Encode()
	return authURL.String(), nil
}

type authCallback struct {
	ch     chan string
	server *http.Server
	Port   int
}

func authCallbackServer(state string) (authCallback, error) {
	ch := make(chan string, 1)
	server, err := createCallbackServer(ch, state)
	if err != nil {
		return authCallback{}, fmt.Errorf("cannot create callback server: %w", err)
	}

	port, err := runServer(server)
	if err != nil {
		return authCallback{}, fmt.Errorf("cannot run authentication server: %w", err)
	}

	return authCallback{
		ch:     ch,
		server: server,
		Port:   port,
	}, nil
}

func (a authCallback) Result() string {
	result := <-a.ch
	_ = a.server.Shutdown(context.Background())
	return result
}

func createCallbackServer(ch chan string, state string) (*http.Server, error) {
	tmpl, err := template.New("login.html").Parse(LOGIN_HTML)
	if err != nil {
		return nil, fmt.Errorf("could not parse login callback template: %w", err)
	}
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			w.WriteHeader(400)
			return
		}

		ch <- q.Get("jwt")

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

	if token := settings.GetToken(); len(token) == 0 {
		fmt.Println("No user logged in.")
		return nil
	}

	if err := invalidateSessionsIfRequested(); err != nil {
		return err
	}

	settings.SetToken("")
	settings.SetUsername("")
	fmt.Println("Logged out.")

	return nil
}

func invalidateSessionsIfRequested() error {
	if !flags.All() {
		return nil
	}

	turso, err := authedTursoClient()
	if err != nil {
		return err
	}

	from, err := turso.Tokens.Invalidate()
	if err != nil {
		return err
	}

	formatted := time.Unix(from, 0).UTC().Format(time.DateTime)
	fmt.Printf("Invalidated all sessions started before %s UTC.\n", formatted)
	return nil
}

func whoAmI(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true
	client, err := authedTursoClient()
	if err != nil {
		return err
	}
	user, err := client.Users.GetUser()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", user.Username)
	return nil
}

func checkEnvAuth(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	token := os.Getenv(ENV_ACCESS_TOKEN)
	if token != "" {
		return fmt.Errorf("a token is set in the %q environment variable, please unset it before running %s", ENV_ACCESS_TOKEN, internal.Emph(cmd.CommandPath()))
	}
	return nil
}
