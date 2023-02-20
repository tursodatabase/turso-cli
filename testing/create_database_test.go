//go:build integration
// +build integration

package main

import (
	qt "github.com/frankban/quicktest"
	"os"
	"os/exec"
	"testing"
)

func testDestroy(c *qt.C, dbName string) {
	output, err := turso("db", "destroy", "--yes", dbName)
	c.Assert(err, qt.IsNil)
	c.Assert(output, qt.Contains, "Destroyed database "+dbName)
}

func testCreate(c *qt.C, dbName string) {
	output, err := turso("db", "create", dbName)
	defer testDestroy(c, dbName)
	c.Assert(err, qt.IsNil)
	c.Assert(output, qt.Contains, "Created database "+dbName)
}

func TestDbCreation(t *testing.T) {
	c := qt.New(t)
	testCreate(c, "t1")
}

func turso(args ...string) (string, error) {
	cmd := exec.Command("../cmd/turso/turso", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
