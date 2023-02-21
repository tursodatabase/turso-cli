package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "turso",
	Version: version,
	Long:    "Turso CLI",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("token", "t", "", "Access token used for authorization")
	if err := viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token")); err != nil {
		fmt.Fprintf(os.Stderr, "error binding token flag: %s", err)
	}
	rootCmd.PersistentFlags().StringP("config-path", "c", "", "Path to the directory with config file")
	if err := viper.BindPFlag("config-path", rootCmd.PersistentFlags().Lookup("config-path")); err != nil {
		fmt.Fprintf(os.Stderr, "error binding token flag: %s", err)
	}
}
