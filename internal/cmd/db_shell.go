package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt/spinner"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/libsql/libsql-shell-go/pkg/shell"
	"github.com/libsql/libsql-shell-go/pkg/shell/enums"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(shellCmd)
	addFromFileFlag(shellCmd, "execute SQL commands from a file line by line")
	addBatchFlag(shellCmd, "sets the size of the batch of operations executed at once when using the --from-file flag")
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

		spinner := spinner.New("Connecting to database")
		if len(args) == 1 {
			spinner.Start()
			defer spinner.Stop()
		}

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return fmt.Errorf("could not create turso client: %w", err)
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("could not read settings: %w", err)
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
			dbUrl = getDatabaseHttpUrl(config, db)
			dbUrl = addTokenAsQueryParameter(dbUrl, token)
		}

		connectionInfo := getConnectionInfo(nameOrUrl, db, config)

		shellConfig := shell.ShellConfig{
			DbPath:         dbUrl,
			InF:            cmd.InOrStdin(),
			OutF:           cmd.OutOrStdout(),
			ErrF:           cmd.ErrOrStderr(),
			HistoryMode:    enums.PerDatabaseHistory,
			HistoryName:    "turso",
			WelcomeMessage: &connectionInfo,
			AfterDbConnectionCallback: func() {
				spinner.Stop()
			},
		}

		if fromFileFlag != "" {
			shellConfig.AfterDbConnectionCallback = func() {}
			return shellFromFile(fromFileFlag, batchFlag, shellConfig, spinner)
		}

		if len(args) == 2 {
			if len(args[1]) == 0 {
				return fmt.Errorf("no SQL command to execute")
			}
			return shell.RunShellLine(shellConfig, args[1])
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

	return nil, fmt.Errorf("hostname '%s' not found", parsed.Hostname())
}

func tokenFromDb(db *turso.Database, client *turso.Client) (string, error) {
	if db == nil {
		return "", nil
	}

	return client.Databases.Token(db.Name, "1d", false)
}

func getConnectionInfo(nameOrUrl string, db *turso.Database, config *settings.Settings) string {
	msg := fmt.Sprintf("Connected to %s", nameOrUrl)
	if db != nil {
		url := getDatabaseUrl(config, db, false)
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

func shellFromFile(path string, batchSize int, shellConfig shell.ShellConfig, s *spinner.Spinner) error {
	start := time.Now()

	f, err := getSQLFile(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	total := "..."
	lines, err := countFileLines(f)
	if err == nil {
		total = strconv.Itoa(lines)
	}

	err = executeSQLFile(f, shellConfig, path, batchSize, func(i int) {
		s.Text(fmt.Sprintf("Executing SQL commands from file (%d/%s)", i, total))
	})

	if err != nil {
		return err
	}

	s.Stop()
	fmt.Printf("Executed all %s SQL statements from %s in %d seconds.\n", total, internal.Emph(path), int(time.Since(start).Seconds()))
	return nil
}

func getSQLFile(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening SQL file %s: %w", path, err)
	}

	return f, nil
}

func executeSQLFile(f *os.File, config shell.ShellConfig, path string, batchSize int, update func(int)) error {
	idx := 0
	scanner := bufio.NewScanner(f)
	batch := make([]string, 0, batchSize)
	for scanner.Scan() {
		if update != nil {
			update(idx * batchSize)
		}
		batch = append(batch, scanner.Text())
		if len(batch) < batchSize {
			continue
		}

		err := executeBatch(batch, config, path, idx, batchSize)
		if err != nil {
			return err
		}

		idx += 1
		batch = batch[:0]
	}

	return executeBatch(batch, config, path, idx, batchSize)
}

func executeBatch(batch []string, config shell.ShellConfig, path string, idx int, batchSize int) error {
	if len(batch) == 0 {
		return nil
	}

	err := shell.RunShellLine(config, strings.Join(batch, "\n"))
	if err != nil {
		return fmt.Errorf("error executing SQL file %s batch %d of size %d: %w", path, idx, batchSize, err)
	}

	return nil
}

// Derived/copied from https://stackoverflow.com/a/52153000
func countFileLines(f *os.File) (int, error) {
	defer f.Seek(0, 0)
	count := 0
	buf := make([]byte, bufio.MaxScanTokenSize)
	for {
		bufferSize, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}

		var buffPosition int
		for {
			i := bytes.IndexByte(buf[buffPosition:], '\n')
			if i == -1 || bufferSize == buffPosition {
				break
			}
			buffPosition += i + 1
			count++
		}
		if err == io.EOF {
			break
		}
	}

	return count, nil
}
