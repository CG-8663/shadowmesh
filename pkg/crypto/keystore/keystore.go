// Package keystore provides encrypted storage for ShadowMesh hybrid keypairs.
// Uses AES-256-GCM with PBKDF2-derived keys for passphrase-protected storage.
package keystore

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/hybrid"
)

// Save encrypts and saves a hybrid keypair to disk.
//
// Parameters:
//   - keypair: HybridKeypair to save (ML-KEM + X25519 + ML-DSA + Ed25519)
//   - passphrase: User passphrase (must pass ValidatePassphrase)
//   - path: File path to save keystore (will be created with 0600 permissions)
//
// Returns:
//   - error if save fails
//
// Process:
//  1. Validate passphrase (minimum 12 characters)
//  2. Convert keypair to KeystoreData (base64-encoded keys)
//  3. Marshal KeystoreData to JSON
//  4. Generate random salt (32 bytes)
//  5. Derive encryption key using PBKDF2-HMAC-SHA256 (100k iterations)
//  6. Encrypt JSON using AES-256-GCM (generates random IV)
//  7. Create KeystoreFile with ciphertext, IV, KDF params
//  8. Marshal KeystoreFile to JSON and save to disk
//  9. Set file permissions to 0600 (user read/write only)
//
// Security Notes:
//   - Passphrase is never written to disk
//   - Salt and IV are randomly generated for each save
//   - File permissions prevent other users from reading
//   - PBKDF2 makes brute-force attacks computationally expensive
func Save(keypair *hybrid.HybridKeypair, passphrase string, path string) error {
	// Validate passphrase
	if err := ValidatePassphrase(passphrase); err != nil {
		return fmt.Errorf("invalid passphrase: %w", err)
	}

	// Validate keypair
	if keypair == nil {
		return fmt.Errorf("keypair cannot be nil")
	}

	// Convert keypair to KeystoreData
	keystoreData := FromHybridKeypair(keypair)

	// Marshal KeystoreData to JSON (plaintext)
	plaintextJSON, err := json.Marshal(keystoreData)
	if err != nil {
		return fmt.Errorf("failed to marshal keypair data: %w", err)
	}

	// Generate random salt for PBKDF2
	var salt [SaltSize]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key from passphrase using PBKDF2
	key, err := DeriveKey(passphrase, salt[:], DefaultIterations)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// Encrypt plaintext JSON using AES-256-GCM
	encrypted, err := Encrypt(plaintextJSON, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt keystore: %w", err)
	}

	// Zero the key from memory after use
	for i := range key {
		key[i] = 0
	}

	// Create KeystoreFile structure
	keystoreFile := KeystoreFile{
		Version: KeystoreVersion,
		KDF:     DefaultKDF,
		KDFParams: KDFParams{
			Iterations: DefaultIterations,
			Salt:       base64.StdEncoding.EncodeToString(salt[:]),
		},
		Cipher:     DefaultCipher,
		Ciphertext: base64.StdEncoding.EncodeToString(encrypted.Ciphertext),
		IV:         base64.StdEncoding.EncodeToString(encrypted.IV[:]),
	}

	// Marshal KeystoreFile to JSON
	keystoreJSON, err := json.MarshalIndent(keystoreFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keystore file: %w", err)
	}

	// Write to file with restricted permissions (0600 = user read/write only)
	if err := os.WriteFile(path, keystoreJSON, 0600); err != nil {
		return fmt.Errorf("failed to write keystore file: %w", err)
	}

	return nil
}

// Load decrypts and loads a hybrid keypair from disk.
//
// Parameters:
//   - passphrase: User passphrase (must match the one used for Save)
//   - path: File path to keystore
//
// Returns:
//   - *HybridKeypair containing decrypted keys
//   - error if load fails, wrong passphrase, or corrupted data
//
// Process:
//  1. Read keystore file from disk
//  2. Unmarshal JSON to KeystoreFile
//  3. Validate keystore format (version, KDF, cipher)
//  4. Decode base64 salt, IV, ciphertext
//  5. Derive encryption key using PBKDF2 (same params as Save)
//  6. Decrypt ciphertext using AES-256-GCM
//  7. Unmarshal decrypted JSON to KeystoreData
//  8. Convert KeystoreData to HybridKeypair
//  9. Return keypair
//
// Error Handling:
//   - Returns ErrWrongPassphrase if passphrase is incorrect
//   - Returns ErrInvalidKeystore if file is corrupted
//   - Returns ErrInvalidKeystoreVersion if version mismatch
func Load(passphrase string, path string) (*hybrid.HybridKeypair, error) {
	// Validate passphrase
	if err := ValidatePassphrase(passphrase); err != nil {
		return nil, fmt.Errorf("invalid passphrase: %w", err)
	}

	// Read keystore file
	keystoreJSON, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore file: %w", err)
	}

	// Unmarshal JSON to KeystoreFile
	var keystoreFile KeystoreFile
	if err := json.Unmarshal(keystoreJSON, &keystoreFile); err != nil {
		return nil, fmt.Errorf("%w: failed to parse JSON: %v", ErrInvalidKeystore, err)
	}

	// Validate keystore format
	if err := keystoreFile.Validate(); err != nil {
		return nil, err
	}

	// Decode base64 salt
	salt, err := base64.StdEncoding.DecodeString(keystoreFile.KDFParams.Salt)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid salt encoding: %v", ErrInvalidKeystore, err)
	}
	if len(salt) != SaltSize {
		return nil, fmt.Errorf("%w: salt size mismatch", ErrInvalidKeystore)
	}

	// Decode base64 IV
	ivBytes, err := base64.StdEncoding.DecodeString(keystoreFile.IV)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid IV encoding: %v", ErrInvalidKeystore, err)
	}
	if len(ivBytes) != IVSize {
		return nil, fmt.Errorf("%w: IV size mismatch", ErrInvalidKeystore)
	}
	var iv [IVSize]byte
	copy(iv[:], ivBytes)

	// Decode base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(keystoreFile.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ciphertext encoding: %v", ErrInvalidKeystore, err)
	}

	// Derive encryption key from passphrase using PBKDF2
	key, err := DeriveKey(passphrase, salt, keystoreFile.KDFParams.Iterations)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	// Decrypt ciphertext using AES-256-GCM
	encrypted := &EncryptedData{
		Ciphertext: ciphertext,
		IV:         iv,
	}
	plaintextJSON, err := Decrypt(encrypted, key)
	if err != nil {
		// Decryption failure could be wrong passphrase or corrupted data
		return nil, ErrWrongPassphrase
	}

	// Zero the key from memory after use
	for i := range key {
		key[i] = 0
	}

	// Unmarshal decrypted JSON to KeystoreData
	var keystoreData KeystoreData
	if err := json.Unmarshal(plaintextJSON, &keystoreData); err != nil {
		return nil, fmt.Errorf("%w: failed to parse decrypted data: %v", ErrInvalidKeystore, err)
	}

	// Convert KeystoreData to HybridKeypair
	keypair, err := keystoreData.ToHybridKeypair()
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct keypair: %w", err)
	}

	return keypair, nil
}

// Exists checks if a keystore file exists at the given path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Remove deletes a keystore file from disk.
// This is a convenience function for testing and key management.
func Remove(path string) error {
	return os.Remove(path)
}
