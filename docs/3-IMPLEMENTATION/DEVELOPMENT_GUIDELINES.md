# Development Guidelines

**Status**: Internal Document - Not for Public Distribution
**Purpose**: Maintain consistent, professional communication in all project materials

---

## Communication Standards

### Core Principles

1. **Factual**: Base all claims on verifiable evidence
2. **Measured**: Avoid superlatives and emotional language
3. **Professional**: Maintain technical credibility
4. **Iterative**: Present as work-in-progress, not finished product
5. **Transparent**: Acknowledge limitations and trade-offs

### Language to AVOID

❌ **Sensationalist**:
- "Revolutionary"
- "World's first" (unless independently verified)
- "Beats/Destroys/Crushes" competitors
- "Amazing", "Incredible", "Mind-blowing"
- Exclamation marks (!!!)
- ALL CAPS for emphasis

❌ **Absolute Claims**:
- "Best", "Perfect", "Ultimate"
- "Never fails", "Always works"
- "Completely secure", "Unhackable"
- "Zero overhead", "Infinite scalability"

❌ **Unverified Comparisons**:
- Claims about competitors without citations
- Performance comparisons without methodology
- Security claims without audits

### Language to USE

✅ **Measured**:
- "Implements", "Provides", "Offers"
- "In testing", "Under development", "Alpha/Beta stage"
- "Designed to", "Intended for", "Aims to"
- "Based on NIST standards"

✅ **Specific**:
- Exact numbers with context
- Citations and references
- Test methodology described
- Limitations acknowledged

✅ **Professional**:
- Technical accuracy
- Peer-reviewed terminology
- Industry-standard comparisons
- Evidence-based claims

---

## Writing Personas

### Technical Documentation

**Tone**: Precise, detailed, referenced
**Audience**: Developers, security researchers, cryptographers
**Style**:
- Short paragraphs
- Code examples included
- Citations to papers/standards
- Limitations clearly stated

**Example**:
```
ShadowMesh implements ML-KEM-1024 (NIST FIPS 203) for key encapsulation.
Initial benchmarks on commodity hardware show ~1ms handshake overhead.
Performance may vary based on hardware acceleration availability.
```

### User Documentation

**Tone**: Clear, helpful, realistic
**Audience**: System administrators, technical users
**Style**:
- Step-by-step instructions
- Prerequisites listed
- Expected behavior described
- Troubleshooting included

**Example**:
```
Installation requires root privileges for TAP device creation.
Expected installation time: 5-10 minutes on a typical system.
If installation fails, check the logs at /var/log/shadowmesh.
```

### Public Communications

**Tone**: Professional, factual, conservative
**Audience**: General technical community
**Style**:
- Lead with facts
- Context before claims
- Acknowledge alpha/beta status
- Invite feedback

**Example**:
```
ShadowMesh is an alpha-stage VPN client implementing post-quantum
cryptography (ML-KEM-1024, ML-DSA-87). We're seeking community feedback
on the implementation and architecture design.
```

---

## Claims Standards

### What We CAN Say

✅ **Factual Implementation**:
- "Uses NIST-standardized ML-KEM-1024"
- "Implements Layer 2 VPN architecture"
- "Written in Go 1.21+"
- "MIT licensed client software"

✅ **Measured Performance** (with data):
- "In our tests: 50ms average latency"
- "Benchmark: 13.8 Mbps throughput"
- "Test environment: UK → Germany → Belgium"
- "Hardware: Raspberry Pi 4, Proxmox VM"

✅ **Development Status**:
- "Alpha release - testing in progress"
- "Client software is feature-complete for core functionality"
- "Seeking community review and feedback"
- "Not yet security audited"

### What We CANNOT Say (Without Evidence)

❌ **Unverified Claims**:
- "World's first post-quantum VPN" → Need independent verification
- "Faster than Tailscale" → Need peer-reviewed benchmarks
- "Production ready" → Need extensive testing + audit
- "Enterprise grade" → Need certifications (SOC 2, etc.)

❌ **Security Claims** (Without Audit):
- "Unhackable", "Completely secure"
- "Quantum-proof" (vs "quantum-resistant")
- "Military-grade security"
- "Zero vulnerabilities"

❌ **Future Promises**:
- Specific release dates
- Guaranteed features
- Performance guarantees
- Support commitments (without infrastructure)

---

## Documentation Review Checklist

Before publishing ANY documentation:

- [ ] No sensationalist language
- [ ] All performance claims include methodology
- [ ] Limitations clearly stated
- [ ] Development status indicated (alpha/beta)
- [ ] No "world's first" without verification
- [ ] Comparisons cite sources
- [ ] No absolute security claims
- [ ] Professional tone throughout
- [ ] Facts verifiable
- [ ] Trade-offs acknowledged

---

## Version Control Practices

### Commit Messages

**Good**:
```
Implement ML-KEM-1024 key exchange

- Add NIST FIPS 203 implementation
- Include test vectors from NIST
- Benchmark shows ~1ms overhead
- Refs: NIST FIPS 203 draft
```

**Bad**:
```
AMAZING new crypto that DESTROYS the competition!!!
```

### Pull Request Descriptions

**Include**:
- What changed
- Why it changed
- Test results
- Known limitations
- Breaking changes

**Avoid**:
- Marketing language
- Hype
- Competitor bashing
- Unsupported claims

---

## README Structure Standards

### Required Sections (in order)

1. **Project Name + Brief Description** (1-2 sentences, factual)
2. **Status Badges** (build, license, version)
3. **Current Status** (alpha/beta/stable, limitations)
4. **Features** (what it does, not why it's better)
5. **Installation** (clear prerequisites)
6. **Quick Start** (minimal working example)
7. **Documentation** (links to detailed docs)
8. **Contributing** (how to help)
9. **License** (clear terms)
10. **Acknowledgments** (credit dependencies)

### NOT Required

- Competitive comparisons in README
- Performance benchmarks (put in docs/)
- Business strategy
- Marketing claims
- Future roadmap promises

---

## Example Transformations

### Before (Sensationalist)
```
🎉 World's First Post-Quantum VPN CRUSHES Tailscale!

ShadowMesh is REVOLUTIONARY - 30% FASTER than competitors with
UNHACKABLE quantum-safe encryption! Perfect for everyone!
```

### After (Professional)
```
ShadowMesh - Post-Quantum VPN Client (Alpha)

An experimental VPN client implementing NIST post-quantum cryptography
standards (ML-KEM-1024, ML-DSA-87). Currently in alpha testing.
Contributions and feedback welcome.
```

---

## Review Frequency

- **Before each release**: Full documentation review
- **Monthly**: Scan for outdated claims
- **After benchmarks**: Update with new data
- **After security findings**: Immediate disclosure update

---

## Accountability

All team members are responsible for:
- Reviewing their own writing against these standards
- Calling out sensationalist language in reviews
- Prioritizing accuracy over excitement
- Maintaining professional credibility

---

## References

- [IETF RFC Guidelines](https://www.ietf.org/standards/rfcs/)
- [NIST PQC Standards](https://csrc.nist.gov/projects/post-quantum-cryptography)
- [Contributor Covenant](https://www.contributor-covenant.org/)
- [Semantic Versioning](https://semver.org/)

---

**Remember**: Credibility is earned through consistent, professional, accurate communication.
One unverified claim can undermine months of technical work.
