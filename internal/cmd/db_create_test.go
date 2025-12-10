package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidCipher(t *testing.T) {
	tests := []struct {
		cipher string
		valid  bool
	}{
		{"aes256gcm", true},
		{"aes128gcm", true},
		{"chacha20poly1305", true},
		{"aegis128l", true},
		{"aegis128x2", true},
		{"aegis128x4", true},
		{"aegis256", true},
		{"aegis256x2", true},
		{"aegis256x4", true},
		{"", false},
		{"invalid", false},
		{"AES256GCM", false},
		{"aes-256-gcm", false},
		{"AEGIS256", false},
		{"aegis-256", false},
	}

	for _, tt := range tests {
		t.Run(tt.cipher, func(t *testing.T) {
			got := isValidCipher(tt.cipher)
			if got != tt.valid {
				t.Errorf("isValidCipher(%q) = %v, want %v", tt.cipher, got, tt.valid)
			}
		})
	}
}

func TestGetRequiredReservedBytes(t *testing.T) {
	tests := []struct {
		cipher        string
		expectedBytes int
		expectedOk    bool
	}{
		{"aes256gcm", 28, true},
		{"aes128gcm", 28, true},
		{"chacha20poly1305", 28, true},
		{"aegis128l", 32, true},
		{"aegis128x2", 32, true},
		{"aegis128x4", 32, true},
		{"aegis256", 48, true},
		{"aegis256x2", 48, true},
		{"aegis256x4", 48, true},
		{"invalid", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.cipher, func(t *testing.T) {
			gotBytes, gotOk := getRequiredReservedBytes(tt.cipher)
			if gotBytes != tt.expectedBytes || gotOk != tt.expectedOk {
				t.Errorf("getRequiredReservedBytes(%q) = (%d, %v), want (%d, %v)",
					tt.cipher, gotBytes, gotOk, tt.expectedBytes, tt.expectedOk)
			}
		})
	}
}

