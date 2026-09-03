package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/bits"
	"sync"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// ChainBatchSize is the number of private keys derived from a single random
// seed. Public keys are chained incrementally (P + G) and converted to affine
// coordinates with a single batched field inversion, which is roughly an order
// of magnitude faster than one full scalar multiplication per key.
const ChainBatchSize = 4096

// chained keys are always kept in the range [2, N-2], so the chain never hits
// the generator G, its negation -G, or the point at infinity.
var (
	chainTwo            = big.NewInt(2)
	chainMaxSeed        = new(big.Int).Sub(secp256k1.S256().Params().N, big.NewInt(ChainBatchSize+1))
	chainGeneratorPoint = newJacobianPoint(secp256k1.S256().Params().Gx, secp256k1.S256().Params().Gy)
)

// newJacobianPoint builds a Jacobian point with Z = 1 from affine big.Ints.
func newJacobianPoint(x, y *big.Int) secp256k1.JacobianPoint {
	var point secp256k1.JacobianPoint
	var xBytes, yBytes [32]byte
	x.FillBytes(xBytes[:])
	y.FillBytes(yBytes[:])
	point.X.SetByteSlice(xBytes[:])
	point.Y.SetByteSlice(yBytes[:])
	point.Z.SetInt(1)
	return point
}

// keccak256Of64 computes Keccak-256 of exactly 64 bytes. For a 64-byte
// message the rate-136 sponge absorbs a single block: 8 little-endian lanes,
// the 0x01 domain separator at byte 64 and the 0x80 padding at byte 135,
// followed by exactly one Keccak-f[1600] permutation.
func keccak256Of64(input *[64]byte) [32]byte {
	var state [25]uint64
	for i := 0; i < 8; i++ {
		state[i] = binary.LittleEndian.Uint64(input[i*8 : i*8+8])
	}
	state[8] ^= 0x01
	state[16] ^= 0x8000000000000000
	keccakF1600(&state)

	var output [32]byte
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(output[i*8:], state[i])
	}
	return output
}

