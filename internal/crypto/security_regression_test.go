package crypto

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for the security audit findings tracked as GitHub issues
// #16, #17, #18 and #19.

// --- Issue #16: secrets in cleartext next to the encrypted artifacts --------

// TestEthereumKeystoreOmitsPasswordFileByDefault covers the acceptance
// criterion "bloco-vgen --prefix ab does not produce a .pwd file without an
// explicit flag".
func TestEthereumKeystoreOmitsPasswordFileByDefault(t *testing.T) {
	dir := t.TempDir()
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: dir,
		KDF:             "scrypt",
	})

	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	address := "0x1234567890123456789012345678901234567890"

	keystore, password, err := service.GenerateKeyStore(privateKey, address, "ethereum")
	if err != nil {
		t.Fatalf("GenerateKeyStore failed: %v", err)
	}
	if err := service.SaveKeyStoreFilesToDisk(address, keystore, password, "ethereum", privateKey); err != nil {
		t.Fatalf("SaveKeyStoreFilesToDisk failed: %v", err)
	}

	for _, entry := range readDirNames(t, dir) {
		if strings.HasSuffix(entry, ".pwd") {
			t.Errorf("password file %q was written without WritePasswordFile", entry)
		}
	}
}

// TestEthereumKeystoreWritesPasswordFileWhenRequested checks the opt-in still
// works for callers that accept the trade-off.
func TestEthereumKeystoreWritesPasswordFileWhenRequested(t *testing.T) {
	dir := t.TempDir()
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:           true,
		OutputDirectory:   dir,
		KDF:               "scrypt",
		WritePasswordFile: true,
	})

	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	address := "0x1234567890123456789012345678901234567890"

	keystore, password, err := service.GenerateKeyStore(privateKey, address, "ethereum")
	if err != nil {
		t.Fatalf("GenerateKeyStore failed: %v", err)
	}
	if err := service.SaveKeyStoreFilesToDisk(address, keystore, password, "ethereum", privateKey); err != nil {
		t.Fatalf("SaveKeyStoreFilesToDisk failed: %v", err)
	}

	found := false
	for _, entry := range readDirNames(t, dir) {
		if strings.HasSuffix(entry, ".pwd") {
			found = true
		}
	}
	if !found {
		t.Error("password file was not written even though WritePasswordFile is set")
	}
}

// TestSolanaKeystoreIsEncryptedAndDecryptable covers two acceptance criteria at
// once: the Solana .json must hold a real KeyStore V3 that decrypts back to the
// generated key, and no plaintext .key may be written by default.
func TestSolanaKeystoreIsEncryptedAndDecryptable(t *testing.T) {
	dir := t.TempDir()
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: dir,
		KDF:             "scrypt",
	})

	// 64-byte Ed25519 private key.
	privateKey := strings.Repeat("ab", 64)
	address := "aAoXxSjjj1SUZMQbwTnMW1MEiGeY6R21gJbCBTGUbmV"

	keystore, password, err := service.GenerateKeyStore(privateKey, address, "solana")
	if err != nil {
		t.Fatalf("GenerateKeyStore for solana failed: %v", err)
	}
	if err := service.SaveKeyStoreFilesToDisk(address, keystore, password, "solana", privateKey); err != nil {
		t.Fatalf("SaveKeyStoreFilesToDisk for solana failed: %v", err)
	}

	for _, entry := range readDirNames(t, dir) {
		if strings.HasSuffix(entry, ".key") {
			t.Errorf("plaintext key file %q was written without WritePlaintextKeyFile", entry)
		}
	}

	// The saved .json must decrypt back to the exact key that was generated.
	saved, err := os.ReadFile(filepath.Join(dir, address+".json"))
	if err != nil {
		t.Fatalf("failed to read saved solana keystore: %v", err)
	}

	parsed, err := FromJSON(saved)
	if err != nil {
		t.Fatalf("saved solana file is not a valid KeyStore V3: %v", err)
	}

	decrypted, err := DecryptPrivateKey(parsed, password)
	if err != nil {
		t.Fatalf("failed to decrypt saved solana keystore: %v", err)
	}

	if got := hex.EncodeToString(decrypted); got != privateKey {
		t.Errorf("decrypted key mismatch:\n got  %s\n want %s", got, privateKey)
	}
}

// TestNoArtifactClaimsEncryptionItDoesNotProvide guards the acceptance
// criterion that no written artifact may claim the key is encrypted when it is
// not. The old Solana .json carried exactly such a note.
func TestNoArtifactClaimsEncryptionItDoesNotProvide(t *testing.T) {
	dir := t.TempDir()
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: dir,
		KDF:             "scrypt",
	})

	privateKey := strings.Repeat("cd", 64)
	address := "aAoXxSjjj1SUZMQbwTnMW1MEiGeY6R21gJbCBTGUbmV"

	keystore, password, err := service.GenerateKeyStore(privateKey, address, "solana")
	if err != nil {
		t.Fatalf("GenerateKeyStore failed: %v", err)
	}
	if err := service.SaveKeyStoreFilesToDisk(address, keystore, password, "solana", privateKey); err != nil {
		t.Fatalf("SaveKeyStoreFilesToDisk failed: %v", err)
	}

	for _, name := range readDirNames(t, dir) {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("failed to read %s: %v", name, err)
		}
		if strings.Contains(string(content), "is encrypted in KeyStore V3 format") {
			t.Errorf("artifact %s still claims an encryption it does not provide", name)
		}
	}
}

