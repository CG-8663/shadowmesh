package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
)

// TLSCertificateManager manages ephemeral TLS certificates for direct P2P connections
type TLSCertificateManager struct {
	// Local certificate
	certificate    *tls.Certificate
	certificateDER []byte // DER-encoded for exchange
	privateKey     *ecdsa.PrivateKey

	// Pinned peer certificate
	pinnedCertDER      []byte
	pinnedCertHash     [32]byte
	pinnedCertVerified bool

	// Signing key for certificate authentication
	signingKey *crypto.HybridSigningKey
}

// NewTLSCertificateManager creates a new TLS certificate manager
func NewTLSCertificateManager(signingKey *crypto.HybridSigningKey) *TLSCertificateManager {
	return &TLSCertificateManager{
		signingKey: signingKey,
	}
}

// GenerateEphemeralCertificate generates a self-signed ECDSA certificate for direct P2P
// The certificate is valid for 24 hours and includes the local IP as Subject Alternative Name
func (tm *TLSCertificateManager) GenerateEphemeralCertificate(localIP string) error {
	// Generate ECDSA P-256 private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour) // 24-hour ephemeral cert

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ShadowMesh P2P"},
			CommonName:   "ShadowMesh Relay",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Add IP address as SAN (Subject Alternative Name)
	if ip := net.ParseIP(localIP); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		// If not a valid IP, add as DNS name (for testing with hostnames)
		template.DNSNames = []string{localIP}
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Convert to tls.Certificate
	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}

	tm.certificate = &tlsCert
	tm.certificateDER = certDER
	tm.privateKey = privateKey

	return nil
}

// GetCertificateDER returns the DER-encoded certificate for exchange with peer
func (tm *TLSCertificateManager) GetCertificateDER() []byte {
	return tm.certificateDER
}

// GetCertificatePEM returns the PEM-encoded certificate (for debugging)
func (tm *TLSCertificateManager) GetCertificatePEM() string {
	if tm.certificateDER == nil {
		return ""
	}

	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tm.certificateDER,
	}

	return string(pem.EncodeToMemory(pemBlock))
}

// GetCertificateFingerprint returns SHA-256 hash of the certificate
func (tm *TLSCertificateManager) GetCertificateFingerprint() [32]byte {
	if tm.certificateDER == nil {
		return [32]byte{}
	}
	return sha256.Sum256(tm.certificateDER)
}

// PinPeerCertificate stores the peer's certificate for later verification
// This implements certificate pinning to prevent MITM attacks
func (tm *TLSCertificateManager) PinPeerCertificate(certDER []byte) error {
	if len(certDER) == 0 {
		return fmt.Errorf("empty certificate")
	}

	// Parse certificate to validate it
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("invalid certificate: %w", err)
	}

	// Verify certificate is not expired
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired")
	}

	// Store pinned certificate
	tm.pinnedCertDER = certDER
	tm.pinnedCertHash = sha256.Sum256(certDER)
	tm.pinnedCertVerified = true

	return nil
}

// VerifyPeerCertificate implements custom certificate verification for pinning
// This is called during TLS handshake to verify the peer's certificate
func (tm *TLSCertificateManager) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if !tm.pinnedCertVerified {
		return fmt.Errorf("no pinned certificate configured")
	}

	if len(rawCerts) == 0 {
		return fmt.Errorf("no certificates provided by peer")
	}

	// Get the leaf certificate (first in chain)
	peerCertDER := rawCerts[0]
	peerCertHash := sha256.Sum256(peerCertDER)

	// Compare with pinned certificate hash
	if peerCertHash != tm.pinnedCertHash {
		return fmt.Errorf("certificate pinning failed: hash mismatch (expected %x, got %x)",
			tm.pinnedCertHash[:8], peerCertHash[:8])
	}

	// Additional validation: parse and check expiry
	cert, err := x509.ParseCertificate(peerCertDER)
	if err != nil {
		return fmt.Errorf("invalid peer certificate: %w", err)
	}

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("peer certificate not valid at current time")
	}

	return nil
}

// GetTLSConfigServer returns TLS config for server (accepting incoming connections)
func (tm *TLSCertificateManager) GetTLSConfigServer() (*tls.Config, error) {
	if tm.certificate == nil {
		return nil, fmt.Errorf("certificate not generated")
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{*tm.certificate},
		MinVersion:   tls.VersionTLS13, // Require TLS 1.3 for maximum security
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		ClientAuth:            tls.RequireAnyClientCert, // Require client certificate
		VerifyPeerCertificate: tm.VerifyPeerCertificate,
	}

	return config, nil
}

// GetTLSConfigClient returns TLS config for client (initiating outgoing connections)
func (tm *TLSCertificateManager) GetTLSConfigClient(serverName string) (*tls.Config, error) {
	if tm.certificate == nil {
		return nil, fmt.Errorf("certificate not generated")
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{*tm.certificate},
		MinVersion:         tls.VersionTLS13, // Require TLS 1.3
		InsecureSkipVerify: true,             // We do manual pinning
		ServerName:         serverName,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		VerifyPeerCertificate: tm.VerifyPeerCertificate,
	}

	return config, nil
}

// SignCertificate signs the TLS certificate with ML-DSA-87 key for authentication
// This binds the ephemeral TLS cert to the long-term PQC identity
func (tm *TLSCertificateManager) SignCertificate() ([]byte, error) {
	if tm.certificateDER == nil {
		return nil, fmt.Errorf("certificate not generated")
	}

	if tm.signingKey == nil {
		return nil, fmt.Errorf("signing key not available")
	}

	// Sign the certificate DER bytes with ML-DSA-87
	signature, err := crypto.Sign(tm.signingKey, tm.certificateDER)
	if err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	return signature, nil
}

// VerifyCertificateSignature verifies the ML-DSA-87 signature on peer's certificate
// This proves the peer's ephemeral TLS cert is bound to their PQC identity
func (tm *TLSCertificateManager) VerifyCertificateSignature(certDER []byte, signature []byte, peerPublicKey *crypto.HybridVerifyKey) error {
	if len(certDER) == 0 {
		return fmt.Errorf("empty certificate")
	}

	if len(signature) == 0 {
		return fmt.Errorf("empty signature")
	}

	if peerPublicKey == nil {
		return fmt.Errorf("peer public key not provided")
	}

	// Verify ML-DSA-87 signature
	err := crypto.Verify(peerPublicKey, certDER, signature)
	if err != nil {
		return fmt.Errorf("certificate signature verification failed: %w", err)
	}

	return nil
}

// IsCertificateValid checks if the local certificate is still valid
func (tm *TLSCertificateManager) IsCertificateValid() bool {
	if tm.certificateDER == nil {
		return false
	}

	cert, err := x509.ParseCertificate(tm.certificateDER)
	if err != nil {
		return false
	}

	now := time.Now()
	return now.After(cert.NotBefore) && now.Before(cert.NotAfter)
}

// RegenerateCertificateIfNeeded checks expiry and regenerates if needed
func (tm *TLSCertificateManager) RegenerateCertificateIfNeeded(localIP string) error {
	if tm.IsCertificateValid() {
		return nil // Still valid
	}

	// Regenerate certificate
	return tm.GenerateEphemeralCertificate(localIP)
}
