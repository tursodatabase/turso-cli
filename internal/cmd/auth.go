package cmd

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

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
	server := createCallbackServer(ch)

	port, err := runServer(server)
	if err != nil {
		return err
	}

	err = beginAuth(port)
	if err != nil {
		return err
	}

	jwt := <-ch
	settings, _ := settings.ReadSettings()
	settings.SetToken(jwt)

	server.Shutdown(context.Background())
	return nil
}

func beginAuth(port int) error {
	authUrl, err := url.Parse(getHost())
	if err != nil {
		return err
	}
	authUrl.RawQuery = url.Values{
		"port":     {strconv.Itoa(port)},
		"redirect": {"true"},
	}.Encode()

	return browser.OpenURL(authUrl.String())
}

func createCallbackServer(jwtCh chan string) *http.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		jwtCh <- q.Get("jwt")

		w.WriteHeader(200)
		// TODO: send nice response to user
	})

	return &http.Server{
		Handler: handler,
	}
}

func runServer(server *http.Server) (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}

	go func() {
		server.Serve(listener)
	}()

	return listener.Addr().(*net.TCPAddr).Port, nil
}
