package cli

import (
	"fmt"
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

			idx := strings.Index(kv, "=")
			if idx < 0 {
				return fmt.Errorf("invalid format: expected KEY=VALUE")
			}

			envKey := kv[:idx]
			envValue := kv[idx+1:]

			key, err := resolveKey(cmd, env)
			if err != nil {
				return err
			}

			encFile := resolveEncFile(cmd, env)
			return krypt.Set(encFile, key, envKey, envValue)
		},
	}
}
