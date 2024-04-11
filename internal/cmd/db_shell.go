package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/libsql/libsql-shell-go/pkg/shell"
	"github.com/libsql/libsql-shell-go/pkg/shell/enums"
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/settings"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var proxy string

func init() {
	dbCmd.AddCommand(shellCmd)
	addInstanceFlag(shellCmd, "Connect to the database at the specified instance.")
	addLocationFlag(shellCmd, "Connect to the database at the specified location.")
	shellCmd.Flags().StringVar(&proxy, "proxy", "", "Proxy to use for the connection.")
	shellCmd.RegisterFlagCompletionFunc("proxy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	})
	flags.AddAttachClaims(shellCmd)
}

func getURL(db *turso.Database, client *turso.Client, http bool) (string, error) {
	scheme := "wss"
	if http {
		scheme = "https"
	}

	if instanceFlag == "" && locationFlag == "" {
		return getUrl(db, nil, scheme), nil
	}

	instances, err := client.Instances.List(db.Name)
	if err != nil {
		return "", err
	}
	for _, instance := range instances {
		if instance.Region == locationFlag || instance.Name == instanceFlag {
			return getUrl(db, &instance, scheme), nil
		}
	}

	if locationFlag != "" {
		return "", fmt.Errorf("location %s for db %s not found", locationFlag, db.Name)
	}
	if instanceFlag != "" {
		return "", fmt.Errorf("instance %s for db %s not found", instanceFlag, db.Name)
	}
	return "", fmt.Errorf("impossible")
}

func getDbURLForDump(u string) string {
	if strings.HasPrefix(u, "wss://") || strings.HasPrefix(u, "ws://") {
		return strings.Replace(u, "ws", "http", 1)
	}
	return u
}

var shellCmd = &cobra.Command{
	Use:               "shell <database-name | replica-url> [sql]",
	Short:             "Start a SQL shell.",
	Long:              "Start a SQL shell.\nWhen database-name is provided, the shell will connect the closest replica of the specified database.\nWhen the --instance flag is provided with a specific instance name, the shell will connect to that instance directly.",
	Example:           "  turso db shell http://127.0.0.1:8080\n  turso db shell name-of-my-amazing-db\n  turso db shell name-of-my-amazing-db --location yyz\n  turso db shell name-of-my-amazing-db --instance a-specific-instance\n  turso db shell name-of-my-amazing-db \"select * from users;\"",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		nameOrUrl := args[0]
		if nameOrUrl == "" {
			return fmt.Errorf("please specify a database name")
		}
		cmd.SilenceUsage = true

		spinner := prompt.StoppedSpinner("Connecting to database")
		if len(args) == 1 {
			spinner.Start()
			defer spinner.Stop()
		}

		dbUrl := nameOrUrl
		urlString := nameOrUrl
		var db *turso.Database = nil
		var authToken string
		nonInteractive := pipeOrRedirect()
		// Makes sure localhost URL or self-hosted will work even if not authenticated
		// to turso. The token code will check for auth
		if !isURL(nameOrUrl) {
			client, err := authedTursoClient()
			if err != nil {
				return fmt.Errorf("could not create turso client: %w", err)
			}

			db, err = databaseFromName(nameOrUrl, client)
			if err != nil {
				return err
			}

			var claim *turso.PermissionsClaim
			if len(flags.AttachClaims()) > 0 {
				err := validateDBNames(client, flags.AttachClaims())
				if err != nil {
					return err
				}
				claim = &turso.PermissionsClaim{
					ReadAttach: turso.Entities{DBNames: flags.AttachClaims()},
				}
			}

			authToken, err = tokenFromDb(db, client, claim)
			if err != nil {
				return err
			}
			dbUrl, err = getURL(db, client, nonInteractive)
			if err != nil {
				return err
			}
			urlString = dbUrl
		} else {
			u, err := url.Parse(dbUrl)
			if err != nil {
				return err
			}
			query := u.Query()
			authTokenSnake := query.Get("auth_token")
			authTokenCamel := query.Get("authToken")
			jwt := query.Get("jwt")
			u.RawQuery = ""

			countNonEmpty := func(slice ...string) int {
				count := 0
				for _, s := range slice {
					if s != "" {
						count++
					}
				}
				return count
			}

			if countNonEmpty(authTokenSnake, authTokenCamel, jwt) > 1 {
				return fmt.Errorf("please use at most one of the following query parameters: 'auth_token', 'authToken', 'jwt'")
			}

			if authTokenSnake != "" {
				authToken = authTokenSnake
			} else if authTokenCamel != "" {
				authToken = authTokenCamel
			} else if jwt != "" {
				authToken = jwt
			} else if strings.HasSuffix(u.Hostname(), ".turso.io") {
				client, err := authedTursoClient()
				if err != nil {
					return fmt.Errorf("could not create turso client: %w", err)
				}
				dbs, err := getDatabases(client)
				if err != nil {
					return err
				}
				for _, d := range dbs {
					if d.Hostname == u.Hostname() {
						db = &d
						break
					}
				}
				if db == nil {
					return fmt.Errorf("could not find a database with the hostname %s", u.Hostname())
				}
				authToken, err = tokenFromDb(db, client, nil)
				if err != nil {
					return err
				}
			}
			dbUrl = u.String()
		}

		connectionInfo := getConnectionInfo(urlString, db)

		shellConfig := shell.ShellConfig{
			DbUri:          dbUrl,
			Proxy:          proxy,
			AuthToken:      authToken,
			InF:            cmd.InOrStdin(),
			OutF:           cmd.OutOrStdout(),
			ErrF:           cmd.ErrOrStderr(),
			HistoryMode:    enums.PerDatabaseHistory,
			HistoryName:    "turso",
			WelcomeMessage: &connectionInfo,
			AfterDbConnectionCallback: func() {
				spinner.Stop()
			},
			DisableAutoCompletion: true,
		}

		dbID := ""
		if db != nil {
			dbID = db.ID
		}
		if len(args) == 2 {
			if len(args[1]) == 0 {
				return fmt.Errorf("no SQL command to execute")
			}
			if args[1] == ".dump" {
				return dump(getDbURLForDump(dbUrl), authToken)
			}
			return runShellLine(dbID, shellConfig, args[1])
		}

		if nonInteractive {
			// TODO: read chunks when interactive transactions are available
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("error reading from stdin: %w", err)
			}
			return runShellLine(dbID, shellConfig, string(b))
		}
		return runShell(dbID, shellConfig)
	},
}

