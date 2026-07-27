package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Clever/csvlint"
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var groupFlag string

func addGroupFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&groupFlag, "group", "", "create the database in the specified group")
	cmd.RegisterFlagCompletionFunc("group", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := authedTursoClient()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		groups, _ := groupNames(client)
		return groups, cobra.ShellCompDirectiveNoFileComp
	})
}

var (
	fromDBFlag    string
	timestampFlag string
)

func addFromDBFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromDBFlag, "from-db", "", "Select another database to copy data from. To use data from a past version of the selected database, see the 'timestamp' flag.")
	cmd.RegisterFlagCompletionFunc("from-db", dbNameArg)
	cmd.Flags().StringVar(&timestampFlag, "timestamp", "", "Set a point in time in the past to copy data from the selected database. Must be used with the 'from-db' flag. Must be in RFC3339 format like '2023-09-29T10:16:13-03:00'")
}

func parseTimestampFlag() (*time.Time, error) {
	if timestampFlag == "" {
		return nil, nil
	}
	if fromDBFlag == "" {
		return nil, errors.New("--timestamp cannot be used without specifying --from-db")
	}

	timestamp, err := time.Parse(time.RFC3339, timestampFlag)
	if err != nil {
		return nil, errors.New("provided timestamp was not in RFC3339 format like '2023-09-29T10:16:13-03:00'")
	}
	return &timestamp, nil
}

func parseDBSeedFlags(client *turso.Client, isAWS bool, cipher string) (*turso.DBSeed, error) {
	if countFlags(fromDBFlag, fromDumpFlag, fromFileFlag, fromDumpURLFlag, fromCSVFlag) > 1 {
		return nil, errors.New("only one of --from prefixed flags can be used at a time")
	}

	timestamp, err := parseTimestampFlag()
	if err != nil {
		return nil, err
	}
	if fromCSVFlag != "" && csvTableNameFlag == "" {
		return nil, errors.New("--csv-table-name must be used with --from-csv")
	}
	if csvTableNameFlag != "" && fromCSVFlag == "" {
		return nil, errors.New("--from-csv must be used with --csv-table-name")
	}

	if fromDBFlag != "" {
		return &turso.DBSeed{Type: "database", Name: fromDBFlag, Timestamp: timestamp}, nil
	}

	if fromFileFlag != "" {
		return handleDBFile(client, fromFileFlag, isAWS, cipher)
	}

	if fromDumpFlag != "" {
		return handleDumpFile(client, fromDumpFlag)
	}

	if fromCSVFlag != "" {
		csvSeparator, err := flags.CSVSeparator()
		if err != nil {
			return nil, err
		}
		return handleCSVFile(client, fromCSVFlag, csvTableNameFlag, csvSeparator, cipher)
	}
	if fromDumpURLFlag != "" {
		return handleDumpURL(fromDumpURLFlag)
	}

	return nil, nil
}

func handleDumpURL(url string) (*turso.DBSeed, error) {
	return &turso.DBSeed{
		Type: "dump",
		URL:  url,
	}, nil
}

func handleDumpFile(client *turso.Client, file string) (*turso.DBSeed, error) {
	dump, err := validateDumpFile(file)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	spinner := prompt.Spinner("Uploading data...")
	defer spinner.Stop()

	dumpURL, err := client.Databases.UploadDump(dump)
	if err != nil {
		return nil, fmt.Errorf("could not upload dump: %w", err)
	}

	spinner.Stop()
	elapsed := time.Since(start)
	fmt.Printf("Uploaded data in %d seconds.\n\n", int(elapsed.Seconds()))

	return handleDumpURL(dumpURL)
}

func validateDumpFile(name string) (*os.File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", name, err)
	}
	fileStat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat file %s: %w", name, err)
	}
	if fileStat.Size() == 0 {
		return nil, errors.New("dump file is empty")
	}
	if fileStat.Size() > MaxDumpFileSizeBytes {
		return nil, errors.New("dump file is too large. max allowed size is 2GB")
	}
	if err := checkDumpFileFirstLines(name, file); err != nil {
		return nil, fmt.Errorf("invalid dump file: %w", err)
	}
	return file, nil
}

func checkDumpFileFirstLines(name string, file *os.File) error {
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	if scanner.Text() != "PRAGMA foreign_keys=OFF;" {
		if checkSQLiteFile(name) == nil {
			return errors.New("you're trying to use a SQLite database file as a dump. Use the --from-db flag instead of --from-dump")
		}
		return errors.New("file doesn't look like a dump: first line should be 'PRAGMA foreign_keys=OFF;'")
	}

	scanner.Scan()
	if scanner.Text() != "BEGIN TRANSACTION;" {
		if checkSQLiteFile(name) == nil {
			return errors.New("you're trying to use a SQLite database file as a dump. Use --from-db instead")
		}
		return errors.New("file doesn't look like a dump: second line should be 'BEGIN TRANSACTION;'")
	}

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("could not seek to the beginning of dump after validating it: %w", err)
	}
	return nil
}

