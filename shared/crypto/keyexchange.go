package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber1024"
	"golang.org/x/crypto/hkdf"
)

const (
	// HybridKDFInfo is the info parameter for HKDF used to derive the final shared secret
	HybridKDFInfo = "ShadowMesh-v1-Hybrid-KEM"

	// SharedSecretSize is the size of the derived shared secret (32 bytes = 256 bits)
	SharedSecretSize = 32
)

var (
	// ErrKEMInvalidCiphertext is returned when the ciphertext is malformed or has incorrect length
	ErrKEMInvalidCiphertext = errors.New("invalid ciphertext")

	// ErrKEMNilKeyPair is returned when a nil keypair is provided
	ErrKEMNilKeyPair = errors.New("keypair cannot be nil")

	// ErrKEMInvalidPublicKey is returned when the public key is invalid
	ErrKEMInvalidPublicKey = errors.New("invalid public key")

	// ErrKEMDecapsulationFailed is returned when decapsulation fails
	ErrKEMDecapsulationFailed = errors.New("decapsulation failed")

	// ErrKEMEncapsulationFailed is returned when encapsulation fails
	ErrKEMEncapsulationFailed = errors.New("encapsulation failed")
)

// HybridKeyPair contains both ML-KEM-1024 and X25519 keypairs
type HybridKeyPair struct {
	// ML-KEM-1024 (post-quantum) keypair
	KEMPublicKey  kem.PublicKey
	KEMPrivateKey kem.PrivateKey

	// X25519 (classical ECDH) keypair
	ECDHPublicKey  *ecdh.PublicKey
	ECDHPrivateKey *ecdh.PrivateKey
}

// PublicKeyBytes returns the serialized public key (KEM public key + ECDH public key)
func (kp *HybridKeyPair) PublicKeyBytes() []byte {
	kemPubBytes, _ := kp.KEMPublicKey.MarshalBinary()
	ecdhPubBytes := kp.ECDHPublicKey.Bytes()

	result := make([]byte, len(kemPubBytes)+len(ecdhPubBytes))
	copy(result, kemPubBytes)
	copy(result[len(kemPubBytes):], ecdhPubBytes)

	return result
}

// HybridPublicKey contains only the public parts needed for encapsulation
type HybridPublicKey struct {
	KEMPublicKey  kem.PublicKey
	ECDHPublicKey *ecdh.PublicKey
}

// PublicKeyBytes returns the serialized public key
func (pk *HybridPublicKey) PublicKeyBytes() []byte {
	kemPubBytes, _ := pk.KEMPublicKey.MarshalBinary()
	ecdhPubBytes := pk.ECDHPublicKey.Bytes()

	result := make([]byte, len(kemPubBytes)+len(ecdhPubBytes))
	copy(result, kemPubBytes)
	copy(result[len(kemPubBytes):], ecdhPubBytes)

	return result
}

// GenerateKeyPair generates a new hybrid keypair combining ML-KEM-1024 and X25519
func GenerateKeyPair() (*HybridKeyPair, error) {
	scheme := kyber1024.Scheme()

	// Generate ML-KEM-1024 keypair
	kemPub, kemPriv, err := scheme.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate KEM keypair: %w", err)
	}

	// Generate X25519 keypair
	ecdhPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDH keypair: %w", err)
	}

	return &HybridKeyPair{
		KEMPublicKey:   kemPub,
		KEMPrivateKey:  kemPriv,
		ECDHPublicKey:  ecdhPriv.PublicKey(),
		ECDHPrivateKey: ecdhPriv,
	}, nil
}

// PublicKey extracts the public key portion from a keypair
func (kp *HybridKeyPair) PublicKey() *HybridPublicKey {
	return &HybridPublicKey{
		KEMPublicKey:  kp.KEMPublicKey,
		ECDHPublicKey: kp.ECDHPublicKey,
	}
}

// Encapsulate performs hybrid key encapsulation against a public key
// Returns the shared secret and the ciphertext (KEM ciphertext + ephemeral ECDH public key)
func Encapsulate(publicKey *HybridPublicKey) (sharedSecret, ciphertext []byte, err error) {
	if publicKey == nil {
		return nil, nil, ErrKEMNilKeyPair
	}

	scheme := kyber1024.Scheme()

	// Perform ML-KEM-1024 encapsulation
	kemCiphertext, kemSecret, err := scheme.Encapsulate(publicKey.KEMPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrKEMEncapsulationFailed, err)
	}

	// Generate ephemeral X25519 keypair for ECDH
	ecdhEphemeralPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ephemeral ECDH key: %w", err)
	}

	// Perform X25519 ECDH
	ecdhSecret, err := ecdhEphemeralPriv.ECDH(publicKey.ECDHPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("ECDH failed: %w", err)
	}

	// Derive the final shared secret from both KEM and ECDH secrets
	hybridSecret := deriveSharedSecret(kemSecret, ecdhSecret)

	// Construct ciphertext: KEM ciphertext || ephemeral ECDH public key
	ecdhEphemeralPub := ecdhEphemeralPriv.PublicKey().Bytes()
	combinedCiphertext := make([]byte, len(kemCiphertext)+len(ecdhEphemeralPub))
	copy(combinedCiphertext, kemCiphertext)
	copy(combinedCiphertext[len(kemCiphertext):], ecdhEphemeralPub)

	return hybridSecret, combinedCiphertext, nil
}

