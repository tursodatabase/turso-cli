//go:build !windows

package internal

import "github.com/fatih/color"

// Color function for emphasising text.
var Emph = color.New(color.FgBlue, color.Bold).SprintFunc()

var Warn = color.New(color.FgYellow, color.Bold).SprintFunc()
