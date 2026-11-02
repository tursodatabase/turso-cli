package flags

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

type FineGrainedPermissions struct {
	TableNames        []string `json:"t"`
	AllowedOperations []string `json:"a"`
}

// ValidActions are the fine-grained actions accepted by the server.
var ValidActions = []string{
	"data_read",
	"data_add",
	"data_update",
	"data_delete",
	"schema_add",
	"schema_update",
	"schema_delete",
}

var fineGrainedPermissions []string

func AddFineGrainedPermissions(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&fineGrainedPermissions, "permissions", "p", nil, "fine-grained permissions in format <table-name|all>:<action1>,...\n(e.g: -p all:data_read -p comments:data_add)\nvalid actions: "+strings.Join(ValidActions, ", "))
}

func FineGrainedPermissionsFlags() ([]FineGrainedPermissions, error) {
	return parseFineGrainedPermissions(fineGrainedPermissions)
}

func parseFineGrainedPermissions(values []string) ([]FineGrainedPermissions, error) {
	permissions := make([]FineGrainedPermissions, 0)
	for _, permission := range values {
		tokens := strings.SplitN(permission, ":", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid permission format: '%v'", permission)
		}
		var tableNames []string
		if tokens[0] != "all" {
			tableNames = append(tableNames, tokens[0])
		}
		actions := strings.Split(tokens[1], ",")
		for _, action := range actions {
			if !slices.Contains(ValidActions, action) {
				return nil, fmt.Errorf("invalid permission action '%v' in '%v': valid actions are %v", action, permission, strings.Join(ValidActions, ", "))
			}
		}
		permissions = append(permissions, FineGrainedPermissions{
			TableNames:        tableNames,
			AllowedOperations: actions,
		})
	}
	return permissions, nil
}
