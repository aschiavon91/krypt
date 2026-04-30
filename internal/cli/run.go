package cli

import (
	"errors"
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
			env, cmdArgs, err := parseRunArgs(cmd.ArgsLenAtDash(), args)
			if err != nil {
				return err
			}
			if len(cmdArgs) == 0 {
				return errors.New("missing command: usage: krypt run [env] -- <command> [args...]")
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

			child := exec.Command(cmdArgs[0], cmdArgs[1:]...) //nolint:gosec // user-provided command, by design
			child.Env = buildChildEnv(secrets)
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
					_ = child.Process.Signal(sig)
				}
			}()

			err = child.Wait()
			signal.Stop(sigCh)
			close(sigCh)

			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}

			return nil
		},
	}
}

func parseRunArgs(dashIdx int, args []string) (env string, cmdArgs []string, err error) {
	switch {
	case dashIdx < 0, dashIdx == 0:
		cmdArgs = args
	case dashIdx == 1:
		env, cmdArgs = args[0], args[1:]
	default:
		err = errors.New("too many arguments before --: expected [env] -- <command>")
	}
	return env, cmdArgs, err
}

func buildChildEnv(secrets map[string]string) []string {
	envMap := make(map[string]string, len(os.Environ()))
	for _, e := range os.Environ() {
		if k, v, ok := strings.Cut(e, "="); ok {
			envMap[k] = v
		}
	}
	maps.Copy(envMap, secrets)

	for k := range envMap {
		if k == "KRYPT_KEY" || strings.HasPrefix(k, "KRYPT_KEY_") {
			delete(envMap, k)
		}
	}

	environ := make([]string, 0, len(envMap))
	for k, v := range envMap {
		environ = append(environ, k+"="+v)
	}
	return environ
}
