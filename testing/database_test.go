//go:build integration
// +build integration

package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

// Change this to true if you want to test canary image
var canary bool = false

type testCase func(c *qt.C, dbName string, configPath *string)

func testDestroy(c *qt.C, dbName string, configPath *string) {
	output, err := turso(configPath, "db", "destroy", "--yes", dbName)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Destroyed database "+dbName)
}

func testCreate(c *qt.C, dbName string, region *string, configPath *string, tc testCase) {
	args := []string{"db", "create", dbName}
	if region != nil {
		args = append(args, "--region", *region)
	}
	if canary {
		args = append(args, "--canary")
	}
	output, err := turso(configPath, args...)
	defer testDestroy(c, dbName, configPath)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Created database "+dbName)

	if region != nil {
		output, err = turso(configPath, "db", "show", dbName)
		c.Assert(err, qt.IsNil, qt.Commentf(output))
		c.Assert(output, qt.Contains, "Regions:  "+*region)
	}

	if tc != nil {
		tc(c, dbName, configPath)
	}
}

func runSql(c *qt.C, dbName string, configPath *string) {
	output, err := turso(configPath, "db", "shell", dbName, "create table test(a int, b text)")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	output, err = turso(configPath, "db", "shell", dbName, "insert into test values(123, 'foobar')")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	output, err = turso(configPath, "db", "shell", dbName, "select * from test")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
}

func TestDbCreation(t *testing.T) {
	var doneWG sync.WaitGroup
	doneWG.Add(4)
	c := qt.New(t)
	go func() {
		defer doneWG.Done()
		dir, err := os.MkdirTemp("", "turso-test-settings-*")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)
		testCreate(c, "t1", nil, &dir, func(c *qt.C, dbName string, configPath *string) {
			runSql(c, dbName, configPath)
		})
	}()
	for _, region := range []string{"waw", "gru", "sea"} {
		go func(region string) {
			defer doneWG.Done()
			dir, err := os.MkdirTemp("", "turso-test-settings-*")
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(dir)
			testCreate(c, "t1-"+region, &region, &dir, runSql)
		}(region)
	}
	doneWG.Wait()
}

func testReplicate(c *qt.C, dbName string, configPath *string) {
	args := []string{"db", "replicate", dbName, "ams"}
	if canary {
		args = append(args, "--canary")
	}
	output, err := turso(configPath, args...)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Replicated database "+dbName)

	output, err = turso(configPath, "db", "show", dbName)
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
	output, err = turso(configPath, "db", "shell", primaryUrl, "create table test(a int, b text)")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Insert row to test on primary
	output, err = turso(configPath, "db", "shell", primaryUrl, "insert into test values(123, 'foobar')")
	c.Assert(err, qt.IsNil, qt.Commentf(output))

	// Create table test2 on replica (forwarded to primary)
	output, err = turso(configPath, "db", "shell", replicaUrl, "create table test2(a int, b text)")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Insert row to test2 on replica (forwarded to primary)
	output, err = turso(configPath, "db", "shell", replicaUrl, "insert into test2 values(123, 'foobar')")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Select row from test2 on primary
	output, err = turso(configPath, "db", "shell", primaryUrl, "select * from test2")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")

	// We have to give replication time to happen
	time.Sleep(5 * time.Second)

	// Select row from test on replica
	output, err = turso(configPath, "db", "shell", replicaUrl, "select * from test")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
	// Select row from test on primary
	output, err = turso(configPath, "db", "shell", primaryUrl, "select * from test")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")

	// Select row from test2 on replica
	output, err = turso(configPath, "db", "shell", replicaUrl, "select * from test2")
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
}

func TestDbReplication(t *testing.T) {
	c := qt.New(t)
	primaryRegion := "waw"
	testCreate(c, "r1", &primaryRegion, nil, testReplicate)
}

func turso(configPath *string, args ...string) (string, error) {
	var cmd *exec.Cmd
	if configPath != nil {
		newArgs := []string{"-c", *configPath}
		for _, arg := range args {
			newArgs = append(newArgs, arg)
		}
		args = newArgs
	}
	cmd = exec.Command("../cmd/turso/turso", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestMain(m *testing.M) {
	output, err := turso(nil, "auth", "token")
	if err != nil {
		log.Fatal(err)
	}
	if strings.Contains(output, "no user logged in") {
		log.Fatal("Tests need a user to be logged in")
	}
	os.Setenv("TURSO_API_TOKEN", output[:len(output)-1])
	os.Exit(m.Run())
}
