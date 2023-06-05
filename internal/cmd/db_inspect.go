package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rodaine/table"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/xwb1989/sqlparser"
	"golang.org/x/sync/errgroup"
)

func init() {
	dbCmd.AddCommand(dbInspectCmd)
	addVerboseFlag(dbInspectCmd)
}

type InspectInstanceInfo struct {
	Location      string
	StorageInfos  []StorageInfo
	RowsReadCount uint64
}

type InspectInfo struct {
	instanceInfos [](*InspectInstanceInfo)
}

type StorageInfo struct {
	Type        string
	Name        string
	SizeTables  uint64
	SizeIndexes uint64
}

func (curr *InspectInstanceInfo) totalTablesSize() uint64 {
	var total uint64
	for _, storageInfo := range curr.StorageInfos {
		total += storageInfo.SizeTables
	}
	return total
}

func (curr *InspectInstanceInfo) totalIndexesSize() uint64 {
	var total uint64
	for _, storageInfo := range curr.StorageInfos {
		total += storageInfo.SizeIndexes
	}
	return total
}

func (curr *InspectInfo) Accumulate(n *InspectInstanceInfo) {
	curr.instanceInfos = append(curr.instanceInfos, n)
}

func (curr *InspectInfo) totalTablesSize() uint64 {
	var total uint64
	for _, instanceInfo := range curr.instanceInfos {
		total += instanceInfo.totalTablesSize()
	}
	return total
}

func (curr *InspectInfo) totalIndexesSize() uint64 {
	var total uint64
	for _, instanceInfo := range curr.instanceInfos {
		total += instanceInfo.totalIndexesSize()
	}
	return total
}

func (curr *InspectInfo) PrintTotalStorage() string {
	return humanize.IBytes(curr.totalTablesSize() + curr.totalIndexesSize())
}

func (curr *InspectInfo) TotalRowsReadCount() uint64 {
	var total uint64
	for _, instanceInfo := range curr.instanceInfos {
		total += instanceInfo.RowsReadCount
	}
	return total
}

func (curr *InspectInfo) show(detailed bool) {
	tables := humanize.IBytes(curr.totalTablesSize())
	indexes := humanize.IBytes(curr.totalIndexesSize())
	rowsRead := fmt.Sprintf("%d", curr.TotalRowsReadCount())
	fmt.Printf("Total space used for tables: %s\n", tables)
	fmt.Printf("Total space used for indexes: %s\n", indexes)
	fmt.Printf("Number of rows read: %s\n", rowsRead)

	if detailed {
		sort.Slice(curr.instanceInfos, func(i, j int) bool {
			return curr.instanceInfos[i].Location < curr.instanceInfos[j].Location
		})
		for _, instanceInfo := range curr.instanceInfos {
			fmt.Println()
			fmt.Printf("For location: %s\n", instanceInfo.Location)
			tbl := table.New("TYPE", "NAME", "SIZE")
			for _, storageInfo := range instanceInfo.StorageInfos {
				tbl.AddRow(storageInfo.Type, storageInfo.Name, humanize.IBytes(storageInfo.SizeTables+storageInfo.SizeIndexes))
			}
			tbl.Print()
		}
	}
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

		inspectInfo, err := inspectInstances(instances, config, db, token)
		if err != nil {
			return err
		}

		inspectInfo.show(verboseFlag)
		return nil
	},
}

func calculateInstancesUsedSize(instances []turso.Instance, config *settings.Settings, db turso.Database, token string) string {
	inspectInfo, err := inspectInstances(instances, config, db, token)
	if err != nil {
		return fmt.Sprintf("fetching size failed: %s", err)
	}
	return inspectInfo.PrintTotalStorage()
}

func inspectInstances(instances []turso.Instance, config *settings.Settings, db turso.Database, token string) (*InspectInfo, error) {
	inspectInfo := &InspectInfo{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	results := make(chan *InspectInstanceInfo, len(instances))
	for _, instance := range instances {
		loopInstance := instance
		g.Go(func() error {
			url := getInstanceHttpUrl(config, &db, &loopInstance)
			ret, err := inspectInstance(ctx, url, token)
			if err != nil {
				return err
			}
			ret.Location = loopInstance.Region
			results <- ret
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return &InspectInfo{}, fmt.Errorf("timeout while inspecting database. It's possible that this database is too old and does not support inspecting or one of the instances is not reachable")
		}
		return &InspectInfo{}, err
	}
	for range instances {
		ret := <-results
		inspectInfo.Accumulate(ret)
	}

	return inspectInfo, nil
}

type GetInstancesInfoReturnType struct {
	size     string
	versions []chan string
	urls     []string
}

func getInstancesInfo(client *turso.Client, instances []turso.Instance, config *settings.Settings, db turso.Database, token string) GetInstancesInfoReturnType {
	versions := [](chan string){}
	urls := []string{}

	for idx, instance := range instances {
		urls = append(urls, getInstanceUrl(config, &db, &instance))
		versions = append(versions, make(chan string, 1))
		go func(idx int, client *turso.Client, config *settings.Settings, db *turso.Database, instance *turso.Instance) {
			versions[idx] <- fetchInstanceVersion(client, config, db, instance)
		}(idx, client, config, &db, &instance)
	}

	instancesInfo := GetInstancesInfoReturnType{
		size:     calculateInstancesUsedSize(instances, config, db, token),
		versions: versions,
		urls:     urls,
	}
	return instancesInfo
}

func inspectInstance(ctx context.Context, url, token string) (*InspectInstanceInfo, error) {
	inspectComputeResult := make(chan uint64, 1)
	go func() {
		rowsRead, err := inspectCompute(ctx, url, token)
		if err != nil {
			rowsRead = 0
		}
		inspectComputeResult <- rowsRead
	}()
	storageInfos, err := inspectStorage(ctx, url, token)
	if err != nil {
		return nil, err
	}
	rowsRead := <-inspectComputeResult
	return &InspectInstanceInfo{
		StorageInfos:  storageInfos,
		RowsReadCount: rowsRead,
	}, nil
}

func inspectCompute(ctx context.Context, url, token string) (uint64, error) {
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

func inspectStorage(ctx context.Context, url, token string) ([]StorageInfo, error) {
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

	stmt := `select name, SUM(pgsize) as size from dbstat
	where name != 'sqlite_schema'
        and name != '_litestream_seq'
        and name != '_litestream_lock'
        and name != 'libsql_wasm_func_table'
	group by name
	order by size desc, name asc`
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

	var errs []string
	var res []StorageInfo
	for _, result := range results {
		if result.Error != nil {
			errs = append(errs, result.Error.Message)
		}
		if result.Results != nil {
			for _, row := range result.Results.Rows {
				type_ := "?"
				name := row[0].(string)
				if t, ok := typeMap[name]; ok {
					type_ = t
				}
				size := uint64(row[1].(float64))
				storageInfo := StorageInfo{}
				storageInfo.Type = type_
				storageInfo.Name = name
				if type_ == "index" {
					storageInfo.SizeIndexes = size
				} else {
					storageInfo.SizeTables = size
				}
				res = append(res, storageInfo)
			}
		}
	}
	if len(errs) > 0 {
		return nil, &SqlError{(strings.Join(errs, "; "))}
	}
	return res, nil
}

type SqlError struct {
	Message string
}

func (e *SqlError) Error() string {
	return e.Message
}

func doQueryContext(ctx context.Context, url, token, stmt string) (*http.Response, error) {
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
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return http.DefaultClient.Do(req)
}
