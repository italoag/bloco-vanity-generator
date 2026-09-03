package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"bloco-vgen/internal/crypto"
	"bloco-vgen/pkg/wallet"
)

// Output formats accepted by --format on the generation command.
const (
	outputFormatText = "text"
	outputFormatJSON = "json"
	outputFormatCSV  = "csv"
)

// outputFormats lists the accepted --format values, for validation and for
// error messages.
var outputFormats = []string{outputFormatText, outputFormatJSON, outputFormatCSV}

// outputConfig holds the resolved --output/--format settings for one run.
type outputConfig struct {
	// Path is the destination file. Empty means stdout.
	Path string
	// Format is one of outputFormats.
	Format string
}

// WritesFile reports whether the results go to a file rather than the terminal.
func (o outputConfig) WritesFile() bool {
	return o.Path != ""
}

// resolveOutputConfig validates the --output and --format flags.
func resolveOutputConfig(path, format string) (outputConfig, error) {
	if format == "" {
		format = outputFormatText
	}

	normalized := strings.ToLower(strings.TrimSpace(format))
	valid := false
	for _, candidate := range outputFormats {
		if normalized == candidate {
			valid = true
			break
		}
	}
	if !valid {
		return outputConfig{}, fmt.Errorf("unsupported output format %q (supported: %s)",
			format, strings.Join(outputFormats, ", "))
	}

	return outputConfig{Path: strings.TrimSpace(path), Format: normalized}, nil
}

// resultCollector accumulates generation results so they can be serialized once
// the run finishes. Results arrive from worker goroutines in the TUI paths, so
// it is guarded by a mutex.
type resultCollector struct {
	mu      sync.Mutex
	results []*wallet.GenerationResult
}

// Record stores a successfully generated result.
func (c *resultCollector) Record(results ...*wallet.GenerationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, result := range results {
		if result != nil && result.Wallet != nil {
			c.results = append(c.results, result)
		}
	}
}

// Results returns the collected results ordered by worker index so repeated
// runs of the same command produce a stable file.
func (c *resultCollector) Results() []*wallet.GenerationResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	ordered := make([]*wallet.GenerationResult, len(c.results))
	copy(ordered, c.results)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Wallet.CreatedAt.Before(ordered[j].Wallet.CreatedAt)
	})
	return ordered
}

// Reset clears the collected results.
func (c *resultCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = nil
}

// secretCollector records the keystore password generated for each address.
// The password is no longer written next to the keystore, so it has to reach
// the user some other way or the keystore becomes unopenable.
type secretCollector struct {
	mu        sync.Mutex
	passwords map[string]string
}

// Record stores the keystore password for an address.
func (c *secretCollector) Record(address, password string) {
	if address == "" || password == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.passwords == nil {
		c.passwords = make(map[string]string)
	}
	c.passwords[address] = password
}

// Passwords returns a copy of the recorded passwords.
func (c *secretCollector) Passwords() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	copied := make(map[string]string, len(c.passwords))
	for address, password := range c.passwords {
		copied[address] = password
	}
	return copied
}

// Reset clears the recorded passwords.
func (c *secretCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.passwords = nil
}

// walletOutputRecord is the serialized shape of one generated wallet. It is a
// dedicated type rather than wallet.Wallet so the file format is stable and
// does not silently gain fields when the internal struct changes.
type walletOutputRecord struct {
	Index      int    `json:"index"`
	Network    string `json:"network"`
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	Mnemonic   string `json:"mnemonic,omitempty"`
	Attempts   int64  `json:"attempts"`
	DurationMS int64  `json:"duration_ms"`
	CreatedAt  string `json:"created_at"`
	// KeystorePassword is the password that opens the generated keystore. It
	// is carried here because it is deliberately not written to disk next to
	// the keystore itself.
	KeystorePassword string `json:"keystore_password,omitempty"`
}

func newWalletOutputRecords(results []*wallet.GenerationResult, passwords map[string]string) []walletOutputRecord {
	records := make([]walletOutputRecord, 0, len(results))
	for i, result := range results {
		network := result.Wallet.Network
		if network == "" {
			network = "ethereum"
		}
		records = append(records, walletOutputRecord{
			Index:      i + 1,
			Network:    network,
			Address:    result.Wallet.Address,
			PrivateKey: result.Wallet.PrivateKey,
			Mnemonic:   result.Wallet.Mnemonic,
			Attempts:   result.Attempts,
			DurationMS: result.Duration.Milliseconds(),
			CreatedAt:  result.Wallet.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),

			KeystorePassword: passwords[result.Wallet.Address],
		})
	}
	return records
}

// renderResults serializes the results in the configured format.
func renderResults(results []*wallet.GenerationResult, format string, passwords map[string]string) ([]byte, error) {
	records := newWalletOutputRecords(results, passwords)

	switch format {
	case outputFormatJSON:
		encoded, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to encode results as JSON: %w", err)
		}
		return append(encoded, '\n'), nil

	case outputFormatCSV:
		var builder strings.Builder
		writer := csv.NewWriter(&builder)
		header := []string{"index", "network", "address", "private_key", "mnemonic", "attempts", "duration_ms", "created_at", "keystore_password"}
		if err := writer.Write(header); err != nil {
			return nil, fmt.Errorf("failed to encode CSV header: %w", err)
		}
		for _, record := range records {
			row := []string{
				strconv.Itoa(record.Index),
				record.Network,
				record.Address,
				record.PrivateKey,
				record.Mnemonic,
				strconv.FormatInt(record.Attempts, 10),
				strconv.FormatInt(record.DurationMS, 10),
				record.CreatedAt,
				record.KeystorePassword,
			}
			if err := writer.Write(row); err != nil {
				return nil, fmt.Errorf("failed to encode CSV row: %w", err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, fmt.Errorf("failed to encode results as CSV: %w", err)
		}
		return []byte(builder.String()), nil

	case outputFormatText:
		var builder strings.Builder
		for _, record := range records {
			fmt.Fprintf(&builder, "Wallet %d:\n", record.Index)
			fmt.Fprintf(&builder, "  Network: %s\n", record.Network)
			fmt.Fprintf(&builder, "  Address: %s\n", record.Address)
			fmt.Fprintf(&builder, "  Private Key: %s\n", record.PrivateKey)
			if record.Mnemonic != "" {
				fmt.Fprintf(&builder, "  Mnemonic: %s\n", record.Mnemonic)
			}
			fmt.Fprintf(&builder, "  Attempts: %d\n", record.Attempts)
			fmt.Fprintf(&builder, "  Duration: %dms\n", record.DurationMS)
			if record.KeystorePassword != "" {
				fmt.Fprintf(&builder, "  Keystore Password: %s\n", record.KeystorePassword)
			}
			fmt.Fprintf(&builder, "  Created At: %s\n\n", record.CreatedAt)
		}
		return []byte(builder.String()), nil

	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}
}

// writeResultsFile writes the rendered results to disk. The file carries
// private keys, so it is created with the same owner-only permission as every
// other artifact this tool writes, and the permission is re-applied when the
// file already exists.
func writeResultsFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, crypto.DefaultKeyStoreFilePerm)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	// O_CREATE only applies the mode when the file is created; an existing file
	// keeps its own, possibly world-readable, permission.
	if err := file.Chmod(crypto.DefaultKeyStoreFilePerm); err != nil {
		return fmt.Errorf("failed to restrict permissions on output file %s: %w", path, err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", path, err)
	}

	return file.Sync()
}
