package crypto

import (
	"crypto/rand"
	"encoding/hex"

	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/wallet"

	"github.com/ethereum/go-ethereum/crypto"
)

// AddressGenerator handles Ethereum address generation
// EthereumGenerator handles Ethereum address generation
type EthereumGenerator struct {
	poolManager *PoolManager
}

// NewEthereumGenerator creates a new address generator
func NewEthereumGenerator(poolManager *PoolManager) *EthereumGenerator {
	return &EthereumGenerator{
		poolManager: poolManager,
	}
}

// GenerateWallet generates a new wallet with private key and address
func (eg *EthereumGenerator) GenerateWallet() (*wallet.Wallet, error) {
	// Get private key buffer from pool
	cryptoPool := eg.poolManager.GetCryptoPool()
	privateKey := cryptoPool.GetPrivateKeyBuffer()
	defer cryptoPool.PutPrivateKeyBuffer(privateKey)

	// Generate 32 random bytes for private key
	_, err := rand.Read(privateKey)
	if err != nil {
		return nil, errors.NewCryptoError("generate_wallet",
			"failed to generate random private key", err)
	}

	// Generate address from private key
	address, err := eg.GenerateAddressFromPrivateKey(privateKey)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_wallet", "failed to generate address from private key")
	}

	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(privateKey),
	}, nil
}

// GenerateAddressFromPrivateKey converts a private key to an Ethereum address
func (eg *EthereumGenerator) GenerateAddressFromPrivateKey(privateKey []byte) (string, error) {
	// Use the optimized generation logic to avoid code duplication and improve performance
	addr, err := eg.OptimizedAddressGeneration(privateKey)
	if err != nil {
		return "", err
	}
	return "0x" + addr, nil
}

// GeneratePrivateKey generates a cryptographically secure private key
func (eg *EthereumGenerator) GeneratePrivateKey() ([]byte, error) {
	cryptoPool := eg.poolManager.GetCryptoPool()
	privateKey := cryptoPool.GetPrivateKeyBuffer()
	defer cryptoPool.PutPrivateKeyBuffer(privateKey)

	_, err := rand.Read(privateKey)
	if err != nil {
		return nil, errors.NewCryptoError("generate_private_key",
			"failed to generate random bytes", err)
	}

	// Create a copy to return (since we're putting the buffer back in the pool)
	result := make([]byte, 32)
	copy(result, privateKey)
	return result, nil
}

// ValidatePrivateKey validates that a private key is valid
func (eg *EthereumGenerator) ValidatePrivateKey(privateKey []byte) error {
	if len(privateKey) != 32 {
		return errors.NewValidationError("validate_private_key",
			"private key must be 32 bytes")
	}

	// Check if private key is zero (invalid)
	isZero := true
	for _, b := range privateKey {
		if b != 0 {
			isZero = false
			break
		}
	}

	if isZero {
		return errors.NewValidationError("validate_private_key",
			"private key cannot be zero")
	}

	// Additional validation could include checking if the key is within
	// the valid range for secp256k1 curve
	return nil
}

// BatchGenerateWallets generates multiple wallets efficiently
func (eg *EthereumGenerator) BatchGenerateWallets(count int) ([]*wallet.Wallet, error) {
	if count <= 0 {
		return nil, errors.NewValidationError("batch_generate_wallets",
			"count must be positive")
	}

	wallets := make([]*wallet.Wallet, 0, count)

	for i := 0; i < count; i++ {
		wallet, err := eg.GenerateWallet()
		if err != nil {
			return wallets, errors.WrapError(err, errors.ErrorTypeCrypto,
				"batch_generate_wallets", "failed to generate wallet in batch")
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

// OptimizedAddressGeneration performs optimized address generation for high-throughput scenarios
func (eg *EthereumGenerator) OptimizedAddressGeneration(privateKey []byte) (string, error) {
	// This is a more optimized version that reuses objects more efficiently
	// and minimizes allocations for high-performance scenarios

	if len(privateKey) != 32 {
		return "", errors.NewValidationError("optimized_address_generation",
			"private key must be 32 bytes")
	}

	// Get objects from pools
	cryptoPool := eg.poolManager.GetCryptoPool()
	hasherPool := eg.poolManager.GetHasherPool()
	bufferPool := eg.poolManager.GetBufferPool()

	privateKeyInt := cryptoPool.GetBigInt()
	hasher := hasherPool.GetKeccak()
	hexBuffer := bufferPool.GetHexBuffer()

	defer func() {
		cryptoPool.PutBigInt(privateKeyInt)
		hasherPool.PutKeccak(hasher)
		bufferPool.PutHexBuffer(hexBuffer)
	}()

	// Convert private key to big int
	privateKeyInt.SetBytes(privateKey)

	// Calculate public key coordinates directly
	x, y := crypto.S256().ScalarBaseMult(privateKey)

	// Create public key bytes directly without intermediate allocations
	hasher.Reset()

	// Write X coordinate (pad to 32 bytes)
	xBytes := x.Bytes()
	padding := 32 - len(xBytes)
	for i := 0; i < padding; i++ {
		hasher.Write([]byte{0})
	}
	hasher.Write(xBytes)

	// Write Y coordinate (pad to 32 bytes)
	yBytes := y.Bytes()
	padding = 32 - len(yBytes)
	for i := 0; i < padding; i++ {
		hasher.Write([]byte{0})
	}
	hasher.Write(yBytes)

	// Get hash and extract address
	hash := hasher.Sum(nil)
	address := hash[len(hash)-20:]

	// Encode to hex
	hex.Encode(hexBuffer, address)
	return string(hexBuffer[:40]), nil
}

// GetPoolManager returns the pool manager (for testing and optimization)
func (eg *EthereumGenerator) GetPoolManager() *PoolManager {
	return eg.poolManager
}
