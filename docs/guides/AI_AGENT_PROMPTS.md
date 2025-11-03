<h1>
  <img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" height="60" style="vertical-align: middle; margin-right: 15px;"/>
  Chronara Group ShadowMesh - AI Development Guide
</h1>

## Purpose
This document contains precise prompt instructions for AI agents (CAIAF, Development Team, etc.) to accurately build the Chronara Group ShadowMesh Decentralized Private Network (DPN). Each section provides context, constraints, and expected outputs to ensure consistency and quality.

---

## General Instructions for All AI Agents

### Core Principles
1. **Security First**: Every line of code must be reviewed for security implications
2. **Go Idiomatic**: Follow Go best practices and conventions
3. **Performance**: Optimize for low latency and high throughput
4. **Testability**: Write code that is easy to unit test and integration test
5. **Documentation**: Every public function must have godoc comments

### Code Quality Standards
```
- Line length: Max 120 characters
- Error handling: Never ignore errors, always return or log
- Naming: Use camelCase for private, PascalCase for public
- Comments: Explain "why", not "what"
- Testing: Minimum 80% code coverage
```

### Security Requirements
- Never log sensitive data (keys, tokens, IPs in production)
- Always validate input from untrusted sources
- Use constant-time comparison for cryptographic operations
- Implement rate limiting on all public endpoints
- Follow OWASP Top 10 guidelines

---

## Module 1: Core Protocol Implementation

### Prompt for Protocol Layer

```
You are an expert Go developer specializing in network protocols and cryptography.

TASK: Implement the ShadowMesh binary protocol frame handler.

REQUIREMENTS:
1. Create a frame structure that includes:
   - Version (4 bits), Type (4 bits), Flags (8 bits)
   - Length field (16 bits, big-endian)
   - Sequence number (32 bits)
   - Peer public key (32 bytes Ed25519)
   - Nonce (24 bytes for ChaCha20-Poly1305)
   - Encrypted payload (variable length)
   - Authentication tag (16 bytes Poly1305)

2. Frame types to support:
   - DATA (0x01): IP packet payload
   - CONTROL (0x02): Protocol control messages
   - KEEPALIVE (0x03): Connection maintenance
   - HANDSHAKE (0x04): Initial key exchange

3. Implement these functions:
   - MarshalFrame(frame *Frame) ([]byte, error)
   - UnmarshalFrame(data []byte) (*Frame, error)
   - ValidateFrame(frame *Frame) error

4. Use golang.org/x/crypto for all cryptographic operations

5. Include comprehensive error handling:
   - ErrInvalidFrameVersion
   - ErrInvalidFrameLength
   - ErrDecryptionFailed
   - ErrSequenceNumberOutOfOrder

CONSTRAINTS:
- Maximum frame size: 64KB (65535 bytes)
- Use binary.BigEndian for all multi-byte fields
- Zero-copy operations where possible (use unsafe if needed)
- Constant-time operations for crypto validation

OUTPUT FORMAT:
- Provide complete Go file with package, imports, types, and functions
- Include unit tests with table-driven test cases
- Add benchmark tests for marshal/unmarshal operations
- Document expected performance (ops/sec)

EXAMPLE TEST CASE:
func TestFrameMarshalUnmarshal(t *testing.T) {
    tests := []struct {
        name    string
        frame   *Frame
        wantErr bool
    }{
        // Test cases here
    }
}
```

### Prompt for Cryptography Module

```
You are a cryptography expert implementing secure communication protocols.

TASK: Implement Ed25519 key management and ChaCha20-Poly1305 encryption for ShadowMesh.

REQUIREMENTS:
1. Key Generation:
   - Generate Ed25519 keypair for device identity
   - Generate X25519 keypair for session key exchange
   - Derive shared secret using ECDH
   - Use HKDF for key derivation

2. Encryption Functions:
   - EncryptFrame(plaintext []byte, sharedSecret [32]byte, nonce [24]byte) ([]byte, error)
   - DecryptFrame(ciphertext []byte, sharedSecret [32]byte, nonce [24]byte) ([]byte, error)
   - GenerateNonce() [24]byte

3. Key Storage:
   - SavePrivateKey(key ed25519.PrivateKey, password []byte) error
   - LoadPrivateKey(password []byte) (ed25519.PrivateKey, error)
   - Encrypt private keys with AES-256-GCM before storage
   - Use OS keychain when available (keyring library)

4. Key Exchange Protocol:
   - Implement X3DH-like protocol for initial handshake
   - Support key rotation every 24 hours or 1GB
   - Maintain forward secrecy

SECURITY REQUIREMENTS:
- Constant-time operations for all crypto comparisons
- Secure random number generation (crypto/rand)
- Zero memory after use (use memguard or similar)
- No hardcoded keys or secrets
- Implement key derivation with proper info strings

PACKAGES TO USE:
- golang.org/x/crypto/ed25519
- golang.org/x/crypto/curve25519
- golang.org/x/crypto/chacha20poly1305
- golang.org/x/crypto/hkdf

OUTPUT:
- Complete crypto.go file with all functions
- Unit tests with known test vectors
- Security audit checklist
- Performance benchmarks (encrypt/decrypt ops per second)

EXAMPLE:
// KeyExchange performs X25519 ECDH and derives shared secret
func KeyExchange(privateKey, publicKey [32]byte) ([32]byte, error) {
    sharedSecret, err := curve25519.X25519(privateKey[:], publicKey[:])
    if err != nil {
        return [32]byte{}, err
    }
    
    // Derive encryption key using HKDF
    kdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("ShadowMesh v1"))
    key := make([]byte, 32)
    if _, err := io.ReadFull(kdf, key); err != nil {
        return [32]byte{}, err
    }
    
    var result [32]byte
    copy(result[:], key)
    return result, nil
}
```

