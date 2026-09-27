package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"bloco-vgen/internal/backup"
	"bloco-vgen/internal/config"
)

func integrationEnv(home string) []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"NO_COLOR=1",
	}
}

func runTool(t *testing.T, home, name string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = home
	cmd.Env = integrationEnv(home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v", name, args, err)
	}
	return out
}

func TestFnoxCLIIntegration(t *testing.T) {
	if os.Getenv("BLOCO_FNOX_INTEGRATION") != "1" {
		t.Skip("set BLOCO_FNOX_INTEGRATION=1 to run")
	}
	if _, err := exec.LookPath("fnox"); err != nil {
		t.Fatal("fnox not found on PATH")
	}
	if _, err := exec.LookPath("age-keygen"); err != nil {
		t.Fatal("age-keygen not found on PATH")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	if out := runTool(t, home, "fnox", "--version"); !strings.Contains(string(out), "1.35.2") {
		t.Fatalf("unsupported fnox version: %s", out)
	}
	if out := runTool(t, home, "age-keygen", "--version"); !strings.Contains(string(out), "1.2.1") {
		t.Fatalf("unsupported age-keygen version: %s", out)
	}

	identity := filepath.Join(home, "identity.txt")
	runTool(t, home, "age-keygen", "-o", identity)
	recipient := strings.TrimSpace(string(runTool(t, home, "age-keygen", "-y", identity)))
	if !strings.HasPrefix(recipient, "age1") {
		t.Fatalf("unexpected recipient: %s", recipient)
	}

	newCLIApp := func() (*Application, *strings.Builder) {
		cfg := config.DefaultConfig()
		cfg.TUI.Enabled = false
		cfg.Logging.Enabled = false
		cfg.Backup.OutputDir = filepath.Join(t.TempDir(), "backups")
		app := NewApplication(cfg, "test", "test", "test")
		root := app.GetRootCommand()
		var buf strings.Builder
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SilenceErrors = true
		root.SilenceUsage = true
		return app, &buf
	}

	listArtifacts := func(dir string) []string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read backup dir: %v", err)
		}
		var names []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".fnox.toml") {
				names = append(names, e.Name())
			}
		}
		return names
	}

	t.Run("doctor", func(t *testing.T) {
		app, buf := newCLIApp()
		app.GetRootCommand().SetArgs([]string{
			"backup", "doctor",
			"--age-recipient", recipient,
			"--age-identity", identity,
		})
		if err := app.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("backup doctor failed: %v", err)
		}
		if !strings.Contains(buf.String(), "Fnox backup provider verified") {
			t.Fatalf("missing verification output: %q", buf.String())
		}
	})

	generate := func(t *testing.T, count int, kdf string, extra ...string) ([]string, string) {
		t.Helper()
		backupDir := filepath.Join(t.TempDir(), "backups")
		app, buf := newCLIApp()
		args := []string{
			"--no-tui", "--no-logging", "--engine", "cpu", "--threads", "1",
			"--backup-store", "fnox",
			"--backup-dir", backupDir,
			"--fnox-bin", "fnox",
			"--age-recipient", recipient,
			"--age-identity", identity,
			"--count", strconv.Itoa(count),
		}
		if kdf != "" {
			args = append(args, "--keystore-kdf", kdf)
		}
		args = append(args, extra...)
		app.GetRootCommand().SetArgs(args)
		stdout, stderr, err := captureStdStreams(t, func() error {
			return app.ExecuteContext(context.Background())
		})
		combined := stdout + stderr + buf.String()
		if err != nil {
			t.Fatalf("generation failed: %v\ncombined:%s", err, combined)
		}
		if !strings.Contains(combined, "Encrypted backup confirmed") {
			t.Fatalf("missing confirmation:\n%s", combined)
		}
		if strings.Contains(combined, "Private Key:") || strings.Contains(combined, "Mnemonic:") {
			t.Fatal("secrets printed to output")
		}
		names := listArtifacts(backupDir)
		if len(names) != count {
			t.Fatalf("expected %d artifacts, got %v", count, names)
		}
		var paths []string
		for _, name := range names {
			path := filepath.Join(backupDir, name)
			info, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Fatalf("artifact mode %v, want 0600", info.Mode().Perm())
			}
			paths = append(paths, path)
		}
		return paths, combined
	}

	loadBundle := func(t *testing.T, path string) *backup.Bundle {
		t.Helper()
		store, err := backup.NewStore(backup.Options{
			Binary:       "fnox",
			OutputDir:    filepath.Dir(path),
			IdentityFile: identity,
			Timeout:      30 * time.Second,
		})
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		b, err := store.Load(ctx, path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		return b
	}

	verify := func(t *testing.T, path string) string {
		t.Helper()
		app, buf := newCLIApp()
		app.GetRootCommand().SetArgs([]string{
			"backup", "verify", path,
			"--age-identity", identity,
			"--fnox-bin", "fnox",
		})
		if err := app.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("backup verify failed: %v\n%s", err, buf.String())
		}
		out := buf.String()
		if !strings.Contains(out, "Backup verified") {
			t.Fatalf("missing verification: %q", out)
		}
		return out
	}

	export := func(t *testing.T, path string) (string, string) {
		t.Helper()
		outDir := filepath.Join(t.TempDir(), "exported")
		app, buf := newCLIApp()
		app.GetRootCommand().SetArgs([]string{
			"backup", "export", path,
			"--output-dir", outDir,
			"--allow-plaintext",
			"--age-identity", identity,
			"--fnox-bin", "fnox",
		})
		if err := app.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("backup export failed: %v\n%s", err, buf.String())
		}
		if !strings.Contains(buf.String(), "plaintext") {
			t.Fatalf("missing plaintext warning: %q", buf.String())
		}
		return outDir, buf.String()
	}

	assertCleanOutput := func(t *testing.T, where, output string, b *backup.Bundle) {
		t.Helper()
		if strings.Contains(output, b.PrivateKey) {
			t.Fatalf("private key leaked in %s", where)
		}
		if b.Mnemonic != "" && strings.Contains(output, b.Mnemonic) {
			t.Fatalf("mnemonic leaked in %s", where)
		}
		if b.KeystorePassword != "" && strings.Contains(output, b.KeystorePassword) {
			t.Fatalf("keystore password leaked in %s", where)
		}
	}

	t.Run("ethereum", func(t *testing.T) {
		paths, genOut := generate(t, 1, "pbkdf2", "--network", "ethereum", "--with-mnemonic")
		b := loadBundle(t, paths[0])
		if b.Network != "ethereum" || !strings.HasPrefix(b.Address, "0x") {
			t.Fatalf("unexpected bundle: %s %s", b.Network, b.Address)
		}
		assertCleanOutput(t, "generation", genOut, b)
		assertCleanOutput(t, "verify", verify(t, paths[0]), b)
		exportDir, exportOut := export(t, paths[0])
		assertCleanOutput(t, "export", exportOut, b)
		for _, name := range []string{"wallet-backup.json", b.Address + ".json", b.Address + ".pwd", b.Address + ".mnemonic"} {
			if _, err := os.Lstat(filepath.Join(exportDir, name)); err != nil {
				t.Fatalf("missing export artifact %s: %v", name, err)
			}
		}
		raw, err := os.ReadFile(filepath.Join(exportDir, "wallet-backup.json"))
		if err != nil {
			t.Fatal(err)
		}
		var decoded backup.Bundle
		if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&decoded); err != nil {
			t.Fatalf("wallet-backup.json invalid: %v", err)
		}
	})

	t.Run("ethereum_count2", func(t *testing.T) {
		paths, genOut := generate(t, 2, "pbkdf2", "--network", "ethereum")
		seen := map[string]bool{}
		for _, path := range paths {
			b := loadBundle(t, path)
			if b.Network != "ethereum" {
				t.Fatalf("unexpected network %s", b.Network)
			}
			if seen[b.Address] {
				t.Fatal("duplicate wallet artifact")
			}
			seen[b.Address] = true
			assertCleanOutput(t, "generation", genOut, b)
			assertCleanOutput(t, "verify", verify(t, path), b)
		}
		entries, err := os.ReadDir(filepath.Dir(paths[0]))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".fnox.toml") {
				t.Fatalf("unexpected plaintext sidecar: %s", e.Name())
			}
		}
	})

	t.Run("solana", func(t *testing.T) {
		paths, genOut := generate(t, 1, "pbkdf2", "--network", "solana")
		b := loadBundle(t, paths[0])
		if b.Network != "solana" {
			t.Fatalf("unexpected network %s", b.Network)
		}
		assertCleanOutput(t, "generation", genOut, b)
		assertCleanOutput(t, "verify", verify(t, paths[0]), b)
		exportDir, exportOut := export(t, paths[0])
		assertCleanOutput(t, "export", exportOut, b)
		raw, err := os.ReadFile(filepath.Join(exportDir, b.Address+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var keyArray []int
		if err := json.Unmarshal(raw, &keyArray); err != nil || len(keyArray) != 64 {
			t.Fatalf("solana export not a 64-element array: %v", err)
		}
	})

	t.Run("bitcoin", func(t *testing.T) {
		paths, genOut := generate(t, 1, "pbkdf2", "--network", "bitcoin")
		b := loadBundle(t, paths[0])
		if b.Network != "bitcoin" || b.MnemonicRole != "unrelated" {
			t.Fatalf("unexpected bundle: %s role=%s", b.Network, b.MnemonicRole)
		}
		assertCleanOutput(t, "generation", genOut, b)
		out := verify(t, paths[0])
		assertCleanOutput(t, "verify", out, b)
		if !strings.Contains(out, "unrelated") {
			t.Fatalf("missing unrelated warning: %q", out)
		}
		exportDir, exportOut := export(t, paths[0])
		assertCleanOutput(t, "export", exportOut, b)
		if _, err := os.Lstat(filepath.Join(exportDir, b.Address+".key")); err != nil {
			t.Fatalf("missing bitcoin key export: %v", err)
		}
	})

	t.Run("ethereum_default_scrypt", func(t *testing.T) {
		paths, genOut := generate(t, 1, "", "--network", "ethereum")
		b := loadBundle(t, paths[0])
		if b.Keystore == nil {
			t.Fatal("ethereum bundle missing keystore")
		}
		if b.Keystore.Crypto.KDF != "scrypt" {
			t.Fatalf("expected default scrypt KDF, got %s", b.Keystore.Crypto.KDF)
		}
		assertCleanOutput(t, "generation", genOut, b)
		assertCleanOutput(t, "verify", verify(t, paths[0]), b)
		_, exportOut := export(t, paths[0])
		assertCleanOutput(t, "export", exportOut, b)
	})

	t.Run("solana_with_mnemonic", func(t *testing.T) {
		paths, genOut := generate(t, 1, "pbkdf2", "--network", "solana", "--with-mnemonic")
		b := loadBundle(t, paths[0])
		if b.Network != "solana" {
			t.Fatalf("unexpected network %s", b.Network)
		}
		if b.Mnemonic != "" {
			t.Fatal("solana bundle must not carry a mnemonic")
		}
		if b.Derivation != nil {
			t.Fatal("solana bundle must not carry derivation metadata")
		}
		if b.KeyOrigin != "random" {
			t.Fatalf("expected random key origin, got %s", b.KeyOrigin)
		}
		assertCleanOutput(t, "generation", genOut, b)
		assertCleanOutput(t, "verify", verify(t, paths[0]), b)
	})
}