// keccakF1600 applies the Keccak-f[1600] permutation to the state. It is
// unrolled over explicit lane variables (as in golang.org/x/crypto/sha3),
// which the Go compiler optimizes substantially better than nested loops.
func keccakF1600(a *[25]uint64) {
	var bc0, bc1, bc2, bc3, bc4, d0, d1, d2, d3, d4, t uint64

	a00 := a[0]
	a01 := a[1]
	a02 := a[2]
	a03 := a[3]
	a04 := a[4]
	a05 := a[5]
	a06 := a[6]
	a07 := a[7]
	a08 := a[8]
	a09 := a[9]
	a10 := a[10]
	a11 := a[11]
	a12 := a[12]
	a13 := a[13]
	a14 := a[14]
	a15 := a[15]
	a16 := a[16]
	a17 := a[17]
	a18 := a[18]
	a19 := a[19]
	a20 := a[20]
	a21 := a[21]
	a22 := a[22]
	a23 := a[23]
	a24 := a[24]

	for round := 0; round < 24; round++ {
		// Theta
		bc0 = a00 ^ a05 ^ a10 ^ a15 ^ a20
		bc1 = a01 ^ a06 ^ a11 ^ a16 ^ a21
		bc2 = a02 ^ a07 ^ a12 ^ a17 ^ a22
		bc3 = a03 ^ a08 ^ a13 ^ a18 ^ a23
		bc4 = a04 ^ a09 ^ a14 ^ a19 ^ a24

		d0 = bc4 ^ bits.RotateLeft64(bc1, 1)
		d1 = bc0 ^ bits.RotateLeft64(bc2, 1)
		d2 = bc1 ^ bits.RotateLeft64(bc3, 1)
		d3 = bc2 ^ bits.RotateLeft64(bc4, 1)
		d4 = bc3 ^ bits.RotateLeft64(bc0, 1)

		a00 ^= d0
		a05 ^= d0
		a10 ^= d0
		a15 ^= d0
		a20 ^= d0
		a01 ^= d1
		a06 ^= d1
		a11 ^= d1
		a16 ^= d1
		a21 ^= d1
		a02 ^= d2
		a07 ^= d2
		a12 ^= d2
		a17 ^= d2
		a22 ^= d2
		a03 ^= d3
		a08 ^= d3
		a13 ^= d3
		a18 ^= d3
		a23 ^= d3
		a04 ^= d4
		a09 ^= d4
		a14 ^= d4
		a19 ^= d4
		a24 ^= d4

		// Rho and Pi
		t = a01
		a01 = bits.RotateLeft64(a06, 44)
		a06 = bits.RotateLeft64(a09, 20)
		a09 = bits.RotateLeft64(a22, 61)
		a22 = bits.RotateLeft64(a14, 39)
		a14 = bits.RotateLeft64(a20, 18)
		a20 = bits.RotateLeft64(a02, 62)
		a02 = bits.RotateLeft64(a12, 43)
		a12 = bits.RotateLeft64(a13, 25)
		a13 = bits.RotateLeft64(a19, 8)
		a19 = bits.RotateLeft64(a23, 56)
		a23 = bits.RotateLeft64(a15, 41)
		a15 = bits.RotateLeft64(a04, 27)
		a04 = bits.RotateLeft64(a24, 14)
		a24 = bits.RotateLeft64(a21, 2)
		a21 = bits.RotateLeft64(a08, 55)
		a08 = bits.RotateLeft64(a16, 45)
		a16 = bits.RotateLeft64(a05, 36)
		a05 = bits.RotateLeft64(a03, 28)
		a03 = bits.RotateLeft64(a18, 21)
		a18 = bits.RotateLeft64(a17, 15)
		a17 = bits.RotateLeft64(a11, 10)
		a11 = bits.RotateLeft64(a07, 6)
		a07 = bits.RotateLeft64(a10, 3)
		a10 = bits.RotateLeft64(t, 1)

		// Chi
		bc0 = a00
		bc1 = a01
		bc2 = a02
		bc3 = a03
		bc4 = a04
		a00 ^= ^bc1 & bc2
		a01 ^= ^bc2 & bc3
		a02 ^= ^bc3 & bc4
		a03 ^= ^bc4 & bc0
		a04 ^= ^bc0 & bc1
		bc0 = a05
		bc1 = a06
		bc2 = a07
		bc3 = a08
		bc4 = a09
		a05 ^= ^bc1 & bc2
		a06 ^= ^bc2 & bc3
		a07 ^= ^bc3 & bc4
		a08 ^= ^bc4 & bc0
		a09 ^= ^bc0 & bc1
		bc0 = a10
		bc1 = a11
		bc2 = a12
		bc3 = a13
		bc4 = a14
		a10 ^= ^bc1 & bc2
		a11 ^= ^bc2 & bc3
		a12 ^= ^bc3 & bc4
		a13 ^= ^bc4 & bc0
		a14 ^= ^bc0 & bc1
		bc0 = a15
		bc1 = a16
		bc2 = a17
		bc3 = a18
		bc4 = a19
		a15 ^= ^bc1 & bc2
		a16 ^= ^bc2 & bc3
		a17 ^= ^bc3 & bc4
		a18 ^= ^bc4 & bc0
		a19 ^= ^bc0 & bc1
		bc0 = a20
		bc1 = a21
		bc2 = a22
		bc3 = a23
		bc4 = a24
		a20 ^= ^bc1 & bc2
		a21 ^= ^bc2 & bc3
		a22 ^= ^bc3 & bc4
		a23 ^= ^bc4 & bc0
		a24 ^= ^bc0 & bc1

		// Iota
		a00 ^= keccakRoundConstants[round]
	}

	a[0] = a00
	a[1] = a01
	a[2] = a02
	a[3] = a03
	a[4] = a04
	a[5] = a05
	a[6] = a06
	a[7] = a07
	a[8] = a08
	a[9] = a09
	a[10] = a10
	a[11] = a11
	a[12] = a12
	a[13] = a13
	a[14] = a14
	a[15] = a15
	a[16] = a16
	a[17] = a17
	a[18] = a18
	a[19] = a19
	a[20] = a20
	a[21] = a21
	a[22] = a22
	a[23] = a23
	a[24] = a24
}

// keccakRoundConstants are the Keccak-f[1600] iota round constants.
var keccakRoundConstants = [24]uint64{
	0x0000000000000001, 0x0000000000008082, 0x800000000000808a, 0x8000000080008000,
	0x000000000000808b, 0x0000000080000001, 0x8000000080008081, 0x8000000000008009,
	0x000000000000008a, 0x0000000000000088, 0x0000000080008009, 0x000000008000000a,
	0x000000008000808b, 0x800000000000008b, 0x8000000000008089, 0x8000000000008003,
	0x8000000000008002, 0x8000000000000080, 0x000000000000800a, 0x800000008000000a,
	0x8000000080008081, 0x8000000000008080, 0x0000000080000001, 0x8000000080008008,
}

// nibblesOf converts a hex string pattern to its nibble values.
func nibblesOf(pattern string) []byte {
	nibbles := make([]byte, 0, len(pattern))
	for i := 0; i < len(pattern); i++ {
		char := pattern[i]
		switch {
		case char >= '0' && char <= '9':
			nibbles = append(nibbles, char-'0')
		case char >= 'a' && char <= 'f':
			nibbles = append(nibbles, char-'a'+10)
		case char >= 'A' && char <= 'F':
			nibbles = append(nibbles, char-'A'+10)
		default:
			return nil
		}
	}
	return nibbles
}

