//go:build integration
// +build integration

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

// Change this to true if you want to test canary image
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

func testReplicate(c *qt.C, dbName string) {
	args := []string{"db", "replicate", dbName, "ams"}
	if canary {
		args = append(args, "--canary")
	}
	output, err := turso(args...)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Replicated database "+dbName)

	output, err = turso("db", "show", dbName)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Regions:  ams, waw")
	c.Assert(output, qt.Contains, "primary     waw")
	c.Assert(output, qt.Contains, "replica     ams")
	primaryPattern := "primary     waw        "
	start := strings.Index(output, primaryPattern) + len(primaryPattern)
	end := start + strings.Index(output[start:], " ")
	primaryUrl := output[start:end]
	replicaPattern := "replica     ams        "
	start = strings.Index(output, replicaPattern) + len(replicaPattern)
	end = start + strings.Index(output[start:], " ")
	replicaUrl := output[start:end]

	// Create table test on primary
	output, err = turso("db", "shell", primaryUrl, "create table test(a int, b text)")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Insert row to test on primary
	output, err = turso("db", "shell", primaryUrl, "insert into test values(123, 'foobar')")
	c.Assert(err, qt.IsNil, qt.Commentf(output))

	// Create table test2 on replica (forwarded to primary)
	output, err = turso("db", "shell", replicaUrl, "create table test2(a int, b text)")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Insert row to test2 on replica (forwarded to primary)
	output, err = turso("db", "shell", replicaUrl, "insert into test2 values(123, 'foobar')")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Select row from test2 on primary
	output, err = turso("db", "shell", primaryUrl, "select * from test2")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")

	// We have to give replication time to happen
	time.Sleep(30 * time.Second)

	// Select row from test on replica
	output, err = turso("db", "shell", replicaUrl, "select * from test")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
	// Select row from test on primary
	output, err = turso("db", "shell", primaryUrl, "select * from test")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")

	// Select row from test2 on replica
	output, err = turso("db", "shell", replicaUrl, "select * from test2")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
}

func TestDbReplication(t *testing.T) {
	c := qt.New(t)
	primaryRegion := "waw"
	testCreate(c, "r1", &primaryRegion, testReplicate)
}

func turso(args ...string) (string, error) {
	cmd := exec.Command("../cmd/turso/turso", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
