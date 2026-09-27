package backup

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bloco-vgen/internal/crypto"
	"bloco-vgen/pkg/wallet"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	bip32 "github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

const testPassword = "Synthetic-Backup-Test-Password1"

func ethKeyFromMnemonic(t *testing.T, mnemonic, passphrase string) string {
	t.Helper()
	seed := bip39.NewSeed(mnemonic, passphrase)
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
	node := masterKey
	for _, child := range path {
		node, err = node.NewChildKey(child)
		if err != nil {
			t.Fatal(err)
		}
	}
	key, err := ethcrypto.ToECDSA(node.Key)
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(ethcrypto.FromECDSA(key))
}

func ethereumBundle(t *testing.T, withMnemonic bool) *Bundle {
	t.Helper()
	var privHex, mnemonic string
	if withMnemonic {
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			t.Fatal(err)
		}
		mnemonic, err = bip39.NewMnemonic(entropy)
		if err != nil {
			t.Fatal(err)
		}
		privHex = ethKeyFromMnemonic(t, mnemonic, "")
	} else {
		key, err := ethcrypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		privHex = hex.EncodeToString(ethcrypto.FromECDSA(key))
	}
	keyBytes, err := hex.DecodeString(privHex)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ethcrypto.ToECDSA(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	address := ethcrypto.PubkeyToAddress(priv.PublicKey).Hex()
	ks, err := crypto.EncryptPrivateKey(privHex, testPassword, "pbkdf2")
	if err != nil {
		t.Fatal(err)
	}
	ks.Address = strings.ToLower(strings.TrimPrefix(address, "0x"))
	w := &wallet.Wallet{
		Address:    address,
		PrivateKey: privHex,
		Mnemonic:   mnemonic,
		Network:    "ethereum",
		CreatedAt:  time.Now().UTC(),
	}
	b, err := NewBundle(w, ks, testPassword)
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}
	return b
}

func bitcoinBundle(t *testing.T, withMnemonic bool) *Bundle {
	t.Helper()
	keyBytes := make([]byte, 32)
	for {
		if _, err := rand.Read(keyBytes); err != nil {
			t.Fatal(err)
		}
		if _, err := ethcrypto.ToECDSA(keyBytes); err == nil {
			break
		}
	}
	address, err := crypto.NewBitcoinGenerator(nil).GenerateAddressFromPrivateKey(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	mnemonic := ""
	if withMnemonic {
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			t.Fatal(err)
		}
		mnemonic, err = bip39.NewMnemonic(entropy)
		if err != nil {
			t.Fatal(err)
		}
	}
	w := &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(keyBytes),
		Mnemonic:   mnemonic,
		Network:    "bitcoin",
		CreatedAt:  time.Now().UTC(),
	}
	b, err := NewBundle(w, nil, "")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}
	return b
}

func solanaBundle(t *testing.T) *Bundle {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	address, err := crypto.NewSolanaGenerator(nil).GenerateAddressFromPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	w := &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(priv),
		Network:    "solana",
		CreatedAt:  time.Now().UTC(),
	}
	b, err := NewBundle(w, nil, "")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}
	return b
}

func TestNewBundleNetworks(t *testing.T) {
	for _, multi := range []bool{false, true} {
		b := ethereumBundle(t, multi)
		if b.Network != "ethereum" || b.KeyEncoding != "hex" {
			t.Fatal("bad ethereum bundle")
		}
		if got := b.Filename(); !strings.HasPrefix(got, "ethereum-") || !strings.HasSuffix(got, ".fnox.toml") {
			t.Fatalf("bad filename: %s", got)
		}
		if multi {
			if b.KeyOrigin != "bip39" || b.MnemonicRole != "derivation" || b.Derivation == nil {
				t.Fatal("bad mnemonic metadata")
			}
		} else {
			if b.KeyOrigin != "random" || b.MnemonicRole != "" || b.Derivation != nil {
				t.Fatal("bad random metadata")
			}
		}
	}
	bb := bitcoinBundle(t, true)
	if bb.KeyOrigin != "random" || bb.MnemonicRole != "unrelated" || bb.Derivation != nil {
		t.Fatal("bad bitcoin mnemonic metadata")
	}
	sb := solanaBundle(t)
	if sb.KeyOrigin != "random" || sb.Mnemonic != "" {
		t.Fatal("bad solana metadata")
	}
}

