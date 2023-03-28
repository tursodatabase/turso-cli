package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/dustin/go-humanize"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(dbInspectCmd)
	dbInspectCmd.Flags().BoolVar(&verboseFlag, "verbose", false, "Show detailed information")
}

type InspectInfo struct {
	SizeTables  uint64
	SizeIndexes uint64
}

func (curr *InspectInfo) Accumulate(n *InspectInfo) {
	curr.SizeTables += n.SizeTables
	curr.SizeIndexes += n.SizeIndexes
}

func (curr *InspectInfo) show() {
	tables := humanize.Bytes(curr.SizeTables)
	indexes := humanize.Bytes(curr.SizeIndexes)
	fmt.Printf("Total space used for tables: %s\nTotal space used for indexes: %s\n", tables, indexes)
}

var dbInspectCmd = &cobra.Command{
	Use:               "inspect {database_name}",
	Short:             "Inspect database.",
	Example:           "turso db inspect name-of-my-amazing-db",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("please specify a database name")
		}
		cmd.SilenceUsage = true

		client, err := createTursoClient()
		if err != nil {
			return err
		}
		db, err := getDatabase(client, name)
		if err != nil {
			return err
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		instances, err := client.Instances.List(db.Name)
		if err != nil {
			return err
		}

		inspectRet := InspectInfo{}
		for _, instance := range instances {
			url := getInstanceHttpUrl(config, &db, &instance)
			ret, err := inspect(url, instance.Region, verboseFlag)
			if err != nil {
				return err
			}
			inspectRet.Accumulate(ret)
		}
		inspectRet.show()
		return nil
	},
}

func inspect(url string, location string, detailed bool) (*InspectInfo, error) {
	inspectRet := InspectInfo{}

	stmt := `select name, pgsize from dbstat where
	name not like 'sqlite_%'
        and name != '_litestream_seq'
        and name != '_litestream_lock'
        and name != 'libsql_wasm_func_table'
	order by pgsize desc, name asc`
	resp, err := doQuery(url, stmt)
	if err != nil {
		return nil, err
	}

	typeStmt := `select name, type from sqlite_schema where
	name not like 'sqlite_%'
        and name != '_litestream_seq'
        and name != '_litestream_lock'
        and name != 'libsql_wasm_func_table'`
	respType, err := doQuery(url, typeStmt)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	defer respType.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(body))
	}

	bodyType, err := io.ReadAll(respType.Body)
	if err != nil {
		return nil, err
	}

	if respType.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(body))
	}

	var results []QueryResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	var typeResults []QueryResult
	if err := json.Unmarshal(bodyType, &typeResults); err != nil {
		return nil, err
	}

	typeMap := make(map[string]string)
	for _, result := range typeResults {
		if result.Results != nil {
			for _, row := range result.Results.Rows {
				typeMap[row[0].(string)] = row[1].(string)
			}
		}
	}

	errs := []string{}
	for _, result := range results {
		if result.Error != nil {
			errs = append(errs, result.Error.Message)
		}
		if result.Results != nil {
			columns := make([]interface{}, 0)
			columns = append(columns, "TYPE")
			columns = append(columns, "NAME")
			columns = append(columns, "SIZE (KB)")
			tbl := table.New(columns...)

			for _, row := range result.Results.Rows {
				type_ := "?"
				name := row[0].(string)
				if t, ok := typeMap[name]; ok {
					type_ = t
				}
				size := uint64(row[1].(float64))
				if type_ == "index" {
					inspectRet.SizeIndexes += size
				} else {
					inspectRet.SizeTables += size
				}
				tbl.AddRow(type_, name, size/1024.0)
			}
			if detailed {
				fmt.Printf("For location: %s\n", location)
				tbl.Print()
				fmt.Println()
			}
		}
	}
	if len(errs) > 0 {
		return nil, &SqlError{(strings.Join(errs, "; "))}
	}

	return &inspectRet, nil
}
