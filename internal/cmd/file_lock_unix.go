//go:build !windows

package cmd

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func isFileLocked(filename string) (bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	fd := int(f.Fd())
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return true, nil
		}
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if err := syscall.Flock(fd, syscall.LOCK_UN); err != nil {
		return false, fmt.Errorf("failed to release lock: %w", err)
	}
	return false, nil
}
