package cmd

import (
	"fmt"
	"strings"
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
	Use:               "unarchive <db-name>",
	Short:             "Unarchive a database",
	Args:              cobra.ExactArgs(1),
	Aliases:           []string{"wakeup"},
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
	s := prompt.Spinner(fmt.Sprintf("Unarchiving database %s... ", internal.Emph(name)))
	defer s.Stop()

	if err := client.Databases.Wakeup(name); err != nil {
		// If the database is part of a group the server tells us so
		// but doesn't tell the user what to run instead. Look up the
		// database to grab its group and suggest the right command.
		// See #918.
		if strings.Contains(err.Error(), "part of a group") {
			if db, lookupErr := client.Databases.Get(name); lookupErr == nil && db.Group != "" {
				cmd := internal.Emph(fmt.Sprintf("turso group unarchive %s", db.Group))
				return fmt.Errorf("%w. Run %s instead", err, cmd)
			}
		}
		return err
	}
	s.Stop()
	elapsed := time.Since(start)
	invalidateDatabasesCache()
	fmt.Printf("Unarchived database %s in %d seconds.\n", internal.Emph(name), int(elapsed.Seconds()))
	return nil
}
