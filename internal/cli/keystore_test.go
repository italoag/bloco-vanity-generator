package cli

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	bip32 "github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"

	"bloco-vgen/internal/config"
	"bloco-vgen/internal/tui"
	"bloco-vgen/internal/worker"
	"bloco-vgen/pkg/wallet"
)

func captureOutput(t *testing.T, fn func() error) error {
	t.Helper()
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		_ = rOut.Close()
		_ = wOut.Close()
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(io.Discard, rOut) }()
	go func() { defer wg.Done(); _, _ = io.Copy(io.Discard, rErr) }()
	os.Stdout, os.Stderr = wOut, wErr
	defer func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
		_ = wOut.Close()
		_ = wErr.Close()
		wg.Wait()
		_ = rOut.Close()
		_ = rErr.Close()
	}()
	return fn()
}

func runGenerateCLI(t *testing.T, extraArgs ...string) error {
	t.Helper()
	cfg := config.DefaultConfig()
	app := NewApplication(cfg, "test", "test", "test")
	root := app.GetRootCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	args := append([]string{
		"--no-tui", "--no-logging", "--engine", "cpu", "--threads", "1", "--prefix", "a",
	}, extraArgs...)
	root.SetArgs(args)
	return captureOutput(t, func() error {
		return app.ExecuteContext(context.Background())
	})
}

func listArtifacts(t *testing.T, dir string) map[string]os.FileInfo {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	files := map[string]os.FileInfo{}
	for _, e := range entries {
		if e.IsDir() {
			t.Fatalf("unexpected subdirectory in keystore dir: %s", e.Name())
		}
		if strings.HasPrefix(e.Name(), "UTC--") {
			t.Fatalf("unexpected UTC-prefixed keystore name: %s", e.Name())
		}
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("file %s has mode %o, expected 0600", e.Name(), info.Mode().Perm())
		}
		files[e.Name()] = info
	}
	return files
}

var wordPasswordPattern = regexp.MustCompile(`^[A-Z][a-z]+[0-9]?[+-_:][A-Z][a-z]+[0-9]?[+-_:][A-Z][a-z]+[0-9]?$`)

func checkWordPassword(t *testing.T, password string) {
	t.Helper()
	if !wordPasswordPattern.MatchString(password) {
		t.Fatal("keystore password does not match word format")
	}
	if len(password) < 12 {
		t.Fatal("password shorter than 12 chars")
	}
	sep := ""
	for _, r := range password {
		if strings.ContainsRune("+-_:", r) {
			if sep == "" {
				sep = string(r)
			} else if string(r) != sep {
				t.Fatal("separators differ")
			}
		}
	}
	digits := 0
	for _, r := range password {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	if digits < 1 || digits > 3 {
		t.Fatalf("expected 1..3 digits, got %d", digits)
	}
}

func addressFromFiles(files map[string]os.FileInfo) []string {
	var addresses []string
	for name := range files {
		if strings.HasSuffix(name, ".json") {
			addresses = append(addresses, strings.TrimSuffix(name, ".json"))
		}
	}
	return addresses
}

func decryptKeystoreArtifact(t *testing.T, dir, base string) *ecdsa.PrivateKey {
	t.Helper()
	jsonData, err := os.ReadFile(filepath.Join(dir, base+".json"))
	if err != nil {
		t.Fatal(err)
	}
	pwdData, err := os.ReadFile(filepath.Join(dir, base+".pwd"))
	if err != nil {
		t.Fatal(err)
	}
	checkWordPassword(t, string(pwdData))
	key, err := keystore.DecryptKey(jsonData, string(pwdData))
	if err != nil {
		t.Fatalf("geth DecryptKey failed: %v", err)
	}
	derived := ethcrypto.PubkeyToAddress(key.PrivateKey.PublicKey).Hex()
	if !strings.EqualFold(derived, base) {
		t.Fatalf("decrypted address does not match file basename")
	}
	return key.PrivateKey
}

func deriveKeyFromMnemonic(t *testing.T, mnemonic string) *ecdsa.PrivateKey {
	t.Helper()
	if !bip39.IsMnemonicValid(mnemonic) {
		t.Fatal("invalid BIP-39 mnemonic")
	}
	seed := bip39.NewSeed(mnemonic, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		t.Fatal(err)
	}
	path := []uint32{
		bip32.FirstHardenedChild + 44,
		bip32.FirstHardenedChild + 60,
		bip32.FirstHardenedChild + 0,
		0,
		0,
	}
	key := masterKey
	for _, child := range path {
		key, err = key.NewChildKey(child)
		if err != nil {
			t.Fatal(err)
		}
	}
	priv, err := ethcrypto.ToECDSA(key.Key)
	if err != nil {
		t.Fatal(err)
	}
	return priv
}

func TestGenerateCommandWritesCanonicalArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := runGenerateCLI(t, "--count", "1", "--keystore-dir", dir); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	files := listArtifacts(t, dir)
	addresses := addressFromFiles(files)
	if len(addresses) != 1 {
		t.Fatalf("expected 1 keystore json, got %d", len(addresses))
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (json+pwd), got %d", len(files))
	}
	for _, base := range addresses {
		if _, ok := files[base+".pwd"]; !ok {
			t.Fatalf("missing %s.pwd companion", base)
		}
		decryptKeystoreArtifact(t, dir, base)
	}
}