---

## Module 2: WebSocket Transport Layer

### Prompt for WebSocket Server

```
You are a Go backend engineer specializing in real-time communications.

TASK: Implement a production-grade WebSocket server for ShadowMesh relay nodes.

REQUIREMENTS:
1. Server Features:
   - Support both WS and WSS (TLS 1.3)
   - Handle 1000+ concurrent connections
   - Graceful shutdown and connection draining
   - Rate limiting per client IP
   - Connection authentication using Ed25519 signatures

2. Connection Management:
   - Track all active connections in memory-efficient way
   - Implement connection pooling
   - Automatic reconnection with exponential backoff
   - Heartbeat/ping-pong every 30 seconds
   - Idle connection timeout (5 minutes)

3. Message Handling:
   - Binary message support only (no text frames)
   - Maximum message size: 64KB
   - Message queue per connection (buffered channel)
   - Backpressure handling (drop oldest messages if queue full)

4. Metrics:
   - Prometheus metrics for monitoring
   - Track: connections, messages sent/received, errors, latency
   - Expose /metrics endpoint

5. Cloudflare Compatibility:
   - Support WebSocket upgrade through Cloudflare proxy
   - Handle Cloudflare-specific headers (CF-Ray, CF-Connecting-IP)
   - Path randomization to avoid pattern detection

IMPLEMENTATION DETAILS:
- Use gorilla/websocket or nhooyr.io/websocket
- Connection struct should include:
  * net.Conn (underlying connection)
  * Public key (Ed25519) for identity
  * Last seen timestamp
  * Message queue (chan []byte)
  * Context for cancellation

PERFORMANCE TARGETS:
- Message latency: < 5ms p99
- Memory per connection: < 10KB
- CPU usage: < 50% on 4-core machine at 1000 connections

OUTPUT:
- Complete websocket_server.go file
- Integration tests simulating 100 concurrent clients
- Load testing script using vegeta or similar
- Configuration file (YAML) for server parameters

EXAMPLE CONFIGURATION:
server:
  listen_addr: "0.0.0.0:8080"
  tls_cert: "/path/to/cert.pem"
  tls_key: "/path/to/key.pem"
  max_connections: 1000
  read_buffer_size: 4096
  write_buffer_size: 4096
  handshake_timeout: 10s
  idle_timeout: 300s
  rate_limit:
    requests_per_second: 100
    burst: 200
```

### Prompt for WebSocket Client

```
You are implementing the client-side WebSocket connection manager.

TASK: Create a robust WebSocket client with automatic reconnection and failover.

REQUIREMENTS:
1. Connection Features:
   - Automatic reconnection with exponential backoff
   - Multiple relay server support (failover)
   - Prefer direct P2P, fallback to relay
   - Connection quality monitoring

2. Reconnection Strategy:
   - Initial backoff: 1 second
   - Max backoff: 60 seconds
   - Jitter to avoid thundering herd
   - Circuit breaker pattern for failed relays

3. Message Queue:
   - In-memory queue for outgoing messages
   - Persistent queue option (write to disk)
   - Maximum queue size with overflow handling
   - Priority queue (control messages > data)

4. Connection States:
   - Disconnected, Connecting, Connected, Reconnecting, Failed
   - State machine with proper transitions
   - Event callbacks for state changes

5. Error Handling:
   - Network errors: retry
   - Authentication errors: notify user
   - Protocol errors: disconnect and report
   - Timeout errors: reconnect

PACKAGES:
- gorilla/websocket or nhooyr.io/websocket
- github.com/cenkalti/backoff for retry logic

OUTPUT:
- websocket_client.go with full implementation
- Unit tests mocking network failures
- Example usage code
- State diagram documentation

EXAMPLE USAGE:
client := NewWebSocketClient(config)
client.OnConnected(func() {
    log.Info("Connected to relay")
})
client.OnDisconnected(func(err error) {
    log.Warn("Disconnected:", err)
})
client.Connect([]string{"wss://relay1.shadowmesh.io", "wss://relay2.shadowmesh.io"})
client.Send(message)
```

