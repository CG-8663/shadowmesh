package hybrid

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/classical"
	"github.com/shadowmesh/shadowmesh/pkg/crypto/mldsa"
	"github.com/shadowmesh/shadowmesh/pkg/crypto/mlkem"
	"golang.org/x/crypto/hkdf"
)

const (
	// HybridKDFSalt is the salt used for HKDF-SHA256 combining both shared secrets
	HybridKDFSalt = "shadowmesh-hybrid-kex"
	// HybridKDFInfo is the info parameter for HKDF
	HybridKDFInfo = "ShadowMesh-v1-Hybrid-KEM"
	// SharedSecretSize is the output size (32 bytes for ChaCha20-Poly1305)
	SharedSecretSize = 32
)

var (
	// ErrInvalidCiphertext indicates the ciphertext format is invalid
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
	// ErrKeyGenerationFailed indicates key generation failed
	ErrKeyGenerationFailed = errors.New("key generation failed")
	// ErrEncapsulationFailed indicates encapsulation failed
	ErrEncapsulationFailed = errors.New("encapsulation failed")
	// ErrDecapsulationFailed indicates decapsulation failed
	ErrDecapsulationFailed = errors.New("decapsulation failed")
)

// HybridKeypair contains both ML-KEM-1024 and X25519 keypairs with metadata
type HybridKeypair struct {
	// Post-quantum keys (ML-KEM-1024)
	MLKEMPublicKey  []byte // 1568 bytes
	MLKEMPrivateKey []byte // 3168 bytes

	// Post-quantum signature keys (ML-DSA-87)
	MLDSAPublicKey  []byte // 2592 bytes
	MLDSAPrivateKey []byte // 4864 bytes

	// Classical keys (X25519)
	X25519PublicKey  []byte // 32 bytes
	X25519PrivateKey []byte // 32 bytes

	// Classical signature keys (Ed25519)
	Ed25519PublicKey  []byte // 32 bytes
	Ed25519PrivateKey []byte // 64 bytes

	// Metadata
	CreatedAt time.Time
	ExpiresAt time.Time
}

// GenerateHybridKeypair creates a new hybrid keypair combining ML-KEM-1024 and X25519
// This operation should complete in <100ms on commodity hardware (4 GHz CPU)
func GenerateHybridKeypair() (*HybridKeypair, error) {
	// Generate ML-KEM-1024 keypair
	mlkemKP, err := mlkem.GenerateKeypair()
	if err != nil {
		return nil, fmt.Errorf("%w: ML-KEM generation failed: %v", ErrKeyGenerationFailed, err)
	}

	// Generate ML-DSA-87 keypair
	mldsaKP, err := mldsa.GenerateKeypair()
	if err != nil {
		return nil, fmt.Errorf("%w: ML-DSA generation failed: %v", ErrKeyGenerationFailed, err)
	}

	// Generate X25519 keypair
	x25519KP, err := classical.GenerateX25519Keypair()
	if err != nil {
		return nil, fmt.Errorf("%w: X25519 generation failed: %v", ErrKeyGenerationFailed, err)
	}

	// Generate Ed25519 keypair
	ed25519KP, err := classical.GenerateEd25519Keypair()
	if err != nil {
		return nil, fmt.Errorf("%w: Ed25519 generation failed: %v", ErrKeyGenerationFailed, err)
	}

	now := time.Now()
	return &HybridKeypair{
		MLKEMPublicKey:    mlkemKP.PublicKey,
		MLKEMPrivateKey:   mlkemKP.PrivateKey,
		MLDSAPublicKey:    mldsaKP.PublicKey,
		MLDSAPrivateKey:   mldsaKP.PrivateKey,
		X25519PublicKey:   x25519KP.PublicKey,
		X25519PrivateKey:  x25519KP.PrivateKey,
		Ed25519PublicKey:  ed25519KP.PublicKey,
		Ed25519PrivateKey: ed25519KP.PrivateKey,
		CreatedAt:         now,
		ExpiresAt:         now.Add(5 * time.Minute), // Default 5-minute expiration
	}, nil
}

