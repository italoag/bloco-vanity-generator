package backup

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	ageHeaderPrefix    = "age-encryption.org/v1\n"
	ageRecipientPrefix = "age1"
	bech32Charset      = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	ageRecipientLen    = 62
	maxRecipients      = 16
	maxCiphertextBytes = 8 * MaxBundleBytes
	maxRunnerOutput    = 256 * 1024
)

type Options struct {
	Binary          string
	OutputDir       string
	Recipients      []string
	IdentityFile    string
	KeychainService string
	KeychainAccount string
	Timeout         time.Duration
}

type Receipt struct {
	Path string
	ID   string
}

type PendingError struct {
	Path string
	Err  error
}

func (e *PendingError) Error() string {
	reason := "operation failed"
	if errors.Is(e.Err, context.Canceled) {
		reason = "canceled"
	} else if errors.Is(e.Err, context.DeadlineExceeded) {
		reason = "timed out"
	}
	return fmt.Sprintf("backup left as encrypted pending artifact at %s: %s", strconv.Quote(e.Path), reason)
}

func (e *PendingError) Unwrap() error {
	return e.Err
}

type Store struct {
	binary          string
	outputDir       string
	recipients      []string
	identityFile    string
	keychainService string
	keychainAccount string
	timeout         time.Duration
	runner          commandRunner
}

func NewStore(opts Options) (*Store, error) {
	binary := opts.Binary
	if binary == "" {
		binary = "fnox"
	}
	resolved, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("fnox binary not found")
	}
	if resolved, err = filepath.Abs(resolved); err != nil {
		return nil, fmt.Errorf("fnox binary not found")
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if timeout < 0 {
		return nil, fmt.Errorf("invalid timeout")
	}
	if opts.IdentityFile != "" && (opts.KeychainService != "" || opts.KeychainAccount != "") {
		return nil, fmt.Errorf("conflicting identity options")
	}
	if opts.IdentityFile != "" && !filepath.IsAbs(opts.IdentityFile) {
		return nil, fmt.Errorf("identity file must be an absolute path")
	}
	if opts.KeychainService == "" {
		opts.KeychainService = "bloco-vgen"
	}
	if opts.KeychainAccount == "" {
		opts.KeychainAccount = "age-identity"
	}
	return &Store{
		binary:          resolved,
		outputDir:       opts.OutputDir,
		recipients:      append([]string{}, opts.Recipients...),
		identityFile:    opts.IdentityFile,
		keychainService: opts.KeychainService,
		keychainAccount: opts.KeychainAccount,
		timeout:         timeout,
		runner:          execRunner{},
	}, nil
}

type portableProvider struct {
	Type       string   `toml:"type"`
	Recipients []string `toml:"recipients"`
}

type portableSecret struct {
	Provider string `toml:"provider"`
	Value    string `toml:"value"`
}

type portableConfig struct {
	Env       *bool                       `toml:"env"`
	IfMissing string                      `toml:"if_missing"`
	Providers map[string]portableProvider `toml:"providers"`
	Secrets   map[string]portableSecret   `toml:"secrets"`
}

func parsePortableData(data []byte) (*portableConfig, error) {
	if len(data) > MaxArtifactBytes {
		return nil, fmt.Errorf("artifact too large")
	}
	var cfg portableConfig
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("invalid artifact format")
	}
	if cfg.Env == nil || *cfg.Env {
		return nil, fmt.Errorf("invalid artifact")
	}
	if cfg.IfMissing != "error" {
		return nil, fmt.Errorf("invalid artifact")
	}
	if len(cfg.Providers) != 1 {
		return nil, fmt.Errorf("invalid artifact")
	}
	provider, ok := cfg.Providers["wallet_age"]
	if !ok || provider.Type != "age" {
		return nil, fmt.Errorf("invalid artifact")
	}
	if len(provider.Recipients) == 0 || len(provider.Recipients) > maxRecipients {
		return nil, fmt.Errorf("invalid artifact recipients")
	}
	for _, r := range provider.Recipients {
		if !validAgeRecipient(r) {
			return nil, fmt.Errorf("invalid artifact recipients")
		}
	}
	if len(cfg.Secrets) != 1 {
		return nil, fmt.Errorf("invalid artifact")
	}
	secret, ok := cfg.Secrets["WALLET_BACKUP"]
	if !ok || secret.Provider != "wallet_age" {
		return nil, fmt.Errorf("invalid artifact")
	}
	if len(secret.Value) > maxCiphertextBytes {
		return nil, fmt.Errorf("invalid artifact")
	}
	return &cfg, nil
}

