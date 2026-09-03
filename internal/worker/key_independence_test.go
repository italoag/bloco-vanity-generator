package worker

import (
	"context"
	"math/big"
	"testing"
	"time"

	"bloco-vgen/pkg/wallet"
)

// minKeyDistance is the separation two independently generated secp256k1 keys
// must have. Keys derived from the same chain batch differ by a small integer
// (1, 2, 3, ...), so anything below this bound means one delivered key can be
// recovered from another by brute force.
var minKeyDistance = new(big.Int).Lsh(big.NewInt(1), 128)

// TestDeliveredKeysAreIndependent is the acceptance criterion for issue #15:
// wallets returned by the pool in one run must not be small offsets of each
// other. Before the fix every wallet in a run came from the same chain batch
// as k0, k0+1, k0+2, ...
func TestDeliveredKeysAreIndependent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-wallet generation in short mode")
	}

	const walletCount = 10

	pool := NewPool(2, "ethereum")
	if err := pool.Start(); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("failed to shut down pool: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// No pattern: every candidate matches, so this exercises the delivery path
	// rather than the search.
	criteria := wallet.GenerationCriteria{Network: "ethereum"}

	keys := make([]*big.Int, 0, walletCount)
	for i := 0; i < walletCount; i++ {
		result, err := pool.GenerateWalletWithContext(ctx, criteria)
		if err != nil {
			t.Fatalf("wallet %d failed: %v", i, err)
		}

		key, ok := new(big.Int).SetString(result.Wallet.PrivateKey, 16)
		if !ok {
			t.Fatalf("wallet %d has an unparsable private key %q", i, result.Wallet.PrivateKey)
		}
		keys = append(keys, key)
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			distance := new(big.Int).Sub(keys[i], keys[j])
			distance.Abs(distance)

			if distance.Sign() == 0 {
				t.Fatalf("wallets %d and %d have the same private key", i, j)
			}
			if distance.Cmp(minKeyDistance) < 0 {
				t.Errorf("wallets %d and %d are only %s apart: one key reveals the other",
					i, j, distance.String())
			}
		}
	}
}

// TestRetireChainReplacesTheWorkerChain checks the mechanism the independence
// property rests on: once a chain has delivered a key it is closed and a fresh
// one is built on next use.
func TestRetireChainReplacesTheWorkerChain(t *testing.T) {
	pool := NewPool(1, "ethereum")

	first, err := pool.getChain(0)
	if err != nil {
		t.Fatalf("getChain failed: %v", err)
	}

	same, err := pool.getChain(0)
	if err != nil {
		t.Fatalf("getChain failed: %v", err)
	}
	if same != first {
		t.Fatal("getChain should reuse the chain until it is retired")
	}

	pool.retireChain(0)

	replacement, err := pool.getChain(0)
	if err != nil {
		t.Fatalf("getChain after retire failed: %v", err)
	}
	if replacement == first {
		t.Error("retireChain must replace the chain, not keep the delivered one")
	}

	pool.closeChains()
}
