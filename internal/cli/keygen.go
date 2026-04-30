package cli

import (
	"fmt"

	"github.com/aschiavon91/krypt/pkg/krypt"
	"github.com/spf13/cobra"
)

func newKeygenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keygen",
		Short: "Generate a new encryption key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			key, err := krypt.GenerateKey()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), key)
			return err
		},
	}
}
