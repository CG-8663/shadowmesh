# Symmetric Cryptography Module

This module provides high-performance symmetric encryption for ShadowMesh's Layer 2 network traffic encryption.

## Overview

- **Algorithm**: ChaCha20-Poly1305 AEAD (Authenticated Encryption with Associated Data)
- **Performance**: 6.87 Gbps throughput (850+ MB/s on Apple M1 Max)
- **Security**: IND-CCA2 secure with authenticated encryption
- **Standard**: RFC 8439

## Components

### ChaCha20-Poly1305 AEAD (`chacha20poly1305.go`)

Provides authenticated encryption for Ethernet frames with automatic integrity verification.

```go
import "github.com/shadowmesh/shadowmesh/pkg/crypto/symmetric"

// Generate key and nonce
var key [32]byte
var nonce [12]byte
rand.Read(key[:])
rand.Read(nonce[:])

// Encrypt plaintext
plaintext := []byte("Secret message")
frame, err := symmetric.Encrypt(plaintext, key, nonce)
if err != nil {
    log.Fatal(err)
}

// Decrypt ciphertext
decrypted, err := symmetric.Decrypt(frame, key)
if err != nil {
    log.Fatal(err) // Authentication failed - ciphertext was tampered
}
```

### Nonce Generator (`nonce.go`)

Thread-safe nonce generation with uniqueness guarantees.

```go
// Create nonce generator
ng, err := symmetric.NewNonceGenerator()
if err != nil {
    log.Fatal(err)
}

// Generate unique nonces
for i := 0; i < 1000; i++ {
    nonce, err := ng.GenerateNonce()
    if err != nil {
        log.Fatal(err)
    }
    // Use nonce for encryption...
}
```

## API Reference

### Encryption

```go
func Encrypt(plaintext []byte, key [32]byte, nonce [12]byte) (*EncryptedFrame, error)
```

Encrypts plaintext using ChaCha20-Poly1305 AEAD.

**Parameters:**
- `plaintext` - Data to encrypt (any size)
- `key` - 32-byte encryption key
- `nonce` - 12-byte nonce (must be unique per key)

**Returns:**
- `*EncryptedFrame` - Encrypted frame with nonce and ciphertext
- `error` - Error if encryption fails

**Security:**
- ⚠️ **Never reuse** (key, nonce) pairs - this breaks AEAD security
- Nonce uniqueness is critical - use `NonceGenerator` for automatic management

### Decryption

```go
func Decrypt(frame *EncryptedFrame, key [32]byte) ([]byte, error)
```

Decrypts and authenticates ciphertext using ChaCha20-Poly1305 AEAD.

**Parameters:**
- `frame` - Encrypted frame from `Encrypt()`
- `key` - 32-byte decryption key (must match encryption key)

**Returns:**
- `[]byte` - Decrypted plaintext
- `error` - Error if authentication fails (tampering detected)

**Security:**
- Constant-time tag comparison (timing attack resistant)
- Fails immediately on tag mismatch (no partial decryption)

### Nonce Generation

```go
func NewNonceGenerator() (*NonceGenerator, error)
```

Creates a new nonce generator with random salt.

**Nonce Format:**
```
[6 bytes counter (big-endian)][6 bytes random salt]
```

- **Counter**: 48-bit atomic counter (0 to 2^48-1 = 281 trillion)
- **Salt**: 48-bit random salt (regenerated on counter overflow)
- **Uniqueness**: Guaranteed within a single generator instance
- **Thread-safe**: Can be called concurrently from multiple goroutines

```go
func (ng *NonceGenerator) GenerateNonce() ([12]byte, error)
```

Generates a unique 12-byte nonce.

**Performance:** ~15 ns/op (single-threaded), ~145 ns/op (parallel)

## Performance Characteristics

### Throughput Benchmarks (Apple M1 Max)

| Frame Size | Encrypt | Decrypt | Throughput |
|------------|---------|---------|------------|
| 1 KB       | 1.57 µs | 1.55 µs | 652 MB/s   |
| 10 KB      | 12.4 µs | 12.5 µs | 828 MB/s   |
| 100 KB     | 120 µs  | 121 µs  | 851 MB/s   |
| 1 MB       | 1.22 ms | 1.23 ms | **859 MB/s** |

**Converted to Gbps**: 859 MB/s × 8 = **6.87 Gbps** ✅ (Target: 1+ Gbps)

### Memory Allocation

- **Encryption**: 4 allocations per operation (nonce copy, ciphertext buffer, frame struct, return)
- **Decryption**: 2 allocations per operation (plaintext buffer, return)
- **Nonce Generation**: 0 allocations (stack-only)

### Latency

