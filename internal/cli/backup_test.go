package cli

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"bloco-vgen/internal/backup"
	"bloco-vgen/internal/config"
	"bloco-vgen/internal/crypto"
	"bloco-vgen/internal/tui"
	"bloco-vgen/internal/worker"
	"bloco-vgen/pkg/wallet"
)

type fakeFnoxStore struct {
	dir         string
	doctorCalls int
	saveCalls   int
	loadCalls   int
	doctorErr   error
	saveErr     error
	loadBundle  *backup.Bundle
	loadErr     error
	saved       []*backup.Bundle
}

func (f *fakeFnoxStore) Doctor(context.Context) error {
	f.doctorCalls++
	return f.doctorErr
}

func (f *fakeFnoxStore) Save(_ context.Context, b *backup.Bundle) (backup.Receipt, error) {
	f.saveCalls++
	if err := b.Validate(); err != nil {
		return backup.Receipt{}, fmt.Errorf("invalid bundle handed to store: %w", err)
	}
	f.saved = append(f.saved, b)
	if f.saveErr != nil {
		return backup.Receipt{}, f.saveErr
	}
	return backup.Receipt{Path: filepath.Join(f.dir, b.Filename()), ID: b.ID}, nil
}

func (f *fakeFnoxStore) Load(_ context.Context, _ string) (*backup.Bundle, error) {
	f.loadCalls++
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.loadBundle, nil
}

func fnoxApp(t *testing.T, store *fakeFnoxStore) *Application {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.KeyStore.Enabled = true
	cfg.KeyStore.OutputDir = filepath.Join(t.TempDir(), "keystores")
	cfg.KeyStore.KDFAlgorithm = "pbkdf2"
	cfg.TUI.Enabled = false
	cfg.Backup.Store = "fnox"
	cfg.Backup.OutputDir = t.TempDir()
	app := NewApplication(cfg, "test", "test", "test")
	app.backupStore = store
	return app
}

func newFnoxTestApp(t *testing.T) (*Application, *fakeFnoxStore) {
	t.Helper()
	fake := &fakeFnoxStore{dir: t.TempDir()}
	return fnoxApp(t, fake), fake
}

func fnoxHeadlessApp(t *testing.T, store *fakeFnoxStore, extraOpts ...tea.ProgramOption) *Application {
	t.Helper()
	app := headlessApp(t, filepath.Join(t.TempDir(), "keystores"), extraOpts...)
	app.config.Backup.Store = "fnox"
	app.config.Backup.OutputDir = t.TempDir()
	app.backupStore = store
	return app
}

func captureStdStreams(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()
	dir := t.TempDir()
	outFile, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	errFile, err := os.Create(filepath.Join(dir, "stderr"))
	if err != nil {
		_ = outFile.Close()
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outFile, errFile
	defer func() {
		os.Stdout, os.Stderr = oldOut, oldErr
		_ = outFile.Close()
		_ = errFile.Close()
	}()
	callErr := fn()
	os.Stdout, os.Stderr = oldOut, oldErr
	stdout, rErr := os.ReadFile(outFile.Name())
	if rErr != nil {
		t.Fatalf("stdout capture unreadable: %v", rErr)
	}
	stderr, rErr := os.ReadFile(errFile.Name())
	if rErr != nil {
		t.Fatalf("stderr capture unreadable: %v", rErr)
	}
	return string(stdout), string(stderr), callErr
}

func assertNoSecrets(t *testing.T, output string, results ...*wallet.GenerationResult) {
	t.Helper()
	for _, r := range results {
		if r == nil || r.Wallet == nil {
			continue
		}
		if strings.Contains(output, r.Wallet.PrivateKey) {
			t.Fatal("private key leaked into output")
		}
		if r.Wallet.Mnemonic != "" && strings.Contains(output, r.Wallet.Mnemonic) {
			t.Fatal("mnemonic leaked into output")
		}
	}
}

func assertNoBundleSecrets(t *testing.T, stdout, stderr string, saved []*backup.Bundle) {
	t.Helper()
	for _, b := range saved {
		for _, stream := range []string{stdout, stderr} {
			if strings.Contains(stream, b.PrivateKey) {
				t.Fatal("private key leaked")
			}
			if b.Mnemonic != "" && strings.Contains(stream, b.Mnemonic) {
				t.Fatal("mnemonic leaked")
			}
			if b.KeystorePassword != "" && strings.Contains(stream, b.KeystorePassword) {
				t.Fatal("keystore password leaked")
			}
		}
	}
}

func assertDirEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty directory, found %v", entries)
	}
}

