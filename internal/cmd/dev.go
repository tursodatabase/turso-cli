package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"

	"database/sql"
	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
	"io"
	_ "modernc.org/sqlite"
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
	Results *resultSet  `json:"results,omitempty"`
	Message *hranaError `json:"error,omitempty"`
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

func errorJson(e error) *hranaError {
	return hranaErr(e)
}

var errSqliteUnknown = "SQLITE_UNKNOWN"

type hranaError struct {
	Message string  `json:"message"`
	Code    *string `json:"code,omitempty"`
}

func hranaErr(e error) *hranaError {
	err := new(hranaError)
	err.Message = e.Error()
	err.Code = &errSqliteUnknown
	return err
}

type hranaCol struct {
	Name string `json:"name"`
}

type hranaValue struct {
	Type   string      `json:"type"`
	Value  interface{} `json:"value,omitempty"`
	Base64 string      `json:"base64,omitempty"`
}

type hranaNamedArg struct {
	Name  string     `json:"name"`
	Value hranaValue `json:"value"`
}

type hranaStmt struct {
	Sql       *string          `json:"sql,omitempty"`
	SqlId     *int32           `json:"sql_id,omitempty"`
	Args      *[]hranaValue    `json:"args,omitempty"`
	NamedArgs *[]hranaNamedArg `json:"named_args,omitempty"`
	WantRows  *bool            `json:"want_rows"`
}

type hranaCond struct {
	Type  string       `json:"type"`
	Step  *int         `json:"step,omitempty"`
	Cond  *hranaCond   `json:"cond,omitempty"`
	Conds *[]hranaCond `json:"conds,omitempty"`
}

type hranaBatchStep struct {
	Stmt *hranaStmt `json:"stmt,omitempty"`
	Cond *hranaCond `json:"condition,omitempty"`
}

type hranaBatch struct {
	Steps []hranaBatchStep `json:"steps"`
}

type hranaExecuteReq struct {
	Stmt hranaStmt `json:"stmt"`
}

type hranaBatchReq struct {
	Batch hranaBatch `json:"batch"`
}

type hranaStmtResult struct {
	Cols     []hranaCol     `json:"cols"`
	Rows     [][]hranaValue `json:"rows"`
	Affected int32          `json:"affected_row_count"`
	LastIR   *string        `json:"last_insert_rowid"`
}

type hranaSuccessResponse struct {
	Result hranaStmtResult `json:"result"`
}

type hranaBatchResponse struct {
	Results []*hranaStmtResult `json:"step_results"`
	Errors  []*hranaError      `json:"step_errors"`
}

type hranaBatchSuccessResponse struct {
	Result hranaBatchResponse `json:"result"`
}

func executeHranaStatement(stmt hranaStmt, db *sql.DB) (*hranaStmtResult, *hranaError) {
	var result hranaStmtResult

	statement := *stmt.Sql
	params := make([]interface{}, 0)

	if stmt.Args != nil {
		for _, narg := range *stmt.Args {
			params = append(params, narg.Value)
		}
	}

	if stmt.NamedArgs != nil {
		for _, narg := range *stmt.NamedArgs {
			params = append(params, sql.Named(narg.Name, narg.Value.Value))
		}
	}

	if verboseFlag {
		fmt.Printf("Executing %s, %s\n", internal.Emph(statement), internal.Emph(params))
	}

	rows, err := db.Query(statement, params...)
	if err != nil {
		// FIXME: I can't get the actual SQLite error code from this
		return nil, hranaErr(err)
	}
	defer rows.Close()

	result.Cols = make([]hranaCol, 0)
	columns, _ := rows.Columns()
	if columns == nil {
		columns = []string{}
	}

	for _, col := range columns {
		// Consistent with the response from SQLd
		if strings.ToLower(col) == "null" {
			col = "NULL"
		}
		c := hranaCol{col}
		result.Cols = append(result.Cols, c)
	}

	result.Rows = make([][]hranaValue, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		err := rows.Scan(pointers...)
		if err != nil {
			return nil, hranaErr(err)
		}
		row := make([]hranaValue, 0)
		for _, val := range values {
			switch v := val.(type) {
			case nil:
				row = append(row, hranaValue{Type: "null"})
			case int64:
				row = append(row, hranaValue{Type: "integer", Value: fmt.Sprintf("%d", v)})
			case float64:
				row = append(row, hranaValue{Type: "float", Value: v})
			case []byte:
				row = append(row, hranaValue{Type: "blob", Base64: string(v)})
			case string:
				row = append(row, hranaValue{Type: "text", Value: v})
			default:
				row = append(row, hranaValue{Type: "text", Value: v.(string)})
			}
		}
		result.Rows = append(result.Rows, row)
	}
	return &result, nil
}

