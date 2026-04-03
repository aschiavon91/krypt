package cli

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aschiavon91/krypt/pkg/krypt"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [env] -- <command> [args...]",
		Short: "Run a command with decrypted secrets in env",
		RunE: func(cmd *cobra.Command, args []string) error {
			dashIdx := cmd.ArgsLenAtDash()

			var env string
			var cmdArgs []string

			if dashIdx < 0 {
				// No --, treat all args as command (no env)
				cmdArgs = args
			} else if dashIdx == 0 {
				cmdArgs = args
			} else if dashIdx == 1 {
				env = args[0]
				cmdArgs = args[1:]
			} else {
				return fmt.Errorf("too many arguments before --: expected [env] -- <command>")
			}

			if len(cmdArgs) == 0 {
				return fmt.Errorf("missing command: usage: krypt run [env] -- <command> [args...]")
			}

			key, err := resolveKey(cmd, env)
			if err != nil {
				return err
			}

			encFile := resolveEncFile(cmd, env)
			secrets, err := krypt.Load(encFile, key)
			if err != nil {
				return err
			}

			// Build env map from current env, then overlay secrets
			envMap := make(map[string]string)
			for _, e := range os.Environ() {
				if before, after, ok := strings.Cut(e, "="); ok {
					envMap[before] = after
				}
			}
			maps.Copy(envMap, secrets)

			// Strip encryption keys from child environment — the child
			// should only see the decrypted secrets, not the master key.
			for k := range envMap {
				if k == "KRYPT_KEY" || strings.HasPrefix(k, "KRYPT_KEY_") {
					delete(envMap, k)
				}
			}

			environ := make([]string, 0, len(envMap))
			for k, v := range envMap {
				environ = append(environ, k+"="+v)
			}

			child := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			child.Env = environ
			child.Stdin = os.Stdin
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			if err := child.Start(); err != nil {
				return fmt.Errorf("start command: %w", err)
			}

			go func() {
				for sig := range sigCh {
					child.Process.Signal(sig)
				}
			}()

			err = child.Wait()
			signal.Stop(sigCh)
			close(sigCh)

			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}

			return nil
		},
	}
}
