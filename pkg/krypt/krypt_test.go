package krypt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func testEncFile(t *testing.T, key []byte, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.enc")
	if err := Encrypt([]byte(content), path, key); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEncryptDecryptFile(t *testing.T) {
	key := testKey(t)
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.enc")

	content := []byte("SECRET=value\n")
	if err := Encrypt(content, path, key); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// File should exist and not be plaintext
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == string(content) {
		t.Error("encrypted file contains plaintext")
	}

	got, err := Decrypt(path, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestDecryptWrongKeyFile(t *testing.T) {
	key := testKey(t)
	path := testEncFile(t, key, "SECRET=value\n")

	wrongKey := testKey(t)
	_, err := Decrypt(path, wrongKey)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}

func TestDecryptFileNotFound(t *testing.T) {
	key := testKey(t)
	_, err := Decrypt("/nonexistent/path", key)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad(t *testing.T) {
	key := testKey(t)
	path := testEncFile(t, key, "DB_HOST=localhost\nDB_PORT=5432\n")

	secrets, err := Load(path, key)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if secrets["DB_HOST"] != "localhost" {
		t.Errorf("DB_HOST = %q", secrets["DB_HOST"])
	}
	if secrets["DB_PORT"] != "5432" {
		t.Errorf("DB_PORT = %q", secrets["DB_PORT"])
	}
}

func TestAutoload(t *testing.T) {
	key := testKey(t)
	path := testEncFile(t, key, "KRYPT_TEST_VAR=autoloaded\n")

	t.Cleanup(func() {
		os.Unsetenv("KRYPT_TEST_VAR")
	})

	if err := Autoload(path, key); err != nil {
		t.Fatalf("Autoload: %v", err)
	}

	got := os.Getenv("KRYPT_TEST_VAR")
	if got != "autoloaded" {
		t.Errorf("KRYPT_TEST_VAR = %q, want %q", got, "autoloaded")
	}
}

func TestSet(t *testing.T) {
	key := testKey(t)
	path := testEncFile(t, key, "EXISTING=old\n")

	// Update existing key
	if err := Set(path, key, "EXISTING", "new"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	secrets, _ := Load(path, key)
	if secrets["EXISTING"] != "new" {
		t.Errorf("EXISTING = %q, want %q", secrets["EXISTING"], "new")
	}

	// Add new key
	if err := Set(path, key, "ADDED", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	secrets, _ = Load(path, key)
	if secrets["ADDED"] != "value" {
		t.Errorf("ADDED = %q", secrets["ADDED"])
	}
	if secrets["EXISTING"] != "new" {
		t.Errorf("EXISTING lost after adding new key: %q", secrets["EXISTING"])
	}
}

func TestGet(t *testing.T) {
	key := testKey(t)
	path := testEncFile(t, key, "MY_SECRET=hunter2\n")

	val, err := Get(path, key, "MY_SECRET")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "hunter2" {
		t.Errorf("got %q, want %q", val, "hunter2")
	}
}

func TestGetNotFound(t *testing.T) {
	key := testKey(t)
	path := testEncFile(t, key, "OTHER=value\n")

	_, err := Get(path, key, "MISSING")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("expected ErrKeyNotFound, got: %v", err)
	}
}

func TestEncryptOverwritesFile(t *testing.T) {
	key := testKey(t)
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.enc")

	Encrypt([]byte("V1=old\n"), path, key)
	Encrypt([]byte("V2=new\n"), path, key)

	secrets, _ := Load(path, key)
	if _, ok := secrets["V1"]; ok {
		t.Error("V1 should not exist after overwrite")
	}
	if secrets["V2"] != "new" {
		t.Errorf("V2 = %q", secrets["V2"])
	}
}

// Verify GenerateKey produces keys that work with Encrypt/Decrypt
func TestGenerateKeyIntegration(t *testing.T) {
	hexKey, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		t.Fatalf("decode hex key: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, ".env.enc")

	if err := Encrypt([]byte("KEY=value\n"), path, key); err != nil {
		t.Fatal(err)
	}

	secrets, err := Load(path, key)
	if err != nil {
		t.Fatal(err)
	}
	if secrets["KEY"] != "value" {
		t.Errorf("KEY = %q", secrets["KEY"])
	}
}
