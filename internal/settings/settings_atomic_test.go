package settings

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func isolatedSettings(t *testing.T) (*Settings, string) {
	t.Helper()
	settings = nil
	viper.Reset()
	t.Cleanup(func() {
		settings = nil
		viper.Reset()
	})
	dir := t.TempDir()
	t.Setenv("TURSO_CONFIG_FOLDER", dir)
	s, err := ReadSettings()
	if err != nil {
		t.Fatal(err)
	}
	return s, filepath.Join(dir, "settings.json")
}

func TestPersistReplacesFileWithoutChangingAnExistingReader(t *testing.T) {
	s, filename := isolatedSettings(t)
	s.SetUsername("before")
	if err := TryToPersistChanges(); err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	s.SetUsername("after")
	if err := TryToPersistChanges(); err != nil {
		t.Fatal(err)
	}
	read, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(read, previous) {
		t.Fatalf("an open reader saw its settings change: %q, error %v", read, err)
	}
	current, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := json.Unmarshal(current, &data); err != nil || data["username"] != "after" {
		t.Fatalf("new settings = %q, error %v", current, err)
	}
	st, err := os.Stat(filename)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("replacement permissions: %v, error %v", st, err)
	}
	assertNoTemporarySettings(t, filename)
}

func TestFailedSerializationKeepsPreviousSettings(t *testing.T) {
	s, filename := isolatedSettings(t)
	s.SetUsername("saved")
	if err := TryToPersistChanges(); err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	viper.Set("unsupported", make(chan int))
	if err := TryToPersistChanges(); err == nil {
		t.Fatal("unsupported JSON value did not fail")
	}
	current, err := os.ReadFile(filename)
	if err != nil || !bytes.Equal(current, previous) {
		t.Fatalf("failed save changed the settings: %q, error %v", current, err)
	}
	assertNoTemporarySettings(t, filename)
}

func assertNoTemporarySettings(t *testing.T, filename string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(filename))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "settings.json" {
		t.Fatalf("temporary settings left behind: %v", entries)
	}
}

func TestPersistKeepsSettingsSymlink(t *testing.T) {
	s, filename := isolatedSettings(t)
	target := filepath.Join(t.TempDir(), "linked.json")
	if err := os.Rename(filename, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filename); err != nil {
		t.Fatal(err)
	}
	s.SetUsername("linked")
	if err := TryToPersistChanges(); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Lstat(filename); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("settings symlink was replaced: %v, %v", st, err)
	}
	data, err := os.ReadFile(target)
	if err != nil || !bytes.Contains(data, []byte(`"username": "linked"`)) {
		t.Fatalf("symlink target was not updated: %q, %v", data, err)
	}
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary file left at symlink target: %v, %v", entries, err)
	}
}

func TestFailedReplacementCleansTemporaryFile(t *testing.T) {
	_, filename := isolatedSettings(t)
	destination := filepath.Join(filepath.Dir(filename), "destination")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	viper.SetConfigFile(destination)
	if err := TryToPersistChanges(); err == nil {
		t.Fatal("replacing a directory succeeded")
	}
	entries, err := os.ReadDir(filepath.Dir(filename))
	if err != nil || len(entries) != 2 {
		t.Fatalf("temporary file left after failed replacement: %v, %v", entries, err)
	}
}
