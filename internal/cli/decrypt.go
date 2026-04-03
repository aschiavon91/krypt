package cli

import (
	"fmt"

	"github.com/aschiavon91/krypt/pkg/krypt"
	"github.com/spf13/cobra"
)

func newDecryptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "decrypt [env]",
		Short: "Decrypt a .env.enc file to stdout",
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

			encFile := resolveEncFile(cmd, env)
			plaintext, err := krypt.Decrypt(encFile, key)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), string(plaintext))
			return nil
		},
	}
}
