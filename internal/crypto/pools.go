package crypto

import (
	"crypto/ecdsa"
	"hash"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"golang.org/x/crypto/sha3"
)

// PoolManager manages all cryptographic object pools
type PoolManager struct {
	cryptoPool *CryptoPool
	hasherPool *HasherPool
	bufferPool *BufferPool
	config     PoolConfig
}

// PoolConfig contains configuration for object pools
type PoolConfig struct {
	MaxPoolSize     int
	EnableClearing  bool
	PreallocateSize int
}

// CryptoPool provides object pooling for cryptographic operations
type CryptoPool struct {
	privateKeyPool sync.Pool
	publicKeyPool  sync.Pool
	bigIntPool     sync.Pool
	ecdsaKeyPool   sync.Pool
	config         PoolConfig
}

// HasherPool provides object pooling for Keccak256 hash instances
type HasherPool struct {
	keccakPool sync.Pool
	config     PoolConfig
}

// BufferPool provides object pooling for byte and string buffers
type BufferPool struct {
	byteBufferPool    sync.Pool
	stringBuilderPool sync.Pool
	hexBufferPool     sync.Pool
	config            PoolConfig
}

// NewPoolManager creates a new pool manager with the given configuration
func NewPoolManager(config PoolConfig) *PoolManager {
	return &PoolManager{
		cryptoPool: NewCryptoPool(config),
		hasherPool: NewHasherPool(config),
		bufferPool: NewBufferPool(config),
		config:     config,
	}
}

// GetCryptoPool returns the crypto pool
func (pm *PoolManager) GetCryptoPool() *CryptoPool {
	return pm.cryptoPool
}

// GetHasherPool returns the hasher pool
func (pm *PoolManager) GetHasherPool() *HasherPool {
	return pm.hasherPool
}

// GetBufferPool returns the buffer pool
func (pm *PoolManager) GetBufferPool() *BufferPool {
	return pm.bufferPool
}

// NewCryptoPool creates a new CryptoPool with initialized pools
func NewCryptoPool(config PoolConfig) *CryptoPool {
	return &CryptoPool{
		privateKeyPool: sync.Pool{
			New: func() interface{} {
				slice := make([]byte, 32)
				return &slice
			},
		},
		publicKeyPool: sync.Pool{
			New: func() interface{} {
				slice := make([]byte, 64)
				return &slice
			},
		},
		bigIntPool: sync.Pool{
			New: func() interface{} {
				return new(big.Int)
			},
		},
		ecdsaKeyPool: sync.Pool{
			New: func() interface{} {
				return &ecdsa.PrivateKey{
					PublicKey: ecdsa.PublicKey{
						Curve: crypto.S256(),
					},
				}
			},
		},
		config: config,
	}
}

// GetPrivateKeyBuffer gets a private key buffer from the pool
func (cp *CryptoPool) GetPrivateKeyBuffer() []byte {
	return *cp.privateKeyPool.Get().(*[]byte)
}

// PutPrivateKeyBuffer returns a private key buffer to the pool
func (cp *CryptoPool) PutPrivateKeyBuffer(buf []byte) {
	if cp.config.EnableClearing {
		// Clear the buffer for security
		for i := range buf {
			buf[i] = 0
		}
	}
	cp.privateKeyPool.Put(&buf)
}

// GetPublicKeyBuffer gets a public key buffer from the pool
func (cp *CryptoPool) GetPublicKeyBuffer() []byte {
	return *cp.publicKeyPool.Get().(*[]byte)
}

// PutPublicKeyBuffer returns a public key buffer to the pool
func (cp *CryptoPool) PutPublicKeyBuffer(buf []byte) {
	if cp.config.EnableClearing {
		// Clear the buffer for security
		for i := range buf {
			buf[i] = 0
		}
	}
	cp.publicKeyPool.Put(&buf)
}

// GetBigInt gets a big.Int from the pool
func (cp *CryptoPool) GetBigInt() *big.Int {
	bigInt := cp.bigIntPool.Get().(*big.Int)
	bigInt.SetInt64(0) // Reset to zero
	return bigInt
}

// PutBigInt returns a big.Int to the pool
func (cp *CryptoPool) PutBigInt(bigInt *big.Int) {
	if cp.config.EnableClearing {
		bigInt.SetInt64(0) // Clear for security
	}
	cp.bigIntPool.Put(bigInt)
}

