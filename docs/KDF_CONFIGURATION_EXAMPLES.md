# KDF Configuration Examples

This document provides comprehensive examples for configuring Key Derivation Functions (KDF) in bloco-vgen, including security recommendations and client-specific configurations.

## Overview

The Universal KDF system in bloco-vgen supports multiple KDF algorithms with comprehensive parameter validation and compatibility analysis. This ensures maximum compatibility with Ethereum clients while maintaining appropriate security levels.

## Supported KDF Algorithms

### Scrypt (Recommended)
- **Algorithm**: scrypt
- **Security**: High memory usage makes it resistant to hardware attacks
- **Performance**: Slower but more secure
- **Client Support**: Excellent (geth, Besu, Anvil, Reth, Firefly)

### PBKDF2
- **Algorithm**: PBKDF2 with SHA-256 or SHA-512
- **Security**: Good computational security
- **Performance**: Faster than scrypt
- **Client Support**: Universal

## Security Level Examples

### High Security (Recommended for Production)

#### Scrypt Configuration
```bash
# Generate keystore with high-security scrypt parameters
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Parameters Explanation:**
- `n`: 262144 (2^18) - Memory cost parameter, must be power of 2
- `r`: 8 - Block size parameter
- `p`: 1 - Parallelization parameter
- `dklen`: 32 - Derived key length in bytes

**Security Analysis:**
- **Memory Usage**: ~128 MB
- **Computational Cost**: Very High
- **Time Estimate**: 2-5 seconds on modern hardware
- **Security Level**: Very High
- **Client Compatibility**: 100% (all major clients)

#### PBKDF2 High Security Configuration
```bash
# Generate keystore with high-security PBKDF2 parameters
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 600000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

**Parameters Explanation:**
- `c`: 600000 - Iteration count (minimum 100000 recommended)
- `prf`: "hmac-sha256" - Pseudo-random function
- `dklen`: 32 - Derived key length in bytes

**Security Analysis:**
- **Memory Usage**: Low (~1 MB)
- **Computational Cost**: High
- **Time Estimate**: 1-3 seconds on modern hardware
- **Security Level**: High
- **Client Compatibility**: 100% (universal support)

### Medium Security (Balanced Performance)

#### Scrypt Balanced Configuration
```bash
# Generate keystore with balanced scrypt parameters
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 131072,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Security Analysis:**
- **Memory Usage**: ~64 MB
- **Computational Cost**: High
- **Time Estimate**: 1-2 seconds on modern hardware
- **Security Level**: High
- **Client Compatibility**: 100%

#### PBKDF2 Balanced Configuration
```bash
# Generate keystore with balanced PBKDF2 parameters
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 300000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

**Security Analysis:**
- **Memory Usage**: Low (~1 MB)
- **Computational Cost**: Medium-High
- **Time Estimate**: 0.5-1.5 seconds on modern hardware
- **Security Level**: Medium-High
- **Client Compatibility**: 100%

### Fast Generation (Development/Testing)

#### Scrypt Fast Configuration
```bash
# Generate keystore with fast scrypt parameters (development only)
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 4096,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Warning**: Only use for development/testing environments.

**Security Analysis:**
- **Memory Usage**: ~2 MB
- **Computational Cost**: Low
- **Time Estimate**: <0.1 seconds
- **Security Level**: Low
- **Client Compatibility**: 100%

#### PBKDF2 Fast Configuration
```bash
# Generate keystore with fast PBKDF2 parameters (development only)
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 10000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

**Warning**: Only use for development/testing environments.

## Client-Specific Configurations

### Geth (Go Ethereum) Optimized
```bash
# Geth-optimized configuration (default geth parameters)
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Compatibility**: Perfect - matches geth default parameters exactly.

### Besu Optimized
```bash
# Besu-compatible configuration
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Compatibility**: Excellent - Besu supports all standard scrypt parameters.

### Anvil (Foundry) Optimized
```bash
# Anvil-compatible configuration (supports both scrypt and PBKDF2)
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 262144,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

**Compatibility**: Excellent - Anvil has robust keystore support.

### Reth Optimized
```bash
# Reth-compatible configuration
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Compatibility**: Excellent - Reth follows Ethereum standards closely.

### Hyperledger Firefly Optimized
```bash
# Firefly-compatible configuration (prefers PBKDF2)
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 100000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

**Compatibility**: Good - Firefly supports standard KeyStore V3 format.

## Performance vs Security Trade-offs

### Ultra-High Security (Enterprise/Long-term Storage)
```bash
# Maximum security configuration
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 1048576,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Trade-offs:**
- **Security**: Maximum (512 MB memory usage)
- **Performance**: Slow (5-15 seconds generation time)
- **Use Case**: Long-term cold storage, enterprise environments
- **Memory Requirement**: 512 MB RAM

### High Security (Production Default)
```bash
# Production-ready configuration
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Trade-offs:**
- **Security**: Very High (128 MB memory usage)
- **Performance**: Good (2-5 seconds generation time)
- **Use Case**: Production applications, user wallets
- **Memory Requirement**: 128 MB RAM

### Balanced (Development/Testing)
```bash
# Balanced configuration
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 65536,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Trade-offs:**
- **Security**: High (32 MB memory usage)
- **Performance**: Fast (0.5-2 seconds generation time)
- **Use Case**: Development, testing, frequent key generation
- **Memory Requirement**: 32 MB RAM

