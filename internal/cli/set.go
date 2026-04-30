package cli

import (
	"errors"
	"strings"

	"github.com/aschiavon91/krypt/pkg/krypt"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY=VALUE [env]",
		Short: "Set a secret in an encrypted file",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv := args[0]
			env := ""
			if len(args) > 1 {
				env = args[1]
			}

			envKey, envValue, ok := strings.Cut(kv, "=")
			if !ok {
				return errors.New("invalid format: expected KEY=VALUE")
			}

			key, err := resolveKey(cmd, env)
			if err != nil {
				return err
			}

			encFile := resolveEncFile(cmd, env)
			return krypt.Set(encFile, key, envKey, envValue)
		},
	}
}