func TestFnoxTextSingleSuccess(t *testing.T) {
	for _, verbose := range []bool{false, true} {
		t.Run(fmt.Sprintf("verbose=%v", verbose), func(t *testing.T) {
			runFnoxTextSingleSuccess(t, verbose)
		})
	}
}

func runFnoxTextSingleSuccess(t *testing.T, verbose bool) {
	t.Helper()
	app, fake := newFnoxTestApp(t)
	app.config.CLI.VerboseOutput = verbose
	result := stubResult(t, true)
	origKey := result.Wallet.PrivateKey

	stdout, stderr, err := captureStdStreams(t, func() error {
		return app.displayWalletResult(result, false)
	})
	if err != nil {
		t.Fatalf("display failed: %v", err)
	}
	output := stdout + stderr
	if !strings.Contains(output, "Encrypted backup confirmed") {
		t.Fatalf("missing confirmation in output:\n%s", output)
	}
	if strings.Contains(output, "Private Key:") || strings.Contains(output, "Mnemonic:") ||
		strings.Contains(output, "Keystore saved") || strings.Contains(output, "Mnemonic saved") {
		t.Fatalf("unexpected plaintext output:\n%s", output)
	}
	assertNoSecrets(t, output, result)
	assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	if result.Wallet.PrivateKey != origKey {
		t.Fatal("wallet mutated by display path")
	}
	if fake.saveCalls != 1 || len(fake.saved) != 1 {
		t.Fatalf("expected 1 save, got %d", fake.saveCalls)
	}
	if fake.saved[0].Keystore == nil || fake.saved[0].KeystorePassword == "" {
		t.Fatal("ethereum bundle missing keystore payload")
	}
	assertDirEmpty(t, app.config.KeyStore.OutputDir)
	assertDirEmpty(t, app.config.Backup.OutputDir)
}

func TestFnoxTextSingleSaveFailure(t *testing.T) {
	for _, quiet := range []bool{false, true} {
		t.Run(fmt.Sprintf("quiet=%v", quiet), func(t *testing.T) {
			app, fake := newFnoxTestApp(t)
			app.config.CLI.QuietMode = quiet
			pending := &backup.PendingError{Path: filepath.Join(t.TempDir(), "backup.fnox.toml"), Err: context.DeadlineExceeded}
			fake.saveErr = pending
			result := stubResult(t, true)

			stdout, stderr, err := captureStdStreams(t, func() error {
				return app.displayWalletResult(result, false)
			})
			if err == nil {
				t.Fatal("expected persistence error, got nil")
			}
			var pe *backup.PendingError
			if !stderrors.As(err, &pe) {
				t.Fatalf("expected PendingError, got %v", err)
			}
			output := stdout + stderr
			if !strings.Contains(output, "Encrypted backup not confirmed") {
				t.Fatalf("missing unconfirmed notice:\n%s", output)
			}
			assertNoSecrets(t, output, result)
			assertNoBundleSecrets(t, stdout, stderr, fake.saved)
			if strings.Contains(output, "Wallet generated successfully") {
				t.Fatal("must not claim success when backup failed")
			}
			assertDirEmpty(t, app.config.Backup.OutputDir)
		})
	}
}

func TestFnoxTextMultiSuccessAndFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		for _, verbose := range []bool{false, true} {
			t.Run(fmt.Sprintf("verbose=%v", verbose), func(t *testing.T) {
				runFnoxTextMultiSuccess(t, verbose)
			})
		}
	})
}

