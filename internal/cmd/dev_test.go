package cmd

import (
	"database/sql"
	"encoding/json"
	qt "github.com/frankban/quicktest"
	"testing"
)

func TestJSONManipulations(t *testing.T) {
	t.Run("executerootstatement", func(t *testing.T) {
		c := qt.New(t)

		db, err := sql.Open("sqlite", devFile)
		c.Assert(err, qt.IsNil)
		defer db.Close()

		var data statementList
		jsonString := `{"statements": [{"q": "select 1"}, { "q": "select $first+$second", "params": {"first": "2", "wrong": "3"}}, {"q": "never executed"} ] }`
		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)

		stmtRes := executeRootStatement(data.Statements, db)
		c.Assert(len(stmtRes), qt.Equals, 3)
		c.Assert(stmtRes[0], qt.IsNotNil)
		c.Assert(stmtRes[1], qt.IsNotNil)
		c.Assert(stmtRes[2], qt.IsNil)

		first := *stmtRes[0]
		second := *stmtRes[1]

		c.Assert(first.Message, qt.IsNil)
		c.Assert(first.Results, qt.IsNotNil)

		jsonRet, err := json.Marshal(first)
		c.Assert(err, qt.IsNil)
		c.Assert(string(jsonRet), qt.Equals, `{"results":{"columns":["1"],"rows":[[1]]}}`)

		c.Assert(second.Message, qt.IsNotNil)
		c.Assert(second.Results, qt.IsNil)

		jsonRet, err = json.Marshal(second)
		c.Assert(err, qt.IsNil)
		c.Assert(string(jsonRet), qt.Matches, `{"error":{"message".*`)

	})

	t.Run("executesimpleselects", func(t *testing.T) {
		c := qt.New(t)

		db, err := sql.Open("sqlite", devFile)
		c.Assert(err, qt.IsNil)
		defer db.Close()

		var data hranaExecuteReq

		jsonString := `{"stmt":{"sql":"SELECT 1","args":[],"named_args":[],"want_rows":true}}`

		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)
		c.Assert(*data.Stmt.Sql, qt.IsNotNil)
		c.Assert(*data.Stmt.Args, qt.IsNotNil)
		c.Assert(*data.Stmt.NamedArgs, qt.IsNotNil)

		c.Assert(*data.Stmt.Sql, qt.Equals, "SELECT 1")
		c.Assert(len(*data.Stmt.Args), qt.Equals, 0)
		c.Assert(len(*data.Stmt.NamedArgs), qt.Equals, 0)

		js, herr := executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNil)

		jsonRet, err := json.Marshal(js)
		c.Assert(err, qt.IsNil)
		c.Assert(string(jsonRet), qt.Equals, `{"cols":[{"name":"1"}],"rows":[[{"type":"integer","value":"1"}]],"affected_row_count":0,"last_insert_rowid":null}`)

		// ==== float type ===
		jsonString = `{"stmt":{"sql":"SELECT 2.3","args":[],"named_args":[],"want_rows":true}}`
		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)

		js, herr = executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNil)
		jsonRet, err = json.Marshal(js)
		c.Assert(err, qt.IsNil)

		c.Assert(string(jsonRet), qt.Equals, `{"cols":[{"name":"2.3"}],"rows":[[{"type":"float","value":2.3}]],"affected_row_count":0,"last_insert_rowid":null}`)

		// ==== text type ===
		jsonString = `{"stmt":{"sql":"SELECT 'text'","args":[],"named_args":[],"want_rows":true}}`
		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)

		js, herr = executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNil)
		jsonRet, err = json.Marshal(js)
		c.Assert(err, qt.IsNil)

		c.Assert(string(jsonRet), qt.Equals, `{"cols":[{"name":"'text'"}],"rows":[[{"type":"text","value":"text"}]],"affected_row_count":0,"last_insert_rowid":null}`)

		// ==== NULL type ===
		jsonString = `{"stmt":{"sql":"SELECT null","args":[],"named_args":[],"want_rows":true}}`
		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)

		js, herr = executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNil)
		jsonRet, err = json.Marshal(js)
		c.Assert(err, qt.IsNil)
		c.Assert(string(jsonRet), qt.Equals, `{"cols":[{"name":"NULL"}],"rows":[[{"type":"null"}]],"affected_row_count":0,"last_insert_rowid":null}`)

		// === missing optional fields ==
		jsonString = `{"stmt":{"sql":"SELECT 1"}}`
		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)

		js, herr = executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNil)
		jsonRet, err = json.Marshal(js)
		c.Assert(err, qt.IsNil)
		c.Assert(string(jsonRet), qt.Equals, `{"cols":[{"name":"1"}],"rows":[[{"type":"integer","value":"1"}]],"affected_row_count":0,"last_insert_rowid":null}`)
	})

	t.Run("executeparamselect", func(t *testing.T) {
		c := qt.New(t)

		db, err := sql.Open("sqlite", devFile)
		c.Assert(err, qt.IsNil)
		defer db.Close()

		var data hranaExecuteReq

		jsonString := `{"stmt":{"sql":"SELECT ?+?","args":[{"type":"float","value":1},{"type":"float","value":2}],"named_args":[],"want_rows":true}}`

		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)
		c.Assert(*data.Stmt.Sql, qt.IsNotNil)
		c.Assert(*data.Stmt.Args, qt.IsNotNil)
		c.Assert(*data.Stmt.NamedArgs, qt.IsNotNil)

		c.Assert(len(*data.Stmt.Args), qt.Equals, 2)
		c.Assert(len(*data.Stmt.NamedArgs), qt.Equals, 0)

		js, herr := executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNil)

		jsonRet, err := json.Marshal(js)
		c.Assert(err, qt.IsNil)
		c.Assert(string(jsonRet), qt.Equals, `{"cols":[{"name":"?+?"}],"rows":[[{"type":"float","value":3}]],"affected_row_count":0,"last_insert_rowid":null}`)
	})

	t.Run("execute_mismatch_paramselect", func(t *testing.T) {
		c := qt.New(t)

		db, err := sql.Open("sqlite", devFile)
		c.Assert(err, qt.IsNil)
		defer db.Close()

		var data hranaExecuteReq

		jsonString := `{"stmt":{"sql":"SELECT :first+:second","args":[{"type":"float","value":1},{"type":"float","value":2}],"named_args":[],"want_rows":true}}`

		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)
		c.Assert(*data.Stmt.Sql, qt.IsNotNil)
		c.Assert(*data.Stmt.Args, qt.IsNotNil)
		c.Assert(*data.Stmt.NamedArgs, qt.IsNotNil)

		c.Assert(len(*data.Stmt.Args), qt.Equals, 2)
		c.Assert(len(*data.Stmt.NamedArgs), qt.Equals, 0)

		_, herr := executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNotNil)
	})

	t.Run("executenamedparamselect", func(t *testing.T) {
		c := qt.New(t)

		db, err := sql.Open("sqlite", devFile)
		c.Assert(err, qt.IsNil)
		defer db.Close()

		var data hranaExecuteReq

		jsonString := `{"stmt":{"sql":"SELECT ?+?","args":[],"named_args":[{"name":"first","value":{"type":"float","value":1}},{"name":"second","value":{"type":"float","value":2}}],"want_rows":true}}`

		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)
		c.Assert(*data.Stmt.Sql, qt.IsNotNil)
		c.Assert(*data.Stmt.Args, qt.IsNotNil)
		c.Assert(*data.Stmt.NamedArgs, qt.IsNotNil)

		c.Assert(len(*data.Stmt.Args), qt.Equals, 0)
		c.Assert(len(*data.Stmt.NamedArgs), qt.Equals, 2)

		js, herr := executeHranaStatement(data.Stmt, db)
		c.Assert(herr, qt.IsNil)

		jsonRet, err := json.Marshal(js)
		c.Assert(err, qt.IsNil)
		c.Assert(string(jsonRet), qt.Equals, `{"cols":[{"name":"?+?"}],"rows":[[{"type":"float","value":3}]],"affected_row_count":0,"last_insert_rowid":null}`)
	})

	t.Run("executesimplebatch", func(t *testing.T) {
		c := qt.New(t)

		db, err := sql.Open("sqlite", devFile)
		c.Assert(err, qt.IsNil)
		defer db.Close()

		var data hranaBatchReq

		jsonString := `{"batch": {"steps":[{"stmt":{"sql":"BEGIN","want_rows":false}},{"condition":{"type":"ok","step":0},"stmt":{"sql":"SELECT 1","args":[],"named_args":[],"want_rows":true}},{"condition":{"type":"ok","step":1},"stmt":{"sql":"SELECT 2","args":[],"named_args":[],"want_rows":true}},{"condition":{"type":"ok","step":2},"stmt":{"sql":"COMMIT","want_rows":false}},{"condition":{"type":"not","cond":{"type":"ok","step":3}},"stmt":{"sql":"ROLLBACK","want_rows":false}}]}}`

		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)
		c.Assert(len(data.Batch.Steps), qt.Equals, 5)

		batch := executeHranaBatch(data.Batch, db)
		results := batch.Results
		errors := batch.Errors

		c.Assert(len(errors), qt.Equals, 5)
		c.Assert(len(results), qt.Equals, 5)

		for _, e := range errors {
			c.Assert(e, qt.IsNil)
		}

		c.Assert(results[0], qt.IsNotNil)
		c.Assert(results[1], qt.IsNotNil)
		c.Assert(results[2], qt.IsNotNil)
		c.Assert(results[3], qt.IsNotNil)
		// rollback skipped
		c.Assert(results[4], qt.IsNil)

		_, err = json.Marshal(results)
		c.Assert(err, qt.IsNil)

		_, err = json.Marshal(errors)
		c.Assert(err, qt.IsNil)
	})

	t.Run("executefailedbatch", func(t *testing.T) {
		c := qt.New(t)

		db, err := sql.Open("sqlite", devFile)
		c.Assert(err, qt.IsNil)
		defer db.Close()

		var data hranaBatchReq

		jsonString := `{"batch": {"steps":[{"stmt":{"sql":"BEGIN","want_rows":false}},{"condition":{"type":"ok","step":0},"stmt":{"sql":"SELECT foo","args":[],"named_args":[],"want_rows":true}},{"condition":{"type":"ok","step":1},"stmt":{"sql":"SELECT 2","args":[],"named_args":[],"want_rows":true}},{"condition":{"type":"ok","step":2},"stmt":{"sql":"COMMIT","want_rows":false}},{"condition":{"type":"not","cond":{"type":"ok","step":3}},"stmt":{"sql":"ROLLBACK","want_rows":false}}]}}`

		err = json.Unmarshal([]byte(jsonString), &data)
		c.Assert(err, qt.IsNil)
		c.Assert(len(data.Batch.Steps), qt.Equals, 5)

		batch := executeHranaBatch(data.Batch, db)
		results := batch.Results
		errors := batch.Errors

		c.Assert(len(errors), qt.Equals, 5)
		c.Assert(len(results), qt.Equals, 5)

		c.Assert(errors[0], qt.IsNil)
		c.Assert(errors[1], qt.IsNotNil)
		c.Assert(errors[2], qt.IsNil)
		c.Assert(errors[3], qt.IsNil)
		c.Assert(errors[4], qt.IsNil)

		c.Assert(results[0], qt.IsNotNil)
		c.Assert(results[1], qt.IsNil)
		c.Assert(results[2], qt.IsNil)
		c.Assert(results[3], qt.IsNil)
		c.Assert(results[4], qt.IsNotNil)

		_, err = json.Marshal(results)
		c.Assert(err, qt.IsNil)

		_, err = json.Marshal(errors)
		c.Assert(err, qt.IsNil)
	})

}
