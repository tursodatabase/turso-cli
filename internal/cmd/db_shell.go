package cmd

import (
	"bytes"
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
)

func shellArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		databases, err := getDatabases(createTursoClient())
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		result := make([]string, 0)
		for _, database := range databases {
			name := database.Name
			ty := database.Type
			if ty == "primary" {
				result = append(result, name)
			}
		}
		return result, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var shellCmd = &cobra.Command{
	Use:               "shell database_name [sql]",
	Short:             "Start a SQL shell.",
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
	fmt.Printf("Type \".quit\" to exit the shell.\n\n")
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
		}
		err = query(dbUrl, line)
		if err != nil {
			if _, ok := err.(*SqlError); !ok {
				return err
			} else {
				fmt.Fprintln(os.Stderr, err)
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
			return &SqlError{fmt.Sprintf("Failed to execute SQL statement: %s", stmt)}
		}
		return &SqlError{fmt.Sprintf("Failed to execute SQL statement: %s\n%s", stmt, err_response.Message)}
	}

	var results []QueryResult
	if err := json.Unmarshal(body, &results); err != nil {
		fmt.Printf("error: Failed to parse response from server: %s\n", err.Error())
		return err
	}
	errs := []string{}
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("error: %s\n", result.Error.Message)
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
	rawReq := QueryRequest{
		Statements: []string{stmt},
	}
	req, err := json.Marshal(rawReq)
	if err != nil {
		return nil, err
	}
	return http.Post(url, "application/json", bytes.NewReader(req))
}