func runFnoxTextMultiSuccess(t *testing.T, verbose bool) {
	t.Helper()
	app, fake := newFnoxTestApp(t)
	app.config.CLI.VerboseOutput = verbose
	results := []*wallet.GenerationResult{stubResult(t, true), stubResult(t, false)}
	stdout, stderr, err := captureStdStreams(t, func() error {
		return app.displayMultipleWalletResults(results, 2, time.Millisecond, false)
	})
	if err != nil {
		t.Fatalf("display failed: %v", err)
	}
	output := stdout + stderr
	if !strings.Contains(output, "Encrypted backups confirmed: 2/2") {
		t.Fatalf("missing summary:\n%s", output)
	}
	if strings.Contains(output, "Keystore:") || strings.Contains(output, "Private Key:") {
		t.Fatalf("unexpected plaintext output:\n%s", output)
	}
	assertNoSecrets(t, output, results...)
	assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	if fake.saveCalls != 2 {
		t.Fatalf("expected 2 saves, got %d", fake.saveCalls)
	}
}

func TestFnoxTextMultiFailure(t *testing.T) {
	app, fake := newFnoxTestApp(t)
	fake.saveErr = stderrors.New("operation failed")
	results := []*wallet.GenerationResult{stubResult(t, true), stubResult(t, false)}
	stdout, stderr, err := captureStdStreams(t, func() error {
		return app.displayMultipleWalletResults(results, 2, time.Millisecond, false)
	})
	if err == nil {
		t.Fatal("expected joined persistence error, got nil")
	}
	output := stdout + stderr
	if !strings.Contains(output, "Encrypted backup: not confirmed") ||
		!strings.Contains(output, "Encrypted backup errors: 2/2") {
		t.Fatalf("missing failure reporting:\n%s", output)
	}
	assertNoSecrets(t, output, results...)
	assertNoBundleSecrets(t, stdout, stderr, fake.saved)
}

func TestFnoxTUISingleMasksSecrets(t *testing.T) {
	fake := &fakeFnoxStore{dir: t.TempDir()}
	var gotMsg *tui.WalletResultMsg
	app := fnoxHeadlessApp(t, fake,
		tea.WithFilter(quitAfterWalletResults(1, func(m tui.WalletResultMsg) {
			captured := m
			gotMsg = &captured
		})),
	)

	result := stubResult(t, true)
	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return result, nil },
	}

	ctx := t.Context()
	stdout, stderr, err := captureStdStreams(t, func() error {
		return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
	})
	if err != nil {
		t.Fatalf("TUI run failed: %v", err)
	}
	output := stdout + stderr
	if gotMsg == nil {
		t.Fatal("expected WalletResultMsg intercepted")
	}
	if gotMsg.Result.PrivateKey != "[encrypted backup]" {
		t.Fatal("TUI received secret instead of placeholder")
	}
	if !strings.Contains(output, "Encrypted backup confirmed") {
		t.Fatalf("missing post-TUI confirmation:\n%s", output)
	}
	assertNoSecrets(t, output, result)
	assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	if result.Wallet.PrivateKey == gotMsg.Result.PrivateKey {
		t.Fatal("source wallet mutated")
	}
	if fake.saveCalls != 1 {
		t.Fatalf("expected 1 save, got %d", fake.saveCalls)
	}
	assertDirEmpty(t, app.config.Backup.OutputDir)
}