---

## Module 3: P2P Networking and NAT Traversal

### Prompt for NAT Traversal

```
You are a network engineer implementing NAT traversal techniques.

TASK: Implement WebSocket-based NAT hole punching for ShadowMesh P2P connections.

CONTEXT:
Unlike traditional UDP hole punching, we use WebSocket connections which are TCP-based. This requires different techniques:
1. Simultaneous TCP open
2. Relay-assisted connection establishment
3. TURN-like relay fallback

REQUIREMENTS:
1. Connection Establishment:
   - Try direct connection first (if public IPs)
   - Attempt hole punching via relay coordination
   - Fallback to relay mode if P2P fails
   - Timeout each attempt (5 seconds max)

2. Relay Coordination Protocol:
   - Peer A sends connection request to relay
   - Relay forwards request to Peer B
   - Both peers initiate simultaneous connect
   - First successful connection wins

3. Connection Types:
   - Direct: Both peers have public IPs
   - Relay-Assisted: Hole punching with relay help
   - Relayed: All traffic through relay (fallback)

4. NAT Type Detection:
   - Implement STUN-like protocol over WebSocket
   - Detect: Full Cone, Restricted, Port Restricted, Symmetric
   - Choose strategy based on NAT types

5. Optimization:
   - Cache successful connection methods
   - Learn NAT behavior over time
   - Prefer connection types with higher success rate

ALGORITHM:
1. Exchange public addresses through relay
2. Detect NAT types for both peers
3. If compatible NAT types:
   a. Send "prepare" message through relay
   b. Both peers initiate outbound connections
   c. Use first successful connection
4. Else: Use relay mode

OUTPUT:
- nat_traversal.go implementation
- Success rate metrics by NAT type combination
- Integration tests with simulated NAT environments
- Documentation of supported NAT scenarios

SUCCESS RATE TARGETS:
- Full Cone NAT: 95%+
- Restricted Cone NAT: 85%+
- Port Restricted NAT: 75%+
- Symmetric NAT (both): 0% (use relay)
```

### Prompt for DHT Implementation

```
You are implementing a distributed hash table for peer discovery.

TASK: Create a lightweight DHT for ShadowMesh peer discovery and routing.

REQUIREMENTS:
1. DHT Protocol:
   - Kademlia-based design (similar to BitTorrent DHT)
   - 160-bit node IDs derived from Ed25519 public keys
   - K-buckets for routing table (K=20)
   - Iterative lookup with alpha=3 parallel requests

2. Operations:
   - Store(key, value): Store peer information
   - FindNode(nodeID): Find K closest nodes
   - FindValue(key): Retrieve stored value
   - Ping(node): Check if node is alive

3. Peer Information Stored:
   - Ed25519 public key
   - WebSocket addresses (multiple relays)
   - Last seen timestamp
   - Announced services/capabilities

4. Bootstrap Process:
   - Hardcoded bootstrap nodes (3-5 reliable nodes)
   - DNS-based bootstrapping (TXT records)
   - Blockchain-based node discovery (smart contract)

5. Security:
   - Rate limiting on lookups
   - Node ID verification (must match public key)
   - Sybil attack prevention
   - Eclipse attack detection

IMPLEMENTATION:
- Use goroutine pool for parallel requests
- LRU cache for frequently accessed values
- Periodic routing table maintenance (every 10 minutes)
- Metric tracking for network health

OUTPUT:
- dht.go with full Kademlia implementation
- Unit tests for routing table operations
- Integration tests with 100+ simulated nodes
- Performance benchmarks (lookups per second)

EXAMPLE:
dht := NewDHT(myPublicKey, bootstrapNodes)
dht.Bootstrap()
// Announce self
dht.Store(myPublicKey, myPeerInfo)
// Find peer
peerInfo, found := dht.FindValue(peerPublicKey)
```

---

## Module 4: Blockchain Integration

### Prompt for Smart Contract Development

