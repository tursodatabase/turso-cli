package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"database/sql"
	"github.com/chiselstrike/iku-turso-cli/internal"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"io"
)

func init() {
	rootCmd.AddCommand(devCmd)
	addDevPortFlag(devCmd)
	addDevFileFlag(devCmd)
	addVerboseFlag(devCmd)
}

type resultSet struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

type statementSuccessResponse struct {
	Results resultSet `json:"results"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type statement struct {
	Query  string `json:"q"`
	Params []interface{}
}

func (s statement) String() string {
	var buf bytes.Buffer
	buf.WriteString(s.Query)
	buf.WriteString("; ")
	jsonData, err := json.Marshal(s.Params)
	if err != nil {
		panic("can't convert json data from params")
	}
	buf.WriteString(string(jsonData))
	return buf.String()
}

type statementList struct {
	Statements []statement `json:"statements"`
}

func (s *statement) UnmarshalJSON(data []byte) error {
	var query interface{}
	err := json.Unmarshal(data, &query)
	if err != nil {
		return err
	}

	switch q := query.(type) {
	case string:
		s.Query = q
	case map[string]interface{}:
		qValue, ok := q["q"].(string)
		if !ok {
			return fmt.Errorf("invalid JSON: 'q' field is missing or not a string")
		}
		s.Query = qValue

		switch p := q["params"].(type) {
		case []interface{}:
			s.Params = p
		case map[string]interface{}:
			var named []interface{}
			for k, v := range p {
				named = append(named, sql.Named(k, v))
			}
			s.Params = named
		}
	}

	return nil
}

func errorJson(e error) []errorResponse {
	return []errorResponse{
		{
			Error: struct {
				Message string `json:"message"`
			}{
				Message: e.Error(),
			},
		},
	}
}

var devCmd = &cobra.Command{
	Use:               "dev",
	Short:             "starts a local development server for Turso",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		port := fmt.Sprintf(":%d", devPort)
		var filename string
		var dbLocation string
		if devFile == "" {
			filename = "file:somefile.db"
			devFile = ":memory:"
			dbLocation = internal.Emph("in-memory")
		} else {
			filename = fmt.Sprintf("file:%s", devFile)
			dbLocation = fmt.Sprintf("at %s", internal.Emph(devFile))
		}

		fmt.Printf("%s In particular, some Turso features are currently not present in local-server mode:\n", internal.Warn("This is experimental:"))
		fmt.Printf("%s extensions.\n", internal.Warn("→  "))
		fmt.Printf("%s interactive transactions/libsql-based URLs.\n", internal.Warn("→  "))
		fmt.Println()
		fmt.Printf("For environments where a filesystem is available:\n")
		fmt.Printf("%s use %s as a URL. This server is not needed. (Ctrl-C)\n", internal.Emph("→  "), internal.Emph(filename))
		fmt.Printf("For all others environments:\n")
		fmt.Printf("%s Database is %s. Connect to %s%s.\n", internal.Emph("→  "), dbLocation, internal.Emph("http://localhost"), internal.Emph(port))

		gin.SetMode(gin.ReleaseMode)
		r := gin.New()
		if !verboseFlag {
			r.Use(gin.LoggerWithWriter(io.Discard))
		}

		db, err := sql.Open("sqlite3", devFile)
		if err != nil {
			fmt.Println("Failed to connect to SQLite database:", err)
			return err
		}
		defer db.Close()

		r.POST("/", func(c *gin.Context) {
			var reqBody statementList

			err := c.BindJSON(&reqBody)
			if err != nil {
				c.JSON(http.StatusBadRequest, errorJson(err))
				return
			}

			statements := reqBody.Statements

			var stmtRes = []statementSuccessResponse{}

			for _, statement := range statements {
				if verboseFlag {
					fmt.Printf("Executing %s\n", internal.Emph(statement))
				}
				rows, err := db.Query(statement.Query, statement.Params...)
				if err != nil {
					c.JSON(http.StatusOK, errorJson(err))
					return
				}

				columns, err := rows.Columns()
				if err != nil {
					// FIXME: 200 or 400 for this one?
					c.JSON(http.StatusOK, errorJson(err))
					return
				}
				if columns == nil {
					columns = []string{}
				}

				var rowsData = [][]interface{}{}
				for rows.Next() {
					values := make([]interface{}, len(columns))
					pointers := make([]interface{}, len(columns))
					for i := range values {
						pointers[i] = &values[i]
					}
					err := rows.Scan(pointers...)
					if err != nil {
						// FIXME: 200 or 400 for this one?
						c.JSON(http.StatusOK, errorJson(err))
						return
					}
					rowsData = append(rowsData, values)
				}

				var resultSet = resultSet{columns, rowsData}
				var response = statementSuccessResponse{resultSet}
				stmtRes = append(stmtRes, response)
				defer rows.Close()
			}

			c.JSON(http.StatusOK, stmtRes)
		})

		r.Run(port)
		return nil
	},
}
