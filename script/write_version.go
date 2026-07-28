//go:build ignore

// Writes version.txt into the current working directory (internal/cmd when
// invoked via go:generate).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var semverTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

func version() string {
	head, err := git("describe", "--tags", "HEAD")
	if err == nil && semverTag.MatchString(head) {
		return head
	}

	left, leftErr := git("describe", "--tags", "HEAD^1")
	right, rightErr := git("describe", "--tags", "HEAD^2")
	if rightErr != nil {
		hash, hashErr := git("log", "-1", "--pretty=%h")
		if hashErr != nil {
			return "dev"
		}
		return hash
	}

	if leftErr == nil && semverTag.MatchString(left) {
		if semverTag.MatchString(right) {
			if left > right {
				return left
			}
			return right
		}
		return left
	}
	if semverTag.MatchString(right) {
		return right
	}

	hash, hashErr := git("log", "-1", "--pretty=%h")
	if hashErr != nil {
		return "dev"
	}
	return hash
}

func main() {
	v := version()
	if err := os.WriteFile("version.txt", []byte(v), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write_version: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(v)
}