// EncapsulateKEM performs only KEM encapsulation (without ECDH)
// Returns the KEM shared secret and KEM ciphertext only
func EncapsulateKEM(publicKey kem.PublicKey) (sharedSecret, ciphertext []byte, err error) {
	if publicKey == nil {
		return nil, nil, ErrKEMInvalidPublicKey
	}

	scheme := kyber1024.Scheme()

	// Perform ML-KEM-1024 encapsulation
	kemCiphertext, kemSecret, err := scheme.Encapsulate(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrKEMEncapsulationFailed, err)
	}

	return kemSecret, kemCiphertext, nil
}

// PerformECDH performs an ECDH key exchange
// Returns the ECDH shared secret
func PerformECDH(privateKey *ecdh.PrivateKey, publicKey *ecdh.PublicKey) ([]byte, error) {
	if privateKey == nil || publicKey == nil {
		return nil, ErrKEMInvalidPublicKey
	}

	ecdhSecret, err := privateKey.ECDH(publicKey)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	return ecdhSecret, nil
}

// Decapsulate performs hybrid key decapsulation using the private key and ciphertext
// Returns the shared secret
func Decapsulate(privateKey *HybridKeyPair, ciphertext []byte) (sharedSecret []byte, err error) {
	if privateKey == nil {
		return nil, ErrKEMNilKeyPair
	}

	scheme := kyber1024.Scheme()
	kemCiphertextSize := scheme.CiphertextSize()
	ecdhPublicKeySize := 32 // X25519 public key size

	// Validate ciphertext length
	expectedSize := kemCiphertextSize + ecdhPublicKeySize
	if len(ciphertext) != expectedSize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrKEMInvalidCiphertext, expectedSize, len(ciphertext))
	}

	// Split ciphertext into KEM ciphertext and ephemeral ECDH public key
	kemCiphertext := ciphertext[:kemCiphertextSize]
	ecdhEphemeralPubBytes := ciphertext[kemCiphertextSize:]

	// Perform ML-KEM-1024 decapsulation
	kemSecret, err := scheme.Decapsulate(privateKey.KEMPrivateKey, kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKEMDecapsulationFailed, err)
	}

	// Parse ephemeral ECDH public key
	ecdhEphemeralPub, err := ecdh.X25519().NewPublicKey(ecdhEphemeralPubBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ephemeral ECDH public key: %w", err)
	}

	// Perform X25519 ECDH
	ecdhSecret, err := privateKey.ECDHPrivateKey.ECDH(ecdhEphemeralPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	// Derive the final shared secret from both KEM and ECDH secrets
	hybridSecret := deriveSharedSecret(kemSecret, ecdhSecret)

	return hybridSecret, nil
}

// DecapsulateKEM performs only KEM decapsulation (without ECDH)
// Returns the KEM shared secret only
func DecapsulateKEM(privateKey kem.PrivateKey, kemCiphertext []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, ErrKEMNilKeyPair
	}

	scheme := kyber1024.Scheme()
	kemCiphertextSize := scheme.CiphertextSize()

	// Validate ciphertext length
	if len(kemCiphertext) != kemCiphertextSize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrKEMInvalidCiphertext, kemCiphertextSize, len(kemCiphertext))
	}

	// Perform ML-KEM-1024 decapsulation
	kemSecret, err := scheme.Decapsulate(privateKey, kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKEMDecapsulationFailed, err)
	}

	return kemSecret, nil
}

// deriveSharedSecret combines the KEM and ECDH shared secrets using HKDF-SHA256
func deriveSharedSecret(kemSecret, ecdhSecret []byte) []byte {
	// Concatenate both secrets as input key material
	ikm := make([]byte, len(kemSecret)+len(ecdhSecret))
	copy(ikm, kemSecret)
	copy(ikm[len(kemSecret):], ecdhSecret)

	// Use HKDF to derive the final shared secret
	hkdfReader := hkdf.New(sha256.New, ikm, nil, []byte(HybridKDFInfo))
	sharedSecret := make([]byte, SharedSecretSize)

	// This should never fail with proper parameters
	if _, err := io.ReadFull(hkdfReader, sharedSecret); err != nil {
		panic(fmt.Sprintf("HKDF failed: %v", err))
	}

	return sharedSecret
}

