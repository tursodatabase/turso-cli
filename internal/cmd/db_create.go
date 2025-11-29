package cmd

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/athoscouto/codename"
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func showSchemaDeprecationNotice() {
	fmt.Println(internal.Warn("Notice: Schema Databases are deprecated."))
	fmt.Println(internal.Warn("For more information, visit: https://tur.so/schema-deprecated\n"))
}

const MaxDumpFileSizeBytes = 8 << 30

// cipherReservedBytes maps cipher names to their required reserved bytes
var cipherReservedBytes = map[string]int{
	"aes256gcm":        28,
	"aes128gcm":        28,
	"chacha20poly1305": 28,
	"aegis128l":        32,
	"aegis128x2":       32,
	"aegis128x4":       32,
	"aegis256":         48,
	"aegis256x2":       48,
	"aegis256x4":       48,
}

func isValidCipher(cipher string) bool {
	_, ok := cipherReservedBytes[cipher]
	return ok
}

func getRequiredReservedBytes(cipher string) (int, bool) {
	bytes, ok := cipherReservedBytes[cipher]
	return bytes, ok
}

func init() {
	dbCmd.AddCommand(createCmd)
	addGroupFlag(createCmd)
	addFromDBFlag(createCmd)
	addDbFromDumpFlag(createCmd)
	addDbFromDumpURLFlag(createCmd)
	addDbFromFileFlag(createCmd)
	addDbFromCSVFlag(createCmd)
	addCSVTableNameFlag(createCmd)
	flags.AddCSVSeparator(createCmd)
	addLocationFlag(createCmd, "Location ID. If no ID is specified, closest location to you is used by default.")
	addWaitFlag(createCmd, "Wait for the database to be ready to receive requests.")
	addEnableExtensionsFlag(createCmd)
	addSchemaFlag(createCmd)
	addTypeFlag(createCmd)
	addSizeLimitFlag(createCmd)
	addRemoteEncryptionCipherFlag(createCmd)
	addRemoteEncryptionKeyFlag(createCmd)
}

var createCmd = &cobra.Command{
	Use:               "create [flags] [database-name]",
	Short:             "Create a database.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: noFilesArg,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if schemaFlag != "" || typeFlag == "schema" {
			showSchemaDeprecationNotice()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		name, err := getDatabaseName(args)
		if err != nil {
			return err
		}
		return CreateDatabase(name)
	},
}

func CreateDatabase(name string) error {
	client, err := authedTursoClient()
	if err != nil {
		return err
	}

	groups, err := client.Groups.List()
	if err != nil {
		return err
	}

	group, err := groupFromFlag(groups)
	if err != nil {
		return err
	}
	groupName := group.Name

	location, err := locationFromFlag(client, group, groups)
	if err != nil {
		return err
	}

	isAWS := strings.HasPrefix(group.Primary, "aws-")

	if err = validateEncryptionFlags(); err != nil {
		return err
	}

	seed, err := parseDBSeedFlags(client, isAWS, remoteEncryptionCipherFlag, multipartFlag)
	if err != nil {
		return err
	}

	if err := ensureGroup(client, groupName, groups, location, "latest"); err != nil {
		return err
	}

	start := time.Now()
	spinner := prompt.Spinner(fmt.Sprintf("Creating database %s in group %s...", internal.Emph(name), internal.Emph(groupName)))
	defer spinner.Stop()

	if _, err = client.Databases.Create(name, location, "", "", groupName, schemaFlag, typeFlag == "schema", seed, sizeLimitFlag, remoteEncryptionCipherFlag, remoteEncryptionKeyFlag(), spinner); err != nil {
		return fmt.Errorf("could not create database %s: %w", name, err)
	}

	spinner.Stop()
	elapsed := time.Since(start)
	fmt.Printf("Created database %s at group %s in %s.\n\n", internal.Emph(name), internal.Emph(groupName), elapsed.Round(time.Millisecond).String())

	fmt.Printf("Start an interactive SQL shell with:\n\n")
	fmt.Printf("   %s\n\n", internal.Emph("turso db shell "+name))
	fmt.Printf("To see information about the database, including a connection URL, run:\n\n")
	fmt.Printf("   %s\n\n", internal.Emph("turso db show "+name))
	fmt.Printf("To get an authentication token for the database, run:\n\n")
	fmt.Printf("   %s\n\n", internal.Emph("turso db tokens create "+name))
	invalidateDatabasesCache()
	return nil
}

