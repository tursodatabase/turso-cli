package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/dustin/go-humanize"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(dbInspectCmd)
	addVerboseFlag(dbInspectCmd)
}

type InspectInstanceInfo struct {
	Location      string
	Name          string
	Type          string
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
	return humanize.Bytes(curr.totalTablesSize() + curr.totalIndexesSize())
}

func (curr *InspectInfo) TotalRowsReadCount() uint64 {
	var total uint64
	for _, instanceInfo := range curr.instanceInfos {
		total += instanceInfo.RowsReadCount
	}
	return total
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

		instances, dbUsage, err := instancesAndUsage(client, db.Name)
		if err != nil {
			return err
		}

		fmt.Printf("Total space used: %s\n", humanize.Bytes(dbUsage.Usage.Total.StorageBytesUsed))
		fmt.Printf("Number of rows read: %d\n", dbUsage.Usage.Total.RowsRead)
		fmt.Printf("Number of rows written: %d\n", dbUsage.Usage.Total.RowsWritten)

		if !verboseFlag {
			return nil
		}

		instancesUsage := getInstanceUsageMap(dbUsage.Usage.Instances)
		tbl := table.New("LOCATION", "TYPE", "INSTANCE NAME", "ROWS READ", "ROWS WRITTEN", "TOTAL STORAGE")
		for _, instance := range instances {
			fmt.Printf("Instance: %s\n", instance.Uuid)
			usg, ok := instancesUsage[instance.Uuid]
			if !ok {
				tbl.AddRow(instance.Region, instance.Type, instance.Name, "-", "-", "-")
				continue
			}
			tbl.AddRow(instance.Region, instance.Type, instance.Name, usg.RowsRead, usg.RowsWritten, humanize.Bytes(usg.StorageBytesUsed))
		}

		fmt.Println()
		tbl.Print()
		fmt.Println()

		return nil
	},
}

func getInstanceUsageMap(usages []turso.Usage) map[string]turso.Usage {
	m := make(map[string]turso.Usage)
	for _, usg := range usages {
		m[usg.UUID] = usg
	}
	return m
}