func TestGenerateCommandWithMnemonic(t *testing.T) {
	dir := t.TempDir()
	if err := runGenerateCLI(t, "--count", "1", "--with-mnemonic", "--keystore-dir", dir); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	files := listArtifacts(t, dir)
	addresses := addressFromFiles(files)
	if len(addresses) != 1 {
		t.Fatalf("expected 1 keystore json, got %d", len(addresses))
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files (json+pwd+mnemonic), got %d", len(files))
	}
	for _, base := range addresses {
		mnemonicBytes, err := os.ReadFile(filepath.Join(dir, base+".mnemonic"))
		if err != nil {
			t.Fatalf("missing %s.mnemonic companion: %v", base, err)
		}
		jsonKey := decryptKeystoreArtifact(t, dir, base)
		mnemonicKey := deriveKeyFromMnemonic(t, strings.TrimSpace(string(mnemonicBytes)))
		if jsonKey.D.Cmp(mnemonicKey.D) != 0 {
			t.Fatal("mnemonic-derived private key does not match decrypted keystore key")
		}
	}
}

func TestGenerateCommandMultipleWithMnemonic(t *testing.T) {
	dir := t.TempDir()
	if err := runGenerateCLI(t, "--count", "2", "--with-mnemonic", "--keystore-dir", dir); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	files := listArtifacts(t, dir)
	addresses := addressFromFiles(files)
	if len(addresses) != 2 {
		t.Fatalf("expected 2 keystore json files, got %d", len(addresses))
	}
	if len(files) != 6 {
		t.Fatalf("expected 6 files, got %d", len(files))
	}
	for _, base := range addresses {
		decryptKeystoreArtifact(t, dir, base)
		if _, err := os.Stat(filepath.Join(dir, base+".mnemonic")); err != nil {
			t.Fatalf("missing mnemonic file: %v", err)
		}
	}
}

func TestGenerateCommandQuietStillPersists(t *testing.T) {
	dir := t.TempDir()
	if err := runGenerateCLI(t, "--count", "1", "--with-mnemonic", "--quiet", "--keystore-dir", dir); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	files := listArtifacts(t, dir)
	if len(files) != 3 {
		t.Fatalf("expected 3 files in quiet mode, got %d", len(files))
	}
}

func TestGenerateCommandPersistenceFailureReturnsError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}

	for _, quiet := range []bool{false, true} {
		args := []string{"--count", "1", "--keystore-dir", blocker}
		if quiet {
			args = append(args, "--quiet")
		}
		err := runGenerateCLI(t, args...)
		if err == nil {
			t.Fatalf("quiet=%v: expected persistence error, got nil", quiet)
		}
	}

	err := runGenerateCLI(t, "--count", "2", "--keystore-dir", blocker)
	if err == nil {
		t.Fatal("multi-wallet: expected persistence error, got nil")
	}
}

type stubWorkerPool struct {
	stats *worker.StatsCollector
	next  func() (*wallet.GenerationResult, error)
	calls int
}