func TestFnoxTUIMultiMasksSecrets(t *testing.T) {
	fake := &fakeFnoxStore{dir: t.TempDir()}
	var captured []tui.WalletResultMsg
	app := fnoxHeadlessApp(t, fake,
		tea.WithFilter(quitAfterWalletResults(2, func(m tui.WalletResultMsg) {
			captured = append(captured, m)
		})),
	)

	results := []*wallet.GenerationResult{stubResult(t, true), stubResult(t, false)}
	i := 0
	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next: func() (*wallet.GenerationResult, error) {
			r := results[i]
			i++
			return r, nil
		},
	}

	ctx := t.Context()
	stdout, stderr, err := captureStdStreams(t, func() error {
		return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 2, tui.EngineInfo{Engine: "cpu"})
	})
	if err != nil {
		t.Fatalf("TUI run failed: %v", err)
	}
	output := stdout + stderr
	if len(captured) != 2 {
		t.Fatalf("expected 2 result messages, got %d", len(captured))
	}
	for _, m := range captured {
		if m.Result.PrivateKey != "[encrypted backup]" {
			t.Fatal("TUI received secret instead of placeholder")
		}
	}
	assertNoSecrets(t, output, results...)
	assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	if fake.saveCalls != 2 {
		t.Fatalf("expected 2 saves, got %d", fake.saveCalls)
	}
	if !strings.Contains(output, "Wallet 1:") || !strings.Contains(output, "Wallet 2:") {
		t.Fatalf("missing per-wallet backup summary:\n%s", output)
	}
}

func TestFnoxTUIPersistenceFailureRecoversMetadata(t *testing.T) {
	fake := &fakeFnoxStore{dir: t.TempDir(), saveErr: stderrors.New("operation failed")}
	app := fnoxHeadlessApp(t, fake)

	result := stubResult(t, true)
	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return result, nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stdout, stderr, err := captureStdStreams(t, func() error {
		return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
	})
	if err == nil {
		t.Fatal("expected persistence error, got nil")
	}
	output := stdout + stderr
	if !strings.Contains(output, "Encrypted backup not confirmed") {
		t.Fatalf("missing recovery metadata:\n%s", output)
	}
	assertNoSecrets(t, output, result)
	assertNoBundleSecrets(t, stdout, stderr, fake.saved)
}

func TestFnoxTUIMultiPersistenceFailure(t *testing.T) {
	fake := &fakeFnoxStore{dir: t.TempDir(), saveErr: stderrors.New("operation failed")}
	app := fnoxHeadlessApp(t, fake)

	result := stubResult(t, true)
	pool := &stubWorkerPool{
		stats: worker.NewStatsCollector(),
		next:  func() (*wallet.GenerationResult, error) { return result, nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stdout, stderr, err := captureStdStreams(t, func() error {
		return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 2, tui.EngineInfo{Engine: "cpu"})
	})
	if err == nil {
		t.Fatal("expected persistence error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to persist wallet") {
		t.Fatalf("expected persistence error, got: %v", err)
	}
	if pool.calls != 1 || fake.saveCalls != 1 {
		t.Fatalf("expected fail-fast after 1 save, got calls=%d saves=%d", pool.calls, fake.saveCalls)
	}
	output := stdout + stderr
	assertNoSecrets(t, output, result)
	assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	if !strings.Contains(output, "Encrypted backup not confirmed") {
		t.Fatalf("missing recovery metadata:\n%s", output)
	}
}

func TestFnoxTUIInterruptAfterCompletedSave(t *testing.T) {
	cases := []struct {
		name  string
		multi bool
	}{
		{"single", false},
		{"multi", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeFnoxStore{dir: t.TempDir()}
			var program *tea.Program
			var intercepted *tui.WalletResultMsg
			app := fnoxHeadlessApp(t, fake,
				func(p *tea.Program) { program = p },
				tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
					if m, ok := msg.(tui.WalletResultMsg); ok {
						if m.Result.PrivateKey != "[encrypted backup]" {
							t.Error("TUI received secret instead of placeholder")
						}
						captured := m
						intercepted = &captured
						program.Kill()
						return nil
					}
					return msg
				}),
			)

			result := stubResult(t, true)
			origKey := result.Wallet.PrivateKey
			pool := &stubWorkerPool{
				stats: worker.NewStatsCollector(),
				next:  func() (*wallet.GenerationResult, error) { return result, nil },
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			stdout, stderr, err := captureStdStreams(t, func() error {
				if tc.multi {
					return app.generateMultipleWalletsTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 1, tui.EngineInfo{Engine: "cpu"})
				}
				return app.generateSingleWalletTUI(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, tui.EngineInfo{Engine: "cpu"})
			})
			if err == nil {
				t.Fatal("expected error from killed TUI, got nil")
			}
			if intercepted == nil {
				t.Fatal("expected WalletResultMsg before kill")
			}
			if pool.calls != 1 || fake.saveCalls != 1 {
				t.Fatalf("calls=%d saves=%d", pool.calls, fake.saveCalls)
			}
			output := stdout + stderr
			if !strings.Contains(output, result.Wallet.Address) ||
				!strings.Contains(output, "Encrypted backup confirmed") {
				t.Fatalf("missing receipt/metadata recovery:\n%s", output)
			}
			assertNoSecrets(t, output, result)
			assertNoBundleSecrets(t, stdout, stderr, fake.saved)
			if result.Wallet.PrivateKey != origKey {
				t.Fatal("source wallet mutated")
			}
		})
	}
}