func runShell(dbID string, config shell.ShellConfig) error {
	err := shell.RunShell(config)
	if isAuthError(err) && dbID != "" {
		clearDBTokenCache(dbID)
	}
	return err
}

func runShellLine(dbID string, config shell.ShellConfig, line string) error {
	err := shell.RunShellLine(config, line)
	if isAuthError(err) {
		clearDBTokenCache(dbID)
	}
	return err
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// TODO: use a more reliable way to detect auth errors
	return strings.Contains(msg, "401") || strings.Contains(msg, "403")
}

func clearDBTokenCache(dbID string) {
	setDbTokenCache(dbID, "", 0)
	settings.PersistChanges()
}

type QueryRequest struct {
	Statements []string `json:"statements"`
}

type QueryResult struct {
	Results *ResultSet `json:"results"`
	Error   *Error     `json:"error"`
}

type ResultSet struct {
	Columns []string `json:"columns"`
	Rows    []Row    `json:"rows"`
}

type Row []interface{}

type Error struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Message string `json:"error"`
}

func databaseFromName(str string, client *turso.Client) (*turso.Database, error) {
	name := str
	db, err := getDatabase(client, name)
	if err != nil {
		return nil, err
	}

	return &db, nil
}

func isURL(s string) bool {
	_, err := url.ParseRequestURI(s)
	return err == nil
}

func tokenFromDb(db *turso.Database, client *turso.Client, claim *turso.PermissionsClaim) (string, error) {
	if db == nil {
		return "", nil
	}
	// skip cache and always use token from server when claims are attached
	if claim != nil {
		return client.Databases.Token(db.Name, "2d", false, claim)
	}

	if token := dbTokenCache(db.ID); token != "" {
		return token, nil
	}

	token, err := client.Databases.Token(db.Name, "2d", false, nil)
	if err != nil {
		return "", err
	}

	exp := time.Now().Add(time.Hour * 6).Unix()
	setDbTokenCache(db.ID, token, exp)

	return token, nil
}

func getConnectionInfo(nameOrUrl string, db *turso.Database) string {
	msg := fmt.Sprintf("Connected to %s", internal.Emph(nameOrUrl))
	if db != nil && nameOrUrl != "" {
		msg = fmt.Sprintf("Connected to %s at %s", internal.Emph(db.Name), internal.Emph(getDatabaseUrl(db)))
	}

	msg += "\n\n"
	msg += "Welcome to Turso SQL shell!\n\n"
	msg += "Type \".quit\" to exit the shell and \".help\" to list all available commands.\n\n"
	return msg
}

func pipeOrRedirect() bool {
	stat, err := os.Stdin.Stat()
	return err == nil && (stat.Mode()&os.ModeCharDevice) == 0
}

func dump(dbURL, authToken string) error {
	req, err := http.NewRequest("GET", dbURL+"/dump", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+authToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		fmt.Print(line)
		if err == io.EOF {
			return nil
		}
	}
}
