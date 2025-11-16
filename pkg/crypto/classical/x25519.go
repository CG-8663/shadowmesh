package classical

import (
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"fmt"
)

var (
	// ErrInvalidPublicKey indicates the public key format is invalid
	ErrInvalidPublicKey = errors.New("invalid public key format")
	// ErrKeyGenerationFailed indicates key generation failed
	ErrKeyGenerationFailed = errors.New("key generation failed")
	// ErrECDHFailed indicates ECDH operation failed
	ErrECDHFailed = errors.New("ECDH operation failed")
)

// X25519Keypair represents an X25519 ECDH keypair
type X25519Keypair struct {
	PublicKey  []byte // 32 bytes
	PrivateKey []byte // 32 bytes
}

// GenerateX25519Keypair generates a new X25519 keypair using crypto/ecdh
// Returns error if random number generation fails
func GenerateX25519Keypair() (*X25519Keypair, error) {
	// Generate X25519 private key using system entropy
	privKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	return &X25519Keypair{
		PublicKey:  privKey.PublicKey().Bytes(),
		PrivateKey: privKey.Bytes(),
	}, nil
}

// X25519Exchange performs ECDH key exchange with the given keys
// Returns 32-byte shared secret
// This is a constant-time operation per RFC 7748
func X25519Exchange(privateKey, publicKey []byte) (sharedSecret []byte, err error) {
	if len(privateKey) != 32 {
		return nil, fmt.Errorf("%w: private key must be 32 bytes, got %d", ErrECDHFailed, len(privateKey))
	}

	if len(publicKey) != 32 {
		return nil, fmt.Errorf("%w: public key must be 32 bytes, got %d", ErrInvalidPublicKey, len(publicKey))
	}

	// Parse private key
	priv, err := ecdh.X25519().NewPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse private key: %v", ErrECDHFailed, err)
	}

	// Parse public key
	pub, err := ecdh.X25519().NewPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse public key: %v", ErrInvalidPublicKey, err)
	}

	// Perform ECDH (constant-time operation)
	secret, err := priv.ECDH(pub)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrECDHFailed, err)
	}

	return secret, nil
}
