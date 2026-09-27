package backup

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

var testRecipient = "age1" + strings.Repeat("p", 58)

type fakeRunner struct {
	mu            sync.Mutex
	secrets       map[string][]byte
	calls         [][]string
	envs          [][]string
	failSet       bool
	failGet       bool
	wrongIdentity bool
	badCiphertext bool
	badPlaintext  bool
	onSet         func(cfgPath string, stdin []byte)
	onGet         func()
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{secrets: map[string][]byte{}}
}

func (f *fakeRunner) Run(ctx context.Context, binary string, args []string, env []string, cwd string, stdin []byte, maxStdout int) ([]byte, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string{}, args...))
	f.envs = append(f.envs, append([]string{}, env...))
	f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var cfgPath string
	var sub string
	for i, a := range args {
		if a == "--config" && i+1 < len(args) {
			cfgPath = args[i+1]
		}
	}
	if len(args) > 0 && args[len(args)-1] == "--version" || (len(args) == 1 && args[0] == "--version") {
		return []byte("fnox 1.35.2\n"), nil
	}
	for _, a := range args {
		if a == "set" || a == "get" {
			sub = a
		}
	}
	switch sub {
	case "set":
		cfg, err := parsePortableFile(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("fnox command failed")
		}
		raw := append([]byte(ageHeaderPrefix), stdin...)
		if f.badCiphertext {
			raw = []byte("not-age-ciphertext")
		}
		ct := base64.StdEncoding.EncodeToString(raw)
		f.mu.Lock()
		f.secrets[ct] = append([]byte{}, stdin...)
		f.mu.Unlock()
		if f.onSet != nil {
			f.onSet(cfgPath, stdin)
		}
		if err := rewriteFileSync(cfgPath, renderPortableConfig(cfg.Providers["wallet_age"].Recipients, ct)); err != nil {
			return nil, fmt.Errorf("fnox command failed")
		}
		if f.failSet {
			return nil, fmt.Errorf("fnox command failed")
		}
		return nil, nil
	case "get":
		if f.onGet != nil {
			f.onGet()
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if f.failGet {
			return nil, fmt.Errorf("fnox command failed")
		}
		if f.wrongIdentity {
			return nil, fmt.Errorf("fnox command failed")
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("fnox command failed")
		}
		var loose struct {
			Secrets map[string]struct {
				Value string `toml:"value"`
			} `toml:"secrets"`
		}
		if err := toml.Unmarshal(data, &loose); err != nil {
			return nil, fmt.Errorf("fnox command failed")
		}
		ct := loose.Secrets["WALLET_BACKUP"].Value
		f.mu.Lock()
		plain, ok := f.secrets[ct]
		f.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("fnox command failed")
		}
		if f.badPlaintext {
			plain = []byte(`{"version":99}`)
		}
		return append(append([]byte{}, plain...), '\n'), nil
	}
	return nil, fmt.Errorf("fnox command failed")
}

