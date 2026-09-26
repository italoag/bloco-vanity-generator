package crypto

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

func TestCryptoPoolECDSAKeyClearing(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(func() string {
			if enabled {
				return "clearing_enabled"
			}
			return "clearing_disabled"
		}(), func(t *testing.T) {
			key, err := crypto.ToECDSA(bytes.Repeat([]byte{1}, 32))
			if err != nil {
				t.Fatal(err)
			}
			scalar := ecies.ImportECDSA(key).D
			if scalar == nil {
				t.Fatal("expected shared scalar big.Int")
			}
			words := scalar.Bits()
			wantScalar := new(big.Int).Set(scalar)
			curve := key.Curve

			pool := NewCryptoPool(PoolConfig{EnableClearing: enabled})
			pool.PutECDSAKey(key)

			if enabled {
				if scalar.Sign() != 0 {
					t.Fatal("expected scalar to be cleared")
				}
				for i, w := range words {
					if w != 0 {
						t.Fatalf("word %d not cleared: %d", i, w)
					}
				}
				if d := ecies.ImportECDSA(key).D; d != nil {
					t.Fatal("expected key D to be nil")
				}
				if key.X != nil || key.Y != nil {
					t.Fatal("expected X/Y to be cleared")
				}
				if key.Curve != curve {
					t.Fatal("expected Curve to be preserved")
				}
			} else {
				if scalar.Cmp(wantScalar) != 0 {
					t.Fatal("expected scalar to be preserved")
				}
				if d := ecies.ImportECDSA(key).D; d == nil || d.Cmp(wantScalar) != 0 {
					t.Fatal("expected key to remain intact")
				}
			}
		})
	}
}

func TestCryptoPoolGetECDSAKeyResets(t *testing.T) {
	pool := NewCryptoPool(PoolConfig{EnableClearing: true})
	synthetic, err := crypto.ToECDSA(bytes.Repeat([]byte{2}, 32))
	if err != nil {
		t.Fatal(err)
	}
	pool.ecdsaKeyPool.New = func() interface{} { return synthetic }

	key := pool.GetECDSAKey()
	if key.Curve != crypto.S256() {
		t.Fatal("expected Curve to be preserved")
	}
	if d := ecies.ImportECDSA(key).D; d != nil {
		t.Fatal("expected D to be nil after reset")
	}
	if key.X != nil || key.Y != nil {
		t.Fatal("expected X/Y to be nil after reset")
	}
}

func TestCryptoPoolPutECDSAKeyEmpty(t *testing.T) {
	pool := NewCryptoPool(PoolConfig{EnableClearing: true})
	pool.PutECDSAKey(pool.GetECDSAKey())
}