```
You are a Solidity smart contract developer focused on security.

TASK: Develop Ethereum smart contracts for ShadowMesh device authentication.

REQUIREMENTS:
1. DeviceRegistry Contract:
   - Register device with Ed25519 public key hash
   - Pay registration fee (prevents spam)
   - Set expiration time (default: 1 year)
   - Owner can revoke device
   - Query device status (active/inactive/expired)

2. Access Control:
   - Only owner can revoke own devices
   - Admin can ban malicious devices (multisig required)
   - Upgradeability via proxy pattern (OpenZeppelin)

3. Events:
   - DeviceRegistered(bytes32 indexed deviceId, address indexed owner, uint256 expiration)
   - DeviceRevoked(bytes32 indexed deviceId, address indexed owner)
   - DeviceBanned(bytes32 indexed deviceId, string reason)

4. Gas Optimization:
   - Pack variables to minimize storage slots
   - Use events instead of storage when possible
   - Batch operations support

5. Security:
   - ReentrancyGuard on payable functions
   - Pausable in case of emergency
   - Rate limiting on registrations per address
   - No floating point math (use fixed point)

SOLIDITY VERSION: 0.8.20 or higher (use native overflow checks)

DEPENDENCIES:
- @openzeppelin/contracts (latest)

OUTPUT:
- DeviceRegistry.sol with full implementation
- Deploy script (Hardhat or Foundry)
- Unit tests with 100% coverage
- Gas usage report
- Security audit checklist

EXAMPLE CONTRACT STRUCTURE:
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/security/Pausable.sol";

contract DeviceRegistry is Ownable, ReentrancyGuard, Pausable {
    struct Device {
        bytes32 publicKeyHash;
        uint256 registrationTime;
        uint256 expirationTime;
        bool active;
    }
    
    mapping(bytes32 => Device) public devices;
    mapping(address => bytes32[]) public userDevices;
    
    uint256 public constant REGISTRATION_FEE = 0.01 ether;
    uint256 public constant DEFAULT_EXPIRATION = 365 days;
    
    // Implementation here...
}
```

### Prompt for Blockchain Client

```
You are implementing blockchain interaction layer in Go.

TASK: Create Go client for interacting with ShadowMesh smart contracts.

REQUIREMENTS:
1. Contract Interaction:
   - Generate Go bindings using abigen
   - Wrap contract calls with retry logic
   - Cache contract instances
   - Support multiple RPC providers (failover)

2. Transaction Management:
   - Automatic gas estimation with 10% buffer
   - Nonce management (sequential, no gaps)
   - Transaction status monitoring
   - Resubmission on failure with higher gas price

3. Event Monitoring:
   - Subscribe to DeviceRegistered events
   - Process events in order (by block number)
   - Handle reorgs (wait for confirmations)
   - Store last processed block (checkpoint)

4. Key Management:
   - Load private key from encrypted keystore
   - Support hardware wallets (Ledger, Trezor)
   - Sign transactions offline when possible
   - HD wallet support (BIP-44)

5. Provider Configuration:
   - Support Infura, Alchemy, Quicknode
   - Automatic failover on provider errors
   - Rate limiting per provider
   - WebSocket for events, HTTP for calls

PACKAGES:
- github.com/ethereum/go-ethereum
- github.com/ethereum/go-ethereum/accounts/abi/bind

OUTPUT:
- blockchain_client.go implementation
- Contract bindings (generated from Solidity)
- Integration tests using Hardhat local node
- Example configuration file

EXAMPLE USAGE:
client := NewBlockchainClient(config)
defer client.Close()

// Register device
tx, err := client.RegisterDevice(publicKeyHash, opts)
if err != nil {
    log.Fatal(err)
}

// Wait for confirmation
receipt, err := client.WaitForTransaction(tx.Hash(), 3) // 3 confirmations

// Monitor events
events := client.SubscribeDeviceEvents()
for event := range events {
    log.Info("Device registered:", event.DeviceId)
}
```

---

## Module 5: Cloud Infrastructure as Code

### Prompt for AWS Terraform Module

```
You are a DevOps engineer specializing in AWS infrastructure.

TASK: Create Terraform module for deploying ShadowMesh relay nodes on AWS.

REQUIREMENTS:
1. VPC Configuration:
   - VPC with CIDR 10.0.0.0/16
   - Public subnet in 3 AZs (multi-AZ for HA)
   - Private subnet for management
   - Internet Gateway for public subnets
   - NAT Gateway for private subnets (1 per AZ)

2. Compute Resources:
   - Auto Scaling Group for relay nodes
   - Launch Template with user data script
   - Instance type: t3.medium (or configurable)
   - AMI: Latest Ubuntu 22.04 LTS
   - Key pair for SSH access

3. Load Balancing:
   - Application Load Balancer (ALB)
   - Target group with health checks
   - HTTPS listener (port 443) with ACM certificate
   - WebSocket upgrade support
   - Connection draining (5 minutes)

4. Security:
   - Security groups with minimal permissions
   - Allow inbound: 443 (HTTPS/WSS), 22 (SSH from bastion)
   - Allow outbound: 443 (HTTPS), 80 (HTTP for updates)
   - KMS key for EBS encryption
   - IAM role with least privilege

5. Monitoring:
   - CloudWatch logs for application logs
   - CloudWatch metrics for resource monitoring
   - CloudWatch alarms for critical metrics
   - SNS topic for alert notifications

6. Storage:
   - S3 bucket for configuration files (encrypted)
   - DynamoDB table for state (optional)
   - Backup retention: 30 days

7. Networking:
   - Route53 hosted zone for domain
   - Route53 records for load balancer
   - CloudFront distribution (optional, for global distribution)

TERRAFORM STRUCTURE:
modules/
  aws/
    vpc/
    compute/
    alb/
    security/
    monitoring/
    main.tf
    variables.tf
    outputs.tf
    versions.tf

OUTPUT:
- Complete Terraform module with all .tf files
- variables.tf with descriptions and defaults
- outputs.tf with useful outputs (ALB DNS, etc.)
- README.md with usage instructions
- Example tfvars file

EXAMPLE MAIN.TF:
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

module "vpc" {
  source = "./vpc"
  
  vpc_cidr = var.vpc_cidr
  azs      = var.availability_zones
  
  public_subnet_cidrs  = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
  
  tags = var.tags
}

# More modules...
```

