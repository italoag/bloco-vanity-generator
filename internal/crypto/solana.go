package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"

	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/wallet"

	"github.com/gagliardetto/solana-go"
)

// SolanaGenerator handles Solana address generation
type SolanaGenerator struct {
	poolManager *PoolManager
}

// NewSolanaGenerator creates a new Solana address generator
func NewSolanaGenerator(poolManager *PoolManager) *SolanaGenerator {
	return &SolanaGenerator{
		poolManager: poolManager,
	}
}

// GenerateWallet generates a new wallet with private key and address
func (sg *SolanaGenerator) GenerateWallet() (*wallet.Wallet, error) {
	// Generate Ed25519 key pair
	// Solana uses Ed25519, which has 64-byte private keys (32 byte seed + 32 byte pub key)
	// But standard crypto/ed25519 GenerateKey returns the full 64 bytes.

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, errors.NewCryptoError("generate_wallet",
			"failed to generate random private key", err)
	}

	// Convert to Solana wallet
	// Solana private keys are usually represented as the 64-byte array

	address := solana.PublicKeyFromBytes(pub).String()

	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(priv),
	}, nil
}

// GenerateAddressFromPrivateKey converts a private key to a Solana address
func (sg *SolanaGenerator) GenerateAddressFromPrivateKey(privateKey []byte) (string, error) {
	// Expecting 64-byte Ed25519 private key
	if len(privateKey) != ed25519.PrivateKeySize {
		// If it's 32 bytes, we assume it's the seed
		if len(privateKey) == ed25519.SeedSize {
			priv := ed25519.NewKeyFromSeed(privateKey)
			pub := priv.Public().(ed25519.PublicKey)
			return solana.PublicKeyFromBytes(pub).String(), nil
		}
		return "", errors.NewValidationError("generate_address",
			"invalid private key length for Solana")
	}

	priv := ed25519.PrivateKey(privateKey)
	pub := priv.Public().(ed25519.PublicKey)
	return solana.PublicKeyFromBytes(pub).String(), nil
}
