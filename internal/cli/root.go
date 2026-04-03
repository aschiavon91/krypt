package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd creates and returns the root command with all subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "krypt",
		Short:        "Encrypted .env credential manager",
		SilenceUsage: true,
	}

	rootCmd.PersistentFlags().StringP("key", "k", "", "encryption key (hex-encoded, visible in ps — prefer env var or --key-file)")
	rootCmd.PersistentFlags().String("key-file", "", "read encryption key from file (first line, hex-encoded)")
	rootCmd.PersistentFlags().StringP("file", "f", "", "override encrypted file path")
	rootCmd.PersistentFlags().String("source", "", "override source .env file path (encrypt only)")

	rootCmd.AddCommand(
		newKeygenCmd(),
		newEncryptCmd(),
		newDecryptCmd(),
		newEditCmd(),
		newRunCmd(),
		newSetCmd(),
		newGetCmd(),
	)

	return rootCmd
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