### Prompt for Multi-Cloud Abstraction

```
You are creating a cloud-agnostic deployment system.

TASK: Implement abstraction layer for deploying across AWS, Azure, GCP, and UpCloud.

REQUIREMENTS:
1. Common Interface:
   - Define CloudProvider interface in Go
   - Methods: CreateNetwork, CreateInstance, CreateLoadBalancer, etc.
   - Implementations for each cloud provider

2. Resource Mapping:
   - Map generic resources to cloud-specific resources
   - Handle provider-specific quirks
   - Validate configurations before deployment

3. Deployment Workflow:
   - Parse user configuration (YAML)
   - Select cloud provider(s)
   - Generate provider-specific IaC (Terraform/Pulumi)
   - Execute deployment
   - Monitor and report status

4. Configuration Format:
deployment:
  name: shadowmesh-relay-cluster
  regions:
    - provider: aws
      region: us-west-2
      nodes: 3
    - provider: gcp
      region: us-central1
      nodes: 2
  instance_type: medium  # Maps to t3.medium, n1-standard-2, etc.
  storage: 50GB
  network:
    cidr: 10.0.0.0/16

5. Cost Optimization:
   - Use spot/preemptible instances when available
   - Auto-scaling based on load
   - Reserved instances for predictable workloads
   - Cost estimation before deployment

OUTPUT:
- cloud_provider.go with interface definition
- Implementations: aws_provider.go, azure_provider.go, gcp_provider.go, upcloud_provider.go
- config_parser.go for YAML parsing
- deployer.go for orchestration
- CLI tool for deployment management

EXAMPLE INTERFACE:
type CloudProvider interface {
    // Network operations
    CreateNetwork(ctx context.Context, config NetworkConfig) (*Network, error)
    DeleteNetwork(ctx context.Context, networkID string) error
    
    // Compute operations
    CreateInstance(ctx context.Context, config InstanceConfig) (*Instance, error)
    DeleteInstance(ctx context.Context, instanceID string) error
    
    // Load balancer operations
    CreateLoadBalancer(ctx context.Context, config LBConfig) (*LoadBalancer, error)
    DeleteLoadBalancer(ctx context.Context, lbID string) error
    
    // Status and monitoring
    GetInstanceStatus(ctx context.Context, instanceID string) (Status, error)
    GetMetrics(ctx context.Context, instanceID string) (*Metrics, error)
}
```

---

## Module 6: AI Agent Integration

### Prompt for Network Optimizer Agent