func newTestStore(t *testing.T, dir string, runner commandRunner) *Store {
	t.Helper()
	if info, err := os.Lstat(dir); err == nil && info.IsDir() {
		if err := os.Chmod(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return &Store{
		binary:          "fnox",
		outputDir:       dir,
		recipients:      []string{testRecipient},
		identityFile:    filepath.Join(t.TempDir(), "identity.txt"),
		keychainService: "bloco-vgen",
		keychainAccount: "age-identity",
		timeout:         5 * time.Second,
		runner:          runner,
	}
}

func scanForPlaintext(t *testing.T, root string, sentinels ...string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}
	for _, e := range entries {
		path := filepath.Join(root, e.Name())
		if e.IsDir() {
			scanForPlaintext(t, path, sentinels...)
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range sentinels {
			if strings.Contains(string(data), s) {
				t.Fatalf("plaintext secret found in %s", path)
			}
		}
	}
}

func TestStoreSaveLoadRoundtrip(t *testing.T) {
	runner := newFakeRunner()
	dir := t.TempDir()
	outDir := filepath.Join(dir, "backups")
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, true)

	receipt, err := store.Save(context.Background(), b)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if receipt.ID != b.ID {
		t.Fatal("receipt id mismatch")
	}
	info, err := os.Lstat(receipt.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("artifact mode %o", info.Mode().Perm())
	}
	dinfo, err := os.Lstat(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if dinfo.Mode().Perm() != 0700 {
		t.Fatalf("output dir mode %o", dinfo.Mode().Perm())
	}
	cfg, err := parsePortableFile(receipt.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !validAgeCiphertext(cfg.Secrets["WALLET_BACKUP"].Value) {
		t.Fatal("no valid ciphertext")
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(entries))
	}
	scanForPlaintext(t, outDir, b.PrivateKey, b.Mnemonic, b.KeystorePassword)

	other := newTestStore(t, t.TempDir(), runner)
	other.identityFile = filepath.Join(t.TempDir(), "other-identity.txt")
	loaded, err := other.Load(context.Background(), receipt.Path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.PrivateKey != b.PrivateKey || loaded.ID != b.ID {
		t.Fatal("loaded bundle mismatch")
	}
}

func TestStoreSaveExistingArtifacts(t *testing.T) {
	for _, kind := range []string{"file", "dir", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			runner := newFakeRunner()
			outDir := t.TempDir()
			store := newTestStore(t, outDir, runner)
			b := ethereumBundle(t, false)
			target := filepath.Join(outDir, b.Filename())
			switch kind {
			case "file":
				if err := os.WriteFile(target, []byte("sentinel"), 0600); err != nil {
					t.Fatal(err)
				}
			case "dir":
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(outDir, "nowhere"), target); err != nil {
					t.Fatal(err)
				}
			}
			_, err := store.Save(context.Background(), b)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, os.ErrExist) {
				t.Fatalf("expected ErrExist, got %v", err)
			}
			if len(runner.calls) != 0 {
				t.Fatal("runner invoked despite existing target")
			}
			switch kind {
			case "file":
				data, rerr := os.ReadFile(target)
				if rerr != nil || string(data) != "sentinel" {
					t.Fatal("existing file modified")
				}
			case "dir":
				info, lerr := os.Lstat(target)
				if lerr != nil || !info.IsDir() {
					t.Fatal("existing dir modified")
				}
			case "symlink":
				got, lerr := os.Readlink(target)
				if lerr != nil || got != filepath.Join(outDir, "nowhere") {
					t.Fatal("existing symlink modified")
				}
			}
		})
	}
}

func TestStoreSaveConcurrentCollision(t *testing.T) {
	runner := newFakeRunner()
	var ready sync.WaitGroup
	ready.Add(2)
	runner.onGet = func() {
		ready.Done()
		ready.Wait()
	}
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)

	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.Save(context.Background(), b)
			results[i] = err
		}(i)
	}
	wg.Wait()
	runner.onGet = nil
	successes := 0
	pendings := 0
	for _, err := range results {
		if err == nil {
			successes++
			continue
		}
		var pe *PendingError
		if errors.As(err, &pe) {
			pendings++
			if _, lerr := os.Lstat(pe.Path); lerr != nil {
				t.Fatal("pending artifact not retained")
			}
			cfg, perr := parsePortableFile(pe.Path)
			if perr != nil || !validAgeCiphertext(cfg.Secrets["WALLET_BACKUP"].Value) {
				t.Fatal("pending artifact has no valid ciphertext")
			}
		} else {
			t.Fatalf("unexpected non-pending error: %v", err)
		}
	}
	if successes != 1 || pendings != 1 {
		t.Fatalf("expected one success one pending, got %d/%d", successes, pendings)
	}
	finalPath := filepath.Join(outDir, b.Filename())
	if _, err := os.Lstat(finalPath); err != nil {
		t.Fatal("final artifact missing")
	}
	loaded, err := store.Load(context.Background(), finalPath)
	if err != nil {
		t.Fatalf("final artifact Load failed: %v", err)
	}
	if loaded.PrivateKey != b.PrivateKey {
		t.Fatal("loaded bundle mismatch")
	}
}

func TestStoreSaveSetFailureNoCiphertext(t *testing.T) {
	runner := newFakeRunner()
	runner.failSet = true
	runner.badCiphertext = true
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)

	_, err := store.Save(context.Background(), b)
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *PendingError
	if errors.As(err, &pe) {
		t.Fatal("unexpected pending error")
	}
	entries, rerr := os.ReadDir(outDir)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(entries) != 0 {
		t.Fatalf("scratch not cleaned: %v", entries)
	}
}

