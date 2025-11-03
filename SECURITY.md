# Security Policy

<img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" width="120" align="right"/>

**Chronara Group ShadowMesh - Security Policy**

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

Currently, we are in alpha release. Security updates will be provided for the latest version only.

## Reporting a Vulnerability

**Please do NOT report security vulnerabilities through public GitHub issues.**

We take security seriously. If you discover a security vulnerability in ShadowMesh, please report it responsibly.

### How to Report

1. **Email**: Send details to security@shadowmesh.dev (or create a private security advisory on GitHub)
2. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)
   - Your contact information

### What to Expect

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 7 days
- **Status updates**: Every 7 days until resolved
- **Public disclosure**: Coordinated with you after fix is released

### Responsible Disclosure Timeline

1. **Report received**: Security team acknowledges
2. **Triage** (1-7 days): Assess severity and impact
3. **Fix development** (varies by severity):
   - Critical: 1-7 days
   - High: 7-14 days
   - Medium: 14-30 days
   - Low: 30-90 days
4. **Testing**: Verify fix doesn't break functionality
5. **Release**: Deploy fix to users
6. **Public disclosure**: 7-14 days after release (or as agreed)

### Severity Levels

**Critical**:
- Remote code execution
- Key extraction or cryptographic bypass
- Authentication bypass

**High**:
- Denial of service (relay or client)
- Information disclosure (private keys, sensitive data)
- Privilege escalation

**Medium**:
- Local denial of service
- Information leak (non-sensitive)
- Timing attacks

**Low**:
- Minor information disclosure
- Configuration issues

## Security Best Practices

### For Users

1. **Keep updated**: Always use the latest version
2. **Protect keys**: Store private keys securely (`~/.shadowmesh/keys/`)
3. **Use strong passwords**: If you encrypt key files
4. **Verify downloads**: Check SHA256 hashes
5. **Monitor logs**: Watch for unusual activity

### For Developers

1. **Input validation**: Sanitize all external inputs
2. **Key management**: Use OS keychain when possible
3. **Constant-time operations**: For crypto comparisons
4. **Memory safety**: Clear sensitive data after use
5. **Dependency updates**: Keep libraries current

## Cryptographic Security

### Algorithms Used

- **Post-Quantum KEM**: ML-KEM-1024 (Kyber, NIST Security Level 5)
- **Post-Quantum Signatures**: ML-DSA-87 (Dilithium, NIST Security Level 5)
- **Symmetric Encryption**: ChaCha20-Poly1305
- **Classical KEM** (hybrid): X25519
- **Classical Signatures** (hybrid): Ed25519

All algorithms are NIST-standardized or widely-reviewed.

### Key Rotation

- **Default**: Every 60 minutes
- **Configurable**: 10 seconds to 24 hours
- **Automatic**: No user intervention required

### Perfect Forward Secrecy

Session keys are ephemeral. Compromise of long-term keys does not compromise past sessions.

## Known Security Considerations

### Current Status (Alpha v0.1.x)

✅ **Implemented**:
- Post-quantum key exchange (ML-KEM-1024)
- Post-quantum signatures (ML-DSA-87)
- ChaCha20-Poly1305 frame encryption
- Automatic key rotation
- Replay attack protection
- Perfect forward secrecy

⚠️ **Limitations**:
- No formal security audit (planned for beta)
- Alpha software - use at own risk
- Limited platform testing
- Relay server code not open source

🔜 **Planned** (Beta):
- Third-party security audit
- Formal verification of protocol
- TPM/SGX attestation for relay nodes
- Multi-hop routing
- Traffic obfuscation

### Threat Model

**What ShadowMesh Protects Against**:
- ✅ Quantum computer attacks (via post-quantum crypto)
- ✅ Eavesdropping (end-to-end encryption)
- ✅ Man-in-the-middle (authenticated key exchange)
- ✅ Replay attacks (nonce + counter)
- ✅ Key compromise (perfect forward secrecy)

**What ShadowMesh Does NOT Protect Against**:
- ❌ Compromised endpoints (malware on client/relay)
- ❌ Traffic analysis (metadata leakage)
- ❌ Side-channel attacks (timing, power)
- ❌ Zero-day vulnerabilities (in dependencies)
- ❌ Social engineering

### Dependencies

We rely on well-vetted cryptographic libraries:
- `github.com/cloudflare/circl` - Cloudflare's crypto library (PQC)
- `golang.org/x/crypto` - Go's extended crypto library
- `crypto/*` - Go standard library

Vulnerabilities in dependencies are tracked and patched promptly.

## Security Audits

### Status

- **Alpha (current)**: No formal audit
- **Beta (planned)**: Third-party security audit
- **v1.0 (planned)**: Full penetration testing

Audit reports will be published after remediation of findings.

## Bug Bounty Program

**Status**: Not yet available

We plan to launch a bug bounty program for beta release. Details will be announced.

## Security Updates

Subscribe to security advisories:
- GitHub Security Advisories: Watch this repository
- Releases: https://github.com/CG-8663/shadowmesh/releases
- Security mailing list: (Coming soon)

## Questions?

For security-related questions that are NOT vulnerabilities:
- Open a GitHub Discussion
- Email: security@shadowmesh.dev

---

**Thank you for helping keep ShadowMesh secure!** 🔒