func TestSaveFnoxWalletEmptyNetworkIsEthereum(t *testing.T) {
	app, fake := newFnoxTestApp(t)
	w := newTestWallet(t, false)
	w.Network = ""
	if err := app.saveFnoxWallet(context.Background(), w); err != nil {
		t.Fatalf("saveFnoxWallet failed: %v", err)
	}
	if fake.saveCalls != 1 || len(fake.saved) != 1 {
		t.Fatalf("expected 1 save, got %d", fake.saveCalls)
	}
	b := fake.saved[0]
	if b.Network != "ethereum" || b.Keystore == nil || b.KeystorePassword == "" {
		t.Fatalf("expected ethereum bundle with keystore, got %s", b.Network)
	}
	if _, ok := app.backupReceipt(w); !ok {
		t.Fatal("receipt not recorded")
	}
	if w.Network != "" {
		t.Fatal("w.Network must not be mutated")
	}
}

func TestSaveFnoxWalletContextSentinels(t *testing.T) {
	app, fake := newFnoxTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.saveFnoxWallet(ctx, newTestWallet(t, false)); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}
	if fake.saveCalls != 0 {
		t.Fatal("save must not run on canceled context")
	}

	fake.saveErr = &backup.PendingError{Path: filepath.Join(t.TempDir(), "backup.fnox.toml"), Err: context.DeadlineExceeded}
	err := app.saveFnoxWallet(context.Background(), newTestWallet(t, false))
	var pe *backup.PendingError
	if !stderrors.As(err, &pe) {
		t.Fatalf("expected PendingError preserved, got %v", err)
	}
	if fake.saveCalls != 1 || len(fake.saved) != 1 {
		t.Fatalf("expected exactly 1 save, got %d", fake.saveCalls)
	}
}

func TestFnoxDoctorPreflightFailure(t *testing.T) {
	fake := &fakeFnoxStore{dir: t.TempDir(), doctorErr: stderrors.New("fnox decryption failed")}
	app := fnoxApp(t, fake)
	root := app.GetRootCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{
		"--no-tui", "--no-logging", "--engine", "cpu", "--threads", "1",
		"--backup-store", "fnox",
	})
	err := app.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected doctor failure, got nil")
	}
	if !strings.Contains(err.Error(), "preflight") {
		t.Fatalf("expected preflight error, got: %v", err)
	}
	if fake.doctorCalls != 1 || fake.saveCalls != 0 {
		t.Fatalf("doctor=%d save=%d", fake.doctorCalls, fake.saveCalls)
	}
}

func TestFilesStoreNeverInvokesFnox(t *testing.T) {
	fake := &fakeFnoxStore{dir: t.TempDir()}
	cfg := config.DefaultConfig()
	cfg.TUI.Enabled = false
	cfg.KeyStore.OutputDir = t.TempDir()
	cfg.KeyStore.KDFAlgorithm = "pbkdf2"
	cfg.Logging.Enabled = false
	app := NewApplication(cfg, "test", "test", "test")
	app.backupStore = fake
	root := app.GetRootCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{
		"--no-tui", "--no-logging", "--engine", "cpu", "--threads", "1", "--prefix", "a",
	})
	if err := app.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("files generation failed: %v", err)
	}
	if fake.doctorCalls != 0 || fake.saveCalls != 0 || fake.loadCalls != 0 {
		t.Fatal("files mode must not invoke fnox store")
	}
}

