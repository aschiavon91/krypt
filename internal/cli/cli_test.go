package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschiavon91/krypt/pkg/krypt"
)

func testSetup(t *testing.T) (key []byte, hexKey string, dir string) {
	t.Helper()
	key = make([]byte, 32)
	rand.Read(key)
	hexKey = hex.EncodeToString(key)
	dir = t.TempDir()
	return
}

func execCmd(t *testing.T, args ...string) (stdout string, stderr string, err error) {
	t.Helper()
	cmd := NewRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestKeygenCommand(t *testing.T) {
	stdout, _, err := execCmd(t, "keygen")
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	output := strings.TrimSpace(stdout)
	if len(output) != 64 {
		t.Errorf("expected 64 hex chars, got %d: %q", len(output), output)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	// Create a plain .env file
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("SECRET=mysecret\nDB=postgres\n"), 0o644)

	encFile := filepath.Join(dir, ".env.enc")

	// Encrypt
	_, _, err := execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Verify encrypted file exists
	if _, err := os.Stat(encFile); err != nil {
		t.Fatalf("encrypted file not created: %v", err)
	}

	// Decrypt via library to verify content
	got, err := krypt.Decrypt(encFile, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(got) != "SECRET=mysecret\nDB=postgres\n" {
		t.Errorf("content mismatch: %q", got)
	}

	// Decrypt via CLI
	stdout, _, err := execCmd(t, "decrypt", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if stdout != "SECRET=mysecret\nDB=postgres\n" {
		t.Errorf("decrypt output: %q", stdout)
	}
}

func TestEncryptMissingSource(t *testing.T) {
	_, hexKey, dir := testSetup(t)
	nonexistent := filepath.Join(dir, ".env.nope")

	_, _, err := execCmd(t, "encrypt", "--key", hexKey, "--source", nonexistent)
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
}

func TestDecryptMissingFile(t *testing.T) {
	_, hexKey, dir := testSetup(t)
	nonexistent := filepath.Join(dir, ".env.enc.nope")

	_, _, err := execCmd(t, "decrypt", "--key", hexKey, "--file", nonexistent)
	if err == nil {
		t.Fatal("expected error for missing encrypted file")
	}
}

func TestDecryptInvalidKey(t *testing.T) {
	_, hexKey, dir := testSetup(t)

	// Create encrypted file
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("SECRET=val\n"), 0o644)
	encFile := filepath.Join(dir, ".env.enc")
	execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile)

	// Try to decrypt with wrong key
	wrongKey := make([]byte, 32)
	rand.Read(wrongKey)
	wrongHex := hex.EncodeToString(wrongKey)

	_, _, err := execCmd(t, "decrypt", "--key", wrongHex, "--file", encFile)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}

func TestKeyResolutionFromEnv(t *testing.T) {
	key, hexKey, dir := testSetup(t)
	_ = key

	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("VAL=fromenv\n"), 0o644)
	encFile := filepath.Join(dir, ".env.enc")

	// Encrypt with --key flag
	execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile)

	// Decrypt using KRYPT_KEY env var
	t.Setenv("KRYPT_KEY", hexKey)
	stdout, _, err := execCmd(t, "decrypt", "--file", encFile)
	if err != nil {
		t.Fatalf("decrypt with KRYPT_KEY env: %v", err)
	}
	if !strings.Contains(stdout, "VAL=fromenv") {
		t.Errorf("unexpected output: %q", stdout)
	}
}

func TestKeyResolutionFromEnvNamed(t *testing.T) {
	_, hexKey, dir := testSetup(t)

	envFile := filepath.Join(dir, ".env.dev")
	os.WriteFile(envFile, []byte("DEV=true\n"), 0o644)
	encFile := filepath.Join(dir, ".env.dev.enc")

	execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile)

	// Should resolve KRYPT_KEY_DEV
	t.Setenv("KRYPT_KEY_DEV", hexKey)
	stdout, _, err := execCmd(t, "decrypt", "dev", "--file", encFile)
	if err != nil {
		t.Fatalf("decrypt with KRYPT_KEY_DEV: %v", err)
	}
	if !strings.Contains(stdout, "DEV=true") {
		t.Errorf("unexpected output: %q", stdout)
	}
}

