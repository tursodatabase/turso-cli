package cmd

import "github.com/spf13/cobra"

var authJwtFile string

func addAuthJwtFileFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&authJwtFile, "auth-jwt-key-file", "a", "", "Path to a file with a JWT decoding key used to authenticate clients in the Hrana and HTTP APIs. The key is either a PKCS#8-encoded Ed25519 public key in PEM, or just plain bytes of the Ed25519 public key in URL-safe base64.")
}
