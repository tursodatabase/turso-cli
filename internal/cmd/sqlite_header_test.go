package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeHeaderFile(t *testing.T, magic string, readVersion, writeVersion byte) string {
	t.Helper()
	header := make([]byte, 100)
	copy(header, magic)
	header[18] = readVersion
	header[19] = writeVersion
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, header, 0644))
	return path
}

func TestSniffSQLiteFileFormat(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		magic                     string
		readVersion, writeVersion byte
		expected                  sqliteFileFormat
	}{
		{"rollback journal", sqliteMagic, 1, 1, fileFormatRollback},
		{"wal", sqliteMagic, 2, 2, fileFormatWAL},
		{"mvcc", sqliteMagic, 255, 255, fileFormatMVCC},
		{"mixed version bytes", sqliteMagic, 2, 255, fileFormatUnknown},
		{"unknown version bytes", sqliteMagic, 42, 42, fileFormatUnknown},
		{"wrong magic", "Not a SQLite db\x00", 2, 2, fileFormatNotSQLite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeHeaderFile(t, tc.magic, tc.readVersion, tc.writeVersion)
			format, err := sniffSQLiteFileFormat(path)
			require.NoError(t, err)
			require.Equal(t, tc.expected, format)
		})
	}

	t.Run("file shorter than header", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "short.db")
		require.NoError(t, os.WriteFile(path, []byte("SQLite"), 0644))
		format, err := sniffSQLiteFileFormat(path)
		require.NoError(t, err)
		require.Equal(t, fileFormatNotSQLite, format)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := sniffSQLiteFileFormat(filepath.Join(t.TempDir(), "missing.db"))
		require.Error(t, err)
	})
}

func TestSniffSQLiteFileFormatOnRealDatabases(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	t.Run("wal database", func(t *testing.T) {
		dbPath := createTestDatabase(t, 10*1024)
		format, err := sniffSQLiteFileFormat(dbPath)
		require.NoError(t, err)
		require.Equal(t, fileFormatWAL, format)
	})

	t.Run("rollback database converts to wal", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rollback.db")
		cmd := exec.Command("sqlite3", "-list", dbPath,
			"PRAGMA page_size=4096;",
			"CREATE TABLE data (id INTEGER PRIMARY KEY);")
		require.NoError(t, cmd.Run(), "failed to create test database")

		format, err := sniffSQLiteFileFormat(dbPath)
		require.NoError(t, err)
		require.Equal(t, fileFormatRollback, format)

		// the conversion handleDBFileAWS performs for rollback files
		out, err := exec.Command("sqlite3", dbPath, "PRAGMA journal_mode=WAL;").CombinedOutput()
		require.NoError(t, err, "convert to WAL: %s", out)

		format, err = sniffSQLiteFileFormat(dbPath)
		require.NoError(t, err)
		require.Equal(t, fileFormatWAL, format)
	})
}
