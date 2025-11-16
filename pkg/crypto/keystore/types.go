// Package keystore provides encrypted storage for ShadowMesh hybrid keypairs.
// Uses AES-256-GCM with PBKDF2-derived keys for passphrase-protected storage.
package keystore

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/hybrid"
)

const (
	// KeystoreVersion is the current keystore format version
	KeystoreVersion = "1.0"
	// DefaultKDF is the key derivation function
	DefaultKDF = "pbkdf2-hmac-sha256"
	// DefaultCipher is the encryption cipher
	DefaultCipher = "aes-256-gcm"
	// DefaultIterations is the PBKDF2 iteration count
	DefaultIterations = 100000
	// SaltSize is the size of the PBKDF2 salt in bytes
	SaltSize = 32
	// IVSize is the size of the AES-GCM IV (nonce) in bytes
	IVSize = 12
)

var (
	// ErrInvalidKeystoreVersion indicates unsupported keystore version
	ErrInvalidKeystoreVersion = errors.New("invalid or unsupported keystore version")
	// ErrInvalidKDF indicates unsupported KDF
	ErrInvalidKDF = errors.New("invalid or unsupported KDF")
	// ErrInvalidCipher indicates unsupported cipher
	ErrInvalidCipher = errors.New("invalid or unsupported cipher")
	// ErrInvalidKeystore indicates corrupted or malformed keystore
	ErrInvalidKeystore = errors.New("invalid keystore format")
	// ErrWrongPassphrase indicates passphrase does not match
	ErrWrongPassphrase = errors.New("wrong passphrase or corrupted keystore")
)

// KeystoreFile is the JSON structure saved to disk
type KeystoreFile struct {
	Version    string    `json:"version"`    // "1.0"
	KDF        string    `json:"kdf"`        // "pbkdf2-hmac-sha256"
	KDFParams  KDFParams `json:"kdf_params"` // PBKDF2 parameters
	Cipher     string    `json:"cipher"`     // "aes-256-gcm"
	Ciphertext string    `json:"ciphertext"` // Base64-encoded encrypted data
	IV         string    `json:"iv"`         // Base64-encoded IV (12 bytes)
	// Note: GCM tag is included in Ciphertext
}

// KDFParams contains PBKDF2 parameters
type KDFParams struct {
	Iterations int    `json:"iterations"` // 100000
	Salt       string `json:"salt"`       // Base64-encoded (32 bytes)
}

// KeystoreData is the plaintext data structure (encrypted in keystore)
type KeystoreData struct {
	// ML-KEM-1024 keys (1568 + 3168 bytes)
	MLKEMPublicKey  string `json:"mlkem_public_key"`  // Base64
	MLKEMPrivateKey string `json:"mlkem_private_key"` // Base64

	// X25519 keys (32 + 32 bytes)
	X25519PublicKey  string `json:"x25519_public_key"`  // Base64
	X25519PrivateKey string `json:"x25519_private_key"` // Base64

	// ML-DSA-87 keys (2592 + 4864 bytes)
	MLDSAPublicKey  string `json:"mldsa_public_key"`  // Base64
	MLDSAPrivateKey string `json:"mldsa_private_key"` // Base64

	// Ed25519 keys (32 + 64 bytes)
	Ed25519PublicKey  string `json:"ed25519_public_key"`  // Base64
	Ed25519PrivateKey string `json:"ed25519_private_key"` // Base64

	// Metadata
	CreatedAt string `json:"created_at"` // RFC3339 timestamp
	ExpiresAt string `json:"expires_at"` // RFC3339 timestamp (optional, can be empty)
}