// --- Issue #17: Bitcoin backup that cannot restore the wallet ---------------

// TestBitcoinMnemonicRestoresGeneratedWallet is the acceptance criterion for
// issue #17: the mnemonic attached to a wallet must derive that wallet's exact
// key and address.
func TestBitcoinMnemonicRestoresGeneratedWallet(t *testing.T) {
	generator := NewBitcoinGenerator(NewPoolManager(DefaultPoolConfig()))

	generated, err := generator.GenerateWalletFromMnemonic()
	if err != nil {
		t.Fatalf("GenerateWalletFromMnemonic failed: %v", err)
	}
	if generated.Mnemonic == "" {
		t.Fatal("expected a mnemonic on the generated wallet")
	}

	// Restore exactly the way a user would: from the saved phrase alone.
	restoredKey, err := BitcoinPrivateKeyFromMnemonic(generated.Mnemonic)
	if err != nil {
		t.Fatalf("failed to derive key from the saved mnemonic: %v", err)
	}

	if got := hex.EncodeToString(restoredKey); got != generated.PrivateKey {
		t.Errorf("mnemonic does not restore the generated private key:\n got  %s\n want %s",
			got, generated.PrivateKey)
	}

	restoredAddress, err := generator.GenerateAddressFromPrivateKey(restoredKey)
	if err != nil {
		t.Fatalf("failed to derive address from the restored key: %v", err)
	}
	if restoredAddress != generated.Address {
		t.Errorf("mnemonic does not restore the generated address:\n got  %s\n want %s",
			restoredAddress, generated.Address)
	}
}

// TestBitcoinDefaultWalletCarriesNoFakeMnemonic guards against the regression
// that produced a mnemonic unrelated to the key.
func TestBitcoinDefaultWalletCarriesNoFakeMnemonic(t *testing.T) {
	generator := NewBitcoinGenerator(NewPoolManager(DefaultPoolConfig()))

	generated, err := generator.GenerateWallet()
	if err != nil {
		t.Fatalf("GenerateWallet failed: %v", err)
	}

	if generated.Mnemonic != "" {
		t.Error("the fast Bitcoin path must not attach a mnemonic: it would not derive the key")
	}
}

// TestBitcoinKeystoreRestoresGeneratedWallet checks the default Bitcoin backup
// (an encrypted keystore) actually restores the wallet.
func TestBitcoinKeystoreRestoresGeneratedWallet(t *testing.T) {
	dir := t.TempDir()
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: dir,
		KDF:             "scrypt",
	})

	generator := NewBitcoinGenerator(NewPoolManager(DefaultPoolConfig()))
	generated, err := generator.GenerateWallet()
	if err != nil {
		t.Fatalf("GenerateWallet failed: %v", err)
	}

	keystore, password, err := service.GenerateKeyStore(generated.PrivateKey, generated.Address, "bitcoin")
	if err != nil {
		t.Fatalf("GenerateKeyStore for bitcoin failed: %v", err)
	}
	if err := service.SaveKeyStoreFilesToDisk(generated.Address, keystore, password, "bitcoin", generated.PrivateKey); err != nil {
		t.Fatalf("SaveKeyStoreFilesToDisk for bitcoin failed: %v", err)
	}

	saved, err := os.ReadFile(filepath.Join(dir, generated.Address+".json"))
	if err != nil {
		t.Fatalf("bitcoin keystore was not written: %v", err)
	}

	parsed, err := FromJSON(saved)
	if err != nil {
		t.Fatalf("saved bitcoin file is not a valid KeyStore V3: %v", err)
	}

	decrypted, err := DecryptPrivateKey(parsed, password)
	if err != nil {
		t.Fatalf("failed to decrypt saved bitcoin keystore: %v", err)
	}
	if got := hex.EncodeToString(decrypted); got != generated.PrivateKey {
		t.Errorf("bitcoin keystore does not restore the generated key:\n got  %s\n want %s",
			got, generated.PrivateKey)
	}
}

// --- Issue #18: configured KDF parameters and the security floor ------------

// TestConfiguredKDFParamsReachTheCipher is the acceptance criterion that
// --kdf-params must actually change the generated keystore.
func TestConfiguredKDFParamsReachTheCipher(t *testing.T) {
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: t.TempDir(),
		KDF:             "scrypt",
		KDFParams: map[string]interface{}{
			"n": 32768, "r": 8, "p": 1, "dklen": 32,
		},
	})

	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	address := "0x1234567890123456789012345678901234567890"

	keystore, _, err := service.GenerateKeyStore(privateKey, address, "ethereum")
	if err != nil {
		t.Fatalf("GenerateKeyStore failed: %v", err)
	}

	params, ok := keystore.Crypto.KDFParams.(ScryptParams)
	if !ok {
		t.Fatalf("expected ScryptParams, got %T", keystore.Crypto.KDFParams)
	}
	if params.N != 32768 {
		t.Errorf("configured n was ignored: got %d, want 32768", params.N)
	}
}

