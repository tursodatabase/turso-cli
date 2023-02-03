//go:build integration
// +build integration

package main

import (
	qt "github.com/frankban/quicktest"
	"os"
	"os/exec"
	"testing"
)

func TestReplication(t *testing.T) {
	c := qt.New(t)
	output, err := turso("db", "create", "t1")
	c.Assert(err, qt.IsNil)
	c.Assert(output, qt.Contains, "Created database t1")
	output, err = turso("db", "destroy", "--yes", "t1")
	c.Assert(err, qt.IsNil)
	c.Assert(output, qt.Contains, "Destroyed database t1")
}

func turso(args ...string) (string, error) {
	cmd := exec.Command("../cmd/turso/turso", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
