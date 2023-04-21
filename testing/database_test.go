//go:build integration
// +build integration

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
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
		args = append(args, "--location", *region)
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
		c.Assert(output, qt.Contains, "Locations:      "+*region)
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
	dbNamePrefix := uuid.NewString()
	go func() {
		defer doneWG.Done()
		dir, err := os.MkdirTemp("", "turso-test-settings-*")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)
		testCreate(c, dbNamePrefix, nil, &dir, func(c *qt.C, dbName string, configPath *string) {
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
			testCreate(c, dbNamePrefix+"-"+region, &region, &dir, runSql)
		}(region)
	}
	doneWG.Wait()
}

func createReplica(c *qt.C, dbName string, configPath *string, replicaName string) {
	args := []string{"db", "replicate", dbName, "ams", replicaName}
	if canary {
		args = append(args, "--canary")
	}
	output, err := turso(configPath, args...)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Replicated database "+dbName)
}

func runSqlOnPrimaryAndReplica(c *qt.C, dbName string, configPath *string, tablePrefix string, replicaName string) {
	output, err := turso(configPath, "db", "show", dbName)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Contains, "Locations:      ams, waw")
	c.Assert(output, qt.Contains, "primary     waw")
	c.Assert(output, qt.Contains, "replica     ams")
	primaryPattern := "primary     waw"
	start := strings.Index(output, primaryPattern) + len(primaryPattern)
	start = start + strings.IndexFunc(output[start:], func(r rune) bool { return r != ' ' })
	start = start + strings.Index(output[start:], " ")
	start = start + strings.IndexFunc(output[start:], func(r rune) bool { return r != ' ' })
	end := start + strings.Index(output[start:], " ")
	primaryUrl := output[start:end]
	output, err = turso(configPath, "db", "show", dbName, "--instance-url", replicaName)
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	replicaUrl := strings.TrimSpace(output)

	// Create table test on primary
	output, err = turso(configPath, "db", "shell", primaryUrl, fmt.Sprintf("create table %s1(a int, b text)", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Insert row to test on primary
	output, err = turso(configPath, "db", "shell", primaryUrl, fmt.Sprintf("insert into %s1 values(123, 'foobar')", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))

	// Create table test2 on replica (forwarded to primary)
	output, err = turso(configPath, "db", "shell", replicaUrl, fmt.Sprintf("create table %s2(a int, b text)", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Insert row to test2 on replica (forwarded to primary)
	output, err = turso(configPath, "db", "shell", replicaUrl, fmt.Sprintf("insert into %s2 values(123, 'foobar')", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	// Select row from test2 on primary
	output, err = turso(configPath, "db", "shell", primaryUrl, fmt.Sprintf("select * from %s2", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")

	// We have to give replication time to happen
	time.Sleep(5 * time.Second)

	// Select row from test on replica
	output, err = turso(configPath, "db", "shell", replicaUrl, fmt.Sprintf("select * from %s1", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
	// Select row from test on primary
	output, err = turso(configPath, "db", "shell", primaryUrl, fmt.Sprintf("select * from %s1", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")

	// Select row from test2 on replica
	output, err = turso(configPath, "db", "shell", replicaUrl, fmt.Sprintf("select * from %s2", tablePrefix))
	c.Assert(err, qt.IsNil, qt.Commentf(output))
	c.Assert(output, qt.Equals, "A    B       \n123  foobar  \n")
}

func TestDbReplication(t *testing.T) {
	c := qt.New(t)
	primaryRegion := "waw"
	dbNamePrefix := uuid.NewString()
	testCreate(c, dbNamePrefix, &primaryRegion, nil, func(c *qt.C, dbName string, configPath *string) {
		replicaName := uuid.NewString()
		createReplica(c, dbName, configPath, replicaName)
		runSqlOnPrimaryAndReplica(c, dbName, configPath, "replication_test_table", replicaName)
	})
}

func changePassword(c *qt.C, dbName string, configPath *string, newPassword string) {
	turso(configPath, "db", "passwd", dbName, "-p", newPassword)
}

func TestChangeDbPassword(t *testing.T) {
	c := qt.New(t)
	var doneWG sync.WaitGroup
	doneWG.Add(2)
	dbNamePrefix := uuid.NewString()
	go func() {
		defer doneWG.Done()
		dir, err := os.MkdirTemp("", "turso-test-settings-*")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)
		primaryRegion := "waw"
		testCreate(c, dbNamePrefix+"1", &primaryRegion, &dir, func(c *qt.C, dbName string, configPath *string) {
			replicaName := uuid.NewString()
			createReplica(c, dbName, configPath, replicaName)
			runSqlOnPrimaryAndReplica(c, dbName, configPath, "change_password_test_table_before", replicaName)
			changePassword(c, dbName, configPath, "new_awesome_password")
			runSqlOnPrimaryAndReplica(c, dbName, configPath, "change_password_test_table_after", replicaName)
		})
	}()
	go func() {
		defer doneWG.Done()
		dir, err := os.MkdirTemp("", "turso-test-settings-*")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)
		primaryRegion := "waw"
		testCreate(c, dbNamePrefix+"2", &primaryRegion, &dir, func(c *qt.C, dbName string, configPath *string) {
			replicaName := uuid.NewString()
			changePassword(c, dbName, configPath, "new_awesome_password")
			createReplica(c, dbName, configPath, replicaName)
			runSqlOnPrimaryAndReplica(c, dbName, configPath, "change_password_test_table_before", replicaName)
		})
	}()
	doneWG.Wait()
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
	if len(os.Getenv("TURSO_API_TOKEN")) == 0 {
		output, err := turso(nil, "auth", "token")
		if err != nil {
			log.Fatal(err)
		}
		if strings.Contains(output, "no user logged in") {
			log.Fatal("Tests need a user to be logged in")
		}
		os.Setenv("TURSO_API_TOKEN", output[:len(output)-1])
	}
	os.Exit(m.Run())
}
