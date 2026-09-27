package backup

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"bloco-vgen/internal/crypto"
	"bloco-vgen/internal/crypto/kdf"
	"bloco-vgen/pkg/wallet"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	bip32 "github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

const FormatVersion = 1
const SupportedFnoxVersion = "1.35.2"
const MaxBundleBytes = 128 * 1024
const MaxArtifactBytes = 512 * 1024

const (
	keyEncodingHex    = "hex"
	keyOriginRandom   = "random"
	keyOriginBIP39    = "bip39"
	roleDerivation    = "derivation"
	roleUnrelated     = "unrelated"
	ethDerivationPath = "m/44'/60'/0'/0/0"
)

const (
	maxMnemonicLen   = 1024
	maxPassphraseLen = 1024
)

type Derivation struct {
	Path       string `json:"path"`
	Passphrase string `json:"passphrase"`
}

type Bundle struct {
	Version          int                `json:"version"`
	ID               string             `json:"id"`
	Network          string             `json:"network"`
	Address          string             `json:"address"`
	CreatedAt        time.Time          `json:"created_at"`
	KeyEncoding      string             `json:"key_encoding"`
	KeyOrigin        string             `json:"key_origin"`
	PrivateKey       string             `json:"private_key"`
	Mnemonic         string             `json:"mnemonic,omitempty"`
	MnemonicRole     string             `json:"mnemonic_role,omitempty"`
	Derivation       *Derivation        `json:"derivation,omitempty"`
	Keystore         *crypto.KeyStoreV3 `json:"keystore,omitempty"`
	KeystorePassword string             `json:"keystore_password,omitempty"`
}

func NewBundle(w *wallet.Wallet, keystore *crypto.KeyStoreV3, password string) (*Bundle, error) {
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}
	network := strings.ToLower(w.Network)
	if network == "" {
		network = "ethereum"
	}
	createdAt := w.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	b := &Bundle{
		Version:     FormatVersion,
		ID:          uuid.NewString(),
		Network:     network,
		CreatedAt:   createdAt,
		KeyEncoding: keyEncodingHex,
		PrivateKey:  strings.ToLower(strings.TrimPrefix(w.PrivateKey, "0x")),
	}
	if len(b.PrivateKey) > 128 {
		return nil, fmt.Errorf("invalid private key length")
	}
	keyBytes, err := hex.DecodeString(b.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key encoding")
	}
	defer func() {
		for i := range keyBytes {
			keyBytes[i] = 0
		}
	}()
	switch network {
	case "ethereum":
		priv, err := ethcrypto.ToECDSA(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("invalid private key")
		}
		canonical := ethcrypto.PubkeyToAddress(priv.PublicKey).Hex()
		if !strings.EqualFold(strings.TrimPrefix(w.Address, "0x"), strings.TrimPrefix(canonical, "0x")) {
			return nil, fmt.Errorf("private key does not match address")
		}
		b.Address = canonical
		if w.Mnemonic != "" {
			b.Mnemonic = w.Mnemonic
			b.MnemonicRole = roleDerivation
			b.KeyOrigin = keyOriginBIP39
			b.Derivation = &Derivation{Path: ethDerivationPath, Passphrase: ""}
		} else {
			b.KeyOrigin = keyOriginRandom
		}
		if keystore == nil || password == "" {
			return nil, fmt.Errorf("ethereum bundle requires keystore and password")
		}
		b.Keystore = keystore
		b.KeystorePassword = password
	case "bitcoin", "solana":
		if keystore != nil || password != "" {
			return nil, fmt.Errorf("keystore is only supported for ethereum")
		}
		b.Address = w.Address
		b.KeyOrigin = keyOriginRandom
		if network == "bitcoin" && w.Mnemonic != "" {
			b.Mnemonic = w.Mnemonic
			b.MnemonicRole = roleUnrelated
		}
		if network == "solana" && w.Mnemonic != "" {
			return nil, fmt.Errorf("solana bundles do not support mnemonics")
		}
	default:
		return nil, fmt.Errorf("unsupported network")
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Bundle) Filename() string {
	return fmt.Sprintf("%s-%s.fnox.toml", b.Network, b.Address)
}

