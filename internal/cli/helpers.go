package cli

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func resolveKey(cmd *cobra.Command, env string) ([]byte, error) {
	// 1. --key flag
	if keyFlag, _ := cmd.Flags().GetString("key"); keyFlag != "" {
		return decodeKey(keyFlag)
	}

	// 2. --key-file flag
	if keyFile, _ := cmd.Flags().GetString("key-file"); keyFile != "" {
		return readKeyFile(keyFile)
	}

	// 3. KRYPT_KEY_<ENV>
	if env != "" {
		envVar := "KRYPT_KEY_" + strings.ToUpper(env)
		if val := os.Getenv(envVar); val != "" {
			return decodeKey(val)
		}
	}

	// 3. KRYPT_KEY
	if val := os.Getenv("KRYPT_KEY"); val != "" {
		return decodeKey(val)
	}

	// 5. Error
	if env != "" {
		return nil, fmt.Errorf("encryption key not found: set KRYPT_KEY_%s or use --key-file", strings.ToUpper(env))
	}
	return nil, errors.New("encryption key not found: set KRYPT_KEY or use --key-file")
}

func readKeyFile(path string) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from --key-file flag, user-controlled by design
	if err != nil {
		return nil, fmt.Errorf("open key file: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only file, close error not actionable

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, fmt.Errorf("key file is empty: %s", path)
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return nil, fmt.Errorf("key file is empty: %s", path)
	}
	return decodeKey(line)
}

func decodeKey(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, errors.New("invalid key: not valid hex")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key: expected 64 hex characters (32 bytes), got %d", len(hexKey))
	}
	return key, nil
}

func resolveEncFile(cmd *cobra.Command, env string) string {
	if f, _ := cmd.Flags().GetString("file"); f != "" {
		return f
	}
	if env == "" {
		return ".env.enc"
	}
	return ".env." + env + ".enc"
}

func resolvePlainFile(cmd *cobra.Command, env string) string {
	if f, _ := cmd.Flags().GetString("source"); f != "" {
		return f
	}
	if env == "" {
		return ".env"
	}
	return ".env." + env
}
