//go:build windows

package cmd

import "errors"

func Update() error {
	return errors.New(
		"automatic updates are not available on Windows; " +
			"download the latest Windows ZIP and replace turso.exe: " +
			"https://github.com/tursodatabase/turso-cli/releases/latest",
	)
}
