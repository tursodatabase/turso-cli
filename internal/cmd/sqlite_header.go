package cmd

import (
	"io"
	"os"
)

type sqliteFileFormat int

const (
	fileFormatNotSQLite sqliteFileFormat = iota
	fileFormatRollback
	fileFormatWAL
	fileFormatMVCC
	fileFormatUnknown
)

const sqliteMagic = "SQLite format 3\x00"

// sniffSQLiteFileFormat classifies a database file by its SQLite header: the
// 16-byte magic string plus the read/write format version bytes at offsets
// 18/19 (1 = rollback journal, 2 = WAL, 255 = MVCC, i.e. tursodb format).
//
// This must run before any sqlite3 shellout: the sqlite3 binary reports
// MVCC-format files as not-a-database, so they have to be routed around the
// sqlite3-based checks entirely.
func sniffSQLiteFileFormat(path string) (sqliteFileFormat, error) {
	file, err := os.Open(path)
	if err != nil {
		return fileFormatNotSQLite, err
	}
	defer file.Close()

	header := make([]byte, 20)
	if _, err := io.ReadFull(file, header); err != nil {
		// too short to hold a SQLite header
		return fileFormatNotSQLite, nil
	}
	if string(header[:len(sqliteMagic)]) != sqliteMagic {
		return fileFormatNotSQLite, nil
	}
	readVersion, writeVersion := header[18], header[19]
	switch {
	case readVersion == 1 && writeVersion == 1:
		return fileFormatRollback, nil
	case readVersion == 2 && writeVersion == 2:
		return fileFormatWAL, nil
	case readVersion == 255 && writeVersion == 255:
		return fileFormatMVCC, nil
	}
	return fileFormatUnknown, nil
}