func countFlags(flags ...string) (count int) {
	for _, flag := range flags {
		if flag != "" {
			count++
		}
	}
	return
}

const MaxAWSDBSizeBytes = 1024 * 1024 * 1024 * 20 // 20 GB

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
		firstLine := scanner.Text()
		return strings.TrimSpace(firstLine) == "PRAGMA foreign_keys=OFF;", nil
	} else {
		return false, scanner.Err()
	}
}

// getReservedBytes retrieves the current reserved bytes setting from a SQLite database
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

// validateReservedBytes checks if the database has the required reserved bytes for the given cipher
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

func sqliteFileIntegrityChecks(file string, cipher string) error {
	if flags.Debug() {
		log.Printf("Running integrity checks on database file %s", file)
	}

	if flags.Debug() {
		log.Printf("Checking if this is a sqlite dump: common mistake!...")
	}

	isDump, err := checkIfDump(file)
	if err != nil {
		return fmt.Errorf("failed to get file header: %w", err)
	}
	if isDump {
		return fmt.Errorf("%s is a sqlite3 dump, not a sqlite3 database. Please import a sqlite database", file)
	}

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

	if flags.Debug() {
		log.Printf("Checking database settings...")
	}
	output, err := exec.Command("sqlite3", "-list", file, ".mode line",
		"select journal_mode as j, page_size as p, auto_vacuum as a, encoding as e from pragma_journal_mode, pragma_page_size, pragma_auto_vacuum, pragma_encoding;").CombinedOutput()
	if err != nil {

		return fmt.Errorf("failed to check database settings: %w", err)
	}

	settings := string(output)
	if !strings.Contains(settings, "j: wal") && !strings.Contains(settings, "j = wal") {
		return fmt.Errorf("database is not in WAL mode. Set it with 'sqlite3 %s 'PRAGMA journal_mode = WAL'", file)
	}
	if !strings.Contains(settings, "p: 4096") && !strings.Contains(settings, "p = 4096") {
		return fmt.Errorf("database must use 4KB page size. you can set it with 'sqlite3 %s 'PRAGMA page_size = 4096; VACUUM;' Note that this is not possible to do if your database is already in WAL mode", file)
	}
	if !strings.Contains(settings, "a: 0") && !strings.Contains(settings, "a = 0") {
		return fmt.Errorf("database must have autovacuum disabled. you can set it with 'sqlite3 %s 'PRAGMA auto_vacuum = 0;'", file)
	}
	if !strings.Contains(settings, "e: UTF-8") && !strings.Contains(settings, "e = UTF-8") {
		return fmt.Errorf("database must use UTF-8 encoding. you can set it with 'sqlite3 %s 'PRAGMA encoding = 'UTF-8'	", file)
	}

	// run quick_check
	if flags.Debug() {
		log.Printf("Running integrity check...")
	}
	spinner := prompt.Spinner(fmt.Sprintf("Validating database file (%s)...", humanReadableSize(fileInfo.Size())))
	err = runQuickCheck(file)
	spinner.Stop()
	if err != nil {
		return err
	}

	// validate reserved bytes if encryption cipher is specified
	if cipher != "" {
		if flags.Debug() {
			log.Printf("Checking reserved bytes for cipher %s...", cipher)
		}
		return validateReservedBytes(file, cipher)
	}

	return nil
}

func runQuickCheck(file string) error {
	cmd := exec.Command("sqlite3", "-list", file, "pragma quick_check;")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}
	return nil
}

// checkpointWALBeforeUpload folds any pending WAL frames into the main
// database file and verifies no data is left behind in sidecar files. The
// upload ships only the main database file, so frames still sitting in
// <file>-wal (or a hot rollback journal) would silently be missing from the
// imported database.
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

// tursodbLogPath returns the logical-log sidecar path tursodb uses for a
// database file, mirroring turso_core's `with_extension("db-log")`: the
// file's extension (if any) is replaced with "db-log".
func tursodbLogPath(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file)) + ".db-log"
}

