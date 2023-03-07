package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
	"github.com/xwb1989/sqlparser"

	"github.com/chiselstrike/libsql-shell/src/lib"
)

func shellArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return getDatabaseNames(createTursoClient()), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var TURSO_WELCOME_MESSAGE = "Welcome to Turso SQL shell!\n\nType \".quit\" to exit the shell, \".tables\" to list all tables, and \".schema\" to show table schemas.\n\n"

var shellCmd = &cobra.Command{
	Use:               "shell {database_name | replica_url} [sql]",
	Short:             "Start a SQL shell.",
	Long:              "Start a SQL shell.\nWhen database_name is provided, the shell will connect to the closest replica of the specified database.\nWhen a url of a particular replica is provided, the shell will connect to that replica directly.",
	Example:           "turso db shell name-of-my-amazing-db\nturso db shell https://<login>:<password>@<replica-url>\nturso db shell https://alice:94l6z30w1Kq8p7ob@e784400f26d083-my-amazing-db-replica-url.turso.io",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: shellArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("please specify a database name")
		}
		cmd.SilenceUsage = true

		dbUrl, err := getDatabaseURL(name)
		if err != nil {
			return err
		}

		db, err := lib.NewDb(dbUrl)
		if err != nil {
			return err
		}
		defer db.Close()

		if len(args) == 1 {
			return runShell(cmd, db, name, dbUrl)
		} else {
			if len(args[1]) == 0 {
				return fmt.Errorf("no SQL command to execute")
			}
			return query(db, cmd.OutOrStderr(), args[1])
		}
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

func getTables() string {
	return `select name from sqlite_schema where
	type = 'table'
	and name not like 'sqlite_%'
	and name != '_litestream_seq'
	and name != '_litestream_lock'
	and name != 'libsql_wasm_func_table'
	order by name`
}

func getSchema() string {
	return `select sql from sqlite_schema where
	name not like 'sqlite_%'
	and name != '_litestream_seq'
	and name != '_litestream_lock'
	and name != 'libsql_wasm_func_table'
	order by name`
}

func getDatabaseURL(name string) (string, error) {
	config, err := settings.ReadSettings()
	if err != nil {
		return "", err
	}
	// If name is a valid URL, let's just is it directly to connect instead
	// of looking up an URL from settings.
	_, err = url.ParseRequestURI(name)
	var dbUrl string
	if err != nil {
		dbSettings, err := config.FindDatabaseByName(name)
		if err != nil {
			return "", err
		}
		dbUrl = dbSettings.GetURL()
	} else {
		dbUrl = name
	}
	resp, err := doQuery(dbUrl, "SELECT 1")
	if err != nil {
		return "", fmt.Errorf("failed to connect: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to connect: %s", resp.Status)
	}
	return dbUrl, nil
}

func runShell(cmd *cobra.Command, db *lib.Db, name, dbUrl string) error {
	if name != dbUrl {
		fmt.Printf("Connected to %s at %s\n\n", turso.Emph(name), dbUrl)
	} else {
		fmt.Printf("Connected to %s\n\n", dbUrl)
	}

	shellConfig := lib.ShellConfig{
		InF:            cmd.InOrStdin(),
		OutF:           cmd.OutOrStdout(),
		ErrF:           cmd.ErrOrStderr(),
		HistoryFile:    fmt.Sprintf("%s/.turso_shell_history", os.Getenv("HOME")),
		WelcomeMessage: &TURSO_WELCOME_MESSAGE,
	}

	return db.RunShell(shellConfig)
}

type SqlError struct {
	Message string
}

func (e *SqlError) Error() string {
	return e.Message
}

func query(db *lib.Db, outF io.Writer, statements string) error {
	results, err := db.ExecuteStatements(statements)
	if err != nil {
		return err
	}
	err = lib.PrintStatementsResult(results, outF, false)
	if err != nil {
		return err
	}
	return nil
}

func doQuery(url, stmt string) (*http.Response, error) {
	stmts, err := sqlparser.SplitStatementToPieces(stmt)
	if err != nil {
		return nil, err
	}
	rawReq := QueryRequest{
		Statements: stmts,
	}
	req, err := json.Marshal(rawReq)
	if err != nil {
		return nil, err
	}
	return http.Post(url, "application/json", bytes.NewReader(req))
}
