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

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
}

func login(cmd *cobra.Command, args []string) error {
	ch := make(chan string, 1)
	server, err := createCallbackServer(ch)
	if err != nil {
		return err
	}

	port, err := runServer(server)
	if err != nil {
		return err
	}

	err = beginAuth(port)
	if err != nil {
		return err
	}

	jwt := <-ch
	settings, err := settings.ReadSettings()
	if err != nil {
		return fmt.Errorf("could not retrieve local config: %w", err)
	}

	err = settings.SetToken(jwt)
	server.Shutdown(context.Background())

	if err != nil {
		return fmt.Errorf("error persisting token on local config: %w", err)
	}
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

	browser.OpenURL(authUrl.String())
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
