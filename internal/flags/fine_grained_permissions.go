package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type FineGrainedPermissions struct {
	TableNames        []string `json:"t"`
	AllowedOperations []string `json:"a"`
}

var fineGrainedPermissions []string

func AddFineGrainedPermissions(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&fineGrainedPermissions, "permissions", "p", nil, "fine-grained permissions in format <table-name|all>:<action1>,...\n(e.g: -p all:data_read -p comments:data_insert)")
}

func FineGrainedPermissionsFlags() ([]FineGrainedPermissions, error) {
	permissions := make([]FineGrainedPermissions, 0)
	for _, permission := range fineGrainedPermissions {
		tokens := strings.SplitN(permission, ":", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid permission format: '%v'", permission)
		}
		var tableNames []string
		if tokens[0] != "all" {
			tableNames = append(tableNames, tokens[0])
		}
		permissions = append(permissions, FineGrainedPermissions{
			TableNames:        tableNames,
			AllowedOperations: strings.Split(tokens[1], ","),
		})
	}
	return permissions, nil
}
