// version_prod.go
//go:build prod
// +build prod

package cmd

import (
	_ "embed"
)

//go:generate sh -c "printf %s $(../../script/version.sh) > version.txt"
//go:embed version.txt
var version string