func TestNewBundleCanonicalAddressFilename(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	privHex := hex.EncodeToString(ethcrypto.FromECDSA(key))
	checksum := ethcrypto.PubkeyToAddress(key.PublicKey).Hex()
	ks, err := crypto.EncryptPrivateKey(privHex, testPassword, "pbkdf2")
	if err != nil {
		t.Fatal(err)
	}
	ks.Address = strings.ToLower(strings.TrimPrefix(checksum, "0x"))
	for _, addr := range []string{checksum, strings.ToLower(strings.TrimPrefix(checksum, "0x")), strings.ToLower(checksum), strings.TrimPrefix(checksum, "0x")} {
		b, err := NewBundle(&wallet.Wallet{Address: addr, PrivateKey: privHex, Network: "ethereum"}, ks, testPassword)
		if err != nil {
			t.Fatalf("NewBundle(%s) failed: %v", addr, err)
		}
		if b.Address != checksum {
			t.Fatalf("address %s not canonicalized to %s", addr, checksum)
		}
		if b.Filename() != "ethereum-"+checksum+".fnox.toml" {
			t.Fatalf("filename mismatch: %s", b.Filename())
		}
	}
}

func TestBundleValidateRejects(t *testing.T) {
	t.Run("address_mismatch", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.Address = strings.Repeat("0", 40)
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("wrong_key", func(t *testing.T) {
		b := ethereumBundle(t, false)
		key, err := ethcrypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		b.PrivateKey = hex.EncodeToString(ethcrypto.FromECDSA(key))
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("altered_ed25519_half", func(t *testing.T) {
		b := solanaBundle(t)
		keyBytes, err := hex.DecodeString(b.PrivateKey)
		if err != nil {
			t.Fatal(err)
		}
		keyBytes[63] ^= 0x01
		b.PrivateKey = hex.EncodeToString(keyBytes)
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("bitcoin_mnemonic_not_derivation", func(t *testing.T) {
		b := bitcoinBundle(t, true)
		b.MnemonicRole = "derivation"
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing_keystore", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.Keystore = nil
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing_password", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.KeystorePassword = ""
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("wrong_password", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.KeystorePassword = "wrong-password"
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("wrong_derivation_path", func(t *testing.T) {
		b := ethereumBundle(t, true)
		b.Derivation.Path = "m/44'/60'/0'/0/1"
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("wrong_passphrase", func(t *testing.T) {
		b := ethereumBundle(t, true)
		b.Derivation.Passphrase = "other"
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("future_version", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.Version = 2
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("bad_id", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.ID = "not-a-uuid"
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("zero_created", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.CreatedAt = time.Time{}
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("keystore_on_bitcoin", func(t *testing.T) {
		b := bitcoinBundle(t, false)
		b.Keystore = &crypto.KeyStoreV3{}
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("solana_mnemonic", func(t *testing.T) {
		b := solanaBundle(t)
		b.Mnemonic = "abandon abandon abandon"
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("oversize_mnemonic", func(t *testing.T) {
		b := ethereumBundle(t, true)
		b.Mnemonic = strings.Repeat("abandon ", 200)
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("kdf_resource_bound", func(t *testing.T) {
		b := ethereumBundle(t, false)
		b.Keystore.SetScryptParams(1<<21, 8, 1, 32, make([]byte, 32))
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("ethereum_mnemonic_validates_derivation", func(t *testing.T) {
		b := ethereumBundle(t, true)
		key, err := ethcrypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		b.PrivateKey = hex.EncodeToString(ethcrypto.FromECDSA(key))
		if err := b.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDecodeBundleStrict(t *testing.T) {
	b := ethereumBundle(t, true)
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeBundle(data); err != nil {
		t.Fatalf("valid bundle rejected: %v", err)
	}
	t.Run("unknown_field", func(t *testing.T) {
		mutated := strings.Replace(string(data), `"version"`, `"unknown_field":1,"version"`, 1)
		if _, err := decodeBundle([]byte(mutated)); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("trailing", func(t *testing.T) {
		if _, err := decodeBundle(append(data, ' ', '{', '}')); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("oversize", func(t *testing.T) {
		if _, err := decodeBundle(make([]byte, MaxBundleBytes+1)); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestBitcoinBundleMnemonicValidation(t *testing.T) {
	t.Run("invalid_checksum_rejected", func(t *testing.T) {
		b := bitcoinBundle(t, true)
		b.Mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
		if err := b.Validate(); err == nil {
			t.Fatal("invalid checksum must fail validation")
		}
	})
	t.Run("nonwordlist_rejected", func(t *testing.T) {
		b := bitcoinBundle(t, true)
		b.Mnemonic = "notaword abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
		if err := b.Validate(); err == nil {
			t.Fatal("non-wordlist mnemonic must fail validation")
		}
	})
	t.Run("empty_accepted", func(t *testing.T) {
		b := bitcoinBundle(t, false)
		if err := b.Validate(); err != nil {
			t.Fatalf("empty mnemonic must stay valid: %v", err)
		}
	})
	t.Run("valid_unrelated_accepted", func(t *testing.T) {
		b := bitcoinBundle(t, true)
		if err := b.Validate(); err != nil {
			t.Fatalf("valid unrelated mnemonic must pass: %v", err)
		}
	})
}
