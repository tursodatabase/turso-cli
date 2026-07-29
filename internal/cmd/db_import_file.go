package cmd

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
	_ "turso.tech/database/tursogo"
)

const MaxAWSDBSizeBytes = 1024 * 1024 * 1024 * 20 // 20 GB

const databaseSettingsQuery = "select journal_mode as j, page_size as p, auto_vacuum as a, encoding as e from pragma_journal_mode, pragma_page_size, pragma_auto_vacuum, pragma_encoding;"

type databaseEngine uint8

const (
	databaseEngineUnknown databaseEngine = iota
	databaseEngineSQLite
	databaseEngineTursoDB
)

func (e databaseEngine) name() string {
	switch e {
	case databaseEngineSQLite:
		return "SQLite"
	case databaseEngineTursoDB:
		return "TursoDB"
	default:
		return "unknown"
	}
}

func (e databaseEngine) settingsCommand() (string, bool) {
	if e == databaseEngineSQLite {
		return "sqlite3", true
	}
	return "", false
}

type databaseFileChecker struct {
	engine      databaseEngine
	journalMode string
	settings    func(string) (databaseSettings, error)
	quickCheck  func(string) error
}

type databaseSettings struct {
	journalMode string
	pageSize    string
	autoVacuum  string
	encoding    string
}

func humanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func checkIfDump(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()) == "PRAGMA foreign_keys=OFF;", nil
	}
	return false, scanner.Err()
}

func validateDatabaseFileSize(file string) error {
	if flags.Debug() {
		log.Printf("Checking file size...")
	}
	fileInfo, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	if fileInfo.Size() > MaxAWSDBSizeBytes {
		return errors.New("database file size exceeds maximum allowed size of 20 GB")
	}
	return nil
}

func sqliteFileIntegrityChecks(file string, cipher string) error {
	return databaseFileIntegrityChecks(file, cipher, databaseFileChecker{
		engine:      databaseEngineSQLite,
		journalMode: "wal",
		settings:    sqliteDatabaseSettings,
		quickCheck:  runQuickCheck,
	})
}

func tursoDBFileIntegrityChecks(file string) error {
	return databaseFileIntegrityChecks(file, "", databaseFileChecker{
		engine:      databaseEngineTursoDB,
		journalMode: "mvcc",
		settings:    tursoDBDatabaseSettings,
		quickCheck:  runTursoDBQuickCheck,
	})
}

func databaseFileIntegrityChecks(file, cipher string, checker databaseFileChecker) error {
	if flags.Debug() {
		log.Printf("Running %s integrity checks on database file %s", checker.engine.name(), file)
		log.Printf("Checking database settings...")
	}

	settings, err := checker.settings(file)
	if err != nil {
		return err
	}
	if err := validateDatabaseSettings(file, settings, checker); err != nil {
		return err
	}

	fileInfo, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	if flags.Debug() {
		log.Printf("Running integrity check...")
	}
	spinner := prompt.Spinner(fmt.Sprintf("Validating database file (%s)...", humanReadableSize(fileInfo.Size())))
	err = checker.quickCheck(file)
	spinner.Stop()
	if err != nil {
		return err
	}

	if cipher != "" {
		if flags.Debug() {
			log.Printf("Checking reserved bytes for cipher %s...", cipher)
		}
		return validateReservedBytes(file, cipher)
	}

	return nil
}

func validateDatabaseSettings(file string, settings databaseSettings, checker databaseFileChecker) error {
	settingsCommand, hasSettingsCommand := checker.engine.settingsCommand()
	if !strings.EqualFold(settings.journalMode, checker.journalMode) {
		if !hasSettingsCommand {
			return fmt.Errorf("database is not in %s mode", strings.ToUpper(checker.journalMode))
		}
		return fmt.Errorf("database is not in %s mode. Set it with '%s %s \"PRAGMA journal_mode = %s;\"'", strings.ToUpper(checker.journalMode), settingsCommand, file, strings.ToUpper(checker.journalMode))
	}
	if settings.pageSize != "4096" {
		if !hasSettingsCommand {
			return errors.New("database must use 4KB page size")
		}
		return fmt.Errorf("database must use 4KB page size. You can set it with '%s %s \"PRAGMA page_size = 4096; VACUUM;\"'", settingsCommand, file)
	}
	if settings.autoVacuum != "0" {
		if !hasSettingsCommand {
			return errors.New("database must have autovacuum disabled")
		}
		return fmt.Errorf("database must have autovacuum disabled. You can set it with '%s %s \"PRAGMA auto_vacuum = 0;\"'", settingsCommand, file)
	}
	if !strings.EqualFold(settings.encoding, "UTF-8") {
		if !hasSettingsCommand {
			return errors.New("database must use UTF-8 encoding")
		}
		return fmt.Errorf("database must use UTF-8 encoding. You can set it with '%s %s \"PRAGMA encoding = 'UTF-8';\"'", settingsCommand, file)
	}
	return nil
}

