package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

//go:embed login.html
var LOGIN_HTML string

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
		settings, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("could not retrieve local config: %w", err)
		}
		token := settings.GetToken()
		if !isJwtTokenValid(token) {
			return fmt.Errorf("No user logged in. Run `turso auth login` to log in and get a token.")
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
}

func isJwtTokenValid(token string) bool {
	if len(token) == 0 {
		return false
	}
	resp, err := createTursoClient().Get("/v2/validate/token", nil)
	return err == nil && resp.StatusCode == http.StatusOK
}

func login(cmd *cobra.Command, args []string) error {
	settings, err := settings.ReadSettings()
	if err != nil {
		return fmt.Errorf("could not retrieve local config: %w", err)
	}
	if isJwtTokenValid(settings.GetToken()) {
		fmt.Println("✔  Success! Existing JWT still valid")
		return nil
	}
	fmt.Println("Waiting for authentication...")
	ch := make(chan string, 1)
	server, err := createCallbackServer(ch)
	if err != nil {
		return fmt.Errorf("Internal error. Cannot create callback: %w", err)
	}

	port, err := runServer(server)
	if err != nil {
		return fmt.Errorf("Internal error. Cannot run authentication server: %w", err)
	}

	err = beginAuth(port)
	if err != nil {
		return fmt.Errorf("Internal error. Cannot initiate auth flow: %w", err)
	}

	jwt := <-ch

	err = settings.SetToken(jwt)
	server.Shutdown(context.Background())

	if err != nil {
		return fmt.Errorf("error persisting token on local config: %w", err)
	}
	fmt.Println("✔  Success!")
	return nil
}

func beginAuth(port int) error {
	authUrl, err := url.Parse(getHost())
	if err != nil {
		return fmt.Errorf("error parsing auth URL: %w", err)
	}
	authUrl.RawQuery = url.Values{
		"port":     {strconv.Itoa(port)},
		"redirect": {"true"},
	}.Encode()

	err = browser.OpenURL(authUrl.String())
	if err != nil {
		return fmt.Errorf("error opening browser for auth flow: %w", err)
	}
	return nil
}

func createCallbackServer(jwtCh chan string) (*http.Server, error) {
	tmpl, err := template.New("login.html").Parse(LOGIN_HTML)
	if err != nil {
		return nil, fmt.Errorf("could not parse login callback template: %w", err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		jwtCh <- q.Get("jwt")

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
	settings, err := settings.ReadSettings()
	if err != nil {
		return fmt.Errorf("could not retrieve local config: %w", err)
	}

	token := settings.GetToken()
	if len(token) == 0 {
		fmt.Println("No user logged in.")
	} else {
		settings.SetToken("")
		fmt.Println("Logged out.")
	}

	return nil
}
