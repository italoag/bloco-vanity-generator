package cli

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bloco-vgen/internal/crypto"
	"bloco-vgen/pkg/wallet"
)

// Regression tests for GitHub issue #21: --output and --format were registered
// but never read on the generation path, so private keys went to the terminal
// while the user believed they had been written to a file.

func sampleResults() []*wallet.GenerationResult {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return []*wallet.GenerationResult{
		{
			Wallet: &wallet.Wallet{
				Address:    "0xabc0000000000000000000000000000000000001",
				PrivateKey: "1111111111111111111111111111111111111111111111111111111111111111",
				Network:    "ethereum",
				CreatedAt:  created,
			},
			Attempts: 42,
			Duration: 250 * time.Millisecond,
		},
		{
			Wallet: &wallet.Wallet{
				Address:    "0xabc0000000000000000000000000000000000002",
				PrivateKey: "2222222222222222222222222222222222222222222222222222222222222222",
				Network:    "ethereum",
				CreatedAt:  created.Add(time.Second),
			},
			Attempts: 7,
			Duration: 100 * time.Millisecond,
		},
	}
}

func TestResolveOutputConfig(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		format     string
		wantFormat string
		wantError  bool
	}{
		{"default format", "", "", outputFormatText, false},
		{"json", "out.json", "json", outputFormatJSON, false},
		{"csv is case-insensitive", "out.csv", "CSV", outputFormatCSV, false},
		{"unsupported format", "", "yaml", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolveOutputConfig(tt.path, tt.format)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected format %q to be rejected", tt.format)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.Format != tt.wantFormat {
				t.Errorf("format: got %q, want %q", resolved.Format, tt.wantFormat)
			}
			if resolved.WritesFile() != (tt.path != "") {
				t.Errorf("WritesFile: got %v for path %q", resolved.WritesFile(), tt.path)
			}
		})
	}
}

func TestRenderResultsJSON(t *testing.T) {
	results := sampleResults()
	passwords := map[string]string{results[0].Wallet.Address: "s3cret-pass"}

	rendered, err := renderResults(results, outputFormatJSON, passwords)
	if err != nil {
		t.Fatalf("renderResults failed: %v", err)
	}

	var decoded []walletOutputRecord
	if err := json.Unmarshal(rendered, &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 records, got %d", len(decoded))
	}
	if decoded[0].PrivateKey != results[0].Wallet.PrivateKey {
		t.Error("private key missing from the JSON output")
	}
	if decoded[0].KeystorePassword != "s3cret-pass" {
		t.Error("keystore password missing from the JSON output")
	}
	if decoded[1].KeystorePassword != "" {
		t.Error("a wallet without a recorded password must not carry one")
	}
}

func TestRenderResultsCSV(t *testing.T) {
	rendered, err := renderResults(sampleResults(), outputFormatCSV, nil)
	if err != nil {
		t.Fatalf("renderResults failed: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(string(rendered))).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v", err)
	}
	if len(rows) != 3 { // header + 2 wallets
		t.Fatalf("expected 3 CSV rows, got %d", len(rows))
	}
	if rows[0][0] != "index" || rows[0][3] != "private_key" {
		t.Errorf("unexpected CSV header: %v", rows[0])
	}
}

// TestWriteResultsFileIsOwnerOnly is the acceptance criterion that the output
// file carrying private keys must not be readable by other local users.
func TestWriteResultsFileIsOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")

	if err := writeResultsFile(path, []byte("{}")); err != nil {
		t.Fatalf("writeResultsFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("output file was not created: %v", err)
	}
	if got := info.Mode().Perm(); got != crypto.DefaultKeyStoreFilePerm {
		t.Errorf("output file mode: got %04o, want %04o", got, crypto.DefaultKeyStoreFilePerm)
	}
}

// TestWriteResultsFileTightensExistingFile covers the O_CREATE gap: a file that
// already exists keeps its own, possibly world-readable, mode.
func TestWriteResultsFileTightensExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")

	if err := os.WriteFile(path, []byte("stale"), 0o644); err != nil {
		t.Fatalf("failed to seed the output file: %v", err)
	}
	if err := writeResultsFile(path, []byte("{}")); err != nil {
		t.Fatalf("writeResultsFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}
	if got := info.Mode().Perm(); got != crypto.DefaultKeyStoreFilePerm {
		t.Errorf("pre-existing output file was not tightened: got %04o, want %04o",
			got, crypto.DefaultKeyStoreFilePerm)
	}
}

func TestResultCollectorIsOrderedAndResettable(t *testing.T) {
	var collector resultCollector

	results := sampleResults()
	collector.Record(results[1], results[0], nil)

	collected := collector.Results()
	if len(collected) != 2 {
		t.Fatalf("expected 2 collected results, got %d", len(collected))
	}
	if collected[0].Wallet.Address != results[0].Wallet.Address {
		t.Error("results should be ordered by creation time")
	}

	collector.Reset()
	if len(collector.Results()) != 0 {
		t.Error("Reset must clear the collected results")
	}
}
