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
	cmd := exec.Command("sqlite3", "-list", dbPath,
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
			cmd = exec.Command("sqlite3", "-list", dbPath,
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

func TestTursodbLogPath(t *testing.T) {
	require.Equal(t, "data.db-log", tursodbLogPath("data.db"))
	require.Equal(t, "data.db-log", tursodbLogPath("data.sqlite"))
	require.Equal(t, "/some/dir/mydb.db-log", tursodbLogPath("/some/dir/mydb.db"))
	require.Equal(t, "noext.db-log", tursodbLogPath("noext"))
}

func TestCheckSidecarEmpty(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing sidecar is fine", func(t *testing.T) {
		require.NoError(t, checkSidecarEmpty(filepath.Join(dir, "missing-wal"), "hint"))
	})

	t.Run("empty sidecar is fine", func(t *testing.T) {
		path := filepath.Join(dir, "empty-wal")
		require.NoError(t, os.WriteFile(path, nil, 0644))
		require.NoError(t, checkSidecarEmpty(path, "hint"))
	})

	t.Run("non-empty sidecar errors with hint", func(t *testing.T) {
		path := filepath.Join(dir, "full-wal")
		require.NoError(t, os.WriteFile(path, []byte("frames"), 0644))
		err := checkSidecarEmpty(path, "close all connections")
		require.Error(t, err)
		require.Contains(t, err.Error(), "importing would lose the data it holds")
		require.Contains(t, err.Error(), "close all connections")
	})
}

func TestCheckpointWALBeforeUpload(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	dbPath := createTestDatabase(t, 10*1024)
	require.NoError(t, checkpointWALBeforeUpload(dbPath))

	// after the checkpoint no data-bearing sidecars may remain
	for _, suffix := range []string{"-wal", "-journal"} {
		if info, err := os.Stat(dbPath + suffix); err == nil {
			require.Zero(t, info.Size(), "%s must be empty after checkpoint", dbPath+suffix)
		}
	}
}
