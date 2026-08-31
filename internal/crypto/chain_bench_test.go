package crypto

import (
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/sha3"
)

func BenchmarkKeccak256Of64(b *testing.B) {
	var input [64]byte
	rand.Read(input[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = keccak256Of64(&input)
	}
}

func BenchmarkSHA3Reference(b *testing.B) {
	var input [64]byte
	rand.Read(input[:])
	h := sha3.NewLegacyKeccak256()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Reset()
		h.Write(input[:])
		var out [32]byte
		h.Sum(out[:0])
	}
}

func BenchmarkChainNextKey(b *testing.B) {
	chain, err := NewPrivateKeyChain()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := chain.NextKey(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChainRefillOnly(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := buildChainBatch(); err != nil {
			b.Fatal(err)
		}
	}
}
