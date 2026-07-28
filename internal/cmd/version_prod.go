// version_prod.go
//go:build prod
// +build prod

package cmd

import (
	_ "embed"
)

//go:generate go run ../../script/write_version.go
//go:embed version.txt
var version string