### Fast (Development Only)
```bash
# Fast configuration (development only)
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 50000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

**Trade-offs:**
- **Security**: Medium (low memory usage)
- **Performance**: Very Fast (<0.5 seconds generation time)
- **Use Case**: Rapid development, automated testing
- **Memory Requirement**: <1 MB RAM

## Advanced Configuration Examples

### Custom Salt Length
```bash
# Generate keystore with custom salt length
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32,
  "salt": "custom_32_byte_salt_hex_encoded_here"
}'
```

### Multi-Hash PBKDF2
```bash
# Use SHA-512 instead of SHA-256
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 300000,
  "prf": "hmac-sha512",
  "dklen": 32
}'
```

### Compatibility Analysis
```bash
# Generate keystore with compatibility analysis
./bloco-vgen --prefix abc --kdf-analysis --security-level high
```

**Output includes:**
- Client compatibility matrix
- Security level assessment
- Parameter optimization suggestions
- Performance estimates

## Environment-Specific Recommendations

### Development Environment
```bash
# Fast generation for development
export BLOCO_KDF_PRESET=development
./bloco-vgen --prefix abc
```

**Preset Parameters:**
- Scrypt: n=4096, r=8, p=1
- PBKDF2: c=10000, prf=hmac-sha256
- Generation time: <0.1 seconds

### Testing Environment
```bash
# Balanced parameters for testing
export BLOCO_KDF_PRESET=testing
./bloco-vgen --prefix abc
```

**Preset Parameters:**
- Scrypt: n=65536, r=8, p=1
- PBKDF2: c=100000, prf=hmac-sha256
- Generation time: 0.5-2 seconds

### Production Environment
```bash
# High security for production
export BLOCO_KDF_PRESET=production
./bloco-vgen --prefix abc
```

**Preset Parameters:**
- Scrypt: n=262144, r=8, p=1
- PBKDF2: c=600000, prf=hmac-sha256
- Generation time: 2-5 seconds

### Enterprise Environment
```bash
# Maximum security for enterprise
export BLOCO_KDF_PRESET=enterprise
./bloco-vgen --prefix abc
```

**Preset Parameters:**
- Scrypt: n=1048576, r=8, p=1
- PBKDF2: c=1000000, prf=hmac-sha512
- Generation time: 5-15 seconds

## Batch Generation Examples

### Generate Multiple Keystores with Different Security Levels
```bash
# Generate 5 keystores with high security
./bloco-vgen --prefix abc --count 5 --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'

# Generate 10 keystores with balanced security
./bloco-vgen --prefix def --count 10 --keystore-kdf pbkdf2 --kdf-params '{
  "c": 300000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

### Mixed KDF Generation
```bash
# Generate with automatic KDF selection based on security level
./bloco-vgen --prefix abc --count 3 --security-level high --auto-kdf
```

## Validation and Testing

### Validate KDF Parameters
```bash
# Validate parameters before generation
./bloco-vgen --validate-kdf --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

### Test Client Compatibility
```bash
# Test generated keystore with specific client
./bloco-vgen --prefix abc --test-client geth
./bloco-vgen --prefix abc --test-client besu
./bloco-vgen --prefix abc --test-client anvil
```

### Benchmark KDF Performance
```bash
# Benchmark different KDF configurations
./bloco-vgen benchmark-kdf --kdf scrypt --params '{
  "n": [65536, 131072, 262144],
  "r": [8],
  "p": [1],
  "dklen": [32]
}'
```

## Security Best Practices

### Parameter Selection Guidelines

1. **Memory Parameter (n for scrypt)**:
   - Minimum: 4096 (development only)
   - Recommended: 262144 (production)
   - Maximum: 1048576 (enterprise)
   - Must be power of 2

2. **Iteration Count (c for PBKDF2)**:
   - Minimum: 10000 (development only)
   - Recommended: 300000+ (production)
   - Maximum: 1000000+ (enterprise)

3. **Block Size (r for scrypt)**:
   - Standard: 8 (recommended)
   - Range: 1-32 (higher values increase memory usage)

4. **Parallelization (p for scrypt)**:
   - Standard: 1 (recommended)
   - Range: 1-4 (higher values increase CPU usage)

### Security Validation
```bash
# Validate security level of generated keystore
./bloco-vgen --analyze-security ./keystores/0xabc123....json
```

**Output includes:**
- Security level assessment (Low/Medium/High/Very High)
- Estimated time to crack
- Memory usage analysis
- Recommendations for improvement

## Troubleshooting Common Issues

### Memory Limit Exceeded
```bash
# Error: Memory usage would exceed 2GB limit
# Solution: Reduce n parameter
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 131072,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

### Invalid Parameter Values
```bash
# Error: n must be power of 2
# Solution: Use valid power of 2 values
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

### Client Compatibility Issues
```bash
# Check compatibility before generation
./bloco-vgen --prefix abc --check-compatibility --target-client geth
```

This comprehensive guide covers all aspects of KDF configuration in bloco-vgen, from basic usage to advanced enterprise configurations. Choose the appropriate security level based on your use case and performance requirements.