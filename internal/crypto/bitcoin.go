package crypto

import (
	"crypto/rand"
	"encoding/hex"

	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/wallet"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// BitcoinGenerator handles Bitcoin address generation
type BitcoinGenerator struct {
	poolManager *PoolManager
	params      *chaincfg.Params
}

// NewBitcoinGenerator creates a new Bitcoin address generator
func NewBitcoinGenerator(poolManager *PoolManager) *BitcoinGenerator {
	return &BitcoinGenerator{
		poolManager: poolManager,
		params:      &chaincfg.MainNetParams,
	}
}

// BitcoinDerivationPath is the BIP-44 path used when a Bitcoin wallet is
// generated from a mnemonic: m/44'/0'/0'/0/0.
const BitcoinDerivationPath = "m/44'/0'/0'/0/0"

// GenerateWallet generates a new wallet from a random private key.
//
// It deliberately does NOT attach a mnemonic. Earlier versions generated a
// BIP-39 mnemonic that had no mathematical relationship to the key and then
// persisted only that mnemonic, so the saved "backup" could never restore the
// wallet. The key is now persisted in an encrypted keystore instead; callers
// that want a mnemonic-backed wallet must use GenerateWalletFromMnemonic.
func (bg *BitcoinGenerator) GenerateWallet() (*wallet.Wallet, error) {
	// Get private key buffer from pool
	cryptoPool := bg.poolManager.GetCryptoPool()
	privateKey := cryptoPool.GetPrivateKeyBuffer()
	defer cryptoPool.PutPrivateKeyBuffer(privateKey)

	// Generate 32 random bytes for private key
	_, err := rand.Read(privateKey)
	if err != nil {
		return nil, errors.NewCryptoError("generate_wallet",
			"failed to generate random private key", err)
	}

	// Generate address from private key
	address, err := bg.GenerateAddressFromPrivateKey(privateKey)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_wallet", "failed to generate address from private key")
	}

	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(privateKey),
		Network:    "bitcoin",
	}, nil
}

// GenerateWalletFromMnemonic generates a wallet whose private key is derived
// from a fresh BIP-39 mnemonic through BIP-32 at BitcoinDerivationPath, so the
// mnemonic is a real backup of the returned wallet.
func (bg *BitcoinGenerator) GenerateWalletFromMnemonic() (*wallet.Wallet, error) {
	mnemonic, err := generateBIP39Mnemonic()
	if err != nil {
		return nil, errors.NewCryptoError("generate_wallet",
			"failed to generate mnemonic", err)
	}

	privateKey, err := BitcoinPrivateKeyFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}

	address, err := bg.GenerateAddressFromPrivateKey(privateKey)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_wallet", "failed to generate address from private key")
	}

	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(privateKey),
		Mnemonic:   mnemonic,
		Network:    "bitcoin",
	}, nil
}

// BitcoinPrivateKeyFromMnemonic derives the Bitcoin private key a mnemonic
// backs up, at BitcoinDerivationPath. Restoring a wallet saved by this tool is
// exactly this function applied to the saved mnemonic.
func BitcoinPrivateKeyFromMnemonic(mnemonic string) ([]byte, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.NewValidationError("derive_private_key",
			"mnemonic is not a valid BIP-39 phrase")
	}

	seed := bip39.NewSeed(mnemonic, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, errors.NewCryptoError("derive_private_key",
			"failed to derive BIP-32 master key", err)
	}

	// m/44'/0'/0'/0/0 - BIP-44 for Bitcoin mainnet, first external address.
	path := []uint32{
		bip32.FirstHardenedChild + 44,
		bip32.FirstHardenedChild + 0,
		bip32.FirstHardenedChild + 0,
		0,
		0,
	}

	key := masterKey
	for _, index := range path {
		key, err = key.NewChildKey(index)
		if err != nil {
			return nil, errors.NewCryptoError("derive_private_key",
				"failed to derive BIP-32 child key", err)
		}
	}

	return key.Key, nil
}

// GenerateAddressFromPrivateKey converts a private key to a Bitcoin address
func (bg *BitcoinGenerator) GenerateAddressFromPrivateKey(privateKey []byte) (string, error) {
	privKey, pubKey := btcec.PrivKeyFromBytes(privateKey)
	_ = privKey // Not used directly, we use pubKey

	// Create address pub key hash (P2PKH)
	// Note: We are using uncompressed public keys for compatibility,
	// but compressed is standard now. Let's use compressed.
	addrPubKey, err := btcutil.NewAddressPubKey(pubKey.SerializeCompressed(), bg.params)
	if err != nil {
		return "", errors.NewCryptoError("generate_address",
			"failed to create address pub key", err)
	}

	return addrPubKey.AddressPubKeyHash().EncodeAddress(), nil
}

// generateBIP39Mnemonic generates a 12-word BIP-39 mnemonic phrase
func generateBIP39Mnemonic() (string, error) {
	// Generate 128 bits of entropy for a 12-word mnemonic
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", err
	}

	// Convert entropy to mnemonic using BIP-39
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}
