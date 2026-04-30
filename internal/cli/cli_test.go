// Package cli provides integration tests for the krypt CLI commands.
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

func testSetup(t *testing.T) (key []byte, hexKey, dir string) {
	t.Helper()
	key = make([]byte, 32)
	rand.Read(key)
	hexKey = hex.EncodeToString(key)
	dir = t.TempDir()
	return key, hexKey, dir
}

func execCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
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

	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("SECRET=mysecret\nDB=postgres\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	encFile := filepath.Join(dir, ".env.enc")

	_, _, err := execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if _, err := os.Stat(encFile); err != nil {
		t.Fatalf("encrypted file not created: %v", err)
	}

	got, err := krypt.Decrypt(encFile, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(got) != "SECRET=mysecret\nDB=postgres\n" {
		t.Errorf("content mismatch: %q", got)
	}

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

	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("SECRET=val\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	encFile := filepath.Join(dir, ".env.enc")
	if _, _, err := execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile); err != nil {
		t.Fatal(err)
	}

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
	if err := os.WriteFile(envFile, []byte("VAL=fromenv\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	encFile := filepath.Join(dir, ".env.enc")
	if _, _, err := execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile); err != nil {
		t.Fatal(err)
	}

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
	if err := os.WriteFile(envFile, []byte("DEV=true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	encFile := filepath.Join(dir, ".env.dev.enc")
	if _, _, err := execCmd(t, "encrypt", "--key", hexKey, "--file", encFile, "--source", envFile); err != nil {
		t.Fatal(err)
	}

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

	mockEditor := filepath.Join(dir, "mock-editor.sh")
	if err := os.WriteFile(mockEditor, []byte("#!/bin/sh\necho 'EDITED_KEY=edited_value' > \"$1\"\n"), 0o755); err != nil { //nolint:gosec // mock editor script requires executable permissions
		t.Fatal(err)
	}
	t.Setenv("VISUAL", mockEditor)

	_, _, err := execCmd(t, "edit", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

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

	encFile := filepath.Join(dir, ".env.enc")
	if err := krypt.Encrypt([]byte("OLD=value\n"), encFile, key); err != nil {
		t.Fatal(err)
	}

	mockEditor := filepath.Join(dir, "mock-editor.sh")
	if err := os.WriteFile(mockEditor, []byte("#!/bin/sh\necho 'NEW=appended' >> \"$1\"\n"), 0o755); err != nil { //nolint:gosec // mock editor script requires executable permissions
		t.Fatal(err)
	}
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
	if err := krypt.Encrypt([]byte("K=V\n"), encFile, key); err != nil {
		t.Fatal(err)
	}

	_, _, err := execCmd(t, "run", "--key", hexKey, "--file", encFile, "--")
	if err == nil {
		t.Fatal("expected error when no command given after --")
	}
}

func TestSetGetRoundTrip(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.enc")
	if err := krypt.Encrypt([]byte("EXISTING=keep\n"), encFile, key); err != nil {
		t.Fatal(err)
	}

	_, _, err := execCmd(t, "set", "NEW_KEY=new_value", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	stdout, _, err := execCmd(t, "get", "NEW_KEY", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("get NEW_KEY: %v", err)
	}
	if strings.TrimSpace(stdout) != "new_value" {
		t.Errorf("NEW_KEY = %q, want %q", strings.TrimSpace(stdout), "new_value")
	}

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
	if err := krypt.Encrypt([]byte("MY_KEY=old\n"), encFile, key); err != nil {
		t.Fatal(err)
	}

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
	if err := krypt.Encrypt([]byte("OTHER=val\n"), encFile, key); err != nil {
		t.Fatal(err)
	}

	_, _, err := execCmd(t, "get", "MISSING", "--key", hexKey, "--file", encFile)
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestSetInvalidFormat(t *testing.T) {
	_, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.enc")
	key, _ := hex.DecodeString(hexKey)
	if err := krypt.Encrypt([]byte("K=V\n"), encFile, key); err != nil {
		t.Fatal(err)
	}

	_, _, err := execCmd(t, "set", "NOEQUALS", "--key", hexKey, "--file", encFile)
	if err == nil {
		t.Fatal("expected error for missing = in KEY=VALUE")
	}
}

func TestSetGetWithEnvName(t *testing.T) {
	key, hexKey, dir := testSetup(t)

	encFile := filepath.Join(dir, ".env.dev.enc")
	if err := krypt.Encrypt([]byte("DEV_VAR=devval\n"), encFile, key); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := execCmd(t, "get", "DEV_VAR", "dev", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("get with env: %v", err)
	}
	if strings.TrimSpace(stdout) != "devval" {
		t.Errorf("DEV_VAR = %q", strings.TrimSpace(stdout))
	}

	_, _, err = execCmd(t, "set", "DEV_VAR=newdev", "dev", "--key", hexKey, "--file", encFile)
	if err != nil {
		t.Fatalf("set with env: %v", err)
	}

	stdout, _, _ = execCmd(t, "get", "DEV_VAR", "dev", "--key", hexKey, "--file", encFile)
	if strings.TrimSpace(stdout) != "newdev" {
		t.Errorf("DEV_VAR = %q", strings.TrimSpace(stdout))
	}
}
