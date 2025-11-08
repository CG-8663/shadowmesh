package client

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudflare/circl/sign/dilithium/mode5"
)

// KeyPair represents an ML-DSA-87 key pair
type KeyPair struct {
	PublicKey  *mode5.PublicKey
	PrivateKey *mode5.PrivateKey
	PeerID     string // Derived from public key
}

// GenerateKeyPair generates a new ML-DSA-87 key pair
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := mode5.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Compute peer ID from public key
	publicKeyBytes := publicKey.Bytes()
	peerID := ComputePeerID(publicKeyBytes)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		PeerID:     peerID,
	}, nil
}

// SaveKeyPair saves the key pair to disk
func (kp *KeyPair) Save(directory string) error {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save private key
	privateKeyPath := filepath.Join(directory, "private_key.bin")
	privateKeyBytes := kp.PrivateKey.Bytes()
	if err := os.WriteFile(privateKeyPath, privateKeyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key
	publicKeyPath := filepath.Join(directory, "public_key.bin")
	publicKeyBytes := kp.PublicKey.Bytes()
	if err := os.WriteFile(publicKeyPath, publicKeyBytes, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Save peer ID
	peerIDPath := filepath.Join(directory, "peer_id.txt")
	if err := os.WriteFile(peerIDPath, []byte(kp.PeerID), 0644); err != nil {
		return fmt.Errorf("failed to write peer ID: %w", err)
	}

	return nil
}

// LoadKeyPair loads a key pair from disk
func LoadKeyPair(directory string) (*KeyPair, error) {
	// Load private key
	privateKeyPath := filepath.Join(directory, "private_key.bin")
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	if len(privateKeyBytes) != mode5.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}

	var privateKey mode5.PrivateKey
	if err := privateKey.UnmarshalBinary(privateKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	// Load public key
	publicKeyPath := filepath.Join(directory, "public_key.bin")
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	if len(publicKeyBytes) != mode5.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}

	var publicKey mode5.PublicKey
	if err := publicKey.UnmarshalBinary(publicKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	// Compute peer ID
	peerID := ComputePeerID(publicKeyBytes)

	return &KeyPair{
		PublicKey:  &publicKey,
		PrivateKey: &privateKey,
		PeerID:     peerID,
	}, nil
}

// Sign signs a message with the private key
func (kp *KeyPair) Sign(message []byte) ([]byte, error) {
	signature := make([]byte, mode5.SignatureSize)
	mode5.SignTo(kp.PrivateKey, message, signature)
	return signature, nil
}

// PublicKeyHex returns the public key as hex string
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey.Bytes())
}

// ComputePeerID computes peer ID from public key (first 20 bytes of public key)
func ComputePeerID(publicKey []byte) string {
	if len(publicKey) >= 20 {
		return hex.EncodeToString(publicKey[:20])
	}
	return hex.EncodeToString(publicKey)
}
