// Package editor provides a secure temp-file editor integration for editing secrets.
package editor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Edit writes content to a secure temp file, opens $EDITOR, and returns the edited content.
func Edit(content []byte) ([]byte, error) {
	dir := tempDir()

	// Set restrictive umask so the temp file is created 0600 from the start,
	// eliminating the TOCTOU window between CreateTemp and Chmod.
	restoreUmask := applyUmask(0o177)
	f, err := os.CreateTemp(dir, "krypt-*")
	restoreUmask()
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of temp file

	if _, err := f.Write(content); err != nil {
		f.Close() //nolint:errcheck,gosec // already returning a write error, close error not actionable
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	editorPath := findEditor()
	if editorPath == "" {
		return nil, errors.New("no editor found: set $EDITOR or $VISUAL")
	}

	cmd := exec.Command(editorPath, tmpPath) //nolint:gosec // editorPath from $VISUAL/$EDITOR, tmpPath is our own temp file
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("editor exited with status %d, changes discarded", exitErr.ExitCode())
		}
		return nil, fmt.Errorf("run editor: %w", err)
	}

	return os.ReadFile(tmpPath) //nolint:gosec // tmpPath is our own temp file created above
}

func tempDir() string {
	if runtime.GOOS == "linux" {
		if info, err := os.Stat("/dev/shm"); err == nil && info.IsDir() {
			return "/dev/shm"
		}
	}
	return os.TempDir()
}

func findEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if path, err := exec.LookPath("vi"); err == nil {
		return path
	}
	return ""
}
