package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var groupFilter string
var schemaFilter string

func init() {
	dbCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&groupFilter, "group", "g", "", "Filter databases by group")
	listCmd.Flags().StringVarP(&schemaFilter, "schema", "s", "", "Filter databases by schema")
}

type DatabaseFetcher struct {
	client *turso.Client
}

func (df *DatabaseFetcher) FetchPage(pageSize int, cursor *string) (turso.ListResponse, error) {
	cursorStr := ""
	if cursor != nil {
		cursorStr = *cursor
	}

	options := turso.DatabaseListOptions{
		Group:  groupFilter,
		Schema: schemaFilter,
		Limit:  pageSize,
		Cursor: cursorStr,
	}

	return df.client.Databases.List(options)
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

		fetcher := &DatabaseFetcher{
			client: client,
		}
		return printDatabaseList(fetcher)
	},
}
