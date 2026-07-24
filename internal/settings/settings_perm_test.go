package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPersistTightensFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows does not honor Unix permission bits the same way; chmod
		// values from os.Stat are not meaningful for this check.
		t.Skip("Unix file mode bits are not enforced on Windows")
	}

	dir := t.TempDir()
	t.Setenv("TURSO_CONFIG_FOLDER", dir)

	s, err := ReadSettings()
	if err != nil {
		t.Fatalf("ReadSettings: %v", err)
	}

	file := filepath.Join(dir, "settings.json")
	st, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat after create: %v", err)
	}
	if got := st.Mode().Perm(); got != 0o600 {
		t.Errorf("fresh file mode = %o, want 600", got)
	}
	stDir, _ := os.Stat(dir)
	if got := stDir.Mode().Perm(); got != 0o700 {
		t.Errorf("fresh dir mode = %o, want 700", got)
	}

	if err := os.Chmod(file, 0o644); err != nil {
		t.Fatal(err)
	}
	s.SetUsername("alice")
	if err := TryToPersistChanges(); err != nil {
		t.Fatalf("TryToPersistChanges: %v", err)
	}
	st, err = os.Stat(file)
	if err != nil {
		t.Fatalf("stat after persist: %v", err)
	}
	if got := st.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode after persist = %o, want 600", got)
	}
}