```
You are designing an AI agent for network optimization using Development Team/Development Team API.

TASK: Create network optimizer agent that analyzes metrics and suggests improvements.

AGENT CAPABILITIES:
1. Data Analysis:
   - Collect metrics from Prometheus
   - Parse network topology graph
   - Identify bottlenecks and inefficiencies
   - Predict future traffic patterns

2. Optimization Strategies:
   - Suggest relay node placement for optimal latency
   - Recommend instance type changes for cost/performance
   - Propose routing changes to reduce congestion
   - Identify underutilized resources

3. Action Execution:
   - Generate Terraform changes for infrastructure
   - Update routing tables programmatically
   - Trigger auto-scaling events
   - Schedule maintenance windows

4. Reporting:
   - Weekly optimization reports
   - Real-time alerts for anomalies
   - Cost savings calculations
   - Performance improvement metrics

IMPLEMENTATION:
1. Metrics Collection:
   - Query Prometheus for last 7 days data
   - Aggregate by relay node, region, connection type
   - Calculate p50, p95, p99 latencies
   - Identify trends using time series analysis

2. AI Processing:
   - Construct prompt for Development Team/Development Team with metrics data
   - Request structured JSON response with recommendations
   - Parse and validate suggestions
   - Score recommendations by impact/effort

3. User Interaction:
   - Present recommendations in dashboard
   - Allow user approval before applying changes
   - Track applied vs rejected suggestions
   - Learn from user feedback

PROMPT TEMPLATE FOR AI:
You are a network optimization expert analyzing ShadowMesh VPN metrics.

METRICS DATA:
{metrics_json}

CURRENT TOPOLOGY:
{topology_json}

TASK:
1. Analyze the metrics for anomalies, bottlenecks, and inefficiencies
2. Identify relay nodes with high latency or resource utilization
3. Suggest concrete optimizations with expected impact
4. Prioritize suggestions by ROI (cost savings vs implementation effort)

OUTPUT FORMAT (JSON only):
{
  "analysis": {
    "summary": "Brief overview of findings",
    "anomalies": ["List of detected issues"],
    "bottlenecks": ["Resources at capacity"]
  },
  "recommendations": [
    {
      "id": "unique_id",
      "type": "infrastructure|routing|scaling",
      "priority": "high|medium|low",
      "description": "What to change",
      "impact": "Expected improvement",
      "cost": "Implementation cost/effort",
      "actions": ["Specific steps to implement"]
    }
  ]
}

OUTPUT:
- optimizer_agent.go with full implementation
- Prometheus query templates
- Example recommendations JSON
- Integration tests with mock metrics
- Documentation for adding custom optimization rules
```

### Prompt for Security Auditor Agent

```
You are creating an AI security auditor for ShadowMesh.

TASK: Implement security auditor agent that monitors for vulnerabilities and attacks.

AGENT RESPONSIBILITIES:
1. Log Analysis:
   - Scan logs for suspicious patterns
   - Detect brute force attempts
   - Identify unusual traffic patterns
   - Flag potential data exfiltration

2. Configuration Auditing:
   - Check for insecure configurations
   - Validate encryption settings
   - Review firewall rules
   - Verify principle of least privilege

3. Threat Detection:
   - Monitor blockchain for fraudulent transactions
   - Detect Sybil attacks on DHT
   - Identify relay node compromise attempts
   - Track known malicious IPs

4. Compliance Checking:
   - GDPR compliance verification
   - Key rotation schedule adherence
   - Log retention policy compliance
   - Encryption standard verification

DETECTION RULES:
1. Failed Authentication Pattern:
   - > 5 failures from same IP in 1 minute
   - Action: Temporary IP ban (1 hour)

2. Unusual Traffic Volume:
   - > 10x normal traffic from single device
   - Action: Rate limit and alert

3. Suspicious Blockchain Activity:
   - Device registration from known malicious address
   - Action: Flag for manual review

4. Configuration Drift:
   - Production config differs from approved baseline
   - Action: Alert and generate correction plan

AI INTEGRATION:
- Feed logs and events to Development Team/Development Team
- Request threat assessment and remediation steps
- Use AI to correlate events across systems
- Generate human-readable incident reports

PROMPT TEMPLATE:
You are a cybersecurity analyst reviewing ShadowMesh security logs.

LOG ENTRIES (last 1 hour):
{log_entries}

NETWORK EVENTS:
{events}

TASK:
1. Identify potential security incidents
2. Assess severity (critical|high|medium|low)
3. Recommend immediate actions
4. Suggest preventive measures

OUTPUT (JSON):
{
  "incidents": [
    {
      "id": "unique_id",
      "severity": "critical",
      "type": "brute_force|ddos|data_leak|config_drift",
      "description": "What happened",
      "affected_resources": ["List of IPs, devices, etc."],
      "confidence": 0.95,
      "recommended_actions": ["Immediate steps"],
      "root_cause": "Likely cause"
    }
  ],
  "summary": "Overall security posture",
  "trends": ["Notable patterns over time"]
}

OUTPUT:
- security_auditor.go implementation
- Rule engine for pattern matching
- Integration with SIEM (Splunk, ELK)
- Automated response playbooks
- Weekly security reports
```

---

## Module 7: Testing Strategy

### Prompt for Unit Tests