// GetECDSAKey gets an ECDSA private key from the pool
func (cp *CryptoPool) GetECDSAKey() *ecdsa.PrivateKey {
	key := cp.ecdsaKeyPool.Get().(*ecdsa.PrivateKey)
	// Reset the key
	*key = ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: key.Curve}}
	return key
}

// PutECDSAKey returns an ECDSA private key to the pool
func (cp *CryptoPool) PutECDSAKey(key *ecdsa.PrivateKey) {
	if cp.config.EnableClearing {
		// Clear sensitive data
		if scalar := ecies.ImportECDSA(key).D; scalar != nil {
			clear(scalar.Bits())
			scalar.SetInt64(0)
		}
		curve := key.Curve
		if key.X != nil {
			key.X.SetInt64(0)
		}
		if key.Y != nil {
			key.Y.SetInt64(0)
		}
		*key = ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: curve}}
	}
	cp.ecdsaKeyPool.Put(key)
}

// NewHasherPool creates a new HasherPool with initialized Keccak256 pool
func NewHasherPool(config PoolConfig) *HasherPool {
	return &HasherPool{
		keccakPool: sync.Pool{
			New: func() interface{} {
				return sha3.NewLegacyKeccak256()
			},
		},
		config: config,
	}
}

// GetKeccak gets a Keccak256 hasher from the pool
func (hp *HasherPool) GetKeccak() hash.Hash {
	hasher := hp.keccakPool.Get().(hash.Hash)
	hasher.Reset() // Reset the hasher state
	return hasher
}

// PutKeccak returns a Keccak256 hasher to the pool
func (hp *HasherPool) PutKeccak(hasher hash.Hash) {
	hasher.Reset() // Clear any remaining state
	hp.keccakPool.Put(hasher)
}

// NewBufferPool creates a new BufferPool with initialized buffer pools
func NewBufferPool(config PoolConfig) *BufferPool {
	return &BufferPool{
		byteBufferPool: sync.Pool{
			New: func() interface{} {
				slice := make([]byte, 0, 64) // Pre-allocate capacity for typical use
				return &slice
			},
		},
		stringBuilderPool: sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
		hexBufferPool: sync.Pool{
			New: func() interface{} {
				slice := make([]byte, 64) // For hex encoding/decoding
				return &slice
			},
		},
		config: config,
	}
}

// GetByteBuffer gets a byte buffer from the pool
func (bp *BufferPool) GetByteBuffer() []byte {
	buf := *bp.byteBufferPool.Get().(*[]byte)
	return buf[:0] // Reset length but keep capacity
}

// PutByteBuffer returns a byte buffer to the pool
func (bp *BufferPool) PutByteBuffer(buf []byte) {
	if bp.config.EnableClearing {
		// Clear the buffer for security
		for i := range buf {
			buf[i] = 0
		}
	}
	trimmed := buf[:0]
	bp.byteBufferPool.Put(&trimmed)
}

// GetStringBuilder gets a string builder from the pool
func (bp *BufferPool) GetStringBuilder() *strings.Builder {
	sb := bp.stringBuilderPool.Get().(*strings.Builder)
	sb.Reset() // Clear any existing content
	return sb
}

// PutStringBuilder returns a string builder to the pool
func (bp *BufferPool) PutStringBuilder(sb *strings.Builder) {
	sb.Reset() // Clear content
	bp.stringBuilderPool.Put(sb)
}

// GetHexBuffer gets a hex buffer from the pool
func (bp *BufferPool) GetHexBuffer() []byte {
	return *bp.hexBufferPool.Get().(*[]byte)
}

// PutHexBuffer returns a hex buffer to the pool
func (bp *BufferPool) PutHexBuffer(buf []byte) {
	if bp.config.EnableClearing {
		// Clear the buffer for security
		for i := range buf {
			buf[i] = 0
		}
	}
	bp.hexBufferPool.Put(&buf)
}

// DefaultPoolConfig returns a default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxPoolSize:     100,
		EnableClearing:  true,
		PreallocateSize: 64,
	}
}

// Stats returns statistics about pool usage
func (pm *PoolManager) Stats() PoolStats {
	// This would require additional tracking in a production implementation
	return PoolStats{
		CryptoPoolHits:   0, // Would track actual hits/misses
		HasherPoolHits:   0,
		BufferPoolHits:   0,
		TotalAllocations: 0,
	}
}

// PoolStats contains statistics about pool usage
type PoolStats struct {
	CryptoPoolHits   int64
	HasherPoolHits   int64
	BufferPoolHits   int64
	TotalAllocations int64
}
