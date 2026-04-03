package krypt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrKeyNotFound is returned when a requested env key does not exist in the encrypted file.
	ErrKeyNotFound = errors.New("key not found")
)

// Load decrypts an .enc file and returns key-value pairs.
func Load(path string, key []byte) (map[string]string, error) {
	plaintext, err := Decrypt(path, key)
	if err != nil {
		return nil, err
	}
	entries := parseEnv(plaintext)
	return entriesToMap(entries), nil
}

// Autoload decrypts an .enc file and sets all values as env vars.
func Autoload(path string, key []byte) error {
	secrets, err := Load(path, key)
	if err != nil {
		return err
	}
	for k, v := range secrets {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("set env %s: %w", k, err)
		}
	}
	return nil
}

// Encrypt encrypts plaintext .env content and writes to an .enc file.
// The write is atomic (temp file + rename) to prevent data loss on interruption.
func Encrypt(plaintext []byte, path string, key []byte) error {
	ciphertext, err := encrypt(plaintext, key)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".krypt-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(ciphertext); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("set file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// Decrypt decrypts an .enc file and returns the plaintext bytes.
func Decrypt(path string, key []byte) ([]byte, error) {
	ciphertext, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return decrypt(ciphertext, key)
}

// Set sets a single key-value pair in an encrypted file.
func Set(path string, key []byte, envKey, envValue string) error {
	plaintext, err := Decrypt(path, key)
	if err != nil {
		return err
	}
	entries := parseEnv(plaintext)
	entries = setEntry(entries, envKey, envValue)
	return Encrypt(serializeEntries(entries), path, key)
}

// Get retrieves a single value from an encrypted file.
func Get(path string, key []byte, envKey string) (string, error) {
	secrets, err := Load(path, key)
	if err != nil {
		return "", err
	}
	val, ok := secrets[envKey]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, envKey)
	}
	return val, nil
}