func (b *Bundle) Validate() error {
	if b == nil {
		return fmt.Errorf("bundle is nil")
	}
	if b.Version != FormatVersion {
		return fmt.Errorf("unsupported bundle version")
	}
	if _, err := uuid.Parse(b.ID); err != nil {
		return fmt.Errorf("invalid bundle id")
	}
	if b.CreatedAt.IsZero() {
		return fmt.Errorf("missing creation time")
	}
	network := b.Network
	if network != "ethereum" && network != "bitcoin" && network != "solana" {
		return fmt.Errorf("unsupported network")
	}
	if b.KeyEncoding != keyEncodingHex {
		return fmt.Errorf("unsupported key encoding")
	}
	if len(b.Mnemonic) > maxMnemonicLen || len(b.KeystorePassword) > maxPassphraseLen {
		return fmt.Errorf("secret field too long")
	}
	wantHexLen := 64
	if network == "solana" {
		wantHexLen = 128
	}
	if len(strings.TrimPrefix(b.PrivateKey, "0x")) != wantHexLen {
		return fmt.Errorf("invalid private key length")
	}
	keyBytes, err := hex.DecodeString(strings.TrimPrefix(b.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key encoding")
	}
	defer func() {
		for i := range keyBytes {
			keyBytes[i] = 0
		}
	}()

	if err := b.validateMetadata(); err != nil {
		return err
	}

	switch network {
	case "ethereum":
		if err := b.validateEthereum(keyBytes); err != nil {
			return err
		}
		if err := b.validateKeystore(); err != nil {
			return err
		}
	case "bitcoin":
		if err := b.validateBitcoin(keyBytes); err != nil {
			return err
		}
	case "solana":
		if err := b.validateSolana(keyBytes); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bundle) validateMetadata() error {
	switch b.Network {
	case "ethereum":
		if b.Keystore == nil || b.KeystorePassword == "" {
			return fmt.Errorf("ethereum bundle requires keystore and password")
		}
		if b.Mnemonic != "" {
			if b.MnemonicRole != roleDerivation || b.KeyOrigin != keyOriginBIP39 {
				return fmt.Errorf("inconsistent mnemonic metadata")
			}
			if b.Derivation == nil || b.Derivation.Path != ethDerivationPath {
				return fmt.Errorf("inconsistent derivation metadata")
			}
			if len(b.Derivation.Passphrase) > maxPassphraseLen {
				return fmt.Errorf("secret field too long")
			}
		} else {
			if b.MnemonicRole != "" || b.Derivation != nil || b.KeyOrigin != keyOriginRandom {
				return fmt.Errorf("inconsistent key metadata")
			}
		}
	case "bitcoin":
		if b.Keystore != nil || b.KeystorePassword != "" || b.Derivation != nil {
			return fmt.Errorf("inconsistent keystore metadata")
		}
		if b.KeyOrigin != keyOriginRandom {
			return fmt.Errorf("inconsistent key origin")
		}
		if b.Mnemonic != "" {
			if b.MnemonicRole != roleUnrelated {
				return fmt.Errorf("inconsistent mnemonic metadata")
			}
		} else if b.MnemonicRole != "" {
			return fmt.Errorf("inconsistent mnemonic metadata")
		}
	case "solana":
		if b.Keystore != nil || b.KeystorePassword != "" || b.Derivation != nil {
			return fmt.Errorf("inconsistent keystore metadata")
		}
		if b.Mnemonic != "" || b.MnemonicRole != "" {
			return fmt.Errorf("solana bundles do not support mnemonics")
		}
		if b.KeyOrigin != keyOriginRandom {
			return fmt.Errorf("inconsistent key origin")
		}
	}
	return nil
}

func (b *Bundle) validateEthereum(keyBytes []byte) error {
	priv, err := ethcrypto.ToECDSA(keyBytes)
	if err != nil {
		return fmt.Errorf("invalid private key")
	}
	derived := ethcrypto.PubkeyToAddress(priv.PublicKey).Hex()
	if b.Address != derived {
		return fmt.Errorf("private key does not match address")
	}
	if b.Mnemonic == "" {
		return nil
	}
	if !bip39.IsMnemonicValid(b.Mnemonic) {
		return fmt.Errorf("invalid mnemonic")
	}
	seed := bip39.NewSeed(b.Mnemonic, b.Derivation.Passphrase)
	defer func() {
		for i := range seed {
			seed[i] = 0
		}
	}()
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return fmt.Errorf("failed to derive master key")
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
			return fmt.Errorf("failed to derive key")
		}
	}
	derivedKey, err := ethcrypto.ToECDSA(node.Key)
	if err != nil {
		return fmt.Errorf("failed to derive key")
	}
	if !derivedKey.Equal(priv) {
		return fmt.Errorf("mnemonic does not match private key")
	}
	return nil
}

func (b *Bundle) validateBitcoin(keyBytes []byte) error {
	if _, err := ethcrypto.ToECDSA(keyBytes); err != nil {
		return fmt.Errorf("invalid private key")
	}
	addr, err := crypto.NewBitcoinGenerator(nil).GenerateAddressFromPrivateKey(keyBytes)
	if err != nil {
		return fmt.Errorf("invalid private key")
	}
	if addr != b.Address {
		return fmt.Errorf("private key does not match address")
	}
	return nil
}

