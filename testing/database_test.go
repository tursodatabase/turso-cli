//go:build integration
// +build integration

package main

import (
	"os"
	"os/exec"
	"testing"

	qt "github.com/frankban/quicktest"
)

var canary bool = false

type testCase func(c *qt.C, dbName string)

func testDestroy(c *qt.C, dbName string) {
	output, err := turso("db", "destroy", "--yes", dbName)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Destroyed database "+dbName)
}

func testCreate(c *qt.C, dbName string, region *string, tc testCase) {
	args := []string{"db", "create", dbName}
	if region != nil {
		args = append(args, "--region", *region)
	}
	if canary {
		args = append(args, "--canary")
	}
	output, err := turso(args...)
	defer testDestroy(c, dbName)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Created database "+dbName)

	if region != nil {
		output, err = turso("db", "show", dbName)
		c.Assert(err, qt.IsNil, qt.Commentf(output))
		c.Assert(output, qt.Contains, "Regions:  "+*region)
	}

	if tc != nil {
		tc(c, dbName)
	}
}

func runSql(c *qt.C, dbName string) {
	output, err := turso("db", "shell", dbName, "create table test(a int, b text)")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	output, err = turso("db", "shell", dbName, "insert into test values(123, 'foobar')")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	output, err = turso("db", "shell", dbName, "select * from test")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
}

func TestDbCreation(t *testing.T) {
	c := qt.New(t)
	testCreate(c, "t1", nil, runSql)
	for _, region := range []string{"waw", "gru", "sea"} {
		testCreate(c, "t1-"+region, &region, runSql)
	}
}

func testReplicate(c *qt.C, dbName string, region string) {
	args := []string{"db", "replicate", dbName, region}
	if canary {
		args = append(args, "--canary")
	}
	output, err := turso(args...)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Replicated database "+dbName)
}

func TestDbReplication(t *testing.T) {
	c := qt.New(t)
	primaryRegion := "waw"
	testCreate(c, "r1", &primaryRegion, func(c *qt.C, dbName string) { testReplicate(c, dbName, "ams") })
}

func turso(args ...string) (string, error) {
	cmd := exec.Command("../cmd/turso/turso", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
