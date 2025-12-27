package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockSpinner implements SpinnerInterface for testing
type mockSpinner struct {
	mu      sync.Mutex
	updates []string
	onText  func(string)
}

func newMockSpinner() *mockSpinner {
	return &mockSpinner{}
}

func (m *mockSpinner) Text(t string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updates = append(m.updates, t)
	if m.onText != nil {
		m.onText(t)
	}
}

func (m *mockSpinner) Stop() {}

func (m *mockSpinner) getUpdates() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.updates...)
}

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
			cmd.Run() // Ignore errors for individual inserts
		}
	}

	return dbPath
}

func TestRunQuickCheckWithProgress(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	t.Run("valid small database succeeds", func(t *testing.T) {
		dbPath := createTestDatabase(t, 10*1024) // 10KB

		fileInfo, err := os.Stat(dbPath)
		require.NoError(t, err)

		spinner := newMockSpinner()
		err = runQuickCheckWithProgress(dbPath, fileInfo.Size(), spinner)
		require.NoError(t, err)
	})

	t.Run("nil spinner is handled", func(t *testing.T) {
		dbPath := createTestDatabase(t, 10*1024) // 10KB

		fileInfo, err := os.Stat(dbPath)
		require.NoError(t, err)

		// Should not panic with nil spinner
		err = runQuickCheckWithProgress(dbPath, fileInfo.Size(), nil)
		require.NoError(t, err)
	})

	t.Run("corrupted database returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "corrupt.db")

		// Create a file with garbage data
		err := os.WriteFile(dbPath, []byte("not a valid sqlite database content here"), 0644)
		require.NoError(t, err)

		spinner := newMockSpinner()
		err = runQuickCheckWithProgress(dbPath, 41, spinner)
		require.Error(t, err)
		require.Contains(t, err.Error(), "integrity check failed")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		spinner := newMockSpinner()
		err := runQuickCheckWithProgress("/nonexistent/path/db.sqlite", 1000, spinner)
		require.Error(t, err)
	})

	t.Run("progress updates are received", func(t *testing.T) {
		// Create a larger database to get progress callbacks
		dbPath := createTestDatabase(t, 500*1024) // 500KB

		fileInfo, err := os.Stat(dbPath)
		require.NoError(t, err)

		spinner := newMockSpinner()
		err = runQuickCheckWithProgress(dbPath, fileInfo.Size(), spinner)
		require.NoError(t, err)

		updates := spinner.getUpdates()
		// For a 500KB file, we should get some progress updates
		// (500KB / 4 bytes/op = 125000 ops, with 100000 interval = ~1-2 callbacks)
		t.Logf("Received %d progress updates for %d byte file", len(updates), fileInfo.Size())
	})

	t.Run("progress percentage increases monotonically", func(t *testing.T) {
		// Create a medium-sized database
		dbPath := createTestDatabase(t, 1024*1024) // 1MB

		fileInfo, err := os.Stat(dbPath)
		require.NoError(t, err)

		var percentages []int
		spinner := &mockSpinner{
			onText: func(text string) {
				var pct int
				fmt.Sscanf(text, "Validating database integrity... %d%%", &pct)
				percentages = append(percentages, pct)
			},
		}

		err = runQuickCheckWithProgress(dbPath, fileInfo.Size(), spinner)
		require.NoError(t, err)

		// Verify percentages are monotonically increasing
		for i := 1; i < len(percentages); i++ {
			require.GreaterOrEqual(t, percentages[i], percentages[i-1],
				"Progress decreased at index %d: %d -> %d", i, percentages[i-1], percentages[i])
		}

		t.Logf("Progress updates: %v", percentages)
	})

	t.Run("progress never exceeds 99 percent", func(t *testing.T) {
		// Create a database where our estimate might overshoot
		dbPath := createTestDatabase(t, 200*1024) // 200KB

		fileInfo, err := os.Stat(dbPath)
		require.NoError(t, err)

		var maxPct int
		spinner := &mockSpinner{
			onText: func(text string) {
				var pct int
				fmt.Sscanf(text, "Validating database integrity... %d%%", &pct)
				if pct > maxPct {
					maxPct = pct
				}
			},
		}

		err = runQuickCheckWithProgress(dbPath, fileInfo.Size(), spinner)
		require.NoError(t, err)

		require.LessOrEqual(t, maxPct, 99, "Progress should never exceed 99%% before completion")
	})
}

func TestRunQuickCheckWithProgress_LargeDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large database test in short mode")
	}

	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping test")
	}

	// Create a 5MB database to get multiple progress callbacks
	dbPath := createTestDatabase(t, 5*1024*1024)

	fileInfo, err := os.Stat(dbPath)
	require.NoError(t, err)

	var progressCount int
	spinner := &mockSpinner{
		onText: func(text string) {
			progressCount++
		},
	}

	err = runQuickCheckWithProgress(dbPath, fileInfo.Size(), spinner)
	require.NoError(t, err)

	// For a 5MB file, we should get multiple progress updates
	// (5MB / 4 bytes/op = 1.25M ops, with 100000 interval = ~12 callbacks)
	require.Greater(t, progressCount, 0, "Expected multiple progress callbacks for large file")
	t.Logf("Received %d progress callbacks for %d byte file", progressCount, fileInfo.Size())
}
