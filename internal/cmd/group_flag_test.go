package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTestDatabase creates a valid WAL-mode test database with approximately the given size
func createTestDatabase(t *testing.T, sizeBytes int) string {
	t.Helper()

	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database with correct settings for Turso
	cmd := exec.Command("sqlite3", dbPath,
		"PRAGMA page_size=4096;",
		"PRAGMA journal_mode=WAL;",
		"CREATE TABLE data (id INTEGER PRIMARY KEY, blob BLOB);")
	require.NoError(t, cmd.Run(), "failed to create test database")

	// Fill with data to reach target size
	if sizeBytes > 0 {
		rowSize := 1000 // ~1KB per row
		numRows := sizeBytes / rowSize
		if numRows < 1 {
			numRows = 1
		}
		for i := 0; i < numRows; i++ {
			cmd = exec.Command("sqlite3", dbPath,
				fmt.Sprintf("INSERT INTO data (blob) VALUES (randomblob(%d));", rowSize))
			require.NoError(t, cmd.Run())
		}
	}

	return dbPath
}

func TestRunQuickCheck(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	t.Run("valid database succeeds", func(t *testing.T) {
		dbPath := createTestDatabase(t, 10*1024) // 10KB
		err := runQuickCheck(dbPath)
		require.NoError(t, err)
	})

	t.Run("corrupted database returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "corrupt.db")

		// Create a file with garbage data
		err := os.WriteFile(dbPath, []byte("not a valid sqlite database content here"), 0644)
		require.NoError(t, err)

		err = runQuickCheck(dbPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "integrity check failed")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := runQuickCheck("/nonexistent/path/db.sqlite")
		require.Error(t, err)
	})
}
