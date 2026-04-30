package editor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTempDirPrefersDevShm(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	dir := tempDir()
	if _, err := os.Stat("/dev/shm"); err == nil {
		if dir != "/dev/shm" {
			t.Errorf("expected /dev/shm, got %s", dir)
		}
	}
}

func TestFindEditorVisual(t *testing.T) {
	t.Setenv("VISUAL", "/usr/bin/nano")
	t.Setenv("EDITOR", "/usr/bin/vim")

	editor := findEditor()
	if editor != "/usr/bin/nano" {
		t.Errorf("expected VISUAL, got %s", editor)
	}
}

func TestFindEditorFallback(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "/usr/bin/vim")

	editor := findEditor()
	if editor != "/usr/bin/vim" {
		t.Errorf("expected EDITOR, got %s", editor)
	}
}

func TestEditWithMockEditor(t *testing.T) {
	// Create a mock editor script that appends a line
	dir := t.TempDir()
	mockEditor := filepath.Join(dir, "mock-editor.sh")
	err := os.WriteFile(mockEditor, []byte("#!/bin/sh\necho 'NEW_KEY=appended' >> \"$1\"\n"), 0o755) //nolint:gosec // mock editor script requires executable permissions
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("VISUAL", mockEditor)

	original := []byte("EXISTING=value\n")
	result, err := Edit(original)
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}

	expected := "EXISTING=value\nNEW_KEY=appended\n"
	if string(result) != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestEditEditorNotFound(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	// Also ensure 'vi' is not used by overriding PATH
	t.Setenv("PATH", t.TempDir())

	_, err := Edit([]byte("content"))
	if err == nil {
		t.Fatal("expected error when no editor found")
	}
}

func TestEditEditorExitNonZero(t *testing.T) {
	dir := t.TempDir()
	mockEditor := filepath.Join(dir, "fail-editor.sh")
	err := os.WriteFile(mockEditor, []byte("#!/bin/sh\nexit 1\n"), 0o755) //nolint:gosec // mock editor script requires executable permissions
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("VISUAL", mockEditor)

	_, err = Edit([]byte("content"))
	if err == nil {
		t.Fatal("expected error when editor exits non-zero")
	}
}