func parsePortableFile(path string) (*portableConfig, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("artifact not accessible")
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("invalid artifact")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("artifact not readable")
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || !os.SameFile(info, st) {
		return nil, fmt.Errorf("invalid artifact")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxArtifactBytes+1))
	if err != nil {
		return nil, fmt.Errorf("artifact not readable")
	}
	if len(data) > MaxArtifactBytes {
		return nil, fmt.Errorf("artifact too large")
	}
	return parsePortableData(data)
}

func validAgeRecipient(r string) bool {
	if len(r) != ageRecipientLen || !strings.HasPrefix(r, ageRecipientPrefix) {
		return false
	}
	if r != strings.ToLower(r) {
		return false
	}
	for _, c := range r[len(ageRecipientPrefix):] {
		if !strings.ContainsRune(bech32Charset, c) {
			return false
		}
	}
	return true
}

func validAgeCiphertext(value string) bool {
	if value == "" || len(value) > maxCiphertextBytes {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	return err == nil && bytes.HasPrefix(raw, []byte(ageHeaderPrefix))
}

func renderPortableConfig(recipients []string, ciphertext string) []byte {
	var sb strings.Builder
	sb.WriteString("env = false\nif_missing = \"error\"\n\n[providers.wallet_age]\ntype = \"age\"\nrecipients = [")
	for i, r := range recipients {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strconv.Quote(r))
	}
	sb.WriteString("]\n\n[secrets.WALLET_BACKUP]\nprovider = \"wallet_age\"\n")
	if ciphertext != "" {
		sb.WriteString("value = " + strconv.Quote(ciphertext) + "\n")
	}
	return []byte(sb.String())
}

func (s *Store) renderRuntimeConfig(ciphertext string) []byte {
	var sb strings.Builder
	sb.WriteString("env = false\nif_missing = \"error\"\n\n[providers.wallet_age]\ntype = \"age\"\nrecipients = []\n")
	if s.identityFile != "" {
		sb.WriteString("key_file = " + strconv.Quote(s.identityFile) + "\n")
	} else {
		sb.WriteString("identity = { provider = \"wallet_keychain\", value = " + strconv.Quote(s.keychainAccount) + " }\n")
		sb.WriteString("\n[providers.wallet_keychain]\ntype = \"keychain\"\nservice = " + strconv.Quote(s.keychainService) + "\n")
	}
	sb.WriteString("\n[secrets.WALLET_BACKUP]\nprovider = \"wallet_age\"\nvalue = " + strconv.Quote(ciphertext) + "\n")
	return []byte(sb.String())
}

func fnoxArgs(configPath string, extra ...string) []string {
	args := []string{
		"--config", configPath,
		"--profile", "default",
		"--no-daemon",
		"--non-interactive",
		"--no-color",
		"--if-missing", "error",
	}
	return append(args, extra...)
}

func writeFileSync(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	return writeSyncClose(f, data)
}

func rewriteFileSync(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	return writeSyncClose(f, data)
}

func writeSyncClose(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	if err := d.Sync(); err != nil {
		var sysErr *os.SyscallError
		if errors.As(err, &sysErr) {
			errno, ok := sysErr.Err.(syscall.Errno)
			if ok && (errno == syscall.EINVAL || errno == syscall.ENOTSUP || errno == syscall.ENOTTY) {
				return nil
			}
		}
		return err
	}
	return nil
}

func safeCommandError(err error, message string) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", message, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w", message, context.DeadlineExceeded)
	}
	return errors.New(message)
}

