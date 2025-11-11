# GitHub Security Configuration Guide

This document outlines the security measures implemented for the ShadowMesh repository and additional steps required for full compliance with GitHub security best practices.

## ✅ Implemented Automated Security

### 1. Code Scanning (CodeQL)
- **Location**: `.github/workflows/codeql.yml`
- **Languages**: Go, JavaScript
- **Schedule**: Weekly + on every push/PR
- **Status**: Active
- **Coverage**: OWASP Top 10, CWE Top 25, security vulnerabilities

### 2. Secret Scanning
- **Location**: `.github/workflows/secret-scan.yml`
- **Tool**: TruffleHog OSS
- **Scope**: All commits, verified secrets only
- **Status**: Active

### 3. Dependency Management (Dependabot)
- **Location**: `.github/dependabot.yml`
- **Ecosystems**: Go modules, GitHub Actions
- **Schedule**: Weekly scans
- **Auto-PR**: Enabled for security updates

## 🔧 Required Manual Configuration

### Repository Settings (GitHub UI)

Navigate to **Settings → Security**:

#### Push Protection
```
☑ Enable push protection
   Prevents secrets from being pushed to the repository
   Location: Settings → Code security and analysis → Secret scanning
```

#### Secret Scanning Alerts
```
☑ Enable secret scanning
☑ Enable push protection
   Automatically scans for over 200 secret patterns
   Location: Settings → Code security and analysis
```

#### Private Vulnerability Reporting
```
☑ Enable private vulnerability reporting
   Allows security researchers to report issues privately
   Location: Settings → Security → Private vulnerability reporting
```

#### Branch Protection Rules
```
Navigate to: Settings → Branches → Add rule

Branch name pattern: main

Required settings:
☑ Require a pull request before merging
  ☑ Require approvals (1)
  ☑ Dismiss stale pull request approvals when new commits are pushed
☑ Require status checks to pass before merging
  ☑ Require branches to be up to date before merging
  Required checks:
    - CodeQL Security Scan
    - Secret Scanning
☑ Require conversation resolution before merging
☑ Do not allow bypassing the above settings
```

### Repository Visibility

**Current**: Public
**Recommendation**: Review if repository should remain public

**If keeping public**:
- ✅ No secrets in commit history (verified)
- ✅ No infrastructure IPs in public files (verified)
- ✅ Source code removed pending security audit (verified)
- ✅ Only community documentation public (verified)

**If making private**:
```
Settings → General → Danger Zone → Change repository visibility → Make private
```

## 🔐 Account Security (Repository Owner Actions)

### Two-Factor Authentication (2FA)
```
Profile → Settings → Password and authentication
☑ Enable two-factor authentication
   Recommended: Hardware key (YubiKey, Titan) or authenticator app
```

### Personal Access Tokens (PATs)
```
Profile → Settings → Developer settings → Personal access tokens

Best practices:
- Use fine-grained tokens (not classic)
- Set minimum necessary scopes
- Set expiration dates (max 90 days for production)
- Rotate regularly
- Revoke unused tokens
```

### SSH Keys
```
Profile → Settings → SSH and GPG keys

Best practices:
- Use Ed25519 keys (ssh-keygen -t ed25519)
- Set passphrase on all keys
- Use separate keys for different machines
- Audit and remove old keys
```

### Security Log Monitoring
```
Profile → Settings → Security log
   Review weekly for unauthorized access attempts
```

## 👥 Access Control

### Collaborator Permissions
```
Settings → Collaborators and teams

Principle of least privilege:
- Read: Documentation contributors
- Triage: Issue managers
- Write: Developers (require 2FA)
- Maintain: Project leads (require 2FA)
- Admin: Repository owner only (require 2FA)
```

### CODEOWNERS
```
Create: .github/CODEOWNERS

Example:
# Require review from security team for sensitive files
/.github/workflows/* @chronara-security
/SECURITY.md @chronara-security
*.key @chronara-security
```

## 📊 Security Monitoring

### GitHub Security Dashboard
```
Navigate to: Security tab

Monitor:
- Dependabot alerts (dependencies)
- Code scanning alerts (CodeQL)
- Secret scanning alerts
- Security advisories
```

### Email Notifications
```
Profile → Settings → Notifications

Enable:
☑ Dependabot alerts
☑ Security alerts
☑ Vulnerability alerts on my repositories
```

## 🚫 Secret Management

### Never Commit These Files
```
.env
.env.local
config.yaml
*.key
*.pem
*.p12
credentials.json
secrets.yaml
id_rsa
```

### How to Store Secrets

**Development**:
- Environment variables
- Local `.env` files (in .gitignore)
- Operating system keychains

**CI/CD**:
- GitHub Secrets: Settings → Secrets and variables → Actions
- Environment-specific secrets

**Production**:
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- Google Secret Manager

## 🔍 Pre-Commit Hooks

Prevent secrets from being committed locally:

```bash
# Install gitleaks
brew install gitleaks  # macOS
# or download from https://github.com/gitleaks/gitleaks

# Add pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
gitleaks protect --staged --verbose
EOF

chmod +x .git/hooks/pre-commit
```

## 📋 Security Checklist

### Initial Setup
- [x] CodeQL workflow created
- [x] Dependabot configured
- [x] Secret scanning workflow created
- [x] SECURITY.md exists and is comprehensive
- [ ] Enable push protection (GitHub UI)
- [ ] Enable secret scanning alerts (GitHub UI)
- [ ] Configure branch protection rules (GitHub UI)
- [ ] Enable 2FA on all collaborator accounts
- [ ] Review repository visibility (public vs private)

### Monthly Review
- [ ] Review Dependabot alerts and merge updates
- [ ] Review CodeQL findings
- [ ] Audit collaborator access
- [ ] Check security log for suspicious activity
- [ ] Rotate PATs older than 90 days
- [ ] Review and update SECURITY.md

### Quarterly Review
- [ ] Full security audit of codebase
- [ ] Review and update branch protection rules
- [ ] Audit SSH keys and remove unused
- [ ] Review third-party integrations
- [ ] Test incident response procedures

## 🆘 Incident Response

If a secret is accidentally committed:

1. **Immediate Actions**:
   ```bash
   # Revoke the compromised secret immediately
   # Do NOT just delete the file - secret is in Git history
   ```

2. **Rotate Credentials**:
   - Generate new credentials
   - Update production systems
   - Verify old credentials no longer work

3. **Clean Git History**:
   ```bash
   # Use BFG Repo-Cleaner or git filter-repo
   git filter-repo --path-match 'secrets.yaml' --invert-paths
   git push --force --all
   ```

4. **Verify Removal**:
   ```bash
   # Scan entire history
   gitleaks detect --source . --verbose
   ```

5. **Document**:
   - Record incident in security log
   - Update procedures to prevent recurrence

## 📚 Additional Resources

- [GitHub Security Best Practices](https://docs.github.com/en/code-security)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)

## 🔄 Continuous Improvement

Security is an ongoing process. This document should be reviewed and updated:
- After security incidents
- When GitHub adds new security features
- Quarterly as part of security review

---

**Last Updated**: 2025-01-11
**Maintained By**: Chronara Security Team