func TestFnoxFlagConflicts(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"keystore-dir", []string{"--backup-store", "fnox", "--keystore-dir", "./x"}},
		{"no-keystore", []string{"--backup-store", "fnox", "--no-keystore"}},
		{"invalid store", []string{"--backup-store", "bogus"}},
		{"identity+keychain", []string{"--backup-store", "fnox", "--age-identity", "/tmp/id", "--keychain-service", "svc"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, fake := newFnoxTestApp(t)
			root := app.GetRootCommand()
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SilenceErrors = true
			root.SilenceUsage = true
			args := append([]string{"--no-tui", "--no-logging"}, tc.args...)
			root.SetArgs(args)
			if err := app.ExecuteContext(context.Background()); err == nil {
				t.Fatal("expected flag validation error, got nil")
			}
			if fake.doctorCalls != 0 || fake.saveCalls != 0 {
				t.Fatal("no store calls expected on flag validation failure")
			}
		})
	}
}

func TestFnoxFlagPrecedenceOverEnv(t *testing.T) {
	t.Setenv("BLOCO_BACKUP_DIR", "/env/dir")
	t.Setenv("BLOCO_AGE_RECIPIENTS", "age1env")

	cfg := config.DefaultConfig()
	cfg.TUI.Enabled = false
	cfg.KeyStore.KDFAlgorithm = "pbkdf2"
	cfg.LoadFromEnvironment()
	app := NewApplication(cfg, "test", "test", "test")
	fake := &fakeFnoxStore{dir: t.TempDir()}
	app.backupStore = fake
	root := app.GetRootCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{
		"backup", "doctor",
		"--backup-dir", "/flag/dir",
		"--age-recipient", "age1a",
		"--age-recipient", "age1b",
		"--backup-timeout", "5s",
	})
	if err := app.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if app.config.Backup.OutputDir != "/flag/dir" {
		t.Fatalf("flag did not override env: %q", app.config.Backup.OutputDir)
	}
	if len(app.config.Backup.AgeRecipients) != 2 || app.config.Backup.AgeRecipients[0] != "age1a" {
		t.Fatalf("recipients not parsed: %v", app.config.Backup.AgeRecipients)
	}
	if app.config.Backup.Timeout != 5*time.Second {
		t.Fatalf("timeout not parsed: %v", app.config.Backup.Timeout)
	}
	if fake.doctorCalls != 1 {
		t.Fatalf("expected doctor call, got %d", fake.doctorCalls)
	}

	cfg2 := config.DefaultConfig()
	cfg2.TUI.Enabled = false
	cfg2.LoadFromEnvironment()
	app2 := NewApplication(cfg2, "test", "test", "test")
	app2.backupStore = &fakeFnoxStore{dir: t.TempDir()}
	if _, err := runBackupCommand(t, app2, "backup", "doctor"); err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if app2.config.Backup.OutputDir != "/env/dir" {
		t.Fatalf("env dir not applied: %q", app2.config.Backup.OutputDir)
	}
	if len(app2.config.Backup.AgeRecipients) != 1 || app2.config.Backup.AgeRecipients[0] != "age1env" {
		t.Fatalf("env recipients not applied: %v", app2.config.Backup.AgeRecipients)
	}
}

func runBackupCommand(t *testing.T, app *Application, args ...string) (string, error) {
	t.Helper()
	root := app.GetRootCommand()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs(args)
	err := app.ExecuteContext(context.Background())
	return buf.String(), err
}

func newBitcoinWalletForBackup(t *testing.T) *wallet.Wallet {
	t.Helper()
	key, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	keyBytes := ethcrypto.FromECDSA(key)
	address, err := crypto.NewBitcoinGenerator(nil).GenerateAddressFromPrivateKey(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(keyBytes),
		Mnemonic:   "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
		Network:    "bitcoin",
		CreatedAt:  time.Now().UTC(),
	}
}

