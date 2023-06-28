package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/libsql/libsql-shell-go/pkg/shell"
	"github.com/libsql/libsql-shell-go/pkg/shell/enums"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(shellCmd)
}

var shellCmd = &cobra.Command{
	Use:               "shell {database_name | replica_url} [sql]",
	Short:             "Start a SQL shell.",
	Long:              "Start a SQL shell.\nWhen database_name is provided, the shell will connect to the closest replica of the specified database.\nWhen a url of a particular replica is provided, the shell will connect to that replica directly.",
	Example:           "turso db shell name-of-my-amazing-db\nturso db shell libsql://<replica-url>\nturso db shell libsql://e784400f26d083-my-amazing-db-replica-url.turso.io\nturso db shell name-of-my-amazing-db \"select * from users;\"",
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

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return fmt.Errorf("could not create turso client: %w", err)
		}

		db, err := databaseFromNameOrURL(nameOrUrl, client)
		if err != nil {
			return err
		}

		token, err := tokenFromDb(db, client)
		if err != nil {
			return err
		}

		dbUrl := nameOrUrl
		if db != nil {
			dbUrl = getDatabaseHttpUrl(db)
			dbUrl = addTokenAsQueryParameter(dbUrl, token)
		}

		connectionInfo := getConnectionInfo(nameOrUrl, db)

		shellConfig := shell.ShellConfig{
			DbUri:          dbUrl,
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

		if len(args) == 2 {
			if len(args[1]) == 0 {
				return fmt.Errorf("no SQL command to execute")
			}
			return shell.RunShellLine(shellConfig, args[1])
		}

		if pipeOrRedirect() {
			// TODO: read chunks when iteractive transactions are available
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("error reading from stdin: %w", err)
			}
			return shell.RunShellLine(shellConfig, string(b))
		}

		return shell.RunShell(shellConfig)
	},
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

func databaseFromNameOrURL(str string, client *turso.Client) (*turso.Database, error) {
	if isURL(str) {
		return databaseFromURL(str, client)
	}

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

func databaseFromURL(dbURL string, client *turso.Client) (*turso.Database, error) {
	parsed, err := url.ParseRequestURI(dbURL)
	if err != nil {
		return nil, err
	}

	dbs, err := client.Databases.List()
	if err != nil {
		return nil, err
	}

	for _, db := range dbs {
		if strings.HasSuffix(parsed.Hostname(), db.Hostname) {
			return &db, nil
		}
	}

	return nil, nil
}

func tokenFromDb(db *turso.Database, client *turso.Client) (string, error) {
	if db == nil {
		return "", nil
	}

	return client.Databases.Token(db.Name, "1d", false)
}

func getConnectionInfo(nameOrUrl string, db *turso.Database) string {
	msg := fmt.Sprintf("Connected to %s", nameOrUrl)
	if db != nil {
		url := getDatabaseUrl(db)
		msg = fmt.Sprintf("Connected to %s at %s", internal.Emph(db.Name), url)
	}

	msg += "\n\n"
	msg += "Welcome to Turso SQL shell!\n\n"
	msg += "Type \".quit\" to exit the shell and \".help\" to list all available commands.\n\n"
	return msg
}

func addTokenAsQueryParameter(dbUrl string, token string) string {
	return fmt.Sprintf("%s?jwt=%s", dbUrl, token)
}

func pipeOrRedirect() bool {
	stat, err := os.Stdin.Stat()
	return err == nil && (stat.Mode()&os.ModeCharDevice) == 0
}
