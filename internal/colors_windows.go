//go:build windows

package internal

import "fmt"

// Color function for emphasising text.
var Emph = func(a ...interface{}) string {
	return fmt.Sprint(a...)
}

var Warn = func(a ...interface{}) string {
	return fmt.Sprint(a...)
}
