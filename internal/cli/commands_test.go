package cli

import (
	"strings"
	"testing"

	"bloco-eth/internal/config"
)

func TestGetGenerationCriteriaReadsCaseSensitiveFlag(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := app.rootCmd

	if err := cmd.ParseFlags([]string{"--prefix", "DEAD", "--checksum", "--case-sensitive"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !criteria.CaseSensitive {
		t.Fatalf("expected CaseSensitive to be true")
	}

	if !criteria.IsChecksum {
		t.Fatalf("expected IsChecksum to be true")
	}
}

func TestGetGenerationCriteriaRejectsCaseSensitiveWithoutChecksum(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := app.rootCmd

	if err := cmd.ParseFlags([]string{"--prefix", "DEAD", "--case-sensitive"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := app.getGenerationCriteria(cmd)
	if err == nil {
		t.Fatalf("expected validation error")
	}

	if !strings.Contains(err.Error(), "case-sensitive matching requires checksum validation") {
		t.Fatalf("unexpected error: %v", err)
	}
}