```
You are writing comprehensive unit tests for ShadowMesh.

TASK: Create unit test suite following Go best practices.

REQUIREMENTS:
1. Coverage:
   - Minimum 80% code coverage
   - 100% coverage for crypto and protocol modules
   - Test all error paths
   - Test edge cases and boundary conditions

2. Test Organization:
   - Table-driven tests for multiple scenarios
   - Subtests for logical grouping
   - Parallel tests where appropriate
   - Benchmark tests for performance-critical code

3. Mocking:
   - Mock external dependencies (network, blockchain)
   - Use interfaces for dependency injection
   - Create test helpers and fixtures
   - Avoid brittle tests (don't test implementation details)

4. Assertions:
   - Use testify/assert for readable assertions
   - Clear error messages on failure
   - Compare structs with deep equality
   - Validate invariants

EXAMPLE TEST FILE:
package protocol

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFrameMarshalUnmarshal(t *testing.T) {
    tests := []struct {
        name    string
        frame   *Frame
        wantErr bool
        errType error
    }{
        {
            name: "valid data frame",
            frame: &Frame{
                Version: 1,
                Type: TypeData,
                Length: 100,
                // ... more fields
            },
            wantErr: false,
        },
        {
            name: "invalid version",
            frame: &Frame{
                Version: 99,
                // ...
            },
            wantErr: true,
            errType: ErrInvalidFrameVersion,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            data, err := MarshalFrame(tt.frame)
            if tt.wantErr {
                require.Error(t, err)
                assert.ErrorIs(t, err, tt.errType)
                return
            }
            
            require.NoError(t, err)
            
            unmarshaledFrame, err := UnmarshalFrame(data)
            require.NoError(t, err)
            assert.Equal(t, tt.frame, unmarshaledFrame)
        })
    }
}

func BenchmarkFrameMarshal(b *testing.B) {
    frame := &Frame{/* initialized frame */}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = MarshalFrame(frame)
    }
}

OUTPUT:
- Test files (*_test.go) for all packages
- Test utilities in testutil/ package
- Mock implementations in mocks/ package
- Coverage report and badge
- CI configuration for automated testing
```

### Prompt for Integration Tests

```
You are creating integration tests for ShadowMesh components.

TASK: Write integration tests that verify component interactions.

REQUIREMENTS:
1. Test Scenarios:
   - End-to-end connection establishment
   - P2P connection with fallback to relay
   - Blockchain registration and verification
   - Multi-hop routing through network
   - Failover on relay node failure

2. Test Environment:
   - Docker Compose for local testing
   - Simulate 10+ nodes in network
   - Network chaos testing (latency, packet loss)
   - Blockchain test network (Ganache or Hardhat)

3. Test Data:
   - Generate realistic traffic patterns
   - Multiple device types and configurations
   - Various NAT scenarios
   - Mix of P2P and relayed connections

4. Assertions:
   - Verify end-to-end connectivity
   - Measure latency and throughput
   - Check data integrity (no corruption)
   - Validate security properties

DOCKER COMPOSE SETUP:
version: '3.8'
services:
  relay1:
    build: .
    command: --mode relay --config /config/relay1.yaml
    ports:
      - "8081:8080"
  
  relay2:
    build: .
    command: --mode relay --config /config/relay2.yaml
    ports:
      - "8082:8080"
  
  blockchain:
    image: trufflesuite/ganache:latest
    ports:
      - "8545:8545"
  
  client1:
    build: .
    command: --mode client --config /config/client1.yaml
    depends_on:
      - relay1
      - relay2
      - blockchain

EXAMPLE TEST:
func TestEndToEndConnection(t *testing.T) {
    // Setup: Start relay nodes and blockchain
    ctx := context.Background()
    env := NewTestEnvironment(t)
    defer env.Cleanup()
    
    // Create two clients
    client1 := env.NewClient("client1")
    client2 := env.NewClient("client2")
    
    // Register devices on blockchain
    require.NoError(t, client1.RegisterOnBlockchain())
    require.NoError(t, client2.RegisterOnBlockchain())
    
    // Establish connection
    conn, err := client1.ConnectToPeer(client2.PublicKey())
    require.NoError(t, err)
    defer conn.Close()
    
    // Send data and verify
    testData := []byte("Hello, ShadowMesh!")
    n, err := conn.Write(testData)
    require.NoError(t, err)
    assert.Equal(t, len(testData), n)
    
    receivedData := make([]byte, len(testData))
    n, err = client2.Read(receivedData)
    require.NoError(t, err)
    assert.Equal(t, testData, receivedData[:n])
    
    // Verify connection type
    assert.True(t, conn.IsP2P() || conn.IsRelayed())
    
    // Measure latency
    rtt := conn.MeasureRTT()
    assert.Less(t, rtt, 100*time.Millisecond)
}

OUTPUT:
- integration_test.go with multiple scenarios
- Docker Compose configuration
- Test environment setup utilities
- Network chaos injection tools
- Performance benchmark results
```

---

## Code Review Checklist

When reviewing code (manually or via AI), check:

### Security
- [ ] No hardcoded secrets or credentials
- [ ] Input validation on all external data
- [ ] Constant-time crypto comparisons
- [ ] Proper error handling without information leakage
- [ ] Rate limiting on public endpoints
- [ ] SQL injection prevention (if using SQL)
- [ ] XSS prevention (if rendering HTML)
- [ ] CSRF protection (if using cookies)

