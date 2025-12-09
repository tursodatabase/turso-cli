package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// unexported, don't use directly. Consider using remoteEncryptionKeyFlag()
var remoteEncryptionKeyArg string
var remoteEncryptionCipherFlag string

func addRemoteEncryptionKeyFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&remoteEncryptionKeyArg, "remote-encryption-key", "", "Encryption key (in base64) for accessing encrypted databases on Turso cloud")
}

func addRemoteEncryptionCipherFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&remoteEncryptionCipherFlag, "remote-encryption-cipher", "", "Cipher to use for database encryption")
}

func remoteEncryptionKeyFlag() string {
	if remoteEncryptionKeyArg != "" {
		return remoteEncryptionKeyArg
	}
	return os.Getenv("TURSO_DB_REMOTE_ENCRYPTION_KEY")
}
