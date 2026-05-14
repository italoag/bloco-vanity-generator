# KDF Compatibility Troubleshooting Guide

This guide helps you diagnose and resolve keystore compatibility issues when using the Universal KDF system in bloco-vgen.

## Quick Diagnosis

### Check Keystore Compatibility
```bash
# Analyze keystore compatibility
./bloco-vgen --analyze-keystore ./keystores/0xabc123....json

# Check compatibility with specific client
./bloco-vgen --check-compatibility --keystore ./keystores/0xabc123....json --client geth
```

### Common Error Messages

#### "Keystore not compatible with client"
**Cause**: KDF parameters don't meet client requirements
**Solution**: Use client-specific configuration (see [Client-Specific Solutions](#client-specific-solutions))

#### "Invalid KDF parameters"
**Cause**: Parameters outside acceptable ranges
**Solution**: Validate parameters (see [Parameter Validation](#parameter-validation))

#### "Memory limit exceeded"
**Cause**: Scrypt n parameter too high
**Solution**: Reduce memory usage (see [Memory Issues](#memory-issues))

## Client-Specific Solutions

### Geth (Go Ethereum) Issues

#### Problem: "unsupported KDF"
```bash
# Error when importing keystore to geth
geth account import keystore.json
# Error: unsupported KDF
```

**Diagnosis**:
```bash
./bloco-vgen --analyze-keystore keystore.json
```

**Solution**: Use geth-compatible parameters
```bash
# Generate geth-compatible keystore
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

#### Problem: "invalid key derivation parameters"
**Cause**: Non-standard parameter values
**Solution**: Use standard geth parameters
```bash
# Geth standard configuration
./bloco-vgen --prefix abc --client-preset geth
```

### Besu Issues

#### Problem: "keystore format not supported"
```bash
# Error when starting Besu with keystore
besu --key-file=keystore.json
# Error: keystore format not supported
```

**Diagnosis**:
```bash
./bloco-vgen --analyze-keystore keystore.json --client besu
```

**Solution**: Use Besu-compatible format
```bash
# Generate Besu-compatible keystore
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}' --format besu
```

#### Problem: "MAC verification failed"
**Cause**: Incorrect MAC calculation or corrupted keystore
**Solution**: Regenerate keystore with proper MAC
```bash
# Regenerate with MAC validation
./bloco-vgen --prefix abc --validate-mac --keystore-kdf scrypt
```

### Anvil (Foundry) Issues

#### Problem: "failed to decrypt keystore"
```bash
# Error when using keystore with Anvil
anvil --accounts 1 --keystore keystore.json
# Error: failed to decrypt keystore
```

**Diagnosis**:
```bash
./bloco-vgen --test-decrypt keystore.json --password password.txt
```

**Solution**: Use Anvil-preferred PBKDF2
```bash
# Generate Anvil-optimized keystore
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 262144,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

### Reth Issues

#### Problem: "unsupported keystore version"
**Cause**: Non-standard KeyStore V3 format
**Solution**: Ensure proper V3 format
```bash
# Generate standard V3 keystore
./bloco-vgen --prefix abc --keystore-version 3 --validate-format
```

#### Problem: "invalid cipher parameters"
**Cause**: Non-standard cipher configuration
**Solution**: Use standard AES-128-CTR
```bash
# Generate with standard cipher
./bloco-vgen --prefix abc --cipher aes-128-ctr --validate-cipher
```

### Hyperledger Firefly Issues

#### Problem: "keystore import failed"
**Cause**: Firefly prefers specific KDF configurations
**Solution**: Use Firefly-optimized settings
```bash
# Generate Firefly-compatible keystore
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 100000,
  "prf": "hmac-sha256",
  "dklen": 32
}' --format firefly
```

## Parameter Validation Issues

### Scrypt Parameter Problems

#### Problem: "n must be power of 2"
```bash
# Invalid configuration
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 100000,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
# Error: n must be power of 2
```

**Solution**: Use valid power of 2 values
```bash
# Valid powers of 2: 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

#### Problem: "r parameter out of range"
**Valid range**: 1-32
**Recommended**: 8
```bash
# Fix invalid r parameter
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

#### Problem: "p parameter out of range"
**Valid range**: 1-4
**Recommended**: 1
```bash
# Fix invalid p parameter
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

### PBKDF2 Parameter Problems

#### Problem: "iteration count too low"
**Minimum**: 10000 (development)
**Recommended**: 100000+ (production)
```bash
# Fix low iteration count
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 300000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

#### Problem: "unsupported PRF"
**Supported PRFs**:
- `hmac-sha256` (recommended)
- `hmac-sha512`
- `hmac-sha1` (deprecated)

```bash
# Use supported PRF
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 300000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

## Memory Issues

### Problem: "Memory usage would exceed limit"
```bash
# Error with high n parameter
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 2097152,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
# Error: Memory usage would exceed 2GB limit
```

**Memory Usage Calculation**:
Memory = 128 * n * r * p bytes

**Solutions**:

#### Reduce n parameter
```bash
# Reduce from 2097152 to 1048576
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 1048576,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

#### Use PBKDF2 instead
```bash
# Switch to PBKDF2 for lower memory usage
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 600000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

#### Adjust system memory limit
```bash
# Increase memory limit (if system allows)
./bloco-vgen --prefix abc --max-memory 4GB --keystore-kdf scrypt --kdf-params '{
  "n": 2097152,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

## Performance Issues

### Problem: "Keystore generation too slow"

#### Diagnosis
```bash
# Benchmark current configuration
./bloco-vgen benchmark-kdf --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

#### Solutions

**Option 1: Reduce security for faster generation**
```bash
# Faster scrypt parameters
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 65536,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Option 2: Switch to PBKDF2**
```bash
# PBKDF2 is generally faster
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 300000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

**Option 3: Use development preset**
```bash
# Fast development configuration
./bloco-vgen --prefix abc --preset development
```

### Problem: "High CPU usage during generation"

#### Diagnosis
```bash
# Monitor CPU usage during generation
./bloco-vgen --prefix abc --monitor-cpu --keystore-kdf scrypt
```

#### Solutions

**Reduce parallelization parameter**
```bash
# Reduce p parameter to lower CPU usage
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

**Use CPU-friendly PBKDF2**
```bash
# PBKDF2 with moderate iteration count
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 200000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

## Format and Structure Issues

### Problem: "Invalid keystore format"

#### Diagnosis
```bash
# Validate keystore structure
./bloco-vgen --validate-format ./keystores/0xabc123....json
```

#### Common Format Issues

**Missing required fields**
```json
{
  "address": "required",
  "crypto": {
    "cipher": "required",
    "ciphertext": "required",
    "cipherparams": "required",
    "kdf": "required",
    "kdfparams": "required",
    "mac": "required"
  },
  "id": "required",
  "version": 3
}
```

**Solution**: Regenerate with proper format validation
```bash
./bloco-vgen --prefix abc --validate-format --strict-compliance
```

### Problem: "MAC verification failed"

#### Diagnosis
```bash
# Test MAC verification
./bloco-vgen --verify-mac ./keystores/0xabc123....json --password password.txt
```

#### Solutions

**Regenerate keystore**
```bash
# Generate new keystore with proper MAC
./bloco-vgen --prefix abc --force-regenerate
```

**Fix MAC calculation**
```bash
# Repair MAC in existing keystore
./bloco-vgen --repair-mac ./keystores/0xabc123....json --password password.txt
```

## Salt and Randomness Issues

### Problem: "Weak salt detected"

#### Diagnosis
```bash
# Analyze salt strength
./bloco-vgen --analyze-salt ./keystores/0xabc123....json
```

#### Solution
```bash
# Generate with strong salt
./bloco-vgen --prefix abc --strong-salt --salt-length 32
```

### Problem: "Duplicate salt values"

#### Diagnosis
```bash
# Check for salt collisions
./bloco-vgen --check-salt-uniqueness ./keystores/
```

#### Solution
```bash
# Force unique salt generation
./bloco-vgen --prefix abc --unique-salt --entropy-source /dev/urandom
```

## Cipher Issues

### Problem: "Unsupported cipher"

#### Supported Ciphers
- `aes-128-ctr` (recommended)
- `aes-128-cbc` (legacy support)

#### Solution
```bash
# Use standard cipher
./bloco-vgen --prefix abc --cipher aes-128-ctr
```

### Problem: "Invalid IV length"

#### Diagnosis
```bash
# Check cipher parameters
./bloco-vgen --analyze-cipher ./keystores/0xabc123....json
```

#### Solution
```bash
# Generate with proper IV
./bloco-vgen --prefix abc --cipher aes-128-ctr --validate-iv
```

## Diagnostic Commands

### Comprehensive Keystore Analysis
```bash
# Full keystore analysis
./bloco-vgen --analyze-keystore ./keystores/0xabc123....json --verbose
```

**Output includes**:
- KDF type and parameters
- Security level assessment
- Client compatibility matrix
- Format validation results
- Performance estimates

### Client Compatibility Matrix
```bash
# Test compatibility with all clients
./bloco-vgen --compatibility-matrix ./keystores/0xabc123....json
```

**Output format**:
```
Client Compatibility Report
===========================
Geth:     Compatible
Besu:     Compatible
Anvil:    Compatible
Reth:     Compatible
Firefly:  Partial (see recommendations)

Recommendations:
- Consider using PBKDF2 for better Firefly compatibility
```

### Parameter Optimization
```bash
# Get parameter optimization suggestions
./bloco-vgen --optimize-params --target-client geth --security-level high
```

**Output includes**:
- Recommended parameters for target client
- Security vs performance trade-offs
- Memory usage estimates
- Generation time estimates

## Automated Fixes

### Auto-Fix Common Issues
```bash
# Automatically fix common compatibility issues
./bloco-vgen --auto-fix ./keystores/0xabc123....json --target-client geth
```

### Batch Validation and Repair
```bash
# Validate and repair all keystores in directory
./bloco-vgen --batch-validate ./keystores/ --auto-repair
```

### Migration Between KDF Types
```bash
# Migrate from scrypt to PBKDF2
./bloco-vgen --migrate-kdf ./keystores/0xabc123....json --from scrypt --to pbkdf2 --password password.txt
```

## Prevention Best Practices

### Pre-Generation Validation
```bash
# Validate parameters before generation
./bloco-vgen --validate-params --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}' --target-client geth
```

### Use Presets for Reliability
```bash
# Use tested presets instead of custom parameters
./bloco-vgen --prefix abc --preset production-geth
./bloco-vgen --prefix abc --preset development-anvil
./bloco-vgen --prefix abc --preset enterprise-besu
```

### Regular Compatibility Testing
```bash
# Test keystores with actual clients regularly
./bloco-vgen --test-with-client geth ./keystores/0xabc123....json
./bloco-vgen --test-with-client besu ./keystores/0xabc123....json
```

## FAQ

### Q: Why is my keystore not working with geth?
**A**: Geth requires specific scrypt parameters. Use the geth preset:
```bash
./bloco-vgen --prefix abc --preset geth
```

### Q: How do I make keystores compatible with all clients?
**A**: Use standard parameters that all clients support:
```bash
./bloco-vgen --prefix abc --keystore-kdf scrypt --kdf-params '{
  "n": 262144,
  "r": 8,
  "p": 1,
  "dklen": 32
}'
```

### Q: What's the fastest KDF configuration that's still secure?
**A**: Balanced PBKDF2 configuration:
```bash
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 300000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

### Q: How do I reduce memory usage without compromising security?
**A**: Switch to PBKDF2 with high iteration count:
```bash
./bloco-vgen --prefix abc --keystore-kdf pbkdf2 --kdf-params '{
  "c": 600000,
  "prf": "hmac-sha256",
  "dklen": 32
}'
```

### Q: Can I convert between KDF types?
**A**: Yes, use the migration tool:
```bash
./bloco-vgen --migrate-kdf ./keystore.json --from scrypt --to pbkdf2 --password password.txt
```

### Q: How do I verify my keystore is secure?
**A**: Use the security analysis tool:
```bash
./bloco-vgen --analyze-security ./keystore.json
```

This troubleshooting guide covers the most common issues you'll encounter when working with KDF configurations in bloco-vgen. For additional support, use the built-in diagnostic tools and refer to the [KDF Configuration Examples](KDF_CONFIGURATION_EXAMPLES.md) document.