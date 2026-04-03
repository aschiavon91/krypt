package cli

import (
	"os"

	"github.com/aschiavon91/krypt/internal/editor"
	"github.com/aschiavon91/krypt/pkg/krypt"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [env]",
		Short: "Edit encrypted secrets in $EDITOR",
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

			var plaintext []byte
			if _, statErr := os.Stat(encFile); statErr == nil {
				plaintext, err = krypt.Decrypt(encFile, key)
				if err != nil {
					return err
				}
			}

			edited, err := editor.Edit(plaintext)
			if err != nil {
				return err
			}

			if err := krypt.Encrypt(edited, encFile, key); err != nil {
				return err
			}

			cmd.PrintErrln("saved", encFile)
			return nil
		},
	}
}