func TestStoreSaveSetFailureValidCiphertextGetFails(t *testing.T) {
	runner := newFakeRunner()
	runner.failSet = true
	runner.failGet = true
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)

	_, err := store.Save(context.Background(), b)
	var pe *PendingError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PendingError, got %v", err)
	}
	if _, lerr := os.Lstat(pe.Path); lerr != nil {
		t.Fatal("pending artifact not retained")
	}
	for _, call := range runner.calls {
		for _, a := range call {
			if a == "get" {
				t.Fatal("get invoked despite set failure")
			}
		}
	}
	scanForPlaintext(t, outDir, b.PrivateKey, b.KeystorePassword)
	if _, lerr := os.Lstat(filepath.Join(outDir, b.Filename())); !os.IsNotExist(lerr) {
		t.Fatal("final artifact unexpectedly present")
	}
}

func TestStoreSaveWrongIdentity(t *testing.T) {
	runner := newFakeRunner()
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)
	receipt, err := store.Save(context.Background(), b)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	runner.wrongIdentity = true
	other := newTestStore(t, t.TempDir(), runner)
	if _, err := other.Load(context.Background(), receipt.Path); err == nil {
		t.Fatal("expected decryption failure")
	}
}

func TestStoreSaveBadPlaintextRetainsPending(t *testing.T) {
	runner := newFakeRunner()
	runner.badPlaintext = true
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)
	_, err := store.Save(context.Background(), b)
	var pe *PendingError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PendingError, got %v", err)
	}
	if _, lerr := os.Lstat(pe.Path); lerr != nil {
		t.Fatal("pending artifact not retained")
	}
	scanForPlaintext(t, outDir, b.PrivateKey)
}

func TestStoreSaveCancelledAfterSet(t *testing.T) {
	runner := newFakeRunner()
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	runner.onGet = cancel
	_, err := store.Save(ctx, b)
	var pe *PendingError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PendingError, got %v", err)
	}
	if _, lerr := os.Lstat(pe.Path); lerr != nil {
		t.Fatal("pending artifact not retained")
	}
}

func TestStoreSaveEnvIsolation(t *testing.T) {
	t.Setenv("FNOX_AGE_KEY", "sentinel-key")
	t.Setenv("FNOX_PROFILE", "sentinel-profile")
	t.Setenv("FNOX_CONFIG_DIR", "/sentinel/dir")
	t.Setenv("RUST_LOG", "debug")
	runner := newFakeRunner()
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)
	if _, err := store.Save(context.Background(), b); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if len(runner.envs) == 0 {
		t.Fatal("no runner invocations captured")
	}
	for _, env := range runner.envs {
		for _, kv := range env {
			key := strings.SplitN(kv, "=", 2)[0]
			switch key {
			case "PATH", "HOME", "USER", "LOGNAME", "USERPROFILE", "SYSTEMROOT",
				"TMPDIR", "TEMP", "TMP", "LANG", "NO_COLOR":
			case "FNOX_CONFIG_DIR":
				if !strings.Contains(kv, "isolated") {
					t.Fatalf("FNOX_CONFIG_DIR not isolated: %s", kv)
				}
			default:
				t.Fatalf("unexpected env var leaked: %s", key)
			}
		}
	}
	for _, args := range runner.calls {
		for _, a := range args {
			for _, s := range []string{b.PrivateKey, b.Mnemonic, b.KeystorePassword} {
				if s != "" && strings.Contains(a, s) {
					t.Fatal("secret in argv")
				}
			}
		}
	}
}