// Validate validates the keystore file structure
func (kf *KeystoreFile) Validate() error {
	if kf.Version != KeystoreVersion {
		return fmt.Errorf("%w: got %s, expected %s", ErrInvalidKeystoreVersion, kf.Version, KeystoreVersion)
	}

	if kf.KDF != DefaultKDF {
		return fmt.Errorf("%w: got %s, expected %s", ErrInvalidKDF, kf.KDF, DefaultKDF)
	}

	if kf.Cipher != DefaultCipher {
		return fmt.Errorf("%w: got %s, expected %s", ErrInvalidCipher, kf.Cipher, DefaultCipher)
	}

	if kf.KDFParams.Iterations <= 0 {
		return fmt.Errorf("%w: iterations must be positive", ErrInvalidKeystore)
	}

	if kf.KDFParams.Salt == "" {
		return fmt.Errorf("%w: salt is required", ErrInvalidKeystore)
	}

	if kf.Ciphertext == "" {
		return fmt.Errorf("%w: ciphertext is required", ErrInvalidKeystore)
	}

	if kf.IV == "" {
		return fmt.Errorf("%w: IV is required", ErrInvalidKeystore)
	}

	return nil
}

// ToHybridKeypair converts KeystoreData to HybridKeypair
func (kd *KeystoreData) ToHybridKeypair() (*hybrid.HybridKeypair, error) {
	// Decode base64 keys
	mlkemPub, err := base64.StdEncoding.DecodeString(kd.MLKEMPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-KEM public key: %w", err)
	}

	mlkemPriv, err := base64.StdEncoding.DecodeString(kd.MLKEMPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-KEM private key: %w", err)
	}

	x25519Pub, err := base64.StdEncoding.DecodeString(kd.X25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X25519 public key: %w", err)
	}

	x25519Priv, err := base64.StdEncoding.DecodeString(kd.X25519PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X25519 private key: %w", err)
	}

	mldsaPub, err := base64.StdEncoding.DecodeString(kd.MLDSAPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-DSA public key: %w", err)
	}

	mldsaPriv, err := base64.StdEncoding.DecodeString(kd.MLDSAPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-DSA private key: %w", err)
	}

	ed25519Pub, err := base64.StdEncoding.DecodeString(kd.Ed25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Ed25519 public key: %w", err)
	}

	ed25519Priv, err := base64.StdEncoding.DecodeString(kd.Ed25519PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Ed25519 private key: %w", err)
	}

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, kd.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at timestamp: %w", err)
	}

	var expiresAt time.Time
	if kd.ExpiresAt != "" {
		expiresAt, err = time.Parse(time.RFC3339, kd.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expires_at timestamp: %w", err)
		}
	}

	return &hybrid.HybridKeypair{
		MLKEMPublicKey:    mlkemPub,
		MLKEMPrivateKey:   mlkemPriv,
		X25519PublicKey:   x25519Pub,
		X25519PrivateKey:  x25519Priv,
		MLDSAPublicKey:    mldsaPub,
		MLDSAPrivateKey:   mldsaPriv,
		Ed25519PublicKey:  ed25519Pub,
		Ed25519PrivateKey: ed25519Priv,
		CreatedAt:         createdAt,
		ExpiresAt:         expiresAt,
	}, nil
}

// FromHybridKeypair converts HybridKeypair to KeystoreData
func FromHybridKeypair(keypair *hybrid.HybridKeypair) *KeystoreData {
	return &KeystoreData{
		MLKEMPublicKey:    base64.StdEncoding.EncodeToString(keypair.MLKEMPublicKey),
		MLKEMPrivateKey:   base64.StdEncoding.EncodeToString(keypair.MLKEMPrivateKey),
		X25519PublicKey:   base64.StdEncoding.EncodeToString(keypair.X25519PublicKey),
		X25519PrivateKey:  base64.StdEncoding.EncodeToString(keypair.X25519PrivateKey),
		MLDSAPublicKey:    base64.StdEncoding.EncodeToString(keypair.MLDSAPublicKey),
		MLDSAPrivateKey:   base64.StdEncoding.EncodeToString(keypair.MLDSAPrivateKey),
		Ed25519PublicKey:  base64.StdEncoding.EncodeToString(keypair.Ed25519PublicKey),
		Ed25519PrivateKey: base64.StdEncoding.EncodeToString(keypair.Ed25519PrivateKey),
		CreatedAt:         keypair.CreatedAt.Format(time.RFC3339),
		ExpiresAt:         keypair.ExpiresAt.Format(time.RFC3339),
	}
}