// HybridEncapsulate performs hybrid key encapsulation against a public key
// Returns ciphertext and 32-byte shared secret
// Encapsulation + decapsulation roundtrip should complete in <50ms
func HybridEncapsulate(publicKey *HybridKeypair) (ciphertext []byte, sharedSecret []byte, err error) {
	if publicKey == nil {
		return nil, nil, fmt.Errorf("%w: public key cannot be nil", ErrEncapsulationFailed)
	}

	// Perform ML-KEM-1024 encapsulation
	kemCT, kemSecret, err := mlkem.Encapsulate(publicKey.MLKEMPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: ML-KEM encapsulation failed: %v", ErrEncapsulationFailed, err)
	}

	// Generate ephemeral X25519 keypair for ECDH
	ephemeralKP, err := classical.GenerateX25519Keypair()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: ephemeral X25519 generation failed: %v", ErrEncapsulationFailed, err)
	}

	// Perform X25519 ECDH with recipient's public key
	ecdhSecret, err := classical.X25519Exchange(ephemeralKP.PrivateKey, publicKey.X25519PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: X25519 exchange failed: %v", ErrEncapsulationFailed, err)
	}

	// Derive final shared secret using HKDF-SHA256 with both secrets
	hybridSecret, err := deriveSharedSecret(kemSecret, ecdhSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: HKDF derivation failed: %v", ErrEncapsulationFailed, err)
	}

	// Construct ciphertext: KEM ciphertext (1568 bytes) || ephemeral ECDH public key (32 bytes)
	// Total ciphertext size: 1600 bytes
	combinedCT := make([]byte, len(kemCT)+len(ephemeralKP.PublicKey))
	copy(combinedCT, kemCT)
	copy(combinedCT[len(kemCT):], ephemeralKP.PublicKey)

	return combinedCT, hybridSecret, nil
}

// HybridDecapsulate performs hybrid key decapsulation using private key and ciphertext
// Returns 32-byte shared secret
// Encapsulation + decapsulation roundtrip should complete in <50ms
func HybridDecapsulate(ciphertext []byte, privateKey *HybridKeypair) (sharedSecret []byte, err error) {
	if privateKey == nil {
		return nil, fmt.Errorf("%w: private key cannot be nil", ErrDecapsulationFailed)
	}

	// Expected ciphertext format: KEM ciphertext (1568 bytes) || ECDH ephemeral public key (32 bytes)
	expectedSize := mlkem.Scheme().CiphertextSize() + 32
	if len(ciphertext) != expectedSize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidCiphertext, expectedSize, len(ciphertext))
	}

	// Split ciphertext into KEM ciphertext and ECDH public key
	kemCTSize := mlkem.Scheme().CiphertextSize()
	kemCT := ciphertext[:kemCTSize]
	ecdhEphemeralPub := ciphertext[kemCTSize:]

	// Perform ML-KEM-1024 decapsulation
	kemSecret, err := mlkem.Decapsulate(kemCT, privateKey.MLKEMPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: ML-KEM decapsulation failed: %v", ErrDecapsulationFailed, err)
	}

	// Perform X25519 ECDH with ephemeral public key
	ecdhSecret, err := classical.X25519Exchange(privateKey.X25519PrivateKey, ecdhEphemeralPub)
	if err != nil {
		return nil, fmt.Errorf("%w: X25519 exchange failed: %v", ErrDecapsulationFailed, err)
	}

	// Derive final shared secret using HKDF-SHA256 with both secrets
	hybridSecret, err := deriveSharedSecret(kemSecret, ecdhSecret)
	if err != nil {
		return nil, fmt.Errorf("%w: HKDF derivation failed: %v", ErrDecapsulationFailed, err)
	}

	return hybridSecret, nil
}

// deriveSharedSecret combines ML-KEM and X25519 shared secrets using HKDF-SHA256
// Uses salt "shadowmesh-hybrid-kex" and info "ShadowMesh-v1-Hybrid-KEM"
// Returns 32-byte final shared secret suitable for ChaCha20-Poly1305
func deriveSharedSecret(kemSecret, ecdhSecret []byte) ([]byte, error) {
	// Concatenate both secrets: kemSecret || ecdhSecret
	combinedSecret := make([]byte, len(kemSecret)+len(ecdhSecret))
	copy(combinedSecret, kemSecret)
	copy(combinedSecret[len(kemSecret):], ecdhSecret)

	// Use HKDF-SHA256 to derive final shared secret
	hash := sha256.New
	hkdf := hkdf.New(hash, combinedSecret, []byte(HybridKDFSalt), []byte(HybridKDFInfo))

	// Extract 32 bytes for ChaCha20-Poly1305
	sharedSecret := make([]byte, SharedSecretSize)
	if _, err := io.ReadFull(hkdf, sharedSecret); err != nil {
		return nil, fmt.Errorf("HKDF extraction failed: %w", err)
	}

	return sharedSecret, nil
}
