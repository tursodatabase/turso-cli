package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/xwb1989/sqlparser"
)

func shellArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return getDatabaseNames(createTursoClient()), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

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
		if len(args) == 1 {
			return runShell(name, dbUrl)
		} else {
			if len(args[1]) == 0 {
				return fmt.Errorf("No SQL command to execute")
			}
			return query(dbUrl, args[1])
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

func runShell(name, dbUrl string) error {
	if name != dbUrl {
		fmt.Printf("Connected to %s at %s\n\n", turso.Emph(name), dbUrl)
	} else {
		fmt.Printf("Connected to %s\n\n", dbUrl)
	}
	promptFmt := color.New(color.FgBlue, color.Bold).SprintFunc()
	l, err := readline.NewEx(&readline.Config{
		Prompt:            promptFmt("→  "),
		HistoryFile:       ".turso_history",
		InterruptPrompt:   "^C",
		EOFPrompt:         ".quit",
		HistorySearchFold: true,
	})
	if err != nil {
		return err
	}
	defer l.Close()
	l.CaptureExitSignal()

	fmt.Printf("Welcome to Turso SQL shell!\n\n")
	fmt.Printf("Type \".quit\" to exit the shell, \".tables\" to list all tables, and \".schema\" to show table schemas.\n\n")
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
				err = query(dbUrl, getTables())
				if err != nil {
					return err
				}
			}
			continue
		case ".schema":
			{
				{
					err = query(dbUrl, getSchema())
					if err != nil {
						return err
					}
				}
				continue
			}
		}
		err = query(dbUrl, line)
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

type SqlError struct {
	Message string
}

func (e *SqlError) Error() string {
	return e.Message
}

func query(url, stmt string) error {
	resp, err := doQuery(url, stmt)
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
		return &SqlError{fmt.Sprintf("Failed to execute SQL statement: %s\n%s", stmt, err_response.Message)}
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
									row[idx] = bytes[:count]
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
