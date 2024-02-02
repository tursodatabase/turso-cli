package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	dbCmd.AddCommand(wakeUpDbCmd)
}

var wakeUpDbCmd = &cobra.Command{
	Use:               "wakeup <db-name>",
	Short:             "Wake up a database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		name := args[0]
		return wakeupDatabase(client, name)
	},
}

func wakeupDatabase(client *turso.Client, name string) error {
	start := time.Now()
	s := prompt.Spinner(fmt.Sprintf("Waking up database %s... ", internal.Emph(name)))
	defer s.Stop()

	if err := client.Databases.Wakeup(name); err != nil {
		return err
	}
	s.Stop()
	elapsed := time.Since(start)
	invalidateDatabasesCache()
	fmt.Printf("Waked up database %s in %d seconds.\n", internal.Emph(name), int(elapsed.Seconds()))
	return nil
}
