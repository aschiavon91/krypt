package cli

import (
	"fmt"
	"os"

	"github.com/aschiavon91/krypt/pkg/krypt"
	"github.com/spf13/cobra"
)

func newEncryptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "encrypt [env]",
		Short: "Encrypt a .env file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := ""
			if len(args) > 0 {
				env = args[0]
			}

			key, err := resolveKey(cmd, env)
			if err != nil {
				return err
			}

			plainFile := resolvePlainFile(cmd, env)
			plaintext, err := os.ReadFile(plainFile)
			if err != nil {
				return fmt.Errorf("file not found: %s", plainFile)
			}

			encFile := resolveEncFile(cmd, env)
			if err := krypt.Encrypt(plaintext, encFile, key); err != nil {
				return err
			}

			cmd.PrintErrln("encrypted", plainFile, "->", encFile)
			return nil
		},
	}
}
