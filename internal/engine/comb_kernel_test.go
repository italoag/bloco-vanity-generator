//go:build darwin && arm64 && cgo

package engine

import (
	"encoding/hex"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// TestCombKernelDumpPubkey reads the raw public keys derived by the full comb
// kernel for known private keys and compares them against decred/go-ethereum.
func TestCombKernelDumpPubkey(t *testing.T) {
	if !MetalAvailable() {
		t.Skip("metal backend unavailable in this build")
	}

	table, err := buildMetalCombTable()
	if err != nil {
		t.Fatal(err)
	}

	keys := make([]byte, 0, 32*11)
	for i := 1; i <= 8; i++ {
		k := make([]byte, 32)
		k[31] = byte(i)
		keys = append(keys, k...)
	}
	k256 := make([]byte, 32)
	k256[30] = 1
	keys = append(keys, k256...)
	k252 := make([]byte, 32)
	k252[0] = 0x10
	keys = append(keys, k252...)
	kAllFF := make([]byte, 32)
	for i := range kAllFF {
		kAllFF[i] = 0xff
	}
	keys = append(keys, kAllFF...)

	pubkeys, err := dumpMetalPubkeys(table, keys)
	if err != nil {
		t.Fatal(err)
	}

	totalKeys := len(keys) / 32
	failures := 0
	for i := 0; i < totalKeys; i++ {
		var kb [32]byte
		copy(kb[:], keys[i*32:(i+1)*32])
		var ks secp256k1.ModNScalar
		ks.SetByteSlice(kb[:])
		var ref secp256k1.JacobianPoint
		secp256k1.ScalarBaseMultNonConst(&ks, &ref)
		ref.ToAffine()
		var want [64]byte
		copy(want[:32], ref.X.Bytes()[:])
		copy(want[32:], ref.Y.Bytes()[:])
		got := pubkeys[i*64 : (i+1)*64]
		if string(got) != string(want[:]) {
			t.Logf("key %d (k=%x): kernel pubkey %s, want %s", i, kb, hex.EncodeToString(got), hex.EncodeToString(want[:]))
			failures++
		}
	}
	if failures > 0 {
		t.Fatalf("%d/%d kernel pubkeys mismatch", failures, totalKeys)
	}
	t.Log("kernel pubkeys match for all test keys")
}