func parseDatabaseSettings(output string) (databaseSettings, error) {
	values := strings.Split(strings.TrimSpace(output), "|")
	if len(values) != 4 {
		return databaseSettings{}, fmt.Errorf("unexpected database settings output: %s", strings.TrimSpace(output))
	}
	return databaseSettings{
		journalMode: strings.TrimSpace(values[0]),
		pageSize:    strings.TrimSpace(values[1]),
		autoVacuum:  strings.TrimSpace(values[2]),
		encoding:    strings.TrimSpace(values[3]),
	}, nil
}

func sqliteDatabaseSettings(file string) (databaseSettings, error) {
	output, err := exec.Command("sqlite3", "-list", file, databaseSettingsQuery).CombinedOutput()
	if err != nil {
		return databaseSettings{}, fmt.Errorf("failed to check database settings with sqlite3: %w: %s", err, strings.TrimSpace(string(output)))
	}
	settings, err := parseDatabaseSettings(string(output))
	if err != nil {
		return databaseSettings{}, fmt.Errorf("failed to parse database settings from sqlite3: %w", err)
	}
	return settings, nil
}

func tursoDBDatabaseSettings(file string) (databaseSettings, error) {
	db, err := openTursoDB(file)
	if err != nil {
		return databaseSettings{}, fmt.Errorf("failed to check database settings with TursoDB: %w", err)
	}
	defer db.Close()

	var settings databaseSettings
	err = db.QueryRow(databaseSettingsQuery).Scan(
		&settings.journalMode,
		&settings.pageSize,
		&settings.autoVacuum,
		&settings.encoding,
	)
	if err != nil {
		return databaseSettings{}, fmt.Errorf("failed to query database settings with TursoDB: %w", err)
	}
	return settings, nil
}

func openTursoDB(file string) (*sql.DB, error) {
	db, err := sql.Open("turso", file)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func runQuickCheck(file string) error {
	cmd := exec.Command("sqlite3", "-list", file, "pragma quick_check;")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}
	return nil
}

func runTursoDBQuickCheck(file string) error {
	db, err := openTursoDB(file)
	if err != nil {
		return fmt.Errorf("TursoDB integrity check failed for %s: %w", file, err)
	}
	defer db.Close()

	rows, err := db.Query("PRAGMA quick_check;")
	if err != nil {
		return fmt.Errorf("TursoDB integrity check failed for %s: %w", file, err)
	}
	defer rows.Close()

	checked := false
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return fmt.Errorf("TursoDB integrity check failed for %s: %w", file, err)
		}
		checked = true
		if result != "ok" {
			return fmt.Errorf("TursoDB integrity check failed for %s: %s", file, result)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("TursoDB integrity check failed for %s: %w", file, err)
	}
	if !checked {
		return fmt.Errorf("TursoDB integrity check failed for %s: no result", file)
	}
	return nil
}

func checkpointWALBeforeUpload(file string) error {
	if out, err := exec.Command("sqlite3", "-list", file, "PRAGMA wal_checkpoint(TRUNCATE);").CombinedOutput(); err != nil {
		return fmt.Errorf("could not checkpoint database %s: %w: %s", file, err, out)
	}
	for _, sidecar := range []struct{ suffix, hint string }{
		{"-wal", "close all connections to the database and retry the import"},
		{"-journal", "the database has a leftover rollback journal; open and cleanly close it with sqlite3 first"},
	} {
		if err := checkSidecarEmpty(file+sidecar.suffix, sidecar.hint); err != nil {
			return err
		}
	}
	return nil
}

func prepareTursoDBFile(file string) error {
	db, err := openTursoDB(file)
	if err != nil {
		return fmt.Errorf("could not prepare %s for TursoDB import: %w", file, err)
	}
	_, execErr := db.Exec("PRAGMA journal_mode = mvcc; PRAGMA wal_checkpoint(TRUNCATE);")
	closeErr := db.Close()
	if execErr != nil {
		return fmt.Errorf("could not prepare %s for TursoDB import: %w", file, execErr)
	}
	if closeErr != nil {
		return fmt.Errorf("could not close %s after preparing it for TursoDB import: %w", file, closeErr)
	}

	format, err := sniffSQLiteFileFormat(file)
	if err != nil {
		return err
	}
	if format != fileFormatMVCC {
		return fmt.Errorf("TursoDB did not convert %s to MVCC format", file)
	}
	return nil
}

