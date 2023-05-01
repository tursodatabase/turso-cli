package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/dustin/go-humanize"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"strings"
	"time"
)

func init() {
	dbCmd.AddCommand(dbInspectCmd)
	addVerboseFlag(dbInspectCmd)
}

type InspectInfo struct {
	StorageInfo   StorageInfo
	RowsReadCount uint64
}

type StorageInfo struct {
	SizeTables  uint64
	SizeIndexes uint64
}

func (curr *InspectInfo) Accumulate(n *InspectInfo) {
	curr.StorageInfo.SizeTables += n.StorageInfo.SizeTables
	curr.StorageInfo.SizeIndexes += n.StorageInfo.SizeIndexes
	curr.RowsReadCount += n.RowsReadCount
}

func (curr *InspectInfo) PrintTotal() string {
	return humanize.IBytes(curr.StorageInfo.SizeTables + curr.StorageInfo.SizeIndexes)
}

func (curr *InspectInfo) show() {
	tables := humanize.IBytes(curr.StorageInfo.SizeTables)
	indexes := humanize.IBytes(curr.StorageInfo.SizeIndexes)
	rowsRead := fmt.Sprintf("%d", curr.RowsReadCount)
	fmt.Printf("Total space used for tables: %s\n", tables)
	fmt.Printf("Total space used for indexes: %s\n", indexes)
	fmt.Printf("Number of rows read: %s\n", rowsRead)
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

		client, err := createTursoClientFromAccessToken(true)
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

		token, err := client.Databases.Token(db.Name, "1d", true)
		if err != nil {
			return err
		}

		inspectRet := InspectInfo{}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		g, ctx := errgroup.WithContext(ctx)
		results := make(chan *InspectInfo, len(instances))
		for _, instance := range instances {
			loopInstance := instance
			g.Go(func() error {
				url := getInstanceHttpUrl(config, &db, &loopInstance)
				ret, err := inspect(ctx, url, token, loopInstance.Region, verboseFlag)
				if err != nil {
					return err
				}
				results <- ret
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("timeout while inspecting database. It's possible that this database is too old and does not support inspecting or one of the instances is not reachable")
			}
			return err
		}
		for range instances {
			ret := <-results
			inspectRet.Accumulate(ret)
		}
		inspectRet.show()
		return nil
	},
}

func inspect(ctx context.Context, url, token string, location string, detailed bool) (*InspectInfo, error) {
	inspectComputeResult := make(chan uint64, 1)
	go func() {
		rowsRead, err := inspectCompute(ctx, url, token, detailed, location)
		if err != nil {
			rowsRead = 0
		}
		inspectComputeResult <- rowsRead
	}()
	storageInfo, err := inspectStorage(ctx, url, token, detailed, location)
	if err != nil {
		return nil, err
	}
	rowsRead := <-inspectComputeResult
	return &InspectInfo{
		StorageInfo:   *storageInfo,
		RowsReadCount: rowsRead,
	}, nil
}

func inspectCompute(ctx context.Context, url, token string, detailed bool, location string) (uint64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url+"/v1/stats", nil)
	if err != nil {
		return 0, err
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var results struct {
		RowsReadCount uint64 `json:"rows_read_count"`
	}
	if err := json.Unmarshal(body, &results); err != nil {
		return 0, err
	}
	return results.RowsReadCount, nil
}

func getTypeMap(ctx context.Context, url, token string) (map[string]string, error) {
	typeStmt := `select name, type from sqlite_schema where
	name != 'sqlite_schema'
        and name != '_litestream_seq'
        and name != '_litestream_lock'
        and name != 'libsql_wasm_func_table'`
	respType, err := doQueryContext(ctx, url, token, typeStmt)
	if err != nil {
		return nil, err
	}
	defer respType.Body.Close()
	bodyType, err := io.ReadAll(respType.Body)
	if err != nil {
		return nil, err
	}

	if respType.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(bodyType))
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

	return typeMap, nil
}

func inspectStorage(ctx context.Context, url, token string, detailed bool, location string) (*StorageInfo, error) {
	typeMapResult := make(chan map[string]string)
	typeMapError := make(chan error)
	go func() {
		typeMap, err := getTypeMap(ctx, url, token)
		if err != nil {
			typeMapError <- err
		} else {
			typeMapResult <- typeMap
		}
	}()

	storageInfo := StorageInfo{}
	stmt := `select name, pgsize from dbstat where
	name != 'sqlite_schema'
        and name != '_litestream_seq'
        and name != '_litestream_lock'
        and name != 'libsql_wasm_func_table'
	order by pgsize desc, name asc`
	resp, err := doQueryContext(ctx, url, token, stmt)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(body))
	}

	var results []QueryResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	var typeMap map[string]string
	select {
	case err := <-typeMapError:
		return nil, err
	case typeMap = <-typeMapResult:
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
					storageInfo.SizeIndexes += size
				} else {
					storageInfo.SizeTables += size
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
	return &storageInfo, nil
}
