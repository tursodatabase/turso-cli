//go:build windows

package cmd

import (
	"errors"
	"fmt"
	"math"
	"os"

	"golang.org/x/sys/windows"
)

func isFileLocked(filename string) (bool, error) {
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(filename),
		windows.GENERIC_READ,
		0, // This is a special mode, it means
		// no sharing, hence, if the file is being used
		// by any other process, it will fail.
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		if errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
			return true, nil
		}
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	windows.CloseHandle(handle)
	return false, nil
}
