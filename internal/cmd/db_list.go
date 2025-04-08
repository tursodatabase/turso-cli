package cmd

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/turso"
	"golang.org/x/exp/slices"
)

var groupFilter string
var schemaFilter string

func init() {
	dbCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&groupFilter, "group", "g", "", "Filter databases by group")
	listCmd.Flags().StringVarP(&schemaFilter, "schema", "s", "", "Filter databases by schema")
	listCmd.Flags().IntVar(&paginationLimit, "limit", 0, "Limit the number of results returned")
	listCmd.Flags().StringVar(&paginationCursor, "cursor", "", "Cursor for pagination")
}

var listCmd = &cobra.Command{
	Use:               "list",
	Short:             "List databases.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		options := turso.DatabaseListOptions{
			Group:  groupFilter,
			Schema: schemaFilter,
			Limit:  paginationLimit,
			Cursor: paginationCursor,
		}
		databases, err := client.Databases.List(options)
		if err != nil {
			return err
		}
		setDatabasesCache(databases)

		printDBListTable(databases)
		return nil
	},
}

func printDBListTable(databases []turso.Database) {
	headers, data := dbListTable(databases)
	if !shouldPrintLocations(databases) {
		headers, data = removeColumn(headers, data, "Locations")
	}
	if !shouldPrintGroups(databases) {
		headers, data = removeColumn(headers, data, "Group")
	}

	if !shouldPrintSleeping(databases) {
		headers, data = removeColumn(headers, data, "Archived")
	}

	printTable(headers, data)
}

func shouldPrintLocations(databases []turso.Database) bool {
	for _, database := range databases {
		if database.Group == "" {
			return true
		}
	}
	return false
}

func shouldPrintGroups(databases []turso.Database) bool {
	mp := map[string]bool{}
	for _, database := range databases {
		mp[database.Group] = true
	}
	return len(mp) > 1
}

func shouldPrintSleeping(databases []turso.Database) bool {
	for _, database := range databases {
		if database.Sleeping {
			return true
		}
	}
	return false
}

func dbListTable(databases []turso.Database) (headers []string, data [][]string) {
	for _, database := range databases {
		row := []string{database.Name, getDatabaseLocations(database), formatGroup(database.Group), getDatabaseUrl(&database), formatBool(database.Sleeping)}
		data = append(data, row)
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i][0] < data[j][0]
	})

	return []string{"Name", "Locations", "Group", "URL", "Archived"}, data
}

func removeColumn(headers []string, data [][]string, column string) ([]string, [][]string) {
	i := slices.Index(headers, column)
	if i == -1 {
		return headers, data
	}

	filtered := make([][]string, 0, len(data))
	for _, row := range data {
		filtered = append(filtered, removeIndex(row, i))
	}

	return removeIndex(headers, i), filtered
}

func removeIndex(arr []string, i int) []string {
	return append(arr[:i], arr[i+1:]...)
}

func formatGroup(group string) string {
	if group == "" {
		return "-"
	}
	return group
}
