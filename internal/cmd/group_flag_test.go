package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestReadPragma(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	dbPath := createTestDatabase(t, 4096)

	t.Run("journal_mode returns wal", func(t *testing.T) {
		v, err := readPragma(dbPath, "journal_mode")
		require.NoError(t, err)
		require.True(t, strings.EqualFold(v, "wal"), "got %q", v)
	})

	t.Run("page_size returns 4096", func(t *testing.T) {
		v, err := readPragma(dbPath, "page_size")
		require.NoError(t, err)
		require.Equal(t, "4096", v)
	})

	t.Run("auto_vacuum returns 0", func(t *testing.T) {
		v, err := readPragma(dbPath, "auto_vacuum")
		require.NoError(t, err)
		require.Equal(t, "0", v)
	})

	t.Run("encoding returns UTF-8", func(t *testing.T) {
		v, err := readPragma(dbPath, "encoding")
		require.NoError(t, err)
		require.Equal(t, "UTF-8", v)
	})

	t.Run("nonexistent file errors", func(t *testing.T) {
		_, err := readPragma("/nonexistent/path/db.sqlite", "journal_mode")
		require.Error(t, err)
	})
}

func TestSqliteFileIntegrityChecks(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	t.Run("valid WAL database passes all settings checks", func(t *testing.T) {
		dbPath := createTestDatabase(t, 4096)
		err := sqliteFileIntegrityChecks(dbPath, "")
		require.NoError(t, err)
	})

	t.Run("non-WAL database returns WAL error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "delete.db")
		cmd := exec.Command("sqlite3", "-list", dbPath,
			"PRAGMA page_size=4096;",
			"PRAGMA journal_mode=DELETE;",
			"CREATE TABLE t (id INTEGER PRIMARY KEY);")
		require.NoError(t, cmd.Run())

		err := sqliteFileIntegrityChecks(dbPath, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not in WAL mode")
	})

	t.Run("wrong page size returns page-size error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "8k.db")
		cmd := exec.Command("sqlite3", "-list", dbPath,
			"PRAGMA page_size=8192;",
			"PRAGMA journal_mode=WAL;",
			"CREATE TABLE t (id INTEGER PRIMARY KEY);")
		require.NoError(t, cmd.Run())

		err := sqliteFileIntegrityChecks(dbPath, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "4KB page size")
	})
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
