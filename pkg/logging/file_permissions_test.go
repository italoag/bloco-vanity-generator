package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for GitHub issue #19: log files used to be created 0644,
// readable by any local user, while every other artifact was 0600.

func TestLogFileIsOwnerOnlyOnCreate(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "bloco.log")

	logger, err := NewSecureLogger(&LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logPath,
		MaxFileSize: 10 * 1024 * 1024,
		MaxFiles:    5,
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	if err := logger.Info("test entry"); err != nil {
		t.Fatalf("failed to write log entry: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("failed to close logger: %v", err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("log file was not created: %v", err)
	}
	if got := info.Mode().Perm(); got != LogFilePerm {
		t.Errorf("log file mode: got %04o, want %04o", got, LogFilePerm)
	}
}

// TestLogFileIsTightenedWhenItAlreadyExists covers the case O_CREATE does not:
// an existing file keeps its own mode, so the permission has to be re-applied.
func TestLogFileIsTightenedWhenItAlreadyExists(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "preexisting.log")

	if err := os.WriteFile(logPath, []byte("old content\n"), 0o644); err != nil {
		t.Fatalf("failed to seed the log file: %v", err)
	}

	logger, err := NewSecureLogger(&LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logPath,
		MaxFileSize: 10 * 1024 * 1024,
		MaxFiles:    5,
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("failed to close logger: %v", err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("failed to stat log file: %v", err)
	}
	if got := info.Mode().Perm(); got != LogFilePerm {
		t.Errorf("pre-existing log file was not tightened: got %04o, want %04o", got, LogFilePerm)
	}
}

// TestRotatedLogFileIsOwnerOnly checks the second open site, which is easy to
// miss: rotation recreates the file.
func TestRotatedLogFileIsOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rotating.log")

	logger, err := NewSecureLogger(&LogConfig{
		Enabled:     true,
		Level:       INFO,
		Format:      TEXT,
		OutputFile:  logPath,
		MaxFileSize: 256, // force rotation quickly
		MaxFiles:    2,
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	for i := 0; i < 50; i++ {
		if err := logger.Info(strings.Repeat("x", 64)); err != nil {
			t.Fatalf("failed to write log entry %d: %v", i, err)
		}
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("failed to close logger: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read log directory: %v", err)
	}

	checked := 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("failed to stat %s: %v", entry.Name(), err)
		}
		if got := info.Mode().Perm(); got != LogFilePerm {
			t.Errorf("log file %s mode: got %04o, want %04o", entry.Name(), got, LogFilePerm)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("expected at least one log file to inspect")
	}
}
