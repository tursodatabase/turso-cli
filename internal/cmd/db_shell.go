package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/xwb1989/sqlparser"
)

func init() {
	dbCmd.AddCommand(shellCmd)
}

var promptFmt = color.New(color.FgBlue, color.Bold).SprintFunc()

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

		spinner := createSpinner("Connecting to database")
		if len(args) == 1 {
			spinner.Start()
			defer spinner.Stop()
		}

		client, err := createTursoClient()
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
		}

		if len(args) == 2 {
			if len(args[1]) == 0 {
				return fmt.Errorf("no SQL command to execute")
			}
			return query(dbUrl, token, args[1])
		}

		reader, err := connectShell(dbUrl, token)
		if err != nil {
			return fmt.Errorf("failed to run shell: %w", err)
		}

		defer reader.Close()

		spinner.Stop()
		printConnectionInfo(nameOrUrl, db, config)

		return shellLoop(dbUrl, token, reader)
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

	return client.Databases.Token(db.Name, "default", false)
}

func printConnectionInfo(nameOrUrl string, db *turso.Database, config *settings.Settings) {
	msg := fmt.Sprintf("Connected to %s", nameOrUrl)
	if db != nil {
		url := getDatabaseUrl(config, db, false)
		msg = fmt.Sprintf("Connected to %s at %s", internal.Emph(db.Name), url)
	}

	fmt.Printf("%s\n\n", msg)
	fmt.Printf("Welcome to Turso SQL shell!\n\n")
	fmt.Printf("Type \".quit\" to exit the shell, \".tables\" to list all tables, and \".schema\" to show table schemas.\n\n")
}

func connectShell(dbUrl, token string) (*readline.Instance, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	historyFile := filepath.Join(user.HomeDir, "/.turso_history")
	l, err := readline.NewEx(&readline.Config{
		Prompt:            promptFmt("→  "),
		HistoryFile:       historyFile,
		InterruptPrompt:   "^C",
		EOFPrompt:         ".quit",
		HistorySearchFold: true,
	})
	if err != nil {
		return nil, err
	}

	l.CaptureExitSignal()

	if err = testDbConnection(dbUrl, token); err != nil {
		return nil, err
	}

	return l, nil
}

func shellLoop(dbUrl, token string, l *readline.Instance) error {
	var cmds []string
replLoop:
	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		switch line {
		case ".quit":
			break replLoop
		case ".tables":
			{
				err = query(dbUrl, token, getTables())
				if err != nil {
					return err
				}
			}
			continue
		case ".schema":
			{
				{
					err = query(dbUrl, token, getSchema())
					if err != nil {
						return err
					}
				}
				continue
			}
		}

		cmds = append(cmds, line)
		if !strings.HasSuffix(line, ";") {
			l.SetPrompt("... ")
			continue
		}
		cmd := strings.Join(cmds, "\n")
		cmds = cmds[:0]
		l.SetPrompt(promptFmt("→  "))

		err = query(dbUrl, token, cmd)

		if err != nil {
			if _, ok := err.(*SqlError); !ok {
				return err
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			}
		}
	}
	return nil
}

func testDbConnection(dbUrl, token string) error {
	resp, err := doQuery(dbUrl, token, "SELECT 1")
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to connect to database: %s", resp.Status)
	}

	return nil
}

type SqlError struct {
	Message string
}

func (e *SqlError) Error() string {
	return e.Message
}

func query(url, token, stmt string) error {
	switch stmt {
	case ".tables":
		stmt = getTables()
	case ".schema":
		stmt = getSchema()
	}
	resp, err := doQuery(url, token, stmt)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var err_response ErrorResponse
		if err := json.Unmarshal(body, &err_response); err != nil {
			return &SqlError{fmt.Sprintf("Failed to execute SQL statement: %s\n%s", stmt, err)}
		}
		var msg string
		if err_response.Message == "interactive transaction not allowed in HTTP queries" {
			msg = "Transactions are only supported in the shell using semicolons to separate each statement.\nFor example: \"BEGIN; [your SQL statements]; END\""
		} else {
			msg = fmt.Sprintf("Failed to execute SQL statement: %s\n%s", stmt, err_response.Message)
		}
		return &SqlError{msg}
	}

	var results []QueryResult
	if err := json.Unmarshal(body, &results); err != nil {
		return err
	}
	errs := []string{}
	for _, result := range results {
		if result.Error != nil {
			errs = append(errs, result.Error.Message)
		}
		if result.Results != nil {
			columns := make([]interface{}, 0)
			for _, column := range result.Results.Columns {
				columns = append(columns, strings.ToUpper(column))
			}
			tbl := table.New(columns...)
			for _, row := range result.Results.Rows {
				for idx, v := range row {
					if f64, ok := v.(float64); ok {
						row[idx] = strconv.FormatFloat(f64, 'f', -1, 64)
					} else if v == nil {
						row[idx] = "NULL"
					} else if m, ok := v.(map[string]interface{}); ok {
						if value, ok := m["base64"]; ok {
							if base64Value, ok := value.(string); ok {
								bytes := make([]byte, base64.StdEncoding.DecodedLen(len(base64Value)))
								count, err := base64.StdEncoding.Decode(bytes, []byte(base64Value))
								if err != nil {
									row[idx] = base64Value
								} else {
									row[idx] = hex.EncodeToString(bytes[:count])
								}
							}
						}
					}
				}
				tbl.AddRow(row...)
			}
			tbl.Print()
		}
	}
	if len(errs) > 0 {
		return &SqlError{(strings.Join(errs, "; "))}
	}

	return nil
}

func doQuery(url, token, stmt string) (*http.Response, error) {
	stmts, err := sqlparser.SplitStatementToPieces(stmt)
	if err != nil {
		return nil, err
	}
	rawReq := QueryRequest{
		Statements: stmts,
	}
	body, err := json.Marshal(rawReq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return http.DefaultClient.Do(req)
}
