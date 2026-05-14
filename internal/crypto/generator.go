package crypto

import "bloco-vgen/pkg/wallet"

// Generator defines the interface for wallet generation
type Generator interface {
	// GenerateWallet generates a new wallet with private key and address
	GenerateWallet() (*wallet.Wallet, error)

	// GenerateAddressFromPrivateKey generates an address from a private key bytes
	GenerateAddressFromPrivateKey(privateKey []byte) (string, error)
}