func (s *stubWorkerPool) Start() error    { return nil }
func (s *stubWorkerPool) Shutdown() error { return nil }
func (s *stubWorkerPool) GetStatsCollector() *worker.StatsCollector {
	return s.stats
}
func (s *stubWorkerPool) GenerateWalletWithContext(ctx context.Context, _ wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.calls++
	return s.next()
}

func newTestWallet(t *testing.T, withMnemonic bool) *wallet.Wallet {
	t.Helper()
	var key *ecdsa.PrivateKey
	var mnemonic string
	if withMnemonic {
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			t.Fatal(err)
		}
		mnemonic, err = bip39.NewMnemonic(entropy)
		if err != nil {
			t.Fatal(err)
		}
		key = deriveKeyFromMnemonic(t, mnemonic)
	} else {
		var err error
		key, err = ethcrypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
	}
	address := ethcrypto.PubkeyToAddress(key.PublicKey).Hex()
	return &wallet.Wallet{
		Address:    strings.TrimPrefix(address, "0x"),
		PrivateKey: hex.EncodeToString(ethcrypto.FromECDSA(key)),
		Mnemonic:   mnemonic,
		Network:    "ethereum",
		CreatedAt:  time.Now(),
	}
}

func headlessApp(t *testing.T, keystoreDir string, extraOpts ...tea.ProgramOption) *Application {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.KeyStore.Enabled = true
	cfg.KeyStore.OutputDir = keystoreDir
	cfg.TUI.Enabled = true
	app := NewApplication(cfg, "test", "test", "test")
	app.tuiProgramOptions = append([]tea.ProgramOption{
		tea.WithInput(strings.NewReader("")),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
	}, extraOpts...)
	return app
}

func stubResult(t *testing.T, withMnemonic bool) *wallet.GenerationResult {
	t.Helper()
	return &wallet.GenerationResult{
		Wallet:   newTestWallet(t, withMnemonic),
		Attempts: 1,
		Duration: time.Millisecond,
	}
}

func TestSingleWalletTUIPersistenceErrorPropagates(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	app := headlessApp(t, blocker)

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, true), nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := captureOutput(t, func() error {
		return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
	})
	if err == nil {
		t.Fatal("expected persistence error from TUI path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to persist wallet") {
		t.Fatalf("expected 'failed to persist wallet' error, got: %v", err)
	}
	if pool.calls != 1 {
		t.Fatalf("expected 1 generation call, got %d", pool.calls)
	}
}

func TestMultipleWalletsTUIPersistenceErrorPropagates(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	app := headlessApp(t, blocker)

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, true), nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := captureOutput(t, func() error {
		return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 2, tui.EngineInfo{Engine: "cpu"})
	})
	if err == nil {
		t.Fatal("expected persistence error from multi TUI path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to persist wallet") {
		t.Fatalf("expected 'failed to persist wallet' error, got: %v", err)
	}
	if pool.calls != 1 {
		t.Fatalf("expected fail-fast after 1 generation call, got %d", pool.calls)
	}
}

func TestSingleWalletTUISuccessWritesFiles(t *testing.T) {
	dir := t.TempDir()
	app := headlessApp(t, dir)

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, true), nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := captureOutput(t, func() error {
		return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files := listArtifacts(t, dir)
	addresses := addressFromFiles(files)
	if len(addresses) != 1 || len(files) != 3 {
		t.Fatalf("expected 1 keystore json and 3 files, got %v", files)
	}
	for _, base := range addresses {
		mnemonicBytes, err := os.ReadFile(filepath.Join(dir, base+".mnemonic"))
		if err != nil {
			t.Fatal(err)
		}
		jsonKey := decryptKeystoreArtifact(t, dir, base)
		mnemonicKey := deriveKeyFromMnemonic(t, strings.TrimSpace(string(mnemonicBytes)))
		if jsonKey.D.Cmp(mnemonicKey.D) != 0 {
			t.Fatal("mnemonic-derived private key does not match decrypted keystore key")
		}
	}
}

func TestMultipleWalletsTUISuccessWritesFiles(t *testing.T) {
	dir := t.TempDir()
	app := headlessApp(t, dir)

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, false), nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := captureOutput(t, func() error {
		return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 2, tui.EngineInfo{Engine: "cpu"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files := listArtifacts(t, dir)
	if len(addressFromFiles(files)) != 2 || len(files) != 4 {
		t.Fatalf("expected 2 keystore json and 4 files, got %v", files)
	}
}

func TestSingleWalletTUIFallbackOnProgramFailure(t *testing.T) {
	dir := t.TempDir()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	app := headlessApp(t, dir, tea.WithContext(canceled))

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, false), nil },
	}

	ctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	err := captureOutput(t, func() error {
		return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
	})
	if err != nil {
		t.Fatalf("expected text fallback success, got: %v", err)
	}
	if pool.calls != 1 {
		t.Fatalf("expected exactly 1 generation call, got %d", pool.calls)
	}
	files := listArtifacts(t, dir)
	if len(addressFromFiles(files)) != 1 || len(files) != 2 {
		t.Fatalf("expected exactly one artifact set, got %v", files)
	}
}