func invalidPortableDocs() map[string]string {
	valid := renderPortableConfig([]string{testRecipient}, base64.StdEncoding.EncodeToString([]byte(ageHeaderPrefix+"x")))
	ct := base64.StdEncoding.EncodeToString([]byte(ageHeaderPrefix + "x"))
	return map[string]string{
		"env_true":            strings.Replace(string(valid), "env = false", "env = true", 1),
		"env_missing":         strings.Replace(string(valid), "env = false\n", "", 1),
		"if_missing":          strings.Replace(string(valid), `if_missing = "error"`, `if_missing = "skip"`, 1),
		"extra_secret":        string(valid) + "\n[secrets.OTHER]\nprovider = \"wallet_age\"\n",
		"extra_provider":      string(valid) + "\n[providers.other]\ntype = \"age\"\nrecipients = []\n",
		"provider_key_file":   strings.Replace(string(valid), "type = \"age\"", "type = \"age\"\nkey_file = \"/tmp/x\"", 1),
		"provider_identity":   strings.Replace(string(valid), "type = \"age\"", "type = \"age\"\nidentity = { provider = \"k\", value = \"v\" }", 1),
		"provider_auth_cmd":   strings.Replace(string(valid), "type = \"age\"", "type = \"age\"\nauth_command = \"x\"", 1),
		"import":              "import = [\"x\"]\n" + string(valid),
		"profiles":            "[profiles.default]\n" + string(valid),
		"default_table":       "[default]\n" + string(valid),
		"secret_json_path":    strings.Replace(string(valid), "provider = \"wallet_age\"\nvalue", "provider = \"wallet_age\"\njson_path = \"x\"\nvalue", 1),
		"secret_as_file":      strings.Replace(string(valid), "provider = \"wallet_age\"\nvalue", "provider = \"wallet_age\"\nas_file = true\nvalue", 1),
		"secret_sync":         strings.Replace(string(valid), "provider = \"wallet_age\"\nvalue", "provider = \"wallet_age\"\nsync = \"x\"\nvalue", 1),
		"secret_default":      strings.Replace(string(valid), "provider = \"wallet_age\"\nvalue", "provider = \"wallet_age\"\ndefault = \"x\"\nvalue", 1),
		"top_sync":            "[sync]\n" + string(valid),
		"mcp":                 "[mcp]\n" + string(valid),
		"top_auth_command":    "auth_command = \"x\"\n" + string(valid),
		"bad_recipient":       string(renderPortableConfig([]string{"age1INVALID"}, ct)),
		"ssh_recipient":       string(renderPortableConfig([]string{"ssh-ed25519 AAAA"}, ct)),
		"empty_recipients":    string(renderPortableConfig(nil, ct)),
		"uppercase_recipient": string(renderPortableConfig([]string{"AGE1" + strings.Repeat("p", 58)}, ct)),
	}
}

