package cli

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
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
		if !jsonKey.Equal(mnemonicKey) {
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
	cfg.KeyStore.KDFAlgorithm = "pbkdf2"
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

func quitAfterWalletResults(expected int, capture func(tui.WalletResultMsg)) func(tea.Model, tea.Msg) tea.Msg {
	received := 0
	return func(_ tea.Model, msg tea.Msg) tea.Msg {
		switch m := msg.(type) {
		case tui.WalletResultMsg:
			received++
			if capture != nil {
				capture(m)
			}
		case tui.ProgressMsg:
			if received >= expected && m.IsComplete {
				return tea.QuitMsg{}
			}
		}
		return msg
	}
}

func TestQuitAfterWalletResults(t *testing.T) {
	walletMsg := func(index int) tui.WalletResultMsg {
		return tui.WalletResultMsg{Result: tui.WalletResult{
			Index:      index,
			Address:    "0xabc",
			PrivateKey: "[encrypted backup]",
			Attempts:   1,
			Error:      "",
		}}
	}
	complete := tui.ProgressMsg{IsComplete: true}
	incomplete := tui.ProgressMsg{IsComplete: false, CompletedWallets: 2, TotalWallets: 2}

	var captured []tui.WalletResultMsg
	filter := quitAfterWalletResults(2, func(m tui.WalletResultMsg) { captured = append(captured, m) })

	if got := filter(nil, complete); got != complete {
		t.Fatalf("completion before results must not quit, got %T", got)
	}
	w1 := walletMsg(1)
	if got := filter(nil, w1); got != w1 {
		t.Fatalf("wallet result must pass through, got %T", got)
	}
	if len(captured) != 1 || captured[0].Result.Index != 1 {
		t.Fatalf("capture did not record first result: %+v", captured)
	}
	if got := filter(nil, complete); got != complete {
		t.Fatalf("completion after 1 of 2 results must not quit, got %T", got)
	}
	w2 := walletMsg(2)
	if got := filter(nil, w2); got != w2 {
		t.Fatalf("wallet result must pass through, got %T", got)
	}
	if len(captured) != 2 || captured[1].Result.Index != 2 {
		t.Fatalf("capture did not record second result: %+v", captured)
	}
	if captured[0].Result.Address != "0xabc" || captured[0].Result.PrivateKey != "[encrypted backup]" {
		t.Fatal("captured message fields not preserved")
	}
	if got := filter(nil, incomplete); got != incomplete {
		t.Fatalf("incomplete progress must not quit, got %T", got)
	}
	got := filter(nil, complete)
	if _, ok := got.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", got)
	}

	var captureCount int
	single := quitAfterWalletResults(1, func(m tui.WalletResultMsg) { captureCount++ })
	if got := single(nil, walletMsg(1)); got != walletMsg(1) {
		t.Fatalf("wallet result must pass through, got %T", got)
	}
	if captureCount != 1 {
		t.Fatalf("capture invoked %d times", captureCount)
	}
	got = single(nil, complete)
	if _, ok := got.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", got)
	}

	nilCapture := quitAfterWalletResults(1, nil)
	if got := nilCapture(nil, walletMsg(1)); got != walletMsg(1) {
		t.Fatalf("wallet result must pass through, got %T", got)
	}
	got = nilCapture(nil, complete)
	if _, ok := got.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg with nil capture, got %T", got)
	}

	var extra []tui.WalletResultMsg
	overflow := quitAfterWalletResults(2, func(m tui.WalletResultMsg) { extra = append(extra, m) })
	for i := 1; i <= 3; i++ {
		m := walletMsg(i)
		if got := overflow(nil, m); got != m {
			t.Fatalf("wallet result %d must pass through, got %T", i, got)
		}
	}
	if len(extra) != 3 {
		t.Fatalf("expected 3 captured results, got %d", len(extra))
	}
	if got := overflow(nil, incomplete); got != incomplete {
		t.Fatalf("incomplete progress must not quit, got %T", got)
	}
	got = overflow(nil, complete)
	if _, ok := got.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg after excess results, got %T", got)
	}
}

