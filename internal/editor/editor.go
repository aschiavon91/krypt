package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// Edit writes content to a secure temp file, opens $EDITOR, and returns the edited content.
func Edit(content []byte) ([]byte, error) {
	dir := tempDir()

	// Set restrictive umask so the temp file is created 0600 from the start,
	// eliminating the TOCTOU window between CreateTemp and Chmod.
	oldUmask := syscall.Umask(0177)
	f, err := os.CreateTemp(dir, "krypt-*")
	syscall.Umask(oldUmask)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)

	if _, err := f.Write(content); err != nil {
		f.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	f.Close()

	editorPath := findEditor()
	if editorPath == "" {
		return nil, fmt.Errorf("no editor found: set $EDITOR or $VISUAL")
	}

	cmd := exec.Command(editorPath, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("editor exited with status %d, changes discarded", exitErr.ExitCode())
		}
		return nil, fmt.Errorf("run editor: %w", err)
	}

	return os.ReadFile(tmpPath)
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