- **Nonce generation**: 15 ns (single-threaded), 145 ns (parallel)
- **1 KB frame**: 1.57 µs encryption + 1.55 µs decryption = **3.12 µs total**
- **1500 byte Ethernet frame**: ~2.4 µs total (estimated)

## Security Properties

### AEAD Security

- **IND-CCA2**: Indistinguishability under chosen-ciphertext attack
- **EUF-CMA**: Existential unforgeability under chosen-message attack
- **128-bit security**: Equivalent to AES-128 (quantum-safe against Grover's algorithm with 64-bit post-quantum security)

### Nonce Uniqueness

- **Counter**: Increments atomically for each frame (guarantees uniqueness)
- **Salt**: Random 48-bit value prevents cross-session collisions
- **Overflow**: After 2^48 frames (~281 trillion), salt regenerates automatically

### Constant-Time Operations

- **Tag comparison**: Uses `subtle.ConstantTimeCompare()` (timing attack resistant)
- **AEAD.Open()**: Constant-time tag validation (no early exit on mismatch)

## Integration with ShadowMesh

### Ethernet Frame Encryption (Epic 2, Story 2.6)

```go
import (
    "github.com/shadowmesh/shadowmesh/pkg/crypto/symmetric"
    "github.com/shadowmesh/shadowmesh/pkg/crypto/hybrid"
)

// Setup: Derive session key from hybrid key exchange
sessionKey, _ := hybrid.DeriveSharedSecret(ciphertext, privateKey)

// Create nonce generator for this session
ng, _ := symmetric.NewNonceGenerator()

// Encrypt Ethernet frame (1500 bytes MTU)
for {
    ethernetFrame := readTAPDevice() // 1500 bytes
    nonce, _ := ng.GenerateNonce()

    var key [32]byte
    copy(key[:], sessionKey)

    encrypted, _ := symmetric.Encrypt(ethernetFrame, key, nonce)
    sendOverWebSocket(encrypted)
}
```

### Key Rotation (Epic 1, Story 1.5)

After key rotation, create a new `NonceGenerator` for the new session:

```go
// After key rotation
newSessionKey := rotateKey(oldSessionKey)

// Create new nonce generator for new session
ng, _ = symmetric.NewNonceGenerator()

// Continue encrypting with new key and new nonce generator
```

## Error Handling

### Common Errors

| Error | Cause | Resolution |
|-------|-------|------------|
| `ErrInvalidKeySize` | Key is not 32 bytes | Use `[32]byte` key or check length |
| `ErrInvalidNonceSize` | Nonce is not 12 bytes | Use `[12]byte` nonce from `NonceGenerator` |
| `ErrDecryptionFailed` | Tag validation failed | Ciphertext was tampered or wrong key |
| `ErrInvalidCiphertext` | Ciphertext too short | Must be at least 16 bytes (tag size) |
| `ErrCounterOverflow` | 2^48 frames exceeded | Automatic salt regeneration (recoverable) |

### Best Practices

1. **Always use `NonceGenerator`** - Don't generate nonces manually
2. **One generator per session** - Create new generator after key rotation
3. **Check errors** - Authentication failures indicate tampering
4. **Secure key storage** - Use encrypted keystore (Story 1.6)
5. **Key rotation** - Rotate keys every 5 minutes (Story 1.5)

## Testing

### Unit Tests

```bash
go test ./pkg/crypto/symmetric/... -v
```

**Coverage**: 81.9% of statements

### Benchmarks

```bash
go test -bench=. -benchmem ./pkg/crypto/symmetric/
```

### Nonce Uniqueness Test

The test suite generates 10,000 nonces sequentially and 100,000 nonces concurrently to verify uniqueness:

```bash
go test -v -run=TestNonceUniqueness ./pkg/crypto/symmetric/
go test -v -run=TestConcurrentNonceGeneration ./pkg/crypto/symmetric/
```

## Migration from Legacy Code

The old `pkg/crypto/chacha20.go` used XChaCha20-Poly1305 (24-byte nonce) and is now deprecated.

**Migration steps:**

1. Import new module: `import "github.com/shadowmesh/shadowmesh/pkg/crypto/symmetric"`
2. Replace `NewChaCha20Cipher()` with `symmetric.NewNonceGenerator()` + `symmetric.Encrypt()`
3. Update nonce size: 24 bytes → 12 bytes
4. Use structured API: `EncryptedFrame` instead of raw bytes

## References

- [RFC 8439](https://datatracker.ietf.org/doc/html/rfc8439) - ChaCha20-Poly1305 AEAD
- [golang.org/x/crypto/chacha20poly1305](https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305) - Go implementation
- [NIST SP 800-38D](https://csrc.nist.gov/publications/detail/sp/800-38d/final) - AEAD modes of operation

## License

Part of ShadowMesh DPN - Post-quantum decentralized private network.