func TestWalletTUISuccessUsesProductionQuit(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(fmt.Sprintf("multi=%v", multi), func(t *testing.T) {
			expected := 1
			if multi {
				expected = 2
			}
			received := 0
			sawQuit := false
			app := headlessApp(t, t.TempDir(), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
				switch msg.(type) {
				case tui.WalletResultMsg:
					received++
				case tui.QuitMsg:
					sawQuit = true
				}
				return msg
			}))
			app.config.KeyStore.Enabled = false
			result := stubResult(t, false)
			pool := &stubWorkerPool{stats: worker.NewStatsCollector(), next: func() (*wallet.GenerationResult, error) { return result, nil }}
			err := captureOutput(t, func() error {
				if multi {
					return app.generateMultipleWalletsTUI(t.Context(), pool, wallet.GenerationCriteria{Network: "ethereum"}, expected, tui.EngineInfo{Engine: "cpu"})
				}
				return app.generateSingleWalletTUI(t.Context(), pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
			})
			if err != nil {
				t.Fatalf("TUI completion failed: %v", err)
			}
			if !sawQuit {
				t.Fatal("production quit message was not observed")
			}
			if received != expected || pool.calls != expected {
				t.Fatalf("received %d results from %d calls; expected %d", received, pool.calls, expected)
			}
		})
	}
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
	app := headlessApp(t, dir, tea.WithFilter(quitAfterWalletResults(1, nil)))

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, true), nil },
	}

	ctx := t.Context()

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
		if !jsonKey.Equal(mnemonicKey) {
			t.Fatal("mnemonic-derived private key does not match decrypted keystore key")
		}
	}
}

func TestMultipleWalletsTUISuccessWritesFiles(t *testing.T) {
	dir := t.TempDir()
	app := headlessApp(t, dir, tea.WithFilter(quitAfterWalletResults(2, nil)))

	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return stubResult(t, false), nil },
	}

	ctx := t.Context()

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

func TestDisplayMultipleWalletResultsPasswordLogging(t *testing.T) {
	sentinel := "Synthetic-Sidecar-Password42"

	for _, verbose := range []bool{false, true} {
		t.Run(fmt.Sprintf("verbose=%v", verbose), func(t *testing.T) {
			dir := t.TempDir()
			cfg := config.DefaultConfig()
			cfg.CLI.QuietMode = true
			cfg.CLI.VerboseOutput = verbose
			cfg.KeyStore.Enabled = true
			cfg.KeyStore.OutputDir = dir
			cfg.KeyStore.KDFAlgorithm = "pbkdf2"
			app := NewApplication(cfg, "test", "test", "test")

			w := newTestWallet(t, false)
			base := w.Address
			if !strings.HasPrefix(base, "0x") {
				base = "0x" + base
			}
			sidecar := filepath.Join(dir, base+".pwd")
			if err := os.WriteFile(sidecar, []byte(sentinel), 0600); err != nil {
				t.Fatal(err)
			}

			outFile, err := os.CreateTemp(t.TempDir(), "stdout-*")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = outFile.Close() }()
			callErr := func() error {
				oldStdout := os.Stdout
				os.Stdout = outFile
				defer func() { os.Stdout = oldStdout }()
				return app.displayMultipleWalletResults([]*wallet.GenerationResult{
					{Wallet: w, Attempts: 1, Duration: time.Second},
				}, 1, time.Second, false)
			}()
			if _, err := outFile.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			outBytes, err := io.ReadAll(outFile)
			if err != nil {
				t.Fatal(err)
			}
			output := string(outBytes)

			if callErr == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(output, "Keystore: Failed to generate") {
				t.Error("expected 'Keystore: Failed to generate' in output")
			}
			if !strings.Contains(output, "Keystore errors: 1/1") {
				t.Error("expected 'Keystore errors: 1/1' in output")
			}
			for _, s := range []string{sidecar, base + ".pwd", sentinel, w.PrivateKey} {
				if s != "" && strings.Contains(output, s) {
					t.Errorf("output leaks sensitive value %q", s)
				}
			}

			content, err := os.ReadFile(sidecar)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != sentinel {
				t.Fatal("existing sidecar was modified")
			}
			if _, err := os.Lstat(filepath.Join(dir, base+".json")); !os.IsNotExist(err) {
				t.Fatal("keystore json unexpectedly present")
			}
		})
	}
}

func runWalletTUI(t *testing.T, app *Application, multi bool, ctx context.Context, pool *stubWorkerPool) error {
	t.Helper()
	return captureOutput(t, func() error {
		if multi {
			return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 1, tui.EngineInfo{Engine: "cpu"})
		}
		return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
	})
}

func TestWalletTUICancellationStopsProgram(t *testing.T) {
	for _, multi := range []bool{false, true} {
		for _, success := range []bool{false, true} {
			t.Run(fmt.Sprintf("multi=%v/success=%v", multi, success), func(t *testing.T) {
				var fired atomic.Bool
				var watchdog *time.Timer
				app := headlessApp(t, t.TempDir(), func(p *tea.Program) {
					watchdog = time.AfterFunc(1500*time.Millisecond, func() {
						fired.Store(true)
						p.Kill()
					})
				})
				defer func() {
					if watchdog != nil {
						watchdog.Stop()
					}
				}()
				app.config.KeyStore.Enabled = false

				result := stubResult(t, false)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				pool := &stubWorkerPool{
					stats: worker.NewStatsCollector(),
					next: func() (*wallet.GenerationResult, error) {
						cancel()
						if success {
							return result, nil
						}
						return nil, context.Canceled
					},
				}

				err := runWalletTUI(t, app, multi, ctx, pool)
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected context cancellation, got %v", err)
				}
				if fired.Load() {
					t.Fatal("TUI ignored caller cancellation until watchdog killed it")
				}
				if pool.calls != 1 {
					t.Fatalf("expected one generation attempt, got %d", pool.calls)
				}
			})
		}
	}
}

func TestWalletTUIDeadlineStopsProgram(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(fmt.Sprintf("multi=%v", multi), func(t *testing.T) {
			var fired atomic.Bool
			var watchdog *time.Timer
			app := headlessApp(t, t.TempDir(), func(p *tea.Program) {
				watchdog = time.AfterFunc(1500*time.Millisecond, func() {
					fired.Store(true)
					p.Kill()
				})
			})
			defer func() {
				if watchdog != nil {
					watchdog.Stop()
				}
			}()
			app.config.KeyStore.Enabled = false

			result := stubResult(t, false)
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			pool := &stubWorkerPool{
				stats: worker.NewStatsCollector(),
				next: func() (*wallet.GenerationResult, error) {
					<-ctx.Done()
					return result, nil
				},
			}

			err := runWalletTUI(t, app, multi, ctx, pool)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected deadline exceeded, got %v", err)
			}
			if fired.Load() {
				t.Fatal("TUI ignored caller deadline until watchdog killed it")
			}
			if pool.calls != 1 {
				t.Fatalf("expected one generation attempt, got %d", pool.calls)
			}
		})
	}
}

func captureStdoutString(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	outFile, err := os.CreateTemp(t.TempDir(), "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outFile.Close() }()
	callErr := func() error {
		oldStdout := os.Stdout
		os.Stdout = outFile
		defer func() { os.Stdout = oldStdout }()
		return fn()
	}()
	if _, err := outFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	outBytes, err := io.ReadAll(outFile)
	if err != nil {
		t.Fatal(err)
	}
	return string(outBytes), callErr
}

func TestWalletTUIInterruptRecoversResults(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(fmt.Sprintf("multi=%v", multi), func(t *testing.T) {
			dir := t.TempDir()
			var program *tea.Program
			app := headlessApp(t, dir, func(p *tea.Program) { program = p })
			app.config.KeyStore.Enabled = false

			result := stubResult(t, true)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			pool := &stubWorkerPool{
				stats: worker.NewStatsCollector(),
				next: func() (*wallet.GenerationResult, error) {
					program.Kill()
					return result, nil
				},
			}

			output, err := captureStdoutString(t, func() error {
				if multi {
					return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 1, tui.EngineInfo{Engine: "cpu"})
				}
				return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(output, "TUI interrupted") {
				t.Error("expected interruption warning in output")
			}
			for _, want := range []string{result.Wallet.Address, result.Wallet.PrivateKey, result.Wallet.Mnemonic} {
				if !strings.Contains(output, want) {
					t.Errorf("output missing wallet material: %s", output)
				}
			}
			if pool.calls != 1 {
				t.Fatalf("expected one generation attempt, got %d", pool.calls)
			}
			if files := listArtifacts(t, dir); len(files) != 0 {
				t.Fatalf("expected no artifacts, got %v", files)
			}
		})
	}
}

func TestWalletTUIInterruptAfterPersistRecoversResults(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(fmt.Sprintf("multi=%v", multi), func(t *testing.T) {
			dir := t.TempDir()
			var program *tea.Program
			var snapshotErr error
			snapshotContents := map[string][]byte{}
			snapshotInfos := map[string]os.FileInfo{}
			app := headlessApp(t, dir,
				func(p *tea.Program) { program = p },
				tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
					if _, ok := msg.(tui.WalletResultMsg); ok {
						entries, err := os.ReadDir(dir)
						if err != nil {
							snapshotErr = err
						}
						for _, entry := range entries {
							path := filepath.Join(dir, entry.Name())
							content, err := os.ReadFile(path)
							if err != nil {
								snapshotErr = err
								continue
							}
							info, err := os.Lstat(path)
							if err != nil {
								snapshotErr = err
								continue
							}
							snapshotContents[entry.Name()] = content
							snapshotInfos[entry.Name()] = info
						}
						program.Kill()
						return nil
					}
					return msg
				}),
			)

			result := stubResult(t, true)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			pool := &stubWorkerPool{
				stats: worker.NewStatsCollector(),
				next:  func() (*wallet.GenerationResult, error) { return result, nil },
			}

			output, err := captureStdoutString(t, func() error {
				if multi {
					return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 1, tui.EngineInfo{Engine: "cpu"})
				}
				return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
			})
			if snapshotErr != nil {
				t.Fatalf("snapshot in filter failed: %v", snapshotErr)
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "TUI failed after wallet generation") {
				t.Fatalf("expected TUI failure error, got %v", err)
			}
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "file exists") {
				t.Fatalf("persistence was attempted a second time: %v", err)
			}
			if !strings.Contains(output, "TUI interrupted") {
				t.Error("expected interruption warning in output")
			}
			for _, want := range []string{result.Wallet.Address, result.Wallet.PrivateKey, result.Wallet.Mnemonic} {
				if !strings.Contains(output, want) {
					t.Errorf("output missing wallet material: %s", output)
				}
			}

			if len(snapshotContents) != 3 {
				t.Fatalf("expected 3 artifacts at interrupt, got %v", snapshotContents)
			}
			for name, want := range snapshotContents {
				path := filepath.Join(dir, name)
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != string(want) {
					t.Fatalf("artifact %s bytes changed after interrupt", name)
				}
				info, err := os.Lstat(path)
				if err != nil {
					t.Fatal(err)
				}
				if !os.SameFile(snapshotInfos[name], info) {
					t.Fatalf("artifact %s inode changed after interrupt", name)
				}
			}
			files := listArtifacts(t, dir)
			addresses := addressFromFiles(files)
			if len(addresses) != 1 || len(files) != 3 {
				t.Fatalf("expected exactly one artifact set, got %v", files)
			}
			jsonKey := decryptKeystoreArtifact(t, dir, addresses[0])
			mnemonicKey := deriveKeyFromMnemonic(t, result.Wallet.Mnemonic)
			if !jsonKey.Equal(mnemonicKey) {
				t.Fatal("persisted artifacts do not match generated wallet")
			}
			if pool.calls != 1 {
				t.Fatalf("expected one generation attempt, got %d", pool.calls)
			}
		})
	}
}

func TestWalletTUIPersistenceErrorRecoversResults(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, multi := range []bool{false, true} {
		t.Run(fmt.Sprintf("multi=%v", multi), func(t *testing.T) {
			app := headlessApp(t, blocker)

			result := stubResult(t, true)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			pool := &stubWorkerPool{
				stats: worker.NewStatsCollector(),
				next:  func() (*wallet.GenerationResult, error) { return result, nil },
			}

			output, err := captureStdoutString(t, func() error {
				if multi {
					return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 1, tui.EngineInfo{Engine: "cpu"})
				}
				return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "failed to persist wallet") {
				t.Fatalf("expected persistence error, got %v", err)
			}
			if !strings.Contains(output, "TUI interrupted") {
				t.Error("expected interruption warning in output")
			}
			for _, want := range []string{result.Wallet.Address, result.Wallet.PrivateKey, result.Wallet.Mnemonic} {
				if !strings.Contains(output, want) {
					t.Errorf("output missing wallet material")
				}
			}
			if pool.calls != 1 {
				t.Fatalf("expected one generation attempt, got %d", pool.calls)
			}
		})
	}
}

func TestDisplayRecoveredWalletResultsQuiet(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CLI.QuietMode = true
	app := NewApplication(cfg, "test", "test", "test")
	w := newTestWallet(t, true)

	output, err := captureStdoutString(t, func() error {
		app.displayRecoveredWalletResults([]*wallet.GenerationResult{
			{Wallet: w, Attempts: 1, Duration: time.Second},
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "TUI interrupted") {
		t.Error("expected interruption warning")
	}
	if !strings.Contains(output, w.Address) {
		t.Error("expected address in quiet output")
	}
	if strings.Contains(output, w.PrivateKey) {
		t.Error("quiet output leaked private key")
	}
	if strings.Contains(output, w.Mnemonic) {
		t.Error("quiet output leaked mnemonic")
	}

	if err := captureOutput(t, func() error {
		app.displayRecoveredWalletResults(nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
