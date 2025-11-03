# Contributing to ShadowMesh

<img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" width="80" align="right"/>

Thank you for your interest in contributing to Chronara Group ShadowMesh! This document provides guidelines for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce** the issue
- **Expected behavior** vs **actual behavior**
- **Environment details** (OS, Go version, hardware)
- **Logs or error messages** (if applicable)

### Suggesting Enhancements

Enhancement suggestions are welcome! Please include:

- **Clear description** of the enhancement
- **Use case** and benefits
- **Potential implementation approach** (if you have ideas)
- **Related issues or PRs** (if applicable)

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Write clear commit messages** describing your changes
3. **Add tests** if you're adding functionality
4. **Update documentation** if needed
5. **Ensure all tests pass** before submitting
6. **Follow the coding standards** (see below)

## Development Process

### Setting Up Development Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/shadowmesh.git
cd shadowmesh

# Install dependencies
go mod download

# Build the client
make build-client

# Run tests
make test
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./client/daemon/
go test ./shared/crypto/
```

### Code Style

- Follow standard Go conventions (`gofmt`, `golint`)
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions focused and small
- Use meaningful variable names

### Commit Messages

Write clear, concise commit messages:

```
Add post-quantum key rotation feature

- Implement automatic key rotation every 60 minutes
- Add configuration option for rotation interval
- Include tests for rotation logic
```

Format:
- First line: Brief summary (50 chars or less)
- Blank line
- Detailed description (if needed)
- Reference issues: `Fixes #123` or `Relates to #456`

## What We're Looking For

### High Priority

- **Bug fixes** for existing issues
- **Performance improvements** (with benchmarks)
- **Documentation improvements** (clarity, examples)
- **Test coverage** for untested code
- **Security enhancements** (responsible disclosure required)

### Welcome Contributions

- **Client improvements** (UX, performance, features)
- **Protocol optimizations** (with backward compatibility)
- **Platform support** (Windows, macOS, ARM)
- **Example configurations** and tutorials
- **CI/CD improvements**

### Not Accepting

- **Server code contributions** (proprietary - relay server is not open source)
- **Breaking changes** without discussion
- **Large refactors** without prior approval
- **Feature additions** without use case justification

## Security Vulnerabilities

**Do NOT open public issues for security vulnerabilities.**

Please see [SECURITY.md](SECURITY.md) for responsible disclosure process.

## Cryptography Changes

Changes to cryptographic code require:

1. **Clear justification** and security analysis
2. **Peer review** by cryptography experts
3. **Test vectors** from NIST or academic sources
4. **Backward compatibility** consideration
5. **Documentation** of algorithm choices

## Testing Requirements

All contributions must include:

- **Unit tests** for new functionality
- **Integration tests** for end-to-end features
- **Benchmarks** for performance-critical code
- **Documentation** for public APIs

## Documentation

- **Code comments** for exported functions
- **README updates** for user-facing changes
- **Architecture docs** for design decisions
- **Examples** for new features

## Review Process

1. **Automated checks** run on all PRs (tests, linting, coverage)
2. **Code review** by maintainers (may request changes)
3. **Testing** on multiple platforms
4. **Approval** from at least one maintainer
5. **Merge** to main branch

## Community

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Pull Requests**: Code contributions

## License

By contributing to ShadowMesh, you agree that your contributions will be licensed under the [MIT License](LICENSE).

## Questions?

If you have questions about contributing, please:

1. Check existing documentation
2. Search closed issues
3. Open a GitHub Discussion
4. Contact the maintainers (if needed)

---

**Thank you for contributing to ShadowMesh!** 🚀

Together, we're building an experimental post-quantum Decentralized Private Network (DPN).
