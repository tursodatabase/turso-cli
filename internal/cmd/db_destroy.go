package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the destruction of all locations of the database.")
	addLocationFlag(destroyCmd, "Pick a database location to destroy.")
	destroyCmd.Flags().StringVar(&instanceFlag, "instance", "", "Pick a specific database instance to destroy.")
	destroyCmd.RegisterFlagCompletionFunc("instance", completeInstanceName)
}

func destroyArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return getDatabaseNames(createTursoClient()), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var destroyCmd = &cobra.Command{
	Use:               "destroy database_name",
	Short:             "Destroy a database.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: destroyArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client := createTursoClient()
		name := args[0]
		if instanceFlag != "" {
			return destroyDatabaseInstance(client, name, instanceFlag)
		}

		if locationFlag != "" {
			return destroyDatabaseRegion(client, name, locationFlag)
		}

		if yesFlag {
			return destroyDatabase(client, name)
		}

		fmt.Printf("Database %s, all its replicas, and data will be destroyed.\n", turso.Emph(name))

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Database destruction avoided.")
			return nil
		}

		return destroyDatabase(client, name)
	},
}