func TestValidateEncryptionFlags(t *testing.T) {
	validKey := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	invalidBase64 := "not-valid-base64!!!"

	tests := []struct {
		name        string
		key         string
		cipher      string
		fromDB      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "no encryption flags - valid",
			key:     "",
			cipher:  "",
			fromDB:  "",
			wantErr: false,
		},
		{
			name:        "cipher without key - invalid",
			key:         "",
			cipher:      "aes256gcm",
			fromDB:      "",
			wantErr:     true,
			errContains: "remote encryption key must be provided",
		},
		{
			name:        "invalid base64 key",
			key:         invalidBase64,
			cipher:      "aes256gcm",
			fromDB:      "",
			wantErr:     true,
			errContains: "not valid base64",
		},
		{
			name:    "valid key and cipher",
			key:     validKey,
			cipher:  "aes256gcm",
			fromDB:  "",
			wantErr: false,
		},
		{
			name:    "valid key and cipher - chacha20poly1305",
			key:     validKey,
			cipher:  "chacha20poly1305",
			fromDB:  "",
			wantErr: false,
		},
		{
			name:    "valid key and cipher - aegis256",
			key:     validKey,
			cipher:  "aegis256",
			fromDB:  "",
			wantErr: false,
		},
		{
			name:        "invalid cipher",
			key:         validKey,
			cipher:      "invalid-cipher",
			fromDB:      "",
			wantErr:     true,
			errContains: "unknown encryption cipher",
		},
		{
			name:        "key without cipher and no from-db - invalid",
			key:         validKey,
			cipher:      "",
			fromDB:      "",
			wantErr:     true,
			errContains: "remote encryption cipher must be provided",
		},
		{
			name:    "key without cipher but with from-db - valid (fork case)",
			key:     validKey,
			cipher:  "",
			fromDB:  "source-database",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set CLI flags directly
			remoteEncryptionKeyArg = tt.key
			remoteEncryptionCipherFlag = tt.cipher
			fromDBFlag = tt.fromDB

			err := validateEncryptionFlags()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEncryptionFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error message should contain %q, got: %s", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestValidateEncryptionFlagsAllValidCiphers(t *testing.T) {
	validKey := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

	for cipher := range cipherReservedBytes {
		t.Run(cipher, func(t *testing.T) {
			remoteEncryptionKeyArg = validKey
			remoteEncryptionCipherFlag = cipher

			err := validateEncryptionFlags()
			if err != nil {
				t.Errorf("validateEncryptionFlags() with valid cipher %q returned error: %v", cipher, err)
			}
		})
	}
}

func TestRemoteEncryptionKeyFlag(t *testing.T) {
	t.Run("flag takes precedence over env var", func(t *testing.T) {
		remoteEncryptionKeyArg = "flag-value"
		t.Setenv("TURSO_DB_REMOTE_ENCRYPTION_KEY", "env-value")

		got := remoteEncryptionKeyFlag()
		if got != "flag-value" {
			t.Errorf("expected flag value, got %q", got)
		}
	})

	t.Run("env var used when flag is empty", func(t *testing.T) {
		remoteEncryptionKeyArg = ""
		t.Setenv("TURSO_DB_REMOTE_ENCRYPTION_KEY", "env-value")

		got := remoteEncryptionKeyFlag()
		if got != "env-value" {
			t.Errorf("expected env value, got %q", got)
		}
	})

	t.Run("empty when both flag and env are empty", func(t *testing.T) {
		remoteEncryptionKeyArg = ""
		os.Unsetenv("TURSO_DB_REMOTE_ENCRYPTION_KEY")

		got := remoteEncryptionKeyFlag()
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}

func TestGetReservedBytes(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping integration test")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cmd := exec.Command("sqlite3", dbPath, "CREATE TABLE test (id INTEGER);")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	reservedBytes, err := getReservedBytes(dbPath)
	if err != nil {
		t.Fatalf("getReservedBytes failed: %v", err)
	}

	if reservedBytes != 0 {
		t.Errorf("getReservedBytes() = %d, want 0 for new database", reservedBytes)
	}
}

func TestValidateReservedBytesAllCiphers(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping integration test")
	}

	for cipher, expectedBytes := range cipherReservedBytes {
		t.Run(cipher, func(t *testing.T) {
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")

			cmd := exec.Command("sqlite3", dbPath, "CREATE TABLE test (id INTEGER);")
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to create test database: %v", err)
			}

			cmd = exec.Command("sqlite3", dbPath,
				fmt.Sprintf(".filectrl reserve_bytes %d", expectedBytes),
				"VACUUM;")
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set reserved bytes: %v", err)
			}

			err := validateReservedBytes(dbPath, cipher)
			if err != nil {
				t.Errorf("validateReservedBytes() unexpected error for cipher %s: %v", cipher, err)
			}
		})
	}
}

func TestCipherReservedBytesCrossValidation(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available, skipping integration test")
	}

	wrongBytesTests := []struct {
		cipher     string
		wrongBytes int
	}{
		{"aes256gcm", 32},        // should be 28
		{"aes128gcm", 48},        // should be 28
		{"chacha20poly1305", 32}, // should be 28
		{"aegis128l", 28},        // should be 32
		{"aegis128x2", 48},       // should be 32
		{"aegis128x4", 28},       // should be 32
		{"aegis256", 28},         // should be 48
		{"aegis256x2", 32},       // should be 48
		{"aegis256x4", 28},       // should be 48
	}

	for _, tt := range wrongBytesTests {
		t.Run(fmt.Sprintf("%s_with_%d_bytes", tt.cipher, tt.wrongBytes), func(t *testing.T) {
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")

			cmd := exec.Command("sqlite3", dbPath, "CREATE TABLE test (id INTEGER);")
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to create test database: %v", err)
			}

			cmd = exec.Command("sqlite3", dbPath,
				fmt.Sprintf(".filectrl reserve_bytes %d", tt.wrongBytes),
				"VACUUM;")
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set reserved bytes: %v", err)
			}

			err := validateReservedBytes(dbPath, tt.cipher)
			if err == nil {
				t.Errorf("expected error for cipher %s with %d reserved bytes", tt.cipher, tt.wrongBytes)
			}

			// verify error message includes the correct expected value
			expectedBytes, _ := getRequiredReservedBytes(tt.cipher)
			if !strings.Contains(err.Error(), fmt.Sprintf("requires %d", expectedBytes)) {
				t.Errorf("error should mention required bytes %d", expectedBytes)
			}
		})
	}
}