func TestMultipleWalletsTUIFallbackOnProgramFailure(t *testing.T) {
	dir := t.TempDir()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	app := headlessApp(t, dir, tea.WithContext(canceled))

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, false), nil },
	}

	ctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	err := captureOutput(t, func() error {
		return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 2, tui.EngineInfo{Engine: "cpu"})
	})
	if err != nil {
		t.Fatalf("expected text fallback success, got: %v", err)
	}
	if pool.calls != 2 {
		t.Fatalf("expected exactly 2 generation calls, got %d", pool.calls)
	}
	files := listArtifacts(t, dir)
	if len(addressFromFiles(files)) != 2 || len(files) != 4 {
		t.Fatalf("expected exactly two artifact sets, got %v", files)
	}
}

func TestGenerateAndSaveKeystoreChecksumFilename(t *testing.T) {
	key, err := ethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	checksumAddress := ethcrypto.PubkeyToAddress(key.PublicKey).Hex()

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.KeyStore.Enabled = true
	cfg.KeyStore.OutputDir = dir
	app := NewApplication(cfg, "test", "test", "test")

	w := &wallet.Wallet{
		Address:    strings.TrimPrefix(checksumAddress, "0x"),
		PrivateKey: strings.Repeat("0", 63) + "1",
		Network:    "ethereum",
		CreatedAt:  time.Now(),
	}

	if err := captureOutput(t, func() error {
		return app.generateAndSaveKeystoreWithVerbose(w, false)
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files := listArtifacts(t, dir)
	if _, ok := files[checksumAddress+".json"]; !ok {
		t.Fatalf("expected checksum-cased keystore file, got %v", files)
	}
	if _, ok := files[checksumAddress+".pwd"]; !ok {
		t.Fatalf("expected checksum-cased password file, got %v", files)
	}
}

func TestGenerateAndSaveKeystoreMnemonicCollision(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.KeyStore.Enabled = true
	cfg.KeyStore.OutputDir = dir
	app := NewApplication(cfg, "test", "test", "test")

	w := newTestWallet(t, true)
	checksumAddress := "0x" + w.Address

	for _, asDir := range []bool{false, true} {
		sub := t.TempDir()
		cfg.KeyStore.OutputDir = sub
		app = NewApplication(cfg, "test", "test", "test")

		mnemonicPath := filepath.Join(sub, checksumAddress+".mnemonic")
		if asDir {
			if err := os.Mkdir(mnemonicPath, 0700); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(mnemonicPath, []byte("existing"), 0600); err != nil {
				t.Fatal(err)
			}
		}

		err := captureOutput(t, func() error {
			return app.generateAndSaveKeystoreWithVerbose(w, false)
		})
		if err == nil {
			t.Fatalf("asDir=%v: expected collision error, got nil", asDir)
		}
		if _, err := os.Lstat(filepath.Join(sub, checksumAddress+".json")); err == nil {
			t.Fatalf("asDir=%v: json artifact was created despite mnemonic collision", asDir)
		}
		if _, err := os.Lstat(filepath.Join(sub, checksumAddress+".pwd")); err == nil {
			t.Fatalf("asDir=%v: pwd artifact was created despite mnemonic collision", asDir)
		}
		if !asDir {
			content, err := os.ReadFile(mnemonicPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "existing" {
				t.Fatal("existing mnemonic file was overwritten")
			}
		}
	}
}
