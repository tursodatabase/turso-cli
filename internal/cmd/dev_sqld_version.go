package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func getSqldVersion(cmd *cobra.Command) {
	cmd.Flags().BoolP("version", "v", false, "sqld version")
}

func sqldVersion() {
	sqld := exec.Command("sqld", "--version")
	sqld.Env = append(os.Environ(), "RUST_LOG=error")
	version, err := sqld.Output()
	if err != nil {
		fmt.Println("Error running sqld --version:", err)
		return
	}
	fmt.Printf("%s\n", version)
}