func handleDBFileAWS(file string, cipher string) (*turso.DBSeed, error) {
	format, err := sniffSQLiteFileFormat(file)
	if err != nil {
		return nil, err
	}

	if format == fileFormatRollback {
		// The server only accepts WAL or MVCC format files. Converting to WAL
		// is exactly the remediation the error message used to instruct users
		// to run themselves, and sqlite3 is already a hard requirement here.
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
		if err := checkpointWALBeforeUpload(file); err != nil {
			return nil, err
		}
		if err := sqliteFileIntegrityChecks(file, cipher); err != nil {
			return nil, err
		}
	case fileFormatMVCC:
		// sqlite3-based checks can't run on MVCC (tursodb format) files: the
		// sqlite3 binary reports them as not-a-database. The server verifies
		// the file with the tursodb engine after upload.
		if !tursoDBFlag {
			return nil, fmt.Errorf("%s is in tursodb (MVCC) format and can only be imported into a tursodb database", file)
		}
		if cipher != "" {
			return nil, errors.New("remote encryption is not supported when importing tursodb (MVCC) format files")
		}
		// The upload ships only the main database file: a non-empty tursodb
		// logical log next to it means the file is not a fully checkpointed
		// snapshot and importing it would lose the log's data.
		if err := checkSidecarEmpty(tursodbLogPath(file), "checkpoint the database with tursodb before importing"); err != nil {
			return nil, err
		}
		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get file info: %w", err)
		}
		if fileInfo.Size() > MaxAWSDBSizeBytes {
			return nil, errors.New("database file size exceeds maximum allowed size of 20 GB")
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
	default:
		return nil, fmt.Errorf("file %s has an unsupported SQLite file format", file)
	}

	seed := &turso.DBSeed{
		Type:     "database_upload",
		Filepath: file,
	}

	return seed, nil
}

func handleDBFile(client *turso.Client, file string, isAWS bool, cipher string) (*turso.DBSeed, error) {
	if err := checkFileExists(file); err != nil {
		return nil, err
	}
	if err := checkSQLiteAvailable(); err != nil {
		return nil, err
	}

	if isAWS {
		return handleDBFileAWS(file, cipher)
	}

	format, err := sniffSQLiteFileFormat(file)
	if err != nil {
		return nil, err
	}
	if format == fileFormatMVCC {
		// non-AWS groups are seeded by replaying a .dump, which sqlite3 cannot
		// produce from an MVCC (tursodb format) file
		return nil, fmt.Errorf("%s is in tursodb (MVCC) format and can only be imported into AWS groups", file)
	}

	if err := checkSQLiteFile(file); err != nil {
		return nil, err
	}

	tmp, err := createTempFile()
	if err != nil {
		return nil, err
	}

	if err := dumpSQLiteDatabase(file, tmp); err != nil {
		return nil, err
	}

	return handleDumpFile(client, tmp.Name())
}

func checkFileExists(file string) error {
	_, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not find file %s", file)
	}
	return err
}

func checkSQLiteAvailable() error {
	_, err := exec.LookPath("sqlite3")
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("could not find sqlite3 on your system. Please install it to use the --from-file flag or use --from-dump instead")
	}
	return err
}

func checkSQLiteFile(file string) error {
	output, err := exec.Command("sqlite3", "-list", file, "pragma quick_check;").CombinedOutput()

	execErr := &exec.ExitError{}
	if errors.As(err, &execErr) && execErr.ExitCode() == 26 {
		return fmt.Errorf("file %s is not a valid SQLite database file", file)
	}

	if err != nil {
		return fmt.Errorf("could not check database file: %w: %s", err, output)
	}
	return nil
}

func createTempFile() (*os.File, error) {
	tmp, err := os.CreateTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary file to dump database file: %w", err)
	}
	return tmp, nil
}

func dumpSQLiteDatabase(database string, dump *os.File) error {
	stdErr := &bytes.Buffer{}
	cmd := exec.Command("sqlite3", "-list", database, ".dump")
	cmd.Stdout = dump
	cmd.Stderr = stdErr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not dump database file: %w: %x", err, stdErr.Bytes())
	}

	return nil
}

func handleCSVFile(client *turso.Client, file, csvTableName string, separator rune, cipher string) (*turso.DBSeed, error) {
	if err := checkFileExists(file); err != nil {
		return nil, err
	}
	if err := checkSQLiteAvailable(); err != nil {
		return nil, err
	}

	csvFile, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("could not open CSV file: %w", err)
	}
	defer csvFile.Close()

	errs, invalid, _ := csvlint.Validate(csvFile, separator, false)
	if invalid {
		return nil, fmt.Errorf("CSV file is not valid: %+v", errs)
	}

	tempDB, err := os.CreateTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary file to dump database file: %w", err)
	}
	defer os.Remove(tempDB.Name())

	err = importCSVIntoSQLite(tempDB, csvFile.Name(), csvTableName, separator)
	if err != nil {
		return nil, err
	}

	seed, err := handleDBFile(client, tempDB.Name(), false, cipher)
	if err != nil {
		return nil, err
	}
	return seed, nil
}

func importCSVIntoSQLite(tempDB *os.File, csvFile, csvTableName string, separator rune) error {
	stdErr := &bytes.Buffer{}
	cmd := exec.Command("sqlite3", "-csv", tempDB.Name(), fmt.Sprintf(".separator %c", separator), fmt.Sprintf(".import %s %s", csvFile, csvTableName))
	cmd.Stderr = stdErr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not load csv into new database file: %w: %x", err, stdErr.Bytes())
	}
	return nil
}