func checkTursoDBSidecars(file string) error {
	for _, sidecar := range []struct{ path, hint string }{
		{file + "-wal", "close all connections to the database and retry the import"},
		{file + "-journal", "the database has a leftover rollback journal; close it cleanly and retry the import"},
		{tursodbLogPath(file), "close all TursoDB connections and retry the import"},
	} {
		if err := checkSidecarEmpty(sidecar.path, sidecar.hint); err != nil {
			return err
		}
	}
	return nil
}

func checkSidecarEmpty(sidecarPath, hint string) error {
	info, err := os.Stat(sidecarPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not check %s: %w", sidecarPath, err)
	}
	if info.Size() > 0 {
		return fmt.Errorf("%s is not empty, importing would lose the data it holds: %s", sidecarPath, hint)
	}
	return nil
}

func tursodbLogPath(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file)) + ".db-log"
}

func handleDBFileAWS(file string, cipher string) (*turso.DBSeed, error) {
	format, err := sniffSQLiteFileFormat(file)
	if err != nil {
		return nil, err
	}

	if format == fileFormatRollback {
		if err := checkSQLiteAvailable(); err != nil {
			return nil, err
		}
		fmt.Printf("File %s uses a rollback journal; converting it to WAL mode for import.\n", file)
		if out, err := exec.Command("sqlite3", file, "PRAGMA journal_mode=WAL;").CombinedOutput(); err != nil {
			return nil, fmt.Errorf("could not convert %s to WAL mode: %w: %s", file, err, out)
		}
		if format, err = sniffSQLiteFileFormat(file); err != nil {
			return nil, err
		}
	}

	switch format {
	case fileFormatWAL:
		if err := checkSQLiteAvailable(); err != nil {
			return nil, err
		}
		if err := checkpointWALBeforeUpload(file); err != nil {
			return nil, err
		}
		if err := sqliteFileIntegrityChecks(file, cipher); err != nil {
			return nil, err
		}
	case fileFormatMVCC:
		if !tursoDBFlag {
			return nil, fmt.Errorf("%s is in tursodb (MVCC) format and can only be imported into a tursodb database", file)
		}
		if cipher != "" {
			return nil, errors.New("remote encryption is not supported when importing tursodb (MVCC) format files")
		}
	case fileFormatNotSQLite:
		isDump, err := checkIfDump(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get file header: %w", err)
		}
		if isDump {
			return nil, fmt.Errorf("%s is a sqlite3 dump, not a sqlite3 database. Please import a sqlite database", file)
		}
		return nil, fmt.Errorf("file %s is not a valid SQLite database file", file)
	case fileFormatUnknown:
		return nil, fmt.Errorf("file %s has unsupported SQLite read/write format versions", file)
	default:
		return nil, fmt.Errorf("file %s has an unsupported SQLite file format", file)
	}

	if tursoDBFlag {
		if format != fileFormatMVCC {
			fmt.Printf("Converting %s to TursoDB (MVCC) format for import.\n", file)
		}
		if err := prepareTursoDBFile(file); err != nil {
			return nil, err
		}
		if err := tursoDBFileIntegrityChecks(file); err != nil {
			return nil, err
		}
		if err := checkTursoDBSidecars(file); err != nil {
			return nil, err
		}
	}

	if err := validateDatabaseFileSize(file); err != nil {
		return nil, err
	}

	return &turso.DBSeed{
		Type:     "database_upload",
		Filepath: file,
	}, nil
}

func getReservedBytes(dbPath string) (int, error) {
	output, err := exec.Command("sqlite3", "-list", dbPath, ".filectrl reserve_bytes").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get reserved bytes: %w", err)
	}
	outputStr := strings.TrimSpace(string(output))
	if strings.Contains(outputStr, ":") {
		parts := strings.Split(outputStr, ":")
		if len(parts) >= 2 {
			outputStr = strings.TrimSpace(parts[1])
		}
	}

	reservedBytes, err := strconv.Atoi(outputStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse reserved bytes from output '%s': %w", string(output), err)
	}
	return reservedBytes, nil
}

func validateReservedBytes(dbPath string, cipher string) error {
	requiredBytes, ok := getRequiredReservedBytes(cipher)
	if !ok {
		return nil
	}

	currentBytes, err := getReservedBytes(dbPath)
	if err != nil {
		return err
	}
	if currentBytes != requiredBytes {
		return fmt.Errorf("database reserved bytes mismatch: found %d, but cipher '%s' requires %d reserved bytes.\nTo fix this, run:\n\n  $ sqlite3 %s\n  sqlite> .filectrl reserve_bytes %d\n  sqlite> VACUUM;",
			currentBytes, cipher, requiredBytes, dbPath, requiredBytes)
	}
	return nil
}