func ensureGroup(client *turso.Client, group string, groups []turso.Group, location, version string) error {
	if !shouldAutoCreateGroup(group, groups) {
		return nil
	}
	if err := createGroup(client, group, location, version); err != nil {
		return err
	}
	return client.Groups.WaitLocation(group, location)
}

func getDatabaseName(args []string) (string, error) {
	if len(args) > 0 && len(args[0]) > 0 {
		return args[0], nil
	}

	rng, err := codename.DefaultRNG()
	if err != nil {
		return "", err
	}
	return codename.Generate(rng, 0), nil
}

// Returns (group, error)
func groupFromFlag(groups []turso.Group) (turso.Group, error) {

	if groupFlag != "" {
		if !groupExists(groups, groupFlag) {
			return turso.Group{}, fmt.Errorf("group %s does not exist. Please double-check the name. You can run 'turso group list' to get a list of your groups, or 'turso group create' to make a new one", groupFlag)
		}
		for _, group := range groups {
			if group.Name == groupFlag {
				return group, nil
			}
		}
		return turso.Group{}, fmt.Errorf("group %s does not exist. Please double-check the name. You can run 'turso group list' to get a list of your groups, or 'turso group create' to make a new one", groupFlag)
	}

	switch {
	case len(groups) == 0:
		return turso.Group{Name: "default"}, nil
	case len(groups) == 1:
		return groups[0], nil
	default:
		return turso.Group{}, fmt.Errorf("you have more than one database group. Please specify one with %s", internal.Emph("--group"))

	}
}

func groupExists(groups []turso.Group, name string) bool {
	for _, group := range groups {
		if group.Name == name {
			return true
		}
	}
	return false
}

func locationFromFlag(client *turso.Client, group turso.Group, groups []turso.Group) (string, error) {
	loc := locationFlag
	groupWillBeAutoCreated := shouldAutoCreateGroup(group.Name, groups)
	if loc == "" {
		if groupWillBeAutoCreated {
			loc, _ = closestLocation(client)
		} else {
			loc = group.Primary
		}
	}
	if !groupWillBeAutoCreated {
		var groupContainsLocation bool
		for _, l := range group.Locations {
			if l == loc {
				groupContainsLocation = true
			}
		}
		if !groupContainsLocation {
			return "", fmt.Errorf("location '%s' is not valid for group '%s'. The group has the following locations: %v. You can use 'turso group locations add' to add a new location to the group", loc, group.Name, strings.Join(group.Locations, ", "))
		}

		return loc, nil
	}
	if !isValidLocation(client, loc) {
		return "", fmt.Errorf("location '%s' is not valid", loc)
	}
	return loc, nil
}

func shouldAutoCreateGroup(name string, groups []turso.Group) bool {
	// we only create the default group automatically
	return name == "default" && len(groups) == 0
}

func validateEncryptionFlags() error {
	remoteEncryptionKey := remoteEncryptionKeyFlag()
	if remoteEncryptionKey == "" && remoteEncryptionCipherFlag == "" {
		return nil
	}
	// if key flag is empty, then user passed only the cipher, which is invalid
	if remoteEncryptionKey == "" {
		return fmt.Errorf("remote encryption key must be provided when remote encryption cipher is set")
	}

	// if key is provided, lets verify its in base64 encoded
	_, err := base64.StdEncoding.DecodeString(remoteEncryptionKey)
	if err != nil {
		return fmt.Errorf("encryption key (%s) is not valid base64: %w", remoteEncryptionKey, err)
	}

	// if cipher is empty, then it is only valid in case of forks and for everything else we need to have it set
	if remoteEncryptionCipherFlag == "" {
		if fromDBFlag == "" {
			return fmt.Errorf("remote encryption cipher must be provided when remote encryption key is set")
		}
		// for forks, cipher can be derived from source database
		return nil
	}

	if !isValidCipher(remoteEncryptionCipherFlag) {
		validCiphers := make([]string, 0, len(cipherReservedBytes))
		for cipher := range cipherReservedBytes {
			validCiphers = append(validCiphers, cipher)
		}
		return fmt.Errorf("unknown encryption cipher: %s. Valid ciphers are: %s", remoteEncryptionCipherFlag, strings.Join(validCiphers, ", "))
	}

	return nil
}
