//go:build darwin && arm64 && cgo

package engine

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

// TestCombTableMatchesGoEthereum verifies every table entry against geth.
func TestCombTableMatchesGoEthereum(t *testing.T) {
	table, err := buildMetalCombTable()
	if err != nil {
		t.Fatal(err)
	}
	order := secp256k1.S256().Params().N
	checked := 0
	for w := 0; w < 64; w++ {
		for j := 0; j < 16; j++ {
			s := new(big.Int).Lsh(big.NewInt(int64(j)), uint(4*w))
			s.Mod(s, order)
			base := (w*16 + j) * 12 * 8
			x := readCombField(table, base)
			y := readCombField(table, base+4*8)
			z := readCombField(table, base+8*8)
			if s.Sign() == 0 {
				if x.Sign() != 0 || y.Sign() != 0 || z.Sign() != 0 {
					t.Fatalf("w=%d j=%d expected infinity, got x=%x y=%x z=%x", w, j, x, y, z)
				}
				continue
			}
			if z.Cmp(big.NewInt(1)) != 0 {
				t.Fatalf("w=%d j=%d expected z=1, got %x", w, j, z)
			}
			var kBytes [32]byte
			s.FillBytes(kBytes[:])
			xRef, yRef := ethcrypto.S256().ScalarBaseMult(kBytes[:])
			if x.Cmp(xRef) != 0 || y.Cmp(yRef) != 0 {
				t.Fatalf("w=%d j=%d (s=%d): table x=%x y=%x, want x=%x y=%x", w, j, s, x, y, xRef, yRef)
			}
			checked++
		}
	}
	t.Logf("verified %d non-infinity comb entries against go-ethereum", checked)
}

func readCombField(table []byte, offset int) *big.Int {
	var b [32]byte
	for i := 0; i < 4; i++ {
		limb := binary.LittleEndian.Uint64(table[offset+i*8:])
		binary.BigEndian.PutUint64(b[8*(3-i):], limb)
	}
	return new(big.Int).SetBytes(b[:])
}

// TestCombAccumulationOrder verifies the comb accumulation algorithm on the
// CPU using the same table and order as the kernel.
func TestCombAccumulationOrder(t *testing.T) {
	table, err := buildMetalCombTable()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []uint64{1, 2, 3, 16, 17, 255, 256, 65536, 0x100000000, 0xabcdef1234567890} {
		var kb [32]byte
		var kb8 [8]byte
		binary.BigEndian.PutUint64(kb8[:], k)
		copy(kb[24:], kb8[:])
		kBig := new(big.Int).SetBytes(kb[:])

		var acc secp256k1.JacobianPoint
		loaded := false
		for w := 63; w >= 0; w-- {
			// window w = bits 4w..4w+3
			shift := uint(4 * w)
			window := new(big.Int).Rsh(new(big.Int).Set(kBig), shift)
			window.And(window, big.NewInt(0xf))
			j := int(window.Int64())
			base := (w*16 + j) * 12 * 8
			x := readCombField(table, base)
			y := readCombField(table, base+4*8)
			z := readCombField(table, base+8*8)
			var point secp256k1.JacobianPoint
			var xb, yb, zb [32]byte
			x.FillBytes(xb[:])
			y.FillBytes(yb[:])
			z.FillBytes(zb[:])
			point.X.SetByteSlice(xb[:])
			point.Y.SetByteSlice(yb[:])
			point.Z.SetByteSlice(zb[:])
			if !loaded {
				acc.Set(&point)
				loaded = true
			} else {
				var result secp256k1.JacobianPoint
				secp256k1.AddNonConst(&acc, &point, &result)
				acc.Set(&result)
			}
		}
		acc.ToAffine()
		xRef, yRef := ethcrypto.S256().ScalarBaseMult(kb[:])
		if acc.X.Normalize().Bytes() == nil || new(big.Int).SetBytes(acc.X.Bytes()[:]).Cmp(xRef) != 0 ||
			new(big.Int).SetBytes(acc.Y.Bytes()[:]).Cmp(yRef) != 0 {
			t.Fatalf("k=%d: comb accumulation mismatch: got (%x, %x) want (%x, %x)",
				k, acc.X.Bytes(), acc.Y.Bytes(), xRef, yRef)
		}
	}
	t.Log("comb accumulation order verified on CPU")
}
func TestCombKernelSingleKey(t *testing.T) {
	if !MetalAvailable() {
		t.Skip("metal backend unavailable in this build")
	}
	benchmarkEngine, err := NewMetalEngine()
	if err != nil {
		t.Fatalf("expected metal engine, got %v", err)
	}
	metalEngine := benchmarkEngine.(*MetalEngine)
	defer metalEngine.Close()

	for _, k := range []uint64{1, 2, 3, 16, 17, 255, 256, 65536, 0x100000000} {
		t.Run(fmt.Sprintf("k=%d", k), func(t *testing.T) {
			privateKeys := make([]byte, 32)
			var kb [8]byte
			binary.BigEndian.PutUint64(kb[:], k)
			copy(privateKeys[24:], kb[:])
			x, y := ethcrypto.S256().ScalarBaseMult(privateKeys)
			var publicKey [64]byte
			x.FillBytes(publicKey[:32])
			y.FillBytes(publicKey[32:])
			address := EthereumAddressBytesFromPublicKey(publicKey[:], sha3.NewLegacyKeccak256())
			prefix, err := patternNibbles(hex.EncodeToString(address[:]))
			if err != nil {
				t.Fatal(err)
			}

			matches, indices, _, _, err := runMetalMatch(metalEngine.context, privateKeys, prefix, nil)
			if err != nil {
				t.Fatal(err)
			}
			if matches != 1 {
				t.Fatalf("expected 1 match for k=%d, got %d (indices %v)", k, matches, indices)
			}
		})
	}
}
