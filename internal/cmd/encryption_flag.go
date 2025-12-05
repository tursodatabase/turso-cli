package cmd

import "github.com/spf13/cobra"

var remoteEncryptionKeyFlag string
var remoteEncryptionCipherFlag string

func addRemoteEncryptionKeyFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&remoteEncryptionKeyFlag, "remote-encryption-key", "", "Encryption key (in base64) for accessing encrypted databases on Turso cloud")
}

func addRemoteEncryptionCipherFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&remoteEncryptionCipherFlag, "remote-encryption-cipher", "", "Cipher to use for database encryption")
}