// addressNibble returns the i-th hex nibble (0-39) of a 20-byte address.
func addressNibble(address *[20]byte, nibble int) byte {
	byteID := nibble >> 1
	value := address[byteID]
	if nibble&1 == 0 {
		return value >> 4
	}
	return value & 0x0f
}

// MatchesAddressNibbles reports whether the address bytes match the given
// case-insensitive prefix/suffix hex patterns, without formatting the
// address as a string. Empty patterns always match.
func MatchesAddressNibbles(address *[20]byte, prefixNibbles, suffixNibbles []byte) bool {
	for i, nibble := range prefixNibbles {
		if addressNibble(address, i) != nibble {
			return false
		}
	}
	for i, nibble := range suffixNibbles {
		if addressNibble(address, 40-len(suffixNibbles)+i) != nibble {
			return false
		}
	}
	return true
}

// NibblePatterns returns the nibble patterns for a case-insensitive
// prefix/suffix pair, or nil if either pattern is not valid hex.
func NibblePatterns(prefix, suffix string) (prefixNibbles, suffixNibbles []byte) {
	prefixNibbles = nibblesOf(prefix)
	if len(prefix) > 0 && prefixNibbles == nil {
		return nil, nil
	}
	suffixNibbles = nibblesOf(suffix)
	if len(suffix) > 0 && suffixNibbles == nil {
		return nil, nil
	}
	return prefixNibbles, suffixNibbles
}

// chainBatch is one filled chain segment: a random seed k0 and the Jacobian
// points of k0*G .. (k0+n-1)*G converted to affine coordinates.
type chainBatch struct {
	k0     [32]byte
	points []secp256k1.JacobianPoint
	z2     []secp256k1.FieldVal
	z3     []secp256k1.FieldVal
	prod   []secp256k1.FieldVal
	prefix []secp256k1.FieldVal
}

// PrivateKeyChain scans candidate secp256k1 private keys as chained batches:
// each batch starts from a random seed k0 in [2, N-2-batch] followed by
// k0+1, k0+2, ... . Public keys are derived incrementally (P + G in Jacobian
// coordinates) and converted to affine coordinates with one batched field
// inversion per batch.
//
// Batches are prefetched on a background goroutine so the fill cost (chain
// build + batched inversion) overlaps with key consumption.
//
// IMPORTANT: keys within one batch are NOT independent. They are consecutive
// scalars, so anyone holding one of them recovers the other 4095 by adding a
// small integer. The chain is a search device only: a caller that hands one of
// these keys to a user as a wallet MUST discard the chain (see Close) so that
// no two delivered keys ever come from the same batch. It is not a source of
// independent random keys and must not be used as one.
type PrivateKeyChain struct {
	batches chan *chainBatch
	current *chainBatch
	pos     int
	errMu   sync.Mutex
	fillErr error

	// done stops the background filler when the chain is discarded.
	done      chan struct{}
	closeOnce sync.Once
}

// NewPrivateKeyChain creates a chain seeded with fresh system entropy.
func NewPrivateKeyChain() (*PrivateKeyChain, error) {
	first, err := buildChainBatch()
	if err != nil {
		return nil, err
	}
	chain := &PrivateKeyChain{
		batches: make(chan *chainBatch, 1),
		current: first,
		done:    make(chan struct{}),
	}
	go chain.filler()
	return chain, nil
}

// Close stops the background filler and releases the chain. Callers must call
// it on every chain they stop using: the filler blocks holding a prebuilt
// batch, so a dropped chain would leak both a goroutine and its batch memory.
// Close is safe to call more than once.
func (c *PrivateKeyChain) Close() {
	c.closeOnce.Do(func() { close(c.done) })
}

// filler prefills batches in the background. On error it records the error
// and closes the channel so consumers stop blocking.
func (c *PrivateKeyChain) filler() {
	for {
		batch, err := buildChainBatch()
		if err != nil {
			c.errMu.Lock()
			c.fillErr = err
			c.errMu.Unlock()
			close(c.batches)
			return
		}
		select {
		case c.batches <- batch:
		case <-c.done:
			return
		}
	}
}

// fillErrSnapshot returns the current background fill error, if any.
func (c *PrivateKeyChain) fillErrSnapshot() error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.fillErr
}

