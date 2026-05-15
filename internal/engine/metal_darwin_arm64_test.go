//go:build darwin && arm64 && cgo

package engine

import (
	"encoding/hex"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

func TestMetalKernelDerivesEthereumAddressFromPrivateKeyWhenAvailable(t *testing.T) {
	if !MetalAvailable() {
		t.Skip("metal backend unavailable in this build")
	}

	benchmarkEngine, err := NewMetalEngine()
	if err != nil {
		t.Fatalf("expected metal engine, got %v", err)
	}
	metalEngine := benchmarkEngine.(*MetalEngine)
	defer metalEngine.Close()

	privateKeys := make([]byte, 32*5)
	for i := 0; i < 4; i++ {
		privateKeys[i*32+31] = byte(i + 1)
	}
	for i := 0; i < 32; i++ {
		privateKeys[4*32+i] = byte(i + 1)
	}
	for i := 0; i < 5; i++ {
		privateKey := privateKeys[i*32 : (i+1)*32]
		x, y := ethcrypto.S256().ScalarBaseMult(privateKey)
		var publicKey [64]byte
		x.FillBytes(publicKey[:32])
		y.FillBytes(publicKey[32:])

		address := EthereumAddressBytesFromPublicKey(publicKey[:], sha3.NewLegacyKeccak256())
		prefix, err := patternNibbles(hex.EncodeToString(address[:]))
		if err != nil {
			t.Fatalf("unexpected prefix error for scalar %d: %v", i+1, err)
		}

		matches, _, kernelDuration, err := runMetalMatch(metalEngine.context, privateKeys, prefix, nil)
		if err != nil {
			t.Fatalf("expected no metal match error for scalar %d, got %v", i+1, err)
		}
		if matches != 1 {
			t.Fatalf("expected one full-address match for scalar %d, got %d", i+1, matches)
		}
		if kernelDuration <= 0 {
			t.Fatalf("expected kernel duration to be recorded for scalar %d", i+1)
		}
	}
}

func TestMetalPrivateKeyBatchCountValidation(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := metalPrivateKeyBatchCount(nil, 1)
		if err == nil || !strings.Contains(err.Error(), "at least one private key") {
			t.Fatalf("expected empty private key error, got %v", err)
		}
	})

	t.Run("non multiple of private key size", func(t *testing.T) {
		_, err := metalPrivateKeyBatchCount(make([]byte, 31), 1)
		if err == nil || !strings.Contains(err.Error(), "multiple of 32 bytes") {
			t.Fatalf("expected private key size error, got %v", err)
		}
	})

	t.Run("non positive maximum", func(t *testing.T) {
		_, err := metalPrivateKeyBatchCount(make([]byte, 32), 0)
		if err == nil || !strings.Contains(err.Error(), "maximum batch size") {
			t.Fatalf("expected maximum batch size error, got %v", err)
		}
	})

	t.Run("too large", func(t *testing.T) {
		_, err := metalPrivateKeyBatchCount(make([]byte, 64), 1)
		if err == nil || !strings.Contains(err.Error(), "batch too large") {
			t.Fatalf("expected batch too large error, got %v", err)
		}
	})

	t.Run("valid", func(t *testing.T) {
		count, err := metalPrivateKeyBatchCount(make([]byte, 64), 2)
		if err != nil {
			t.Fatalf("expected valid batch, got %v", err)
		}
		if count != 2 {
			t.Fatalf("expected count 2, got %d", count)
		}
	})
}