func (b *Bundle) validateSolana(keyBytes []byte) error {
	if !bytes.Equal(ed25519.NewKeyFromSeed(keyBytes[:32]), ed25519.PrivateKey(keyBytes)) {
		return fmt.Errorf("invalid private key")
	}
	addr, err := crypto.NewSolanaGenerator(nil).GenerateAddressFromPrivateKey(keyBytes)
	if err != nil {
		return fmt.Errorf("invalid private key")
	}
	if addr != b.Address {
		return fmt.Errorf("private key does not match address")
	}
	return nil
}

func (b *Bundle) validateKeystore() error {
	ks := b.Keystore
	if !strings.EqualFold(strings.TrimPrefix(ks.Address, "0x"), strings.TrimPrefix(b.Address, "0x")) {
		return fmt.Errorf("keystore address does not match bundle")
	}
	if ks.Crypto.Cipher != "aes-128-ctr" {
		return fmt.Errorf("unsupported keystore cipher")
	}
	iv, err := hex.DecodeString(ks.Crypto.CipherParams.IV)
	if err != nil || len(iv) != 16 {
		return fmt.Errorf("invalid keystore cipher parameters")
	}
	ciphertext, err := hex.DecodeString(ks.Crypto.CipherText)
	if err != nil || len(ciphertext) != 32 {
		return fmt.Errorf("invalid keystore ciphertext")
	}
	mac, err := hex.DecodeString(ks.Crypto.MAC)
	if err != nil || len(mac) != 32 {
		return fmt.Errorf("invalid keystore mac")
	}
	if err := validateKDFBounds(ks); err != nil {
		return err
	}
	cryptoParams, err := ks.ToKDFCryptoParams()
	if err != nil {
		return fmt.Errorf("invalid keystore parameters")
	}
	derivedKey, err := kdf.NewUniversalKDFService().DeriveKey(b.KeystorePassword, cryptoParams)
	if err != nil {
		return fmt.Errorf("keystore derivation failed")
	}
	defer func() {
		for i := range derivedKey {
			derivedKey[i] = 0
		}
	}()
	ok, err := crypto.VerifyMAC(derivedKey, ciphertext, mac)
	if err != nil || !ok {
		return fmt.Errorf("keystore integrity check failed")
	}
	plaintext, err := crypto.DecryptAES128CTR(ciphertext, derivedKey[:16], iv)
	if err != nil {
		return fmt.Errorf("keystore decryption failed")
	}
	defer func() {
		for i := range plaintext {
			plaintext[i] = 0
		}
	}()
	if !strings.EqualFold(hex.EncodeToString(plaintext), strings.TrimPrefix(b.PrivateKey, "0x")) {
		return fmt.Errorf("keystore does not match private key")
	}
	return nil
}

func validateKDFBounds(ks *crypto.KeyStoreV3) error {
	switch ks.Crypto.KDF {
	case "scrypt":
		params, err := ks.GetScryptParams()
		if err != nil {
			return fmt.Errorf("invalid keystore parameters")
		}
		if params.DKLen != 32 {
			return fmt.Errorf("invalid keystore parameters")
		}
		if params.N < 2 || params.N > 1<<20 || params.N&(params.N-1) != 0 {
			return fmt.Errorf("invalid keystore parameters")
		}
		if params.R < 1 || params.R > 32 || params.P < 1 || params.P > 16 {
			return fmt.Errorf("invalid keystore parameters")
		}
		if int64(params.N)*int64(params.R)*128 > 512*1024*1024 {
			return fmt.Errorf("invalid keystore parameters")
		}
		if int64(params.N)*int64(params.R)*int64(params.P) > 1<<26 {
			return fmt.Errorf("invalid keystore parameters")
		}
		salt, err := hex.DecodeString(params.Salt)
		if err != nil || len(salt) < 16 || len(salt) > 64 {
			return fmt.Errorf("invalid keystore parameters")
		}
	case "pbkdf2":
		params, err := ks.GetPBKDF2Params()
		if err != nil {
			return fmt.Errorf("invalid keystore parameters")
		}
		if params.DKLen != 32 || params.C < 1 || params.C > 2000000 {
			return fmt.Errorf("invalid keystore parameters")
		}
		if params.PRF != "hmac-sha256" && params.PRF != "hmac-sha512" {
			return fmt.Errorf("invalid keystore parameters")
		}
		salt, err := hex.DecodeString(params.Salt)
		if err != nil || len(salt) < 16 || len(salt) > 64 {
			return fmt.Errorf("invalid keystore parameters")
		}
	default:
		return fmt.Errorf("unsupported keystore kdf")
	}
	return nil
}