func executeStatement(c *gin.Context, db *sql.DB) {
	var reqBody hranaExecuteReq
	var result hranaSuccessResponse
	err := c.BindJSON(&reqBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, hranaErr(err))
		return
	}

	res, herr := executeHranaStatement(reqBody.Stmt, db)
	if herr != nil {
		c.JSON(http.StatusBadRequest, herr)
		return
	}
	result.Result = *res
	c.JSON(http.StatusOK, result)
}

// 0 -> skipped
// 1 -> error
// 2 -> ok
func evaluateCond(cond hranaCond, stmtRes []int) bool {
	ty := strings.ToLower(cond.Type)
	if ty == "ok" {
		// malformed
		if cond.Step == nil {
			return false
		}
		step := *cond.Step
		return stmtRes[step] == 2
	}
	if ty == "error" {
		// malformed
		if cond.Step == nil {
			return false
		}
		step := *cond.Step
		return stmtRes[step] == 1
	}
	if ty == "not" {
		// malformed
		if cond.Cond == nil {
			return false
		}
		return !evaluateCond(*cond.Cond, stmtRes)
	}
	if ty == "and" {
		// malformed
		if cond.Conds == nil {
			return false
		}

		curr := true
		for _, cond := range *cond.Conds {
			curr = curr && evaluateCond(cond, stmtRes)
		}
		return curr
	}
	if ty == "or" {
		// malformed
		if cond.Conds == nil {
			return false
		}

		curr := false
		for _, cond := range *cond.Conds {
			curr = curr || evaluateCond(cond, stmtRes)
		}
		return curr
	}

	return false
}

func executeHranaBatch(batch hranaBatch, db *sql.DB) hranaBatchResponse {
	stmtRes := make([]int, len(batch.Steps))
	var result hranaBatchResponse

	results := make([]*hranaStmtResult, len(batch.Steps))
	errors := make([]*hranaError, len(batch.Steps))

	for step_idx, step := range batch.Steps {
		var res *hranaStmtResult
		var err *hranaError
		skip := false
		if step.Cond != nil {
			skip = !evaluateCond(*step.Cond, stmtRes)
		}

		if !skip && step.Stmt != nil {
			res, err = executeHranaStatement(*step.Stmt, db)
		}
		results[step_idx] = res
		errors[step_idx] = err

		if res != nil {
			stmtRes[step_idx] = 2
		} else if err != nil {
			stmtRes[step_idx] = 1
		} else {
			stmtRes[step_idx] = 0
		}
	}

	result.Results = results
	result.Errors = errors
	return result
}

func executeBatch(c *gin.Context, db *sql.DB) {
	var reqBody hranaBatchReq
	err := c.BindJSON(&reqBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, hranaErr(err))
		return
	}

	var result hranaBatchSuccessResponse
	result.Result = executeHranaBatch(reqBody.Batch, db)

	c.JSON(http.StatusOK, result)
}

func executeRootStatement(statements []statement, db *sql.DB) []*statementSuccessResponse {
	stmtRes := []*statementSuccessResponse{}
	hasError := false

	for _, statement := range statements {
		if hasError {
			stmtRes = append(stmtRes, nil)
			continue
		}

		if verboseFlag {
			fmt.Printf("Executing %s\n", internal.Emph(statement))
		}
		rows, err := db.Query(statement.Query, statement.Params...)
		if err != nil {
			response := new(statementSuccessResponse)
			response.Message = errorJson(err)
			stmtRes = append(stmtRes, response)
			hasError = true
			continue
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			response := new(statementSuccessResponse)
			response.Message = errorJson(err)
			stmtRes = append(stmtRes, response)
			hasError = true
			continue
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
				response := new(statementSuccessResponse)
				response.Message = errorJson(err)
				stmtRes = append(stmtRes, response)
				hasError = true
				continue
			}
			rowsData = append(rowsData, values)
		}

		var resultSet = resultSet{columns, rowsData}
		var response = new(statementSuccessResponse)
		response.Results = &resultSet
		stmtRes = append(stmtRes, response)
	}
	return stmtRes

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
		fmt.Printf("%s, consider using libsql-server, available at %s\n", internal.Warn("If this server doesn't work for you"), internal.Emph("https://github.com/libsql/sqld"))
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

		db, err := sql.Open("sqlite", devFile)
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

			stmtRes := executeRootStatement(reqBody.Statements, db)

			c.JSON(http.StatusOK, stmtRes)
		})

		r.POST("/v1/execute", func(c *gin.Context) {
			c.Header("Content-Type", "application/json")
			executeStatement(c, db)
		})
		r.POST("/v1/batch", func(c *gin.Context) {
			c.Header("Content-Type", "application/json")
			executeBatch(c, db)
		})
		r.Run(port)
		return nil
	},
}