func decodeBundle(data []byte) (*Bundle, error) {
	if len(data) > MaxBundleBytes {
		return nil, fmt.Errorf("bundle too large")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var b Bundle
	if err := dec.Decode(&b); err != nil {
		return nil, fmt.Errorf("invalid bundle")
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid bundle")
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) Doctor(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(s.recipients) == 0 || len(s.recipients) > maxRecipients {
		return fmt.Errorf("invalid recipients")
	}
	for _, r := range s.recipients {
		if !validAgeRecipient(r) {
			return fmt.Errorf("invalid recipients")
		}
	}
	if s.outputDir != "" {
		if err := s.prepareOutputDir(); err != nil {
			return err
		}
		probe, err := os.CreateTemp(s.outputDir, ".fnox-write-check-*")
		if err != nil {
			return fmt.Errorf("output directory not writable")
		}
		probePath := probe.Name()
		syncErr := probe.Sync()
		closeErr := probe.Close()
		removeErr := os.Remove(probePath)
		if syncErr != nil || closeErr != nil {
			_ = os.Remove(probePath)
			return fmt.Errorf("output directory not writable")
		}
		if removeErr != nil {
			return fmt.Errorf("output directory probe cleanup failed")
		}
	}
	scratch, err := os.MkdirTemp("", "fnox-doctor-")
	if err != nil {
		return fmt.Errorf("scratch setup failed")
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	if err := os.Chmod(scratch, 0700); err != nil {
		return fmt.Errorf("scratch setup failed")
	}
	env, err := isolatedEnv(scratch)
	if err != nil {
		return fmt.Errorf("scratch setup failed")
	}
	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	verErr := checkFnoxVersion(runCtx, s.runner, s.binary, env, scratch)
	cancel()
	if verErr != nil {
		return verErr
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("probe generation failed")
	}
	payload := []byte(`{"probe":"` + hex.EncodeToString(nonce) + `"}`)
	defer func() {
		for i := range payload {
			payload[i] = 0
		}
	}()
	cfgPath := filepath.Join(scratch, "probe.fnox.toml")
	if err := writeFileSync(cfgPath, renderPortableConfig(s.recipients, "")); err != nil {
		return fmt.Errorf("probe setup failed")
	}
	runCtx, cancel = context.WithTimeout(ctx, s.timeout)
	_, setErr := s.runner.Run(runCtx, s.binary, fnoxArgs(cfgPath, "set", "WALLET_BACKUP", "--provider", "wallet_age"), env, scratch, payload, 4096)
	cancel()
	if setErr != nil {
		return safeCommandError(setErr, "fnox encryption failed")
	}
	cfg, parseErr := parsePortableFile(cfgPath)
	if parseErr != nil || !validAgeCiphertext(cfg.Secrets["WALLET_BACKUP"].Value) {
		return fmt.Errorf("fnox produced invalid ciphertext")
	}
	runtimeDir := filepath.Join(scratch, "runtime")
	if err := os.Mkdir(runtimeDir, 0700); err != nil {
		return fmt.Errorf("scratch setup failed")
	}
	runtimeCfg := filepath.Join(runtimeDir, "runtime.fnox.toml")
	if err := writeFileSync(runtimeCfg, s.renderRuntimeConfig(cfg.Secrets["WALLET_BACKUP"].Value)); err != nil {
		return fmt.Errorf("probe setup failed")
	}
	runCtx, cancel = context.WithTimeout(ctx, s.timeout)
	out, getErr := s.runner.Run(runCtx, s.binary, fnoxArgs(runtimeCfg, "get", "WALLET_BACKUP"), env, scratch, nil, maxRunnerOutput)
	cancel()
	if getErr != nil {
		return safeCommandError(getErr, "fnox decryption failed")
	}
	defer func() {
		for i := range out {
			out[i] = 0
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !bytes.Equal(bytes.TrimRight(out, "\r\n"), payload) {
		return fmt.Errorf("fnox roundtrip mismatch")
	}
	return nil
}

func (s *Store) prepareOutputDir() error {
	if s.outputDir == "" {
		return fmt.Errorf("output directory required")
	}
	if !filepath.IsAbs(s.outputDir) {
		return fmt.Errorf("output directory must be absolute")
	}
	info, err := os.Lstat(s.outputDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("output directory not accessible")
		}
		if err := os.MkdirAll(s.outputDir, 0700); err != nil {
			return fmt.Errorf("output directory setup failed")
		}
		return nil
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("output directory invalid")
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("output directory permissions too broad")
	}
	return nil
}

func (s *Store) Save(ctx context.Context, b *Bundle) (Receipt, error) {
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	if err := b.Validate(); err != nil {
		return Receipt{}, err
	}
	if len(s.recipients) == 0 || len(s.recipients) > maxRecipients {
		return Receipt{}, fmt.Errorf("invalid recipients")
	}
	for _, r := range s.recipients {
		if !validAgeRecipient(r) {
			return Receipt{}, fmt.Errorf("invalid recipients")
		}
	}
	if err := s.prepareOutputDir(); err != nil {
		return Receipt{}, err
	}
	data, err := json.Marshal(b)
	if err != nil {
		return Receipt{}, fmt.Errorf("bundle serialization failed")
	}
	defer func() {
		for i := range data {
			data[i] = 0
		}
	}()
	if len(data) > MaxBundleBytes {
		return Receipt{}, fmt.Errorf("bundle too large")
	}
	final := filepath.Join(s.outputDir, b.Filename())
	if _, err := os.Lstat(final); err == nil {
		return Receipt{}, NewSaveExistsError(final)
	} else if !os.IsNotExist(err) {
		return Receipt{}, fmt.Errorf("output path not accessible")
	}
	stageDir, err := os.MkdirTemp(s.outputDir, ".fnox-pending-")
	if err != nil {
		return Receipt{}, fmt.Errorf("staging setup failed")
	}
	encrypted := false
	defer func() {
		if !encrypted {
			_ = os.RemoveAll(stageDir)
		}
	}()
	cfgPath := filepath.Join(stageDir, "backup.fnox.toml")
	if err := writeFileSync(cfgPath, renderPortableConfig(s.recipients, "")); err != nil {
		return Receipt{}, fmt.Errorf("staging setup failed")
	}
	env, err := isolatedEnv(stageDir)
	if err != nil {
		return Receipt{}, fmt.Errorf("staging setup failed")
	}
	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	verErr := checkFnoxVersion(runCtx, s.runner, s.binary, env, stageDir)
	cancel()
	if verErr != nil {
		return Receipt{}, verErr
	}
	runCtx, cancel = context.WithTimeout(ctx, s.timeout)
	_, setErr := s.runner.Run(runCtx, s.binary, fnoxArgs(cfgPath, "set", "WALLET_BACKUP", "--provider", "wallet_age"), env, stageDir, data, 4096)
	cancel()
	cfg, parseErr := parsePortableFile(cfgPath)
	if parseErr != nil || !validAgeCiphertext(cfg.Secrets["WALLET_BACKUP"].Value) {
		_ = os.RemoveAll(stageDir)
		if setErr != nil {
			return Receipt{}, safeCommandError(setErr, "backup encryption failed")
		}
		return Receipt{}, fmt.Errorf("backup encryption failed")
	}
	encrypted = true
	pending := func(err error) (Receipt, error) {
		return Receipt{}, &PendingError{Path: cfgPath, Err: err}
	}
	cfgFile, err := os.OpenFile(cfgPath, os.O_RDWR, 0)
	if err != nil {
		return pending(fmt.Errorf("staging failed"))
	}
	if err := cfgFile.Chmod(0600); err != nil {
		_ = cfgFile.Close()
		return pending(fmt.Errorf("staging failed"))
	}
	if err := cfgFile.Sync(); err != nil {
		_ = cfgFile.Close()
		return pending(fmt.Errorf("staging failed"))
	}
	if err := cfgFile.Close(); err != nil {
		return pending(fmt.Errorf("staging failed"))
	}
	if err := syncDir(stageDir); err != nil {
		return pending(fmt.Errorf("staging failed"))
	}
	if setErr != nil {
		return pending(safeCommandError(setErr, "backup encryption failed"))
	}
	if !equalStringsOrdered(cfg.Providers["wallet_age"].Recipients, s.recipients) {
		return pending(fmt.Errorf("backup verification failed"))
	}
	ciphertext := cfg.Secrets["WALLET_BACKUP"].Value
	runtimeDir, err := os.MkdirTemp(stageDir, "runtime-")
	if err != nil {
		return pending(fmt.Errorf("staging failed"))
	}
	defer func() { _ = os.RemoveAll(runtimeDir) }()
	runtimeCfg := filepath.Join(runtimeDir, "runtime.fnox.toml")
	if err := writeFileSync(runtimeCfg, s.renderRuntimeConfig(ciphertext)); err != nil {
		return pending(fmt.Errorf("staging failed"))
	}
	runCtx, cancel = context.WithTimeout(ctx, s.timeout)
	out, getErr := s.runner.Run(runCtx, s.binary, fnoxArgs(runtimeCfg, "get", "WALLET_BACKUP"), env, stageDir, nil, maxRunnerOutput)
	cancel()
	if getErr != nil {
		return pending(safeCommandError(getErr, "backup verification failed"))
	}
	defer func() {
		for i := range out {
			out[i] = 0
		}
	}()
	if err := ctx.Err(); err != nil {
		return pending(safeCommandError(err, "backup verification failed"))
	}
	roundtrip := bytes.TrimRight(out, "\r\n")
	if _, err := decodeBundle(roundtrip); err != nil {
		return pending(fmt.Errorf("backup verification failed"))
	}
	if !bytes.Equal(roundtrip, data) {
		return pending(fmt.Errorf("backup verification failed"))
	}
	if err := ctx.Err(); err != nil {
		return pending(safeCommandError(err, "backup verification failed"))
	}
	if err := os.Link(cfgPath, final); err != nil {
		return pending(fmt.Errorf("backup publish failed"))
	}
	info, err := os.Lstat(final)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return pending(fmt.Errorf("backup verification failed"))
	}
	stagedInfo, err := os.Lstat(cfgPath)
	if err != nil || !os.SameFile(stagedInfo, info) {
		return pending(fmt.Errorf("backup verification failed"))
	}
	if err := syncDir(s.outputDir); err != nil {
		return pending(fmt.Errorf("backup durability check failed"))
	}
	_ = os.RemoveAll(stageDir)
	return Receipt{Path: final, ID: b.ID}, nil
}

func equalStringsOrdered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func NewSaveExistsError(path string) error {
	return fmt.Errorf("artifact already exists: %s: %w", path, os.ErrExist)
}

func (s *Store) Load(ctx context.Context, path string) (*Bundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg, err := parsePortableFile(path)
	if err != nil {
		return nil, err
	}
	ciphertext := cfg.Secrets["WALLET_BACKUP"].Value
	if !validAgeCiphertext(ciphertext) {
		return nil, fmt.Errorf("invalid artifact ciphertext")
	}
	scratch, err := os.MkdirTemp("", "fnox-load-")
	if err != nil {
		return nil, fmt.Errorf("scratch setup failed")
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	if err := os.Chmod(scratch, 0700); err != nil {
		return nil, fmt.Errorf("scratch setup failed")
	}
	env, err := isolatedEnv(scratch)
	if err != nil {
		return nil, fmt.Errorf("scratch setup failed")
	}
	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	verErr := checkFnoxVersion(runCtx, s.runner, s.binary, env, scratch)
	cancel()
	if verErr != nil {
		return nil, verErr
	}
	runtimeCfg := filepath.Join(scratch, "runtime.fnox.toml")
	if err := writeFileSync(runtimeCfg, s.renderRuntimeConfig(ciphertext)); err != nil {
		return nil, fmt.Errorf("scratch setup failed")
	}
	runCtx, cancel = context.WithTimeout(ctx, s.timeout)
	out, getErr := s.runner.Run(runCtx, s.binary, fnoxArgs(runtimeCfg, "get", "WALLET_BACKUP"), env, scratch, nil, maxRunnerOutput)
	cancel()
	if getErr != nil {
		return nil, safeCommandError(getErr, "backup decryption failed")
	}
	defer func() {
		for i := range out {
			out[i] = 0
		}
	}()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return decodeBundle(bytes.TrimRight(out, "\r\n"))
}