// ParsePublicKey deserializes a public key from bytes
func ParsePublicKey(publicKeyBytes []byte) (*HybridPublicKey, error) {
	scheme := kyber1024.Scheme()
	kemPubKeySize := scheme.PublicKeySize()
	ecdhPubKeySize := 32 // X25519 public key size

	expectedSize := kemPubKeySize + ecdhPubKeySize
	if len(publicKeyBytes) != expectedSize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrKEMInvalidPublicKey, expectedSize, len(publicKeyBytes))
	}

	// Parse KEM public key
	kemPubBytes := publicKeyBytes[:kemPubKeySize]
	kemPub, err := scheme.UnmarshalBinaryPublicKey(kemPubBytes)
	if err != nil || kemPub == nil {
		return nil, fmt.Errorf("%w: failed to unmarshal KEM public key", ErrKEMInvalidPublicKey)
	}

	// Parse ECDH public key
	ecdhPubBytes := publicKeyBytes[kemPubKeySize:]
	ecdhPub, err := ecdh.X25519().NewPublicKey(ecdhPubBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse ECDH public key: %v", ErrKEMInvalidPublicKey, err)
	}

	return &HybridPublicKey{
		KEMPublicKey:  kemPub,
		ECDHPublicKey: ecdhPub,
	}, nil
}

// ParseKEMPublicKey parses only a KEM public key from bytes
func ParseKEMPublicKey(kemPubBytes []byte) (kem.PublicKey, error) {
	scheme := kyber1024.Scheme()
	kemPubKeySize := scheme.PublicKeySize()

	if len(kemPubBytes) != kemPubKeySize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrKEMInvalidPublicKey, kemPubKeySize, len(kemPubBytes))
	}

	kemPub, err := scheme.UnmarshalBinaryPublicKey(kemPubBytes)
	if err != nil || kemPub == nil {
		return nil, fmt.Errorf("%w: failed to unmarshal KEM public key", ErrKEMInvalidPublicKey)
	}

	return kemPub, nil
}

// ParseECDHPublicKey parses only an ECDH public key from bytes
func ParseECDHPublicKey(ecdhPubBytes []byte) (*ecdh.PublicKey, error) {
	if len(ecdhPubBytes) != 32 {
		return nil, fmt.Errorf("%w: expected 32 bytes, got %d", ErrKEMInvalidPublicKey, len(ecdhPubBytes))
	}

	ecdhPub, err := ecdh.X25519().NewPublicKey(ecdhPubBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse ECDH public key: %v", ErrKEMInvalidPublicKey, err)
	}

	return ecdhPub, nil
}

// NewHybridPublicKey creates a HybridPublicKey from separate KEM and ECDH public keys
func NewHybridPublicKey(kemPub kem.PublicKey, ecdhPub *ecdh.PublicKey) *HybridPublicKey {
	return &HybridPublicKey{
		KEMPublicKey:  kemPub,
		ECDHPublicKey: ecdhPub,
	}
}

// hybridKeyPairJSON is used for JSON marshaling/unmarshaling
type hybridKeyPairJSON struct {
	KEMPublicKey   string `json:"kem_public_key"`
	KEMPrivateKey  string `json:"kem_private_key"`
	ECDHPrivateKey string `json:"ecdh_private_key"`
}

// MarshalJSON implements the json.Marshaler interface
func (kp *HybridKeyPair) MarshalJSON() ([]byte, error) {
	if kp == nil {
		return nil, errors.New("cannot marshal nil HybridKeyPair")
	}

	// Serialize KEM public key
	kemPubBytes, err := kp.KEMPublicKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KEM public key: %w", err)
	}

	// Serialize KEM private key
	kemPrivBytes, err := kp.KEMPrivateKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KEM private key: %w", err)
	}

	// ECDH private key bytes
	ecdhPrivBytes := kp.ECDHPrivateKey.Bytes()

	jsonData := hybridKeyPairJSON{
		KEMPublicKey:   base64.StdEncoding.EncodeToString(kemPubBytes),
		KEMPrivateKey:  base64.StdEncoding.EncodeToString(kemPrivBytes),
		ECDHPrivateKey: base64.StdEncoding.EncodeToString(ecdhPrivBytes),
	}

	return json.Marshal(jsonData)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (kp *HybridKeyPair) UnmarshalJSON(data []byte) error {
	var jsonData hybridKeyPairJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	scheme := kyber1024.Scheme()

	// Decode KEM public key
	kemPubBytes, err := base64.StdEncoding.DecodeString(jsonData.KEMPublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode KEM public key: %w", err)
	}
	kemPub, err := scheme.UnmarshalBinaryPublicKey(kemPubBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal KEM public key: %w", err)
	}

	// Decode KEM private key
	kemPrivBytes, err := base64.StdEncoding.DecodeString(jsonData.KEMPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to decode KEM private key: %w", err)
	}
	kemPriv, err := scheme.UnmarshalBinaryPrivateKey(kemPrivBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal KEM private key: %w", err)
	}

	// Decode ECDH private key
	ecdhPrivBytes, err := base64.StdEncoding.DecodeString(jsonData.ECDHPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to decode ECDH private key: %w", err)
	}
	ecdhPriv, err := ecdh.X25519().NewPrivateKey(ecdhPrivBytes)
	if err != nil {
		return fmt.Errorf("failed to create ECDH private key: %w", err)
	}

	kp.KEMPublicKey = kemPub
	kp.KEMPrivateKey = kemPriv
	kp.ECDHPublicKey = ecdhPriv.PublicKey()
	kp.ECDHPrivateKey = ecdhPriv

	return nil
}