// buildChainBatch draws a fresh seed and builds the next chain batch.
func buildChainBatch() (*chainBatch, error) {
	batch := &chainBatch{
		points: make([]secp256k1.JacobianPoint, ChainBatchSize),
		z2:     make([]secp256k1.FieldVal, ChainBatchSize),
		z3:     make([]secp256k1.FieldVal, ChainBatchSize),
		prod:   make([]secp256k1.FieldVal, ChainBatchSize),
		prefix: make([]secp256k1.FieldVal, ChainBatchSize+1),
	}

	for {
		var seed [32]byte
		if _, err := rand.Read(seed[:]); err != nil {
			return nil, fmt.Errorf("failed to read chain seed: %w", err)
		}
		seedBig := new(big.Int).SetBytes(seed[:])
		if seedBig.Cmp(chainTwo) < 0 || seedBig.Cmp(chainMaxSeed) > 0 {
			continue
		}
		copy(batch.k0[:], seed[:])
		break
	}

	var kScalar secp256k1.ModNScalar
	kScalar.SetByteSlice(batch.k0[:])

	var start secp256k1.JacobianPoint
	secp256k1.ScalarBaseMultNonConst(&kScalar, &start)
	batch.points[0].Set(&start)
	for i := 1; i < ChainBatchSize; i++ {
		secp256k1.AddNonConst(&batch.points[i-1], &chainGeneratorPoint, &batch.points[i])
	}
	batchAffine(batch.points, batch.z2, batch.z3, batch.prod, batch.prefix)
	return batch, nil
}

// batchAffine converts N Jacobian points to affine coordinates in place using
// a single field inversion (Montgomery batch inversion of all Z^2*Z^3).
func batchAffine(points []secp256k1.JacobianPoint, z2, z3, prod, prefix []secp256k1.FieldVal) {
	n := len(points)
	var one secp256k1.FieldVal
	one.SetInt(1)
	prefix[0].Set(&one)
	for i := 0; i < n; i++ {
		z2[i].SquareVal(&points[i].Z)
		z3[i].Mul2(&z2[i], &points[i].Z)
		prod[i].Mul2(&z2[i], &z3[i])
		prefix[i+1].Mul2(&prefix[i], &prod[i])
	}

	var inv secp256k1.FieldVal
	inv.Set(&prefix[n])
	inv.Inverse() // inv = 1 / prod[0..n-1]
	for i := n - 1; i >= 0; i-- {
		var invProd secp256k1.FieldVal
		invProd.Mul2(&inv, &prefix[i]) // 1 / (z2[i] * z3[i])
		inv.Mul(&prod[i])              // 1 / prod[j] for j < i
		points[i].X.Mul(&invProd).Mul(&z3[i]).Normalize()
		points[i].Y.Mul(&invProd).Mul(&z2[i]).Normalize()
		points[i].Z.SetInt(1)
	}
}

// NextKey returns the next (private key, uncompressed public key) pair in the
// chain. The private key is 32 big-endian bytes; the public key is the raw
// 64-byte X || Y concatenation (no 0x04 prefix).
func (c *PrivateKeyChain) NextKey() ([32]byte, [64]byte, error) {
	if c.pos >= ChainBatchSize {
		select {
		case <-c.done:
			return [32]byte{}, [64]byte{}, fmt.Errorf("private key chain is closed")
		default:
		}

		batch, ok := <-c.batches
		if !ok {
			err := c.fillErrSnapshot()
			if err == nil {
				err = fmt.Errorf("private key chain fill stopped")
			}
			return [32]byte{}, [64]byte{}, err
		}
		c.current = batch
		c.pos = 0
	}
	batch := c.current

	var key [32]byte
	copy(key[:], batch.k0[:])
	offset := c.pos
	if offset > 0 {
		var kScalar, offsetScalar secp256k1.ModNScalar
		kScalar.SetByteSlice(batch.k0[:])
		offsetScalar.SetInt(uint32(offset))
		kScalar.Add(&offsetScalar) // k0 + pos; stays < N by seed range check
		kScalar.PutBytesUnchecked(key[:])
	}

	point := &batch.points[offset]
	var pub [64]byte
	point.X.PutBytes((*[32]byte)(pub[0:32]))
	point.Y.PutBytes((*[32]byte)(pub[32:64]))
	c.pos++
	return key, pub, nil
}

// AddressFromPublicKey derives the Ethereum address (last 20 bytes of
// Keccak-256) from a raw 64-byte X || Y public key.
func AddressFromPublicKey(pub *[64]byte) [20]byte {
	hash := keccak256Of64(pub)
	var address [20]byte
	copy(address[:], hash[12:])
	return address
}

// Seed returns the current batch seed (for diagnostics and tests).
func (c *PrivateKeyChain) Seed() [32]byte {
	return c.current.k0
}

// Position returns the number of keys consumed from the current batch.
func (c *PrivateKeyChain) Position() int {
	return c.pos
}