// TestConfiguredKDFParamsDoNotReuseSalt guards the copy in resolveKDFParams:
// the configured map is shared across keystores, so writing the salt into it
// would reuse one salt for every wallet in a run.
func TestConfiguredKDFParamsDoNotReuseSalt(t *testing.T) {
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: t.TempDir(),
		KDF:             "scrypt",
		KDFParams: map[string]interface{}{
			"n": 32768, "r": 8, "p": 1, "dklen": 32,
		},
	})

	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	address := "0x1234567890123456789012345678901234567890"

	salts := make(map[string]bool)
	for i := 0; i < 3; i++ {
		keystore, _, err := service.GenerateKeyStore(privateKey, address, "ethereum")
		if err != nil {
			t.Fatalf("GenerateKeyStore failed on iteration %d: %v", i, err)
		}
		params, ok := keystore.Crypto.KDFParams.(ScryptParams)
		if !ok {
			t.Fatalf("expected ScryptParams, got %T", keystore.Crypto.KDFParams)
		}
		if salts[params.Salt] {
			t.Fatalf("salt %s was reused across keystores", params.Salt)
		}
		salts[params.Salt] = true
	}
}

func TestScryptSecurityFloor(t *testing.T) {
	tests := []struct {
		name      string
		n, r      int
		wantError bool
	}{
		{"n=1024 r=1 (128 KiB) rejected", 1024, 1, true},
		{"n=1024 r=8 (1 MiB) rejected", 1024, 8, true},
		{"n=16384 r=1 (2 MiB) rejected", 16384, 1, true},
		{"n=16384 r=8 (16 MiB, low preset) accepted", 16384, 8, false},
		{"n=262144 r=8 (default) accepted", 262144, 8, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScryptSecurityFloor(tt.n, tt.r)
			if tt.wantError && err == nil {
				t.Errorf("expected n=%d r=%d to be rejected", tt.n, tt.r)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected n=%d r=%d to be accepted, got %v", tt.n, tt.r, err)
			}
		})
	}
}

func TestPBKDF2SecurityFloor(t *testing.T) {
	if err := ValidatePBKDF2SecurityFloor(1000); err == nil {
		t.Error("expected c=1000 to be rejected")
	}
	if err := ValidatePBKDF2SecurityFloor(120000); err != nil {
		t.Errorf("expected c=120000 (low preset) to be accepted, got %v", err)
	}
}

// TestWeakConfiguredParamsAreRejectedBeforeEncrypting makes sure the floor is
// enforced at the cipher, not only at the CLI flag parser.
func TestWeakConfiguredParamsAreRejectedBeforeEncrypting(t *testing.T) {
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: t.TempDir(),
		KDF:             "scrypt",
		KDFParams: map[string]interface{}{
			"n": 1024, "r": 1, "p": 1, "dklen": 32,
		},
	})

	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	address := "0x1234567890123456789012345678901234567890"

	if _, _, err := service.GenerateKeyStore(privateKey, address, "ethereum"); err == nil {
		t.Error("expected weak scrypt parameters to be rejected before encrypting")
	}
}

// TestFilePermIsClampedToOwnerOnly checks a loose configured mode cannot leak
// private key material to other local users.
func TestFilePermIsClampedToOwnerOnly(t *testing.T) {
	for _, perm := range []os.FileMode{0o644, 0o666, 0o777, 0} {
		service := NewKeyStoreService(KeyStoreConfig{
			Enabled:         true,
			OutputDirectory: t.TempDir(),
			FilePerm:        perm,
		})
		if got := service.GetConfig().FilePerm; got != DefaultKeyStoreFilePerm {
			t.Errorf("FilePerm %04o should be clamped to %04o, got %04o",
				perm, DefaultKeyStoreFilePerm, got)
		}
	}
}

// --- Issue #19: filesystem isolation ----------------------------------------

func TestOutputDirectoryIsOwnerOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keystores")
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: dir,
		KDF:             "scrypt",
	})

	if err := service.ensureOutputDirectory(); err != nil {
		t.Fatalf("ensureOutputDirectory failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("output directory was not created: %v", err)
	}
	if got := info.Mode().Perm(); got != KeyStoreDirPerm {
		t.Errorf("output directory mode: got %04o, want %04o", got, KeyStoreDirPerm)
	}
}

func TestCheckDirectoryPermissionsRejectsWorldWritable(t *testing.T) {
	dir := t.TempDir()
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: dir,
	})

	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatalf("failed to chmod test directory: %v", err)
	}

	if err := service.CheckDirectoryPermissions(); err == nil {
		t.Error("a world-writable output directory must be rejected")
	}

	if err := os.Chmod(dir, KeyStoreDirPerm); err != nil {
		t.Fatalf("failed to restore test directory mode: %v", err)
	}
	if err := service.CheckDirectoryPermissions(); err != nil {
		t.Errorf("an owner-only output directory must be accepted, got %v", err)
	}
}

func readDirNames(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read %s: %v", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