### Performance
- [ ] No blocking operations in hot paths
- [ ] Efficient data structures chosen
- [ ] Memory allocations minimized
- [ ] Goroutine leaks prevented
- [ ] Database queries optimized
- [ ] Caching used appropriately
- [ ] Benchmarks show acceptable performance

### Reliability
- [ ] Graceful error handling
- [ ] Retries with exponential backoff
- [ ] Circuit breakers for external services
- [ ] Timeouts on all network operations
- [ ] Resource cleanup (defer close())
- [ ] Context cancellation respected
- [ ] No data races (use -race flag)

### Maintainability
- [ ] Clear, descriptive naming
- [ ] Functions under 50 lines
- [ ] Proper separation of concerns
- [ ] DRY principle followed
- [ ] Comments explain "why", not "what"
- [ ] Godoc on all public functions
- [ ] Examples provided for complex usage

### Testing
- [ ] Unit tests for all public functions
- [ ] Edge cases tested
- [ ] Error paths tested
- [ ] Integration tests for interactions
- [ ] Test coverage above threshold
- [ ] Benchmarks for performance-critical code
- [ ] No flaky tests

---

## AI Agent Workflow Example

### Complete Feature Implementation Flow

```
FEATURE: Implement automatic relay node failover

STEP 1: Planning (Development Team Agent)
PROMPT: "Act as a software architect. Design a relay node failover mechanism for ShadowMesh. Consider: health checking, automatic detection, connection migration, and zero downtime. Provide high-level design and component breakdown."

STEP 2: Implementation (Coding Agent)
PROMPT: "Implement relay_failover.go based on this design: {design}. Include health checker, failure detector, and connection migrator components. Follow Go best practices and ShadowMesh coding standards."

STEP 3: Testing (Testing Agent)
PROMPT: "Write comprehensive tests for relay_failover.go. Include unit tests, integration tests simulating relay failures, and benchmarks for failover speed. Target: < 100ms failover time."

STEP 4: Documentation (Documentation Agent)
PROMPT: "Generate documentation for the relay failover feature. Include architecture diagram, configuration options, monitoring metrics, and troubleshooting guide."

STEP 5: Review (Security Agent)
PROMPT: "Review relay_failover.go for security issues. Check: race conditions, resource leaks, denial of service vulnerabilities, and proper error handling."

STEP 6: Optimization (Performance Agent)
PROMPT: "Analyze relay_failover.go performance. Profile CPU and memory usage. Suggest optimizations to reduce overhead. Target: < 1% CPU at idle."

STEP 7: Deployment (DevOps Agent)
PROMPT: "Create deployment plan for relay failover feature. Include: feature flag configuration, rollout strategy, monitoring setup, and rollback procedure."
```

---

## Quick Reference: Go Best Practices

### Error Handling
```go
// Good
if err != nil {
    return fmt.Errorf("failed to connect to relay: %w", err)
}

// Bad
if err != nil {
    log.Println(err)
    return err
}
```

### Concurrency
```go
// Good
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Bad
time.Sleep(30 * time.Second)
```

### Resource Management
```go
// Good
f, err := os.Open("file.txt")
if err != nil {
    return err
}
defer f.Close()

// Bad
f, err := os.Open("file.txt")
// ... might return before closing
f.Close()
```

### Interfaces
```go
// Good: Accept interfaces, return structs
func ProcessConnection(conn Connector) (*Result, error)

// Bad: Return interfaces
func NewConnection() Connector
```

---

## Deployment Checklist

Before deploying to production:

- [ ] All tests passing (unit + integration)
- [ ] Security audit completed
- [ ] Performance benchmarks meet targets
- [ ] Documentation updated
- [ ] Monitoring and alerting configured
- [ ] Backup and recovery procedures documented
- [ ] Disaster recovery plan tested
- [ ] Rate limiting configured
- [ ] DDoS protection enabled
- [ ] SSL/TLS certificates valid
- [ ] Environment variables configured
- [ ] Database migrations tested
- [ ] Rollback procedure documented
- [ ] On-call team notified
- [ ] User communication prepared

---

## Success Metrics

Track these KPIs to measure ShadowMesh success:

### Technical Metrics
- Connection success rate: > 95%
- P2P connection rate: > 80%
- Average latency: < 50ms
- Uptime: > 99.9%
- Bandwidth utilization: > 70%

### Business Metrics
- Active devices: Growth rate
- Monthly active users: Growth rate
- Relay node operators: Number and distribution
- Infrastructure cost per user: < $1/month
- Customer satisfaction: > 4.5/5

### Security Metrics
- Blocked attack attempts: Count per day
- Mean time to detect (MTTD): < 5 minutes
- Mean time to respond (MTTR): < 30 minutes
- Vulnerability disclosure to patch: < 7 days
- Security incidents: 0 per month

