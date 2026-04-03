package cli

import (
	"fmt"

	"github.com/aschiavon91/krypt/pkg/krypt"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY [env]",
		Short: "Get a secret from an encrypted file",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey := args[0]
			env := ""
			if len(args) > 1 {
				env = args[1]
			}

			key, err := resolveKey(cmd, env)
			if err != nil {
				return err
			}

			encFile := resolveEncFile(cmd, env)
			val, err := krypt.Get(encFile, key, envKey)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}
