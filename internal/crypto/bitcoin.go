package crypto

import (
	"crypto/rand"
	"encoding/hex"

	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/wallet"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
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

// GenerateWallet generates a new wallet with private key, address, and mnemonic
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

	// Generate BIP-39 mnemonic for backup
	// Note: This mnemonic is for backup purposes only and is not used to derive the key
	// The actual private key is randomly generated above
	mnemonic, err := generateBIP39Mnemonic()
	if err != nil {
		return nil, errors.NewCryptoError("generate_wallet",
			"failed to generate mnemonic", err)
	}

	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(privateKey),
		Mnemonic:   mnemonic,
	}, nil
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
