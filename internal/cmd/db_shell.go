package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/libsql/libsql-shell-go/pkg/shell"
	"github.com/libsql/libsql-shell-go/pkg/shell/enums"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(shellCmd)
	addInstanceFlag(shellCmd, "Connect to the database at the specified instance.")
	addLocationFlag(shellCmd, "Connect to the database at the specified location.")
}

func getURL(db *turso.Database, client *turso.Client) (string, error) {
	if instanceFlag != "" || locationFlag != "" {
		instances, err := client.Instances.List(db.Name)
		if err != nil {
			return "", err
		}
		for _, instance := range instances {
			if instance.Region == locationFlag {
				return getInstanceHttpUrl(db, &instance), nil
			}
			if instance.Name == instanceFlag {
				return getInstanceHttpUrl(db, &instance), nil
			}
		}
		if locationFlag != "" {
			return "", fmt.Errorf("location %s for db %s not found", locationFlag, db.Name)
		}
		if instanceFlag != "" {
			return "", fmt.Errorf("instance %s for db %s not found", instanceFlag, db.Name)
		}
		return "", fmt.Errorf("impossible")
	} else {
		return getDatabaseHttpUrl(db), nil
	}
}

var shellCmd = &cobra.Command{
	Use:               "shell {database_name | replica_url} [sql]",
	Short:             "Start a SQL shell.",
	Long:              "Start a SQL shell.\nWhen database_name is provided, the shell will connect the closest replica of the specified database.\nWhen the --instance flag is provided with a specific instance name, the shell will connect to that instance directly.",
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
		// Makes sure localhost URL or self-hosted will work even if not authenticated
		// to turso. The token code will check for auth
		if !isURL(nameOrUrl) {
			client, err := createTursoClientFromAccessToken(true)
			if err != nil {
				return fmt.Errorf("could not create turso client: %w", err)
			}

			db, err = databaseFromName(nameOrUrl, client)
			if err != nil {
				return err
			}

			token, err := tokenFromDb(db, client)
			if err != nil {
				return err
			}

			dbUrl, err = getURL(db, client)
			if err != nil {
				return err
			}
			urlString = dbUrl
			dbUrl = addTokenAsQueryParameter(dbUrl, token)
		}

		connectionInfo := getConnectionInfo(urlString, db)

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

func tokenFromDb(db *turso.Database, client *turso.Client) (string, error) {
	if db == nil {
		return "", nil
	}

	settings, _ := settings.ReadSettings()
	if token := settings.DbTokenCache(db.ID); token != "" {
		return token, nil
	}

	token, err := client.Databases.Token(db.Name, "1d", false)
	if err != nil {
		return "", err
	}

	exp := time.Now().Add(time.Hour * 23).Unix()
	settings.SetDbTokenCache(db.ID, token, exp)

	return token, nil
}

func getConnectionInfo(nameOrUrl string, db *turso.Database) string {
	msg := fmt.Sprintf("Connected to %s", internal.Emph(nameOrUrl))
	if db != nil && nameOrUrl != "" {
		msg = fmt.Sprintf("Connected to %s at %s", internal.Emph(db.Name), internal.Emph(nameOrUrl))
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
