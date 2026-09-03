package crypto

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

// TestKeccak256Of64MatchesSHA3 validates the raw Keccak-256 implementation
// against golang.org/x/crypto/sha3 on many random 64-byte inputs.
func TestKeccak256Of64MatchesSHA3(t *testing.T) {
	reference := sha3.NewLegacyKeccak256()
	for i := 0; i < 2000; i++ {
		var input [64]byte
		if _, err := rand.Read(input[:]); err != nil {
			t.Fatal(err)
		}
		got := keccak256Of64(&input)

		reference.Reset()
		reference.Write(input[:])
		var want [32]byte
		reference.Sum(want[:0])
		if got != want {
			t.Fatalf("keccak mismatch on input %d: got %x want %x", i, got, want)
		}
	}

	// Edge cases
	edges := [][64]byte{}
	var allZero [64]byte
	edges = append(edges, allZero)
	var allFF [64]byte
	for i := range allFF {
		allFF[i] = 0xff
	}
	edges = append(edges, allFF)
	var one [64]byte
	one[0] = 1
	edges = append(edges, one)
	for _, input := range edges {
		got := keccak256Of64(&input)
		reference.Reset()
		reference.Write(input[:])
		var want [32]byte
		reference.Sum(want[:0])
		if got != want {
			t.Fatalf("keccak mismatch on edge case %x: got %x want %x", input, got, want)
		}
	}
}

// TestKeccak256Of64KnownVector checks a fixed known vector.
func TestKeccak256Of64KnownVector(t *testing.T) {
	// keccak256 of 64 zero bytes, computed with the reference implementation.
	var input [64]byte
	got := keccak256Of64(&input)
	reference := sha3.NewLegacyKeccak256()
	reference.Write(input[:])
	var want [32]byte
	reference.Sum(want[:0])
	if got != want {
		t.Fatalf("known vector mismatch: got %x want %x", got, want)
	}
}

// TestPrivateKeyChainMatchesGoEthereum walks chains and verifies every key
// against go-ethereum's ScalarBaseMult and Keccak address derivation.
func TestPrivateKeyChainMatchesGoEthereum(t *testing.T) {
	order := ethcrypto.S256().Params().N
	for seed := 0; seed < 3; seed++ {
		chain, err := NewPrivateKeyChain()
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 128; i++ {
			key, pub, err := chain.NextKey()
			if err != nil {
				t.Fatal(err)
			}
			address := AddressFromPublicKey(&pub)

			// Key must be in [1, N-1]
			keyBig := new(big.Int).SetBytes(key[:])
			if keyBig.Sign() <= 0 || keyBig.Cmp(order) >= 0 {
				t.Fatalf("seed %d key %d out of range: %x", seed, i, key)
			}

			// Expected key: k0 + i
			seedBytes := chain.Seed()
			var expected [32]byte
			expectedBig := new(big.Int).Add(new(big.Int).SetBytes(seedBytes[:]), big.NewInt(int64(i)))
			if expectedBig.Cmp(order) >= 0 {
				t.Fatalf("seed %d key %d exceeds curve order", seed, i)
			}
			expectedBig.FillBytes(expected[:])
			if !bytes.Equal(key[:], expected[:]) {
				t.Fatalf("seed %d key %d = %x, want k0+%d = %x", seed, i, key, i, expected)
			}

			// Address must match geth derivation from the same key
			x, y := ethcrypto.S256().ScalarBaseMult(key[:])
			var refPub [64]byte
			x.FillBytes(refPub[:32])
			y.FillBytes(refPub[32:])

			hasher := sha3.NewLegacyKeccak256()
			hasher.Write(refPub[:])
			var hash [32]byte
			hasher.Sum(hash[:0])
			var wantAddress [20]byte
			copy(wantAddress[:], hash[12:])
			if address != wantAddress {
				t.Fatalf("seed %d key %d address mismatch:\n got %x\nwant %x", seed, i, address, wantAddress)
			}
		}
	}
}

// TestPrivateKeyChainBatchBoundary refills the chain across a batch boundary
// and verifies that keys are sequential within a batch and that a fresh seed
// is drawn after the boundary.
func TestPrivateKeyChainBatchBoundary(t *testing.T) {
	chain, err := NewPrivateKeyChain()
	if err != nil {
		t.Fatal(err)
	}
	firstSeed := chain.Seed()

	var previous big.Int
	for i := 0; i < ChainBatchSize+16; i++ {
		key, _, err := chain.NextKey()
		if err != nil {
			t.Fatal(err)
		}
		keyBig := new(big.Int).SetBytes(key[:])
		if i == ChainBatchSize {
			// Fresh batch: must be a new random seed (not k0+batch)
			if chain.Seed() == firstSeed {
				t.Fatalf("expected refill at batch boundary")
			}
			// And the first key of the new batch is its own seed
			seedBytes := chain.Seed()
			if keyBig.Cmp(new(big.Int).SetBytes(seedBytes[:])) != 0 {
				t.Fatalf("first key of new batch is not the new seed")
			}
		} else if i > 0 {
			// Sequential within the batch: k_i = k_{i-1} + 1
			want := new(big.Int).Add(&previous, big.NewInt(1))
			if keyBig.Cmp(want) != 0 {
				t.Fatalf("key %d not sequential: got %x want %x", i, key, want)
			}
		}
		previous.Set(keyBig)
	}
}

// TestNibblePatternsMatchesStringComparison verifies that nibble matching
// agrees with the string-based case-insensitive matching used today.
func TestNibblePatternsMatchesStringComparison(t *testing.T) {
	cases := []struct {
		prefix, suffix string
	}{
		{"ab", ""},
		{"", "cd"},
		{"ab", "cd"},
		{"aB", "Cd"},
		{"ABCDEF", "1234"},
		{"0x", ""}, // nibblesOf rejects the 'x' and yields nil patterns
		{"abc", "def"},
	}
	// Reference: derive addresses from chain and compare both matchers
	chain, err := NewPrivateKeyChain()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		_, pub, err := chain.NextKey()
		if err != nil {
			t.Fatal(err)
		}
		addressBytes := AddressFromPublicKey(&pub)
		address := formatAddressBytes(&addressBytes)
		for _, c := range cases {
			prefixNibbles, suffixNibbles := NibblePatterns(c.prefix, c.suffix)
			if prefixNibbles == nil && c.prefix != "" {
				continue // invalid pattern, skip
			}
			got := MatchesAddressNibbles(&addressBytes, prefixNibbles, suffixNibbles)
			want := matchesString(address, c.prefix, c.suffix)
			if got != want {
				t.Fatalf("case %q/%q address %s: nibble=%v string=%v", c.prefix, c.suffix, address, got, want)
			}
		}
	}
}

func formatAddressBytes(address *[20]byte) string {
	const hexChars = "0123456789abcdef"
	var out [42]byte
	out[0] = '0'
	out[1] = 'x'
	for i := 0; i < 20; i++ {
		out[2+i*2] = hexChars[address[i]>>4]
		out[3+i*2] = hexChars[address[i]&0x0f]
	}
	return string(out[:])
}

// matchesString is the reference case-insensitive matcher.
func matchesString(address, prefix, suffix string) bool {
	addr := address
	if len(addr) > 2 && addr[:2] == "0x" {
		addr = addr[2:]
	}
	if prefix != "" {
		if len(addr) < len(prefix) || !equalFold(addr[:len(prefix)], prefix) {
			return false
		}
	}
	if suffix != "" {
		if len(addr) < len(suffix) || !equalFold(addr[len(addr)-len(suffix):], suffix) {
			return false
		}
	}
	return true
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'F' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'F' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