func TestEditCommand(t *testing.T) {
	_, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.enc")

	// Create a mock editor that writes content
	mockEditor := filepath.Join(dir, "mock-editor.sh")
	os.WriteFile(mockEditor, []byte("#!/bin/sh\necho 'EDITED_KEY=edited_value' > \"$1\"\n"), 0o755)
	t.Setenv("VISUAL", mockEditor)

	// Edit a new file (no existing .enc file)
	_, _, err := execCmd(t, "edit", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

	// Verify the encrypted file was created with the edited content
	stdout, _, err := execCmd(t, "decrypt", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("decrypt after edit: %v", err)
	}
	if !strings.Contains(stdout, "EDITED_KEY=edited_value") {
		t.Errorf("expected edited content, got: %q", stdout)
	}
}

func TestEditExistingFile(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	// Create an existing encrypted file
	encFile := filepath.Join(dir, ".env.enc")
	krypt.Encrypt([]byte("OLD=value\n"), encFile, key)

	// Mock editor appends a line
	mockEditor := filepath.Join(dir, "mock-editor.sh")
	os.WriteFile(mockEditor, []byte("#!/bin/sh\necho 'NEW=appended' >> \"$1\"\n"), 0o755)
	t.Setenv("VISUAL", mockEditor)

	_, _, err := execCmd(t, "edit", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

	stdout, _, _ := execCmd(t, "decrypt", "--key", hexKey, "--file", encFile)
	if !strings.Contains(stdout, "OLD=value") || !strings.Contains(stdout, "NEW=appended") {
		t.Errorf("expected old + new content, got: %q", stdout)
	}
}

func TestRunMissingCommand(t *testing.T) {
	_, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.enc")
	key, _ := hex.DecodeString(hexKey)
	krypt.Encrypt([]byte("K=V\n"), encFile, key)

	_, _, err := execCmd(t, "run", "--key", hexKey, "--file", encFile, "--")
	if err == nil {
		t.Fatal("expected error when no command given after --")
	}
}

func TestSetGetRoundTrip(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	// Create an initial encrypted file
	encFile := filepath.Join(dir, ".env.enc")
	krypt.Encrypt([]byte("EXISTING=keep\n"), encFile, key)

	// Set a new key
	_, _, err := execCmd(t, "set", "NEW_KEY=new_value", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	// Get the new key
	stdout, _, err := execCmd(t, "get", "NEW_KEY", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("get NEW_KEY: %v", err)
	}
	if strings.TrimSpace(stdout) != "new_value" {
		t.Errorf("NEW_KEY = %q, want %q", strings.TrimSpace(stdout), "new_value")
	}

	// Verify existing key is preserved
	stdout, _, err = execCmd(t, "get", "EXISTING", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("get EXISTING: %v", err)
	}
	if strings.TrimSpace(stdout) != "keep" {
		t.Errorf("EXISTING = %q, want %q", strings.TrimSpace(stdout), "keep")
	}
}

func TestSetUpdateExisting(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.enc")
	krypt.Encrypt([]byte("MY_KEY=old\n"), encFile, key)

	_, _, err := execCmd(t, "set", "MY_KEY=updated", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	stdout, _, _ := execCmd(t, "get", "MY_KEY", "--key", hexKey, "--file", encFile)
	if strings.TrimSpace(stdout) != "updated" {
		t.Errorf("MY_KEY = %q, want %q", strings.TrimSpace(stdout), "updated")
	}
}

func TestGetNotFoundCLI(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.enc")
	krypt.Encrypt([]byte("OTHER=val\n"), encFile, key)

	_, _, err := execCmd(t, "get", "MISSING", "--key", hexKey, "--file", encFile)
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestSetInvalidFormat(t *testing.T) {
	_, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.enc")
	key, _ := hex.DecodeString(hexKey)
	krypt.Encrypt([]byte("K=V\n"), encFile, key)

	_, _, err := execCmd(t, "set", "NOEQUALS", "--key", hexKey, "--file", encFile)
	if err == nil {
		t.Fatal("expected error for missing = in KEY=VALUE")
	}
}

func TestSetGetWithEnvName(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.dev.enc")
	krypt.Encrypt([]byte("DEV_VAR=devval\n"), encFile, key)

	// Get with env name
	stdout, _, err := execCmd(t, "get", "DEV_VAR", "dev", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("get with env: %v", err)
	}
	if strings.TrimSpace(stdout) != "devval" {
		t.Errorf("DEV_VAR = %q", strings.TrimSpace(stdout))
	}

	// Set with env name
	_, _, err = execCmd(t, "set", "DEV_VAR=newdev", "dev", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("set with env: %v", err)
	}

	stdout, _, _ = execCmd(t, "get", "DEV_VAR", "dev", "--key", hexKey, "--file", encFile)
	if strings.TrimSpace(stdout) != "newdev" {
		t.Errorf("DEV_VAR = %q", strings.TrimSpace(stdout))
	}
}