func TestParsePortableSchemaStrict(t *testing.T) {
	valid := renderPortableConfig([]string{testRecipient}, base64.StdEncoding.EncodeToString([]byte(ageHeaderPrefix+"x")))
	if _, err := parsePortableData(valid); err != nil {
		t.Fatalf("valid artifact rejected: %v", err)
	}
	for name, doc := range invalidPortableDocs() {
		t.Run(name, func(t *testing.T) {
			if _, err := parsePortableData([]byte(doc)); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestLoadRejectsBeforeRunner(t *testing.T) {
	runner := newFakeRunner()
	store := newTestStore(t, t.TempDir(), runner)
	dir := t.TempDir()
	docs := invalidPortableDocs()
	docs["plaintext_value"] = string(renderPortableConfig([]string{testRecipient}, "not-valid-ciphertext"))
	for name, doc := range docs {
		path := filepath.Join(dir, name+".toml")
		if err := os.WriteFile(path, []byte(doc), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Load(context.Background(), path); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner invoked %d times for invalid artifacts", len(runner.calls))
	}
}

func TestLoadOversizeArtifact(t *testing.T) {
	runner := newFakeRunner()
	store := newTestStore(t, t.TempDir(), runner)
	path := filepath.Join(t.TempDir(), "big.fnox.toml")
	if err := os.WriteFile(path, make([]byte, MaxArtifactBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), path); err == nil {
		t.Fatal("expected error")
	}
	if len(runner.calls) != 0 {
		t.Fatal("runner invoked for oversize artifact")
	}
}

func TestStoreContextSentinels(t *testing.T) {
	runner := newFakeRunner()
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)
	receipt, err := store.Save(context.Background(), b)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Save(ctx, ethereumBundle(t, false)); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}
	if _, err := store.Load(ctx, receipt.Path); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}
	if err := store.Doctor(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}

	slow := &fakeVersionRunner{err: context.DeadlineExceeded}
	slowStore := newTestStore(t, t.TempDir(), slow)
	if _, err := slowStore.Load(context.Background(), receipt.Path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if err := slowStore.Doctor(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestStoreSaveOutputDirRejected(t *testing.T) {
	runner := newFakeRunner()
	b := ethereumBundle(t, false)
	t.Run("symlink_dir", func(t *testing.T) {
		base := t.TempDir()
		real := filepath.Join(base, "real")
		if err := os.Mkdir(real, 0700); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(base, "link")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		store := newTestStore(t, link, runner)
		if _, err := store.Save(context.Background(), b); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("broad_perms", func(t *testing.T) {
		dir := t.TempDir()
		store := newTestStore(t, dir, runner)
		if err := os.Chmod(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Save(context.Background(), b); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("relative_dir", func(t *testing.T) {
		store := newTestStore(t, "relative/path", runner)
		if _, err := store.Save(context.Background(), b); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestStoreSaveErrorsDontLeakSentinel(t *testing.T) {
	runner := newFakeRunner()
	runner.failGet = true
	outDir := t.TempDir()
	store := newTestStore(t, outDir, runner)
	b := ethereumBundle(t, false)
	_, err := store.Save(context.Background(), b)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, s := range []string{b.PrivateKey, b.KeystorePassword, b.Mnemonic} {
		if s != "" && strings.Contains(err.Error(), s) {
			t.Fatal("secret leaked in error")
		}
	}
}

func TestNewStoreValidation(t *testing.T) {
	if _, err := NewStore(Options{Binary: "definitely-missing-fnox-binary"}); err == nil {
		t.Fatal("expected error for missing binary")
	}
	if _, err := NewStore(Options{Binary: "go", Timeout: -time.Second}); err == nil {
		t.Fatal("expected error for negative timeout")
	}
	if _, err := NewStore(Options{Binary: "go", IdentityFile: "/tmp/x", KeychainAccount: "y"}); err == nil {
		t.Fatal("expected error for conflicting identity options")
	}
	if _, err := NewStore(Options{Binary: "go", IdentityFile: "relative"}); err == nil {
		t.Fatal("expected error for relative identity file")
	}
	s, err := NewStore(Options{Binary: "go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.timeout != 30*time.Second {
		t.Fatal("default timeout not applied")
	}
	if s.keychainService != "bloco-vgen" || s.keychainAccount != "age-identity" {
		t.Fatal("keychain defaults not applied")
	}
}

func TestStoreDoctorOutputDirProbe(t *testing.T) {
	recipient := testRecipient

	t.Run("symlink_dir_rejected_before_runner", func(t *testing.T) {
		runner := newFakeRunner()
		target := t.TempDir()
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		store := newTestStore(t, link, runner)
		if err := store.Doctor(context.Background()); err == nil {
			t.Fatal("expected doctor failure for symlink output dir")
		}
		runner.mu.Lock()
		defer runner.mu.Unlock()
		if len(runner.calls) != 0 {
			t.Fatalf("runner invoked %d times before dir rejection", len(runner.calls))
		}
	})

	t.Run("file_output_rejected_before_runner", func(t *testing.T) {
		runner := newFakeRunner()
		f := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		store := newTestStore(t, f, runner)
		if err := store.Doctor(context.Background()); err == nil {
			t.Fatal("expected doctor failure for file output path")
		}
		runner.mu.Lock()
		defer runner.mu.Unlock()
		if len(runner.calls) != 0 {
			t.Fatal("runner invoked before dir rejection")
		}
	})

	t.Run("broad_dir_rejected_before_runner", func(t *testing.T) {
		runner := newFakeRunner()
		dir := t.TempDir()
		if err := os.Chmod(dir, 0755); err != nil {
			t.Fatal(err)
		}
		store := &Store{
			binary:       "fnox",
			outputDir:    dir,
			recipients:   []string{recipient},
			identityFile: filepath.Join(t.TempDir(), "identity.txt"),
			timeout:      5 * time.Second,
			runner:       runner,
		}
		if err := store.Doctor(context.Background()); err == nil {
			t.Fatal("expected doctor failure for broad permissions")
		}
		runner.mu.Lock()
		defer runner.mu.Unlock()
		if len(runner.calls) != 0 {
			t.Fatal("runner invoked before dir rejection")
		}
		info, _ := os.Lstat(dir)
		if info.Mode().Perm() != 0755 {
			t.Fatal("doctor must not chmod existing directory")
		}
	})

	t.Run("probe_created_and_removed", func(t *testing.T) {
		runner := newFakeRunner()
		dir := t.TempDir()
		store := newTestStore(t, dir, runner)
		if err := store.Doctor(context.Background()); err != nil {
			t.Fatalf("doctor failed: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".fnox-write-check-") {
				t.Fatalf("probe left behind: %s", e.Name())
			}
		}
		runner.mu.Lock()
		defer runner.mu.Unlock()
		if len(runner.calls) == 0 {
			t.Fatal("expected runner calls after successful probe")
		}
	})

	t.Run("empty_output_dir_skips_probe", func(t *testing.T) {
		runner := newFakeRunner()
		store := newTestStore(t, "", runner)
		if err := store.Doctor(context.Background()); err != nil {
			t.Fatalf("doctor failed for reader store: %v", err)
		}
	})
}

func TestStoreSavePublicationFailure(t *testing.T) {
	t.Run("sync_failure_retains_both_and_retry_collides", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		outSyncs := 0
		store.publication = &publicationOps{
			syncDir: func(dir string) error {
				if dir == outDir {
					outSyncs++
					if outSyncs == 1 {
						return fmt.Errorf("injected sync failure")
					}
				}
				return syncDir(dir)
			},
		}
		b := ethereumBundle(t, false)
		_, err := store.Save(context.Background(), b)
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if !errors.Is(err, errPublicationUnconfirmed) {
			t.Fatal("unconfirmed marker missing")
		}
		final := filepath.Join(outDir, b.Filename())
		if pe.FinalPath != final {
			t.Fatal("FinalPath must mark unconfirmed publication")
		}
		stageInfo, err := os.Lstat(pe.Path)
		if err != nil {
			t.Fatalf("pending artifact missing: %v", err)
		}
		finalInfo, err := os.Lstat(final)
		if err != nil {
			t.Fatalf("final must be retained: %v", err)
		}
		if !os.SameFile(stageInfo, finalInfo) {
			t.Fatal("final must be the same file as the staged copy")
		}
		if _, err := store.Load(context.Background(), pe.Path); err != nil {
			t.Fatalf("pending artifact not loadable: %v", err)
		}
		retry := newTestStore(t, outDir, runner)
		_, err = retry.Save(context.Background(), b)
		if err == nil || !errors.Is(err, os.ErrExist) {
			t.Fatalf("blind retry must collide with ErrExist, got %v", err)
		}
		if _, err := os.Lstat(pe.Path); err != nil {
			t.Fatal("staged copy must not be deleted")
		}
		if _, err := os.Lstat(final); err != nil {
			t.Fatal("final copy must not be deleted")
		}
	})

	t.Run("postlink_lstat_failure_retains_both", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		b := ethereumBundle(t, false)
		final := filepath.Join(outDir, b.Filename())
		finalStats := 0
		store.publication = &publicationOps{
			lstat: func(path string) (os.FileInfo, error) {
				if path == final {
					finalStats++
					if finalStats == 1 {
						return nil, fmt.Errorf("injected lstat failure")
					}
				}
				return os.Lstat(path)
			},
		}
		_, err := store.Save(context.Background(), b)
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if !errors.Is(err, errPublicationUnconfirmed) {
			t.Fatal("unconfirmed marker missing")
		}
		if pe.FinalPath != final {
			t.Fatal("FinalPath must mark unconfirmed publication")
		}
		if _, err := os.Lstat(final); err != nil {
			t.Fatal("final must be retained")
		}
		if _, err := os.Lstat(pe.Path); err != nil {
			t.Fatal("pending artifact must be retained")
		}
	})

	t.Run("replaced_final_preserved", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		b := ethereumBundle(t, false)
		final := filepath.Join(outDir, b.Filename())
		store.publication = &publicationOps{
			link: func(src, dst string) error {
				if err := os.Link(src, dst); err != nil {
					return err
				}
				if err := os.Remove(dst); err != nil {
					return err
				}
				return os.WriteFile(dst, []byte("unrelated sentinel"), 0600)
			},
		}
		_, err := store.Save(context.Background(), b)
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if !errors.Is(err, errPublicationUnconfirmed) {
			t.Fatal("unconfirmed marker missing")
		}
		if pe.FinalPath != final {
			t.Fatal("FinalPath must mark unconfirmed publication")
		}
		if !strings.Contains(pe.Error(), "the final path was not removed automatically") {
			t.Fatalf("missing retention warning: %s", pe.Error())
		}
		content, err := os.ReadFile(final)
		if err != nil {
			t.Fatalf("read final: %v", err)
		}
		if string(content) != "unrelated sentinel" {
			t.Fatal("unrelated final must be preserved")
		}
		if _, err := os.Lstat(pe.Path); err != nil {
			t.Fatalf("pending artifact missing: %v", err)
		}
	})

	t.Run("late_replacement_after_identity_check", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		b := ethereumBundle(t, false)
		final := filepath.Join(outDir, b.Filename())
		finalStats := 0
		syncCalls := 0
		store.publication = &publicationOps{
			lstat: func(path string) (os.FileInfo, error) {
				if path == final {
					finalStats++
					if finalStats == 1 {
						info, err := os.Lstat(path)
						if err != nil {
							return nil, err
						}
						if err := os.Remove(path); err != nil {
							return nil, err
						}
						if err := os.WriteFile(path, []byte("late sentinel"), 0600); err != nil {
							return nil, err
						}
						return info, nil
					}
				}
				return os.Lstat(path)
			},
			syncDir: func(dir string) error {
				if dir == outDir {
					syncCalls++
					if syncCalls == 1 {
						content, err := os.ReadFile(final)
						if err != nil {
							return err
						}
						if string(content) != "late sentinel" {
							return fmt.Errorf("sentinel must remain during sync")
						}
					}
				}
				return syncDir(dir)
			},
		}
		_, err := store.Save(context.Background(), b)
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if !errors.Is(err, errPublicationUnconfirmed) {
			t.Fatal("unconfirmed marker missing")
		}
		content, err := os.ReadFile(final)
		if err != nil {
			t.Fatalf("read final: %v", err)
		}
		if string(content) != "late sentinel" {
			t.Fatal("late replacement must not be deleted")
		}
		if _, err := os.Lstat(pe.Path); err != nil {
			t.Fatal("staged copy must be retained")
		}
	})

	t.Run("missing_source_preserves_sole_final", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		b := ethereumBundle(t, false)
		final := filepath.Join(outDir, b.Filename())
		var stagedPath string
		store.publication = &publicationOps{
			link: func(src, dst string) error {
				stagedPath = src
				return os.Link(src, dst)
			},
			syncDir: func(dir string) error {
				if dir == outDir {
					if stagedPath != "" {
						if rerr := os.Remove(stagedPath); rerr != nil {
							t.Errorf("staged removal fixture: %v", rerr)
						}
					}
					return fmt.Errorf("injected sync failure")
				}
				return syncDir(dir)
			},
		}
		_, err := store.Save(context.Background(), b)
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if !errors.Is(err, errPublicationUnconfirmed) {
			t.Fatal("unconfirmed marker missing")
		}
		if pe.FinalPath != final {
			t.Fatal("FinalPath must mark unconfirmed publication")
		}
		if _, err := os.Lstat(final); err != nil {
			t.Fatal("sole final copy must be preserved")
		}
	})

	t.Run("replaced_during_successful_sync", func(t *testing.T) {
		cases := map[string]func(final, staged string) error{
			"deleted_then_sentinel": func(final, staged string) error {
				if err := os.Remove(final); err != nil {
					return err
				}
				return os.WriteFile(final, []byte("sentinel"), 0600)
			},
			"missing": func(final, staged string) error {
				return os.Remove(final)
			},
			"symlink": func(final, staged string) error {
				if err := os.Remove(final); err != nil {
					return err
				}
				return os.Symlink(staged, final)
			},
			"perm_change": func(final, staged string) error {
				return os.Chmod(final, 0644)
			},
		}
		for name, mutate := range cases {
			t.Run(name, func(t *testing.T) {
				runner := newFakeRunner()
				outDir := t.TempDir()
				store := newTestStore(t, outDir, runner)
				b := ethereumBundle(t, false)
				final := filepath.Join(outDir, b.Filename())
				var stagedPath string
				mutated := false
				store.publication = &publicationOps{
					link: func(src, dst string) error {
						stagedPath = src
						return os.Link(src, dst)
					},
					syncDir: func(dir string) error {
						if dir == outDir && !mutated {
							mutated = true
							if err := mutate(final, stagedPath); err != nil {
								return err
							}
						}
						return syncDir(dir)
					},
				}
				_, err := store.Save(context.Background(), b)
				var pe *PendingError
				if !errors.As(err, &pe) {
					t.Fatalf("expected PendingError, got %v", err)
				}
				if !errors.Is(err, errPublicationUnconfirmed) {
					t.Fatal("unconfirmed marker missing")
				}
				if _, err := os.Lstat(pe.Path); err != nil {
					t.Fatal("staged copy must be retained")
				}
			})
		}
	})

	t.Run("link_collision_preserved", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		b := ethereumBundle(t, false)
		final := filepath.Join(outDir, b.Filename())
		store.publication = &publicationOps{
			link: func(src, dst string) error {
				if err := os.WriteFile(dst, []byte("competitor"), 0600); err != nil {
					return err
				}
				return os.Link(src, dst)
			},
		}
		_, err := store.Save(context.Background(), b)
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if !errors.Is(err, os.ErrExist) {
			t.Fatal("link collision must preserve os.ErrExist")
		}
		if errors.Is(err, errPublicationUnconfirmed) {
			t.Fatal("link never succeeded; no unconfirmed marker expected")
		}
		if !strings.Contains(pe.Error(), "destination already exists") {
			t.Fatalf("missing actionable reason: %s", pe.Error())
		}
		content, err := os.ReadFile(final)
		if err != nil {
			t.Fatalf("read final: %v", err)
		}
		if string(content) != "competitor" {
			t.Fatal("competitor file must be unchanged")
		}
		if _, err := os.Lstat(pe.Path); err != nil {
			t.Fatal("staged copy must be retained")
		}
	})

	t.Run("link_permission_error_sanitized", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		store.publication = &publicationOps{
			link: func(string, string) error {
				return os.ErrPermission
			},
		}
		_, err := store.Save(context.Background(), ethereumBundle(t, false))
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if !errors.Is(err, os.ErrPermission) {
			t.Fatal("os.ErrPermission must survive wrapping")
		}
		if strings.Contains(pe.Error(), "permission denied") {
			t.Fatalf("raw error must not leak into user message: %s", pe.Error())
		}
	})

	t.Run("link_generic_error_hides_details", func(t *testing.T) {
		runner := newFakeRunner()
		outDir := t.TempDir()
		store := newTestStore(t, outDir, runner)
		store.publication = &publicationOps{
			link: func(string, string) error {
				return fmt.Errorf("secret internal detail")
			},
		}
		_, err := store.Save(context.Background(), ethereumBundle(t, false))
		var pe *PendingError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PendingError, got %v", err)
		}
		if strings.Contains(pe.Error(), "secret internal detail") {
			t.Fatalf("raw error must not leak into user message: %s", pe.Error())
		}
	})
}

func TestPendingErrorReasons(t *testing.T) {
	pending := &PendingError{Path: "/p/f.fnox.toml", Err: context.Canceled}
	if got := pending.Error(); !strings.Contains(got, "canceled") {
		t.Fatalf("missing canceled reason: %s", got)
	}
	pending = &PendingError{Path: "/p/f.fnox.toml", Err: context.DeadlineExceeded}
	if got := pending.Error(); !strings.Contains(got, "timed out") {
		t.Fatalf("missing timeout reason: %s", got)
	}
	pending = &PendingError{Path: "/p/f.fnox.toml", Err: fmt.Errorf("wrap: %w", os.ErrExist)}
	if got := pending.Error(); !strings.Contains(got, "destination already exists; no file was overwritten") {
		t.Fatalf("missing exists reason: %s", got)
	}
	underlying := fmt.Errorf("raw detail must not print")
	pending = &PendingError{Path: "/p/f.fnox.toml", Err: underlying}
	if got := pending.Error(); strings.Contains(got, "raw detail") {
		t.Fatalf("raw error leaked: %s", got)
	}
}

func TestPrepareOutputDirNestedUnderPublicParent(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0755); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(parent)
	if err != nil {
		t.Fatal(err)
	}
	parentMode := info.Mode().Perm()
	if parentMode != 0755 {
		t.Fatalf("fixture parent mode %v, want 0755", parentMode)
	}

	leaf := filepath.Join(parent, "intermediate", "leaf")
	if err := (&Store{outputDir: leaf}).prepareOutputDir(); err != nil {
		t.Fatalf("prepareOutputDir: %v", err)
	}
	for _, dir := range []string{filepath.Join(parent, "intermediate"), leaf} {
		fi, err := os.Lstat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !fi.IsDir() || fi.Mode().Perm() != 0700 {
			t.Fatalf("created dir %s mode %v, want 0700 dir", dir, fi.Mode())
		}
	}
	fi, err := os.Lstat(parent)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != parentMode {
		t.Fatalf("parent mode changed: %v -> %v", parentMode, fi.Mode().Perm())
	}
	if err := (&Store{outputDir: leaf}).prepareOutputDir(); err != nil {
		t.Fatalf("second call must pass: %v", err)
	}

	broad := filepath.Join(parent, "broad")
	if err := os.Mkdir(broad, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(broad, 0755); err != nil {
		t.Fatal(err)
	}
	if err := (&Store{outputDir: broad}).prepareOutputDir(); err == nil {
		t.Fatal("existing broad leaf must still be rejected")
	}

	link := filepath.Join(parent, "linkleaf")
	if err := os.Symlink(leaf, link); err != nil {
		t.Fatal(err)
	}
	if err := (&Store{outputDir: link}).prepareOutputDir(); err == nil {
		t.Fatal("symlink leaf must be rejected")
	}
}
