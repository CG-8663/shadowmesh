package mlkem

import (
	"errors"
	"fmt"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber1024"
)

var (
	// ErrInvalidCiphertext indicates the ciphertext format is invalid or corrupted
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
	// ErrKeyGenerationFailed indicates key generation failed
	ErrKeyGenerationFailed = errors.New("key generation failed")
	// ErrDecapsulationFailed indicates decapsulation operation failed
	ErrDecapsulationFailed = errors.New("decapsulation failed")
)

// MLKEMKeypair represents an ML-KEM-1024 keypair
type MLKEMKeypair struct {
	PublicKey  []byte // 1568 bytes
	PrivateKey []byte // 3168 bytes
}

// GenerateKeypair generates a new ML-KEM-1024 keypair using NIST FIPS 203
// Returns error if random number generation fails
func GenerateKeypair() (*MLKEMKeypair, error) {
	scheme := kyber1024.Scheme()

	// Generate keypair using system entropy
	pk, sk, err := scheme.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	// Pack keys into byte slices
	pkBytes, err := pk.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal public key: %v", ErrKeyGenerationFailed, err)
	}

	skBytes, err := sk.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal private key: %v", ErrKeyGenerationFailed, err)
	}

	return &MLKEMKeypair{
		PublicKey:  pkBytes,
		PrivateKey: skBytes,
	}, nil
}

// Encapsulate performs ML-KEM-1024 encapsulation with the given public key
// Returns ciphertext (1568 bytes) and shared secret (32 bytes)
// This operation is IND-CCA2 secure against quantum attacks per NIST FIPS 203
func Encapsulate(publicKey []byte) (ciphertext []byte, sharedSecret []byte, err error) {
	if len(publicKey) != kyber1024.Scheme().PublicKeySize() {
		return nil, nil, fmt.Errorf("%w: expected %d bytes, got %d",
			ErrInvalidCiphertext, kyber1024.Scheme().PublicKeySize(), len(publicKey))
	}

	scheme := kyber1024.Scheme()

	// Unmarshal public key
	pk, err := scheme.UnmarshalBinaryPublicKey(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to unmarshal public key: %v", ErrInvalidCiphertext, err)
	}

	// Perform encapsulation using system entropy
	// Returns ciphertext as []byte and shared secret as []byte directly
	ct, ss, err := scheme.Encapsulate(pk)
	if err != nil {
		return nil, nil, fmt.Errorf("encapsulation failed: %w", err)
	}

	return ct, ss, nil
}

// Decapsulate performs ML-KEM-1024 decapsulation with the given private key
// Returns shared secret (32 bytes)
// This operation uses constant-time comparisons to prevent timing attacks
func Decapsulate(ciphertext []byte, privateKey []byte) (sharedSecret []byte, err error) {
	if len(privateKey) != kyber1024.Scheme().PrivateKeySize() {
		return nil, fmt.Errorf("%w: invalid private key size: expected %d bytes, got %d",
			ErrDecapsulationFailed, kyber1024.Scheme().PrivateKeySize(), len(privateKey))
	}

	if len(ciphertext) != kyber1024.Scheme().CiphertextSize() {
		return nil, fmt.Errorf("%w: invalid ciphertext size: expected %d bytes, got %d",
			ErrInvalidCiphertext, kyber1024.Scheme().CiphertextSize(), len(ciphertext))
	}

	scheme := kyber1024.Scheme()

	// Unmarshal private key
	sk, err := scheme.UnmarshalBinaryPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal private key: %v", ErrDecapsulationFailed, err)
	}

	// Perform decapsulation (constant-time operation)
	// Ciphertext is passed as []byte directly
	ss, err := scheme.Decapsulate(sk, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecapsulationFailed, err)
	}

	return ss, nil
}

// Scheme returns the ML-KEM-1024 (Kyber1024) KEM scheme
// Useful for accessing size constants and algorithm metadata
func Scheme() kem.Scheme {
	return kyber1024.Scheme()
}