func newSolanaWalletForBackup(t *testing.T) *wallet.Wallet {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	address, err := crypto.NewSolanaGenerator(nil).GenerateAddressFromPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(priv),
		Network:    "solana",
		CreatedAt:  time.Now().UTC(),
	}
}

func TestBackupDoctorCommand(t *testing.T) {
	app, fake := newFnoxTestApp(t)
	out, err := runBackupCommand(t, app, "backup", "doctor")
	if err != nil {
		t.Fatalf("doctor failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Fnox backup provider verified") {
		t.Fatalf("missing verification output: %q", out)
	}
	if fake.doctorCalls != 1 {
		t.Fatalf("expected 1 doctor call, got %d", fake.doctorCalls)
	}

	app2, fake2 := newFnoxTestApp(t)
	fake2.doctorErr = stderrors.New("operation failed")
	if _, err := runBackupCommand(t, app2, "backup", "doctor"); err == nil {
		t.Fatal("expected doctor error, got nil")
	}
}

func TestBackupVerifyCommand(t *testing.T) {
	w := newBitcoinWalletForBackup(t)
	b, err := backup.NewBundle(w, nil, "")
	if err != nil {
		t.Fatalf("NewBundle: %v", err)
	}
	app, fake := newFnoxTestApp(t)
	fake.loadBundle = b

	out, err := runBackupCommand(t, app, "backup", "verify", filepath.Join(t.TempDir(), "bitcoin-x.fnox.toml"))
	if err != nil {
		t.Fatalf("verify failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, b.Address) || !strings.Contains(out, "Backup verified") {
		t.Fatalf("missing public metadata: %q", out)
	}
	if !strings.Contains(out, "unrelated") {
		t.Fatalf("missing unrelated mnemonic warning: %q", out)
	}
	if strings.Contains(out, b.PrivateKey) || strings.Contains(out, w.Mnemonic) {
		t.Fatal("verify leaked secrets")
	}
	if fake.loadCalls != 1 {
		t.Fatalf("expected 1 load, got %d", fake.loadCalls)
	}
}

func TestBackupExportRequiresAcknowledgement(t *testing.T) {
	app, fake := newFnoxTestApp(t)
	_, err := runBackupCommand(t, app, "backup", "export", "file.fnox.toml", "--output-dir", t.TempDir()+"/out")
	if err == nil {
		t.Fatal("expected acknowledgement error, got nil")
	}
	if fake.loadCalls != 0 {
		t.Fatal("Load must not run before --allow-plaintext acknowledgement")
	}

	app2, fake2 := newFnoxTestApp(t)
	_, err = runBackupCommand(t, app2, "backup", "export", "file.fnox.toml", "--allow-plaintext")
	if err == nil {
		t.Fatal("expected missing output-dir error, got nil")
	}
	if fake2.loadCalls != 0 {
		t.Fatal("Load must not run before --output-dir validation")
	}
}

func TestBackupExportCommand(t *testing.T) {
	w := newSolanaWalletForBackup(t)
	b, err := backup.NewBundle(w, nil, "")
	if err != nil {
		t.Fatalf("NewBundle: %v", err)
	}
	app, fake := newFnoxTestApp(t)
	fake.loadBundle = b

	outDir := filepath.Join(t.TempDir(), "export-new")
	out, err := runBackupCommand(t, app, "backup", "export", "input.fnox.toml", "--output-dir", outDir, "--allow-plaintext")
	if err != nil {
		t.Fatalf("export failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "plaintext") {
		t.Fatalf("missing plaintext warning: %q", out)
	}
	if strings.Contains(out, b.PrivateKey) {
		t.Fatal("export leaked secret to output")
	}
	for _, name := range []string{"wallet-backup.json", b.Address + ".json", b.Address + ".key"} {
		if _, err := os.Lstat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("expected exported file %s: %v", name, err)
		}
	}
}

func TestFnoxTextBatchPersistsBeforeNextGeneration(t *testing.T) {
	t.Run("incremental_persistence", func(t *testing.T) {
		app, fake := newFnoxTestApp(t)
		pool := &stubWorkerPool{
			stats: worker.NewStatsCollector(),
		}
		criteria := wallet.GenerationCriteria{Network: "ethereum"}
		var first *wallet.GenerationResult
		pool.next = func() (*wallet.GenerationResult, error) {
			if pool.calls == 2 {
				if fake.saveCalls != 1 {
					t.Error("first wallet must be persisted before second generation")
				}
				if _, ok := app.backupReceipt(first.Wallet); !ok {
					t.Error("receipt for first wallet must exist before second generation")
				}
			}
			res := stubResult(t, true)
			if pool.calls == 1 {
				first = res
			}
			return res, nil
		}
		stdout, stderr, err := captureStdStreams(t, func() error {
			return app.generateMultipleWalletsText(context.Background(), pool, criteria, 2, false)
		})
		if err != nil {
			t.Fatalf("batch failed: %v", err)
		}
		if fake.saveCalls != 2 {
			t.Fatalf("expected 2 saves, got %d", fake.saveCalls)
		}
		if pool.calls != 2 {
			t.Fatalf("expected 2 generations, got %d", pool.calls)
		}
		output := stdout + stderr
		if !strings.Contains(output, "Encrypted backups confirmed: 2/2") {
			t.Fatalf("missing summary:\n%s", output)
		}
		assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	})

	t.Run("cancellation_preserves_first_save", func(t *testing.T) {
		app, fake := newFnoxTestApp(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		pool := &stubWorkerPool{
			stats: worker.NewStatsCollector(),
		}
		pool.next = func() (*wallet.GenerationResult, error) {
			if pool.calls == 2 {
				cancel()
				return nil, ctx.Err()
			}
			return stubResult(t, true), nil
		}
		stdout, stderr, err := captureStdStreams(t, func() error {
			return app.generateMultipleWalletsText(ctx, pool, wallet.GenerationCriteria{Network: "ethereum"}, 2, false)
		})
		if err == nil || !stderrors.Is(err, context.Canceled) {
			t.Fatalf("expected wrapped context.Canceled, got %v", err)
		}
		if fake.saveCalls != 1 {
			t.Fatalf("expected exactly 1 save, got %d", fake.saveCalls)
		}
		output := stdout + stderr
		if !strings.Contains(output, "Wallet 1:") || !strings.Contains(output, "Encrypted backup confirmed") {
			t.Fatalf("first wallet confirmation missing:\n%s", output)
		}
		assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	})

	t.Run("pending_error_stops_batch", func(t *testing.T) {
		app, fake := newFnoxTestApp(t)
		pending := &backup.PendingError{Path: filepath.Join(t.TempDir(), "pending", "w.fnox.toml"), Err: context.DeadlineExceeded}
		fake.saveErr = pending
		pool := &stubWorkerPool{
			stats: worker.NewStatsCollector(),
			next:  func() (*wallet.GenerationResult, error) { return stubResult(t, true), nil },
		}
		stdout, stderr, err := captureStdStreams(t, func() error {
			return app.generateMultipleWalletsText(context.Background(), pool, wallet.GenerationCriteria{Network: "ethereum"}, 2, false)
		})
		if err == nil {
			t.Fatal("expected persistence error")
		}
		var pe *backup.PendingError
		if !stderrors.As(err, &pe) {
			t.Fatalf("expected PendingError preserved, got %v", err)
		}
		if pe.Path != pending.Path {
			t.Fatal("pending path not preserved")
		}
		if pool.calls != 1 {
			t.Fatalf("no generation should happen after failed save, got %d", pool.calls)
		}
		if fake.saveCalls != 1 {
			t.Fatalf("expected 1 save attempt, got %d", fake.saveCalls)
		}
		output := stdout + stderr
		if !strings.Contains(output, "Encrypted backup not confirmed") {
			t.Fatalf("missing unconfirmed notice:\n%s", output)
		}
		assertNoBundleSecrets(t, stdout, stderr, fake.saved)
	})
}
