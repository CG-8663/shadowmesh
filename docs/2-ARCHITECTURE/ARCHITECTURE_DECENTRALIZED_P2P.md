# ShadowMesh Decentralized P2P Architecture

**Version**: 1.0
**Date**: 2025-11-04
**Status**: Design Specification

## Executive Summary

This document specifies the architecture for transforming ShadowMesh from a relay-based VPN into a **fully decentralized, incentivized, quantum-safe peer-to-peer network** using Kademlia DHT and blockchain-based relay coordination.

**Key Innovations**:
- Zero centralized dependencies (no STUN, TURN, DNS)
- Kademlia DHT for peer discovery and coordination
- Smart contract relay registry with staking/slashing
- Token-incentivized bandwidth market
- Self-organizing network bootstrapped from blockchain state
- Preserves post-quantum security guarantees

**Strategic Value**:
- First decentralized quantum-safe VPN
- Web3-native architecture attracts crypto markets
- Censorship-resistant (no single point of failure)
- Scales horizontally without infrastructure costs
- Revenue from token transactions vs. infrastructure overhead

---

## 1. System Architecture

### 1.1 High-Level Components

```
┌─────────────────────────────────────────────────────────────┐
│                    ShadowMesh Network                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐       ┌──────────────┐                    │
│  │  Client A    │◄─────►│  Client B    │   Direct P2P       │
│  │  (NAT Easy)  │  DHT  │  (NAT Easy)  │   Connection       │
│  └──────────────┘       └──────────────┘                    │
│         │                       │                             │
│         │  NAT Hole Punch       │                             │
│         │  Coordination         │                             │
│         ▼                       ▼                             │
│  ┌─────────────────────────────────────┐                    │
│  │       Kademlia DHT Network          │                    │
│  │   (Distributed Hash Table)          │                    │
│  │  - Peer discovery                    │                    │
│  │  - Address storage                   │                    │
│  │  - NAT type mapping                  │                    │
│  │  - Relay selection                   │                    │
│  └─────────────────────────────────────┘                    │
│         ▲                       ▲                             │
│         │                       │                             │
│  ┌──────────────┐       ┌──────────────┐                    │
│  │  Client C    │       │  Relay Node  │   Circuit Relay    │
│  │ (NAT Hard)   │◄─────►│  (Staked)    │   for NAT-Hard     │
│  └──────────────┘       └──────────────┘   Peers            │
│                                 │                             │
│                                 │ Attestation                │
│                                 ▼                             │
│  ┌─────────────────────────────────────┐                    │
│  │    Blockchain Smart Contracts       │                    │
│  │  - Relay Registry                    │                    │
│  │  - Staking/Slashing                  │                    │
│  │  - Reputation System                 │                    │
│  │  - Token Economics                   │                    │
│  │  - TPM Attestation Verification      │                    │
│  └─────────────────────────────────────┘                    │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Component Descriptions

#### 1.2.1 Kademlia DHT Layer
- Distributed hash table storing peer metadata
- XOR distance metric for routing efficiency
- Key = PeerID (hash of post-quantum public key)
- Value = PeerRecord{IP:Port, NAT type, relay preferences, reputation}
- Implements k-bucket routing (k=20 recommended)
- 160-bit keyspace (SHA-256 truncated or BLAKE2b-160)

#### 1.2.2 Peer Types

**Standard Peer**:
- Connects to DHT on startup
- Publishes own address
- Discovers other peers
- Attempts direct connections
- Falls back to relays for hard NAT

**Relay Node**:
- All capabilities of standard peer
- Stakes tokens in smart contract
- Provides circuit relay service
- Submits TPM attestation hourly
- Earns bandwidth tokens
- Subject to slashing for misbehavior

**Bootstrap Node**:
- Long-lived DHT participants
- Addresses derived from blockchain state
- No staking requirement (community-run)
- Help new peers join DHT
- Ephemeral role (network becomes self-sustaining)

#### 1.2.3 Blockchain Layer (Ethereum/Polygon/Arbitrum)

**Smart Contracts**:
- `RelayRegistry.sol` - Registry of active relays
- `StakingManager.sol` - Token staking/slashing logic
- `AttestationVerifier.sol` - TPM attestation validation
- `ReputationOracle.sol` - On-chain reputation tracking
- `BandwidthMarket.sol` - Tokenized bandwidth payments

**Token**: SHM (ShadowMesh Token)
- ERC-20 standard
- Used for staking, payments, governance
- Deflationary (burn on slashing)

---

## 2. Protocol Designs

### 2.1 Peer Identity and Addressing

#### 2.1.1 PeerID Generation

```
# PeerID derived from post-quantum public key
ML-DSA-87 Public Key (2592 bytes)
    ↓
BLAKE2b-256 hash
    ↓
Truncate to 160 bits (DHT key space)
    ↓
PeerID: 160-bit identifier
```

**Format**:
```
PeerID: <160-bit hash>
Example: 0x8f3a4b9c1e2d5f7a6b8c9d0e1f2a3b4c5d6e7f8a
```

**Properties**:
- Unique identifier derived from PQC public key
- Cannot forge without private key
- DHT routing uses XOR distance
- Collision resistance from BLAKE2b

#### 2.1.2 Multiaddress Format

```
/ip4/203.0.113.5/udp/41231/shadowmesh/<peerid>
/ip6/2001:db8::1/udp/41231/shadowmesh/<peerid>
```

Stored in DHT as:
```protobuf
message PeerRecord {
  bytes peer_id = 1;              // 160-bit PeerID
  repeated Multiaddr addrs = 2;   // Multiple addresses (IPv4/IPv6)
  NATType nat_type = 3;           // Easy, Moderate, Hard, Unknown
  RelayPreference relay_pref = 4; // Preferred relay PeerIDs
  uint64 last_seen = 5;           // Unix timestamp
  bytes signature = 6;            // ML-DSA-87 signature over record
  uint32 protocol_version = 7;    // ShadowMesh version
}

enum NATType {
  UNKNOWN = 0;
  EASY = 1;        // Full cone NAT
  MODERATE = 2;    // Restricted cone NAT
  HARD = 3;        // Symmetric NAT
  PUBLIC = 4;      // No NAT
}
```

### 2.2 Kademlia DHT Operations

#### 2.2.1 DHT Routing Table

```
k-bucket structure (k=20):
- 160 buckets (one per bit distance)
- Each bucket holds up to k peers
- Buckets split when full
- LRU eviction policy

Distance Calculation:
distance(a, b) = a XOR b
```

**Example**:
```
MyPeerID:    10101010...
TargetPeerID: 10100110...
XOR Distance: 00001100... (Hamming distance: 2)
```

#### 2.2.2 Core DHT Operations

**FIND_NODE(target_peer_id)**:
```
1. Client queries k closest known peers to target
2. Each peer returns k closest peers from their routing table
3. Iterative lookup continues until no closer peers found
4. Returns k closest peers to target
```

**STORE(key, value, ttl)**:
```
1. Find k closest peers to key using FIND_NODE
2. Send STORE request to all k peers
3. Peers store (key, value) with expiration = now + ttl
4. Republish every ttl/2 to prevent expiration
```

**FIND_VALUE(key)**:
```
1. Query k closest peers for key
2. If any peer has value, return immediately
3. Otherwise, continue FIND_NODE lookup
4. Return value or "not found"
```

#### 2.2.3 DHT Message Format

```protobuf
message DHTMessage {
  enum MessageType {
    PING = 0;
    STORE = 1;
    FIND_NODE = 2;
    FIND_VALUE = 3;
    GET_PROVIDERS = 4;
  }

  MessageType type = 1;
  bytes message_id = 2;        // Random 16-byte ID
  bytes sender_peer_id = 3;    // Sender's PeerID
  bytes key = 4;                // DHT key (for STORE/FIND_VALUE)
  bytes value = 5;              // Payload (PeerRecord)
  repeated PeerInfo closer_peers = 6; // For FIND_NODE responses
  uint64 timestamp = 7;         // Unix timestamp
  bytes signature = 8;          // ML-DSA-87 signature
}

message PeerInfo {
  bytes peer_id = 1;
  repeated bytes addrs = 2;    // Multiaddress encoded
}
```

### 2.3 NAT Traversal Without STUN/TURN

#### 2.3.1 AutoNAT Protocol

Peers discover their external address by querying other peers:

```
Client A (behind NAT) → DHT Peer B (public IP):
  "What address do you see me from?"

DHT Peer B → Client A:
  "I see you at 203.0.113.5:41231"

Client A:
  - Compares with local address
  - Determines NAT type
  - Updates PeerRecord in DHT
```

**NAT Type Detection**:
1. Send probe from local UDP socket
2. Receive response with observed address
3. Compare with local bind address:
   - Same = No NAT (PUBLIC)
   - Different but consistent = Easy NAT
   - Different and varying = Symmetric NAT (HARD)

#### 2.3.2 DHT-Coordinated Hole Punching

For two peers behind NAT:

```
Step 1: Coordination via DHT
  Client A → DHT: "I want to connect to Client B"
  DHT → Client A: "Here's B's PeerRecord"
  DHT → Client B: "Client A wants to connect, prepare hole punch"

Step 2: Simultaneous UDP Punch
  Client A: Sends UDP packet to B's external address
  Client B: Sends UDP packet to A's external address
  (Packets create NAT mappings)

Step 3: Handshake
  Both clients retry for 5 seconds
  Once NAT mapping established, ML-KEM-1024 handshake begins
```

**Coordination Message**:
```protobuf
message HolePunchCoordination {
  bytes requester_peer_id = 1;
  bytes target_peer_id = 2;
  repeated Multiaddr requester_addrs = 3;
  uint64 punch_deadline = 4;    // Unix timestamp to start punching
  bytes signature = 5;           // ML-DSA-87 signature
}
```

#### 2.3.3 Circuit Relay Fallback

When direct connection fails (symmetric NAT on both sides):

```
Client A → Relay Node R → Client B

Relay Selection:
1. Query DHT for relays near Client B's PeerID
2. Filter by:
   - Staked in smart contract
   - Good reputation score
   - Low latency
   - Sufficient bandwidth
3. Establish A↔R and B↔R tunnels
4. Relay forwards encrypted packets
```

**Relay Circuit Message**:
```protobuf
message CircuitRelay {
  bytes relay_peer_id = 1;
  bytes source_peer_id = 2;
  bytes dest_peer_id = 3;
  bytes encrypted_payload = 4;   // End-to-end encrypted
  uint64 sequence_num = 5;
  bytes relay_signature = 6;     // Relay signs to prove relay path
}
```

**Relay Economics**:
- Relay earns X tokens per GB transferred
- Client pays from token balance or uses free tier (limited bandwidth)
- Micropayments via state channels (not every packet on-chain)

### 2.4 Bootstrap Process

#### 2.4.1 Initial Network Join

```
Step 1: Query Blockchain
  - Read RelayRegistry smart contract
  - Get list of staked relay nodes (addresses + PeerIDs)
  - Sort by reputation score
  - Select top 10 as bootstrap candidates

Step 2: Connect to Bootstrap Peers
  - Attempt connection to 3-5 bootstrap relays
  - Send FIND_NODE(my_peer_id) to learn neighbors
  - Receive k closest peers to my PeerID

Step 3: Populate Routing Table
  - Add received peers to k-buckets
  - Perform iterative FIND_NODE for random IDs
  - Discover diverse set of DHT participants

Step 4: Publish Own PeerRecord
  - Determine external address via AutoNAT
  - STORE(my_peer_id, my_peer_record) to DHT
  - Republish every 30 minutes
```

#### 2.4.2 Network Self-Organization

After bootstrap phase:
- No hardcoded servers
- Peers discover each other via DHT
- Relay nodes continuously advertise availability
- Failed relays removed via blockchain slashing
- Network remains operational even if all bootstrap nodes die

---

## 3. Smart Contract Design

### 3.1 RelayRegistry.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract RelayRegistry is Ownable {
    struct RelayNode {
        bytes32 peerId;          // 160-bit PeerID (left-padded)
        address operator;         // Ethereum address of operator
        uint256 stakedAmount;     // SHM tokens staked
        uint256 reputation;       // Reputation score (0-1000)
        uint64 registeredAt;      // Registration timestamp
        uint64 lastAttestationAt; // Last successful attestation
        string[] multiaddrs;      // IP addresses
        bool isActive;            // Can accept relay requests
        uint256 bandwidthServed;  // Total bytes relayed
        uint256 slashCount;       // Number of slashes
    }

    // Minimum stake to become relay (e.g., 1000 SHM)
    uint256 public constant MIN_STAKE = 1000 * 10**18;

    // Attestation must occur every 2 hours
    uint256 public constant ATTESTATION_INTERVAL = 2 hours;

    // Slash amount for missed attestation
    uint256 public constant SLASH_AMOUNT = 10 * 10**18;

    IERC20 public shmToken;

    mapping(bytes32 => RelayNode) public relays;
    bytes32[] public relayList;

    event RelayRegistered(bytes32 indexed peerId, address indexed operator);
    event RelayDeregistered(bytes32 indexed peerId);
    event AttestationSubmitted(bytes32 indexed peerId, uint64 timestamp);
    event RelaySlashed(bytes32 indexed peerId, uint256 amount, string reason);
    event ReputationUpdated(bytes32 indexed peerId, uint256 newReputation);

    constructor(address _shmToken) {
        shmToken = IERC20(_shmToken);
    }

    function registerRelay(
        bytes32 _peerId,
        string[] calldata _multiaddrs,
        uint256 _stakeAmount
    ) external {
        require(_stakeAmount >= MIN_STAKE, "Insufficient stake");
        require(!relays[_peerId].isActive, "Relay already exists");
        require(
            shmToken.transferFrom(msg.sender, address(this), _stakeAmount),
            "Stake transfer failed"
        );

        relays[_peerId] = RelayNode({
            peerId: _peerId,
            operator: msg.sender,
            stakedAmount: _stakeAmount,
            reputation: 500, // Start at middle reputation
            registeredAt: uint64(block.timestamp),
            lastAttestationAt: uint64(block.timestamp),
            multiaddrs: _multiaddrs,
            isActive: true,
            bandwidthServed: 0,
            slashCount: 0
        });

        relayList.push(_peerId);
        emit RelayRegistered(_peerId, msg.sender);
    }

    function submitAttestation(
        bytes32 _peerId,
        bytes calldata _tpmQuote,
        bytes calldata _signature
    ) external {
        RelayNode storage relay = relays[_peerId];
        require(relay.isActive, "Relay not active");
        require(msg.sender == relay.operator, "Not relay operator");

        // Verify TPM attestation (simplified - see AttestationVerifier)
        require(verifyTPMQuote(_tpmQuote, _signature), "Invalid attestation");

        relay.lastAttestationAt = uint64(block.timestamp);

        // Increase reputation for timely attestation
        if (relay.reputation < 1000) {
            relay.reputation += 1;
        }

        emit AttestationSubmitted(_peerId, uint64(block.timestamp));
    }

    function checkAttestations() external {
        // Anyone can call to slash relays with expired attestations
        for (uint i = 0; i < relayList.length; i++) {
            bytes32 peerId = relayList[i];
            RelayNode storage relay = relays[peerId];

            if (relay.isActive) {
                uint256 timeSinceAttestation =
                    block.timestamp - relay.lastAttestationAt;

                if (timeSinceAttestation > ATTESTATION_INTERVAL) {
                    slashRelay(peerId, "Missed attestation");
                }
            }
        }
    }

    function slashRelay(bytes32 _peerId, string memory _reason) internal {
        RelayNode storage relay = relays[_peerId];

        if (relay.stakedAmount >= SLASH_AMOUNT) {
            relay.stakedAmount -= SLASH_AMOUNT;
            relay.slashCount++;

            // Decrease reputation significantly
            if (relay.reputation >= 50) {
                relay.reputation -= 50;
            } else {
                relay.reputation = 0;
            }

            // Burn slashed tokens (deflationary)
            shmToken.transfer(address(0), SLASH_AMOUNT);

            emit RelaySlashed(_peerId, SLASH_AMOUNT, _reason);

            // Deactivate if stake falls below minimum
            if (relay.stakedAmount < MIN_STAKE) {
                relay.isActive = false;
                emit RelayDeregistered(_peerId);
            }
        }
    }

    function reportBandwidth(bytes32 _peerId, uint256 _bytesServed) external {
        // Called by relay operator to claim bandwidth rewards
        RelayNode storage relay = relays[_peerId];
        require(msg.sender == relay.operator, "Not relay operator");

        relay.bandwidthServed += _bytesServed;

        // Increase reputation based on service
        if (relay.reputation < 1000 && _bytesServed > 0) {
            relay.reputation += (_bytesServed / 1e9); // +1 per GB
            if (relay.reputation > 1000) relay.reputation = 1000;
        }

        emit ReputationUpdated(_peerId, relay.reputation);
    }

    function deregisterRelay(bytes32 _peerId) external {
        RelayNode storage relay = relays[_peerId];
        require(msg.sender == relay.operator, "Not relay operator");

        relay.isActive = false;

        // Return stake minus penalties
        shmToken.transfer(relay.operator, relay.stakedAmount);

        emit RelayDeregistered(_peerId);
    }

    function getActiveRelays() external view returns (bytes32[] memory) {
        uint256 activeCount = 0;
        for (uint i = 0; i < relayList.length; i++) {
            if (relays[relayList[i]].isActive) activeCount++;
        }

        bytes32[] memory active = new bytes32[](activeCount);
        uint256 idx = 0;
        for (uint i = 0; i < relayList.length; i++) {
            if (relays[relayList[i]].isActive) {
                active[idx++] = relayList[i];
            }
        }
        return active;
    }

    function verifyTPMQuote(
        bytes calldata _quote,
        bytes calldata _sig
    ) internal pure returns (bool) {
        // Placeholder - actual implementation in AttestationVerifier.sol
        // Verifies TPM 2.0 quote signature and PCR values
        return true;
    }
}
```

### 3.2 AttestationVerifier.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract AttestationVerifier {
    struct TPMQuote {
        bytes pcrDigest;         // SHA-256 of PCR values
        uint64 timestamp;         // Quote generation time
        bytes nonce;              // Freshness nonce
        bytes signature;          // TPM signature (RSA/ECC)
    }

    // Trusted PCR values (whitelisted relay software measurements)
    mapping(bytes32 => bool) public trustedPCRs;

    event PCRWhitelisted(bytes32 indexed pcrHash);
    event AttestationVerified(bytes32 indexed peerId, uint64 timestamp);

    function whitelistPCR(bytes32 _pcrHash) external {
        // Only governance can whitelist
        trustedPCRs[_pcrHash] = true;
        emit PCRWhitelisted(_pcrHash);
    }

    function verifyAttestation(
        bytes32 _peerId,
        bytes calldata _quoteBytes,
        bytes calldata _tpmPublicKey
    ) external view returns (bool) {
        TPMQuote memory quote = decodeQuote(_quoteBytes);

        // 1. Verify quote signature using TPM public key
        require(
            verifyTPMSignature(quote, _tpmPublicKey),
            "Invalid TPM signature"
        );

        // 2. Verify PCR values are in whitelist
        bytes32 pcrHash = keccak256(quote.pcrDigest);
        require(trustedPCRs[pcrHash], "PCR not whitelisted");

        // 3. Verify timestamp freshness (within 5 minutes)
        require(
            block.timestamp - quote.timestamp < 300,
            "Quote too old"
        );

        return true;
    }

    function decodeQuote(bytes calldata _data)
        internal
        pure
        returns (TPMQuote memory)
    {
        // Decode TPM2_Quote structure
        // Simplified - actual implementation uses TPM 2.0 spec
        return TPMQuote({
            pcrDigest: _data[0:32],
            timestamp: uint64(bytes8(_data[32:40])),
            nonce: _data[40:72],
            signature: _data[72:]
        });
    }

    function verifyTPMSignature(
        TPMQuote memory _quote,
        bytes calldata _pubKey
    ) internal pure returns (bool) {
        // Verify RSASSA-PKCS1-v1_5 or ECDSA signature
        // Uses on-chain crypto precompiles or library
        return true; // Placeholder
    }
}
```

### 3.3 BandwidthMarket.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

contract BandwidthMarket {
    IERC20 public shmToken;

    struct ServiceTicket {
        bytes32 clientPeerId;
        bytes32 relayPeerId;
        uint256 bytesQuota;       // Bytes client has paid for
        uint256 bytesConsumed;    // Bytes actually used
        uint256 pricePerGB;       // Price in SHM tokens
        uint64 expiresAt;         // Ticket expiration
        bool isActive;
    }

    mapping(bytes32 => ServiceTicket) public tickets;

    // Dynamic pricing based on relay reputation
    uint256 public constant BASE_PRICE_PER_GB = 1 * 10**16; // 0.01 SHM

    event TicketPurchased(
        bytes32 indexed clientPeerId,
        bytes32 indexed relayPeerId,
        uint256 bytesQuota
    );

    event BandwidthConsumed(
        bytes32 indexed ticketId,
        uint256 bytesConsumed
    );

    constructor(address _shmToken) {
        shmToken = IERC20(_shmToken);
    }

    function purchaseBandwidth(
        bytes32 _clientPeerId,
        bytes32 _relayPeerId,
        uint256 _gigabytes,
        uint256 _pricePerGB
    ) external {
        uint256 totalCost = _gigabytes * _pricePerGB;
        require(
            shmToken.transferFrom(msg.sender, address(this), totalCost),
            "Payment failed"
        );

        bytes32 ticketId = keccak256(
            abi.encodePacked(_clientPeerId, _relayPeerId, block.timestamp)
        );

        tickets[ticketId] = ServiceTicket({
            clientPeerId: _clientPeerId,
            relayPeerId: _relayPeerId,
            bytesQuota: _gigabytes * 1e9,
            bytesConsumed: 0,
            pricePerGB: _pricePerGB,
            expiresAt: uint64(block.timestamp + 30 days),
            isActive: true
        });

        emit TicketPurchased(_clientPeerId, _relayPeerId, _gigabytes * 1e9);
    }

    function consumeBandwidth(
        bytes32 _ticketId,
        uint256 _bytesUsed,
        bytes calldata _proof
    ) external {
        ServiceTicket storage ticket = tickets[_ticketId];
        require(ticket.isActive, "Ticket inactive");
        require(block.timestamp < ticket.expiresAt, "Ticket expired");

        // Verify proof of bandwidth transfer (simplified)
        // Actual implementation uses zero-knowledge proof or signed receipts

        ticket.bytesConsumed += _bytesUsed;
        require(
            ticket.bytesConsumed <= ticket.bytesQuota,
            "Quota exceeded"
        );

        // Pay relay proportionally
        uint256 payment = (_bytesUsed * ticket.pricePerGB) / 1e9;
        shmToken.transfer(msg.sender, payment); // msg.sender = relay operator

        emit BandwidthConsumed(_ticketId, _bytesUsed);

        if (ticket.bytesConsumed >= ticket.bytesQuota) {
            ticket.isActive = false;
        }
    }
}
```

### 3.4 Token Economics Summary

**Token Distribution**:
- 40% - Relay operator rewards (vested over 5 years)
- 25% - Staking pool (locked for slashing)
- 20% - Team and development (4-year vesting)
- 10% - Public sale/liquidity
- 5% - Community grants

**Revenue Streams**:
- Bandwidth payments from clients
- Relay staking deposits
- Transaction fees (small % on bandwidth purchases)

**Deflationary Mechanics**:
- Slashed tokens burned
- % of bandwidth fees burned

---

## 4. Security Analysis

### 4.1 Sybil Attack Mitigation

**Attack**: Attacker creates many DHT peers to control routing.

**Mitigations**:
1. **PeerID derived from PQC keys**: Expensive to generate many valid identities
2. **Relay staking**: Relay nodes require significant token stake (1000+ SHM)
3. **Reputation system**: New peers have low reputation, limited influence
4. **Diverse peer selection**: DHT queries sent to k diverse peers (XOR distance)
5. **Proof-of-work on DHT writes**: Small computational puzzle prevents spam

**Implementation**:
```go
// Before accepting new DHT peer, require PoW
func ValidateDHTPeer(peerRecord PeerRecord) bool {
    // Verify signature
    if !VerifyML_DSA_Signature(peerRecord.Signature, peerRecord.PeerID) {
        return false
    }

    // Verify proof-of-work (must find nonce where hash < difficulty)
    hash := BLAKE2b(peerRecord.Bytes() + peerRecord.Nonce)
    if hash[0:2] != 0x0000 { // Difficulty: 2 leading zero bytes
        return false
    }

    return true
}
```

### 4.2 Eclipse Attack Mitigation

**Attack**: Attacker surrounds victim peer with malicious DHT nodes.

**Mitigations**:
1. **Diverse bootstrap**: Query blockchain for relay list, not hardcoded servers
2. **Periodic routing table refresh**: Re-discover random IDs every hour
3. **Outbound connection limits**: Don't accept all peers from single source
4. **Cross-verification**: Verify peer records with multiple DHT nodes
5. **Reputation filtering**: Prefer high-reputation peers in routing table

**Implementation**:
```go
func RefreshRoutingTable() {
    // Every hour, discover new random IDs
    for i := 0; i < 10; i++ {
        randomID := GenerateRandomPeerID()
        peers := DHT.FindNode(randomID)
        for _, p := range peers {
            if ValidateAndCheckReputation(p) {
                RoutingTable.Add(p)
            }
        }
    }
}
```

### 4.3 Relay Trust Model

**Problem**: Relay can inspect traffic (even if encrypted end-to-end).

**Mitigations**:
1. **End-to-end encryption**: Relay sees only encrypted bytes
2. **Multi-hop routing**: Traffic routes through 2-3 relays (like Tor)
3. **TPM attestation**: Relays prove they run unmodified software
4. **Reputation + slashing**: Misbehaving relays lose stake
5. **Random relay selection**: Clients choose random relays, hard to target specific users

**Trust Assumptions**:
- At least one relay in multi-hop path is honest
- TPM hardware is not compromised
- Blockchain validators are honest majority

### 4.4 DHT Poisoning Attack

**Attack**: Attacker floods DHT with fake peer records.

**Mitigations**:
1. **Signature requirement**: All records signed with ML-DSA-87
2. **Rate limiting**: Limit STORE requests per peer (10/hour)
3. **Proof-of-work**: Require PoW for DHT writes
4. **Record expiration**: Records expire after 1 hour, must republish
5. **Validation on retrieval**: Clients verify signatures before using peer records

### 4.5 Quantum Threat Analysis

**Current Status**: All cryptography is post-quantum:
- ML-KEM-1024 key exchange
- ML-DSA-87 signatures
- ChaCha20-Poly1305 symmetric encryption (quantum-resistant)

**DHT Security**: XOR distance metric is quantum-resistant (no speedup from Grover's algorithm for routing).

**Blockchain Security**:
- Ethereum moving to post-quantum signatures (future upgrade)
- Smart contracts use ECDSA today (vulnerable to quantum)
- Migration path: Replace ECDSA with ML-DSA-87 in future hard fork

---

## 5. Implementation Plan

### 5.1 Phase 1: Core DHT (Weeks 1-4)

**Objectives**:
- Implement Kademlia DHT in Go
- PeerID generation from ML-DSA-87 keys
- k-bucket routing table
- FIND_NODE, STORE, FIND_VALUE operations

**Deliverables**:
- `pkg/dht/kademlia.go` - Core DHT implementation
- `pkg/dht/routing_table.go` - k-bucket management
- `pkg/dht/peer_record.go` - PeerRecord structure
- Unit tests with 90%+ coverage
- Integration test with 100-node local DHT

**Dependencies**:
- `github.com/libp2p/go-libp2p-kad-dht` (reference, not used directly)
- Custom implementation for PQC integration

### 5.2 Phase 2: NAT Traversal (Weeks 5-8)

**Objectives**:
- AutoNAT protocol for address discovery
- DHT-coordinated hole punching
- Circuit relay fallback

**Deliverables**:
- `pkg/nat/autonat.go` - AutoNAT implementation
- `pkg/nat/holepunch.go` - Hole punching coordination
- `pkg/relay/circuit.go` - Circuit relay protocol
- NAT traversal success rate testing (target: >80% for moderate NAT)

**Testing**:
- Simulate various NAT types (Easy, Moderate, Hard)
- Measure connection success rates
- Latency benchmarks

### 5.3 Phase 3: Smart Contracts (Weeks 9-12)

**Objectives**:
- Deploy RelayRegistry, AttestationVerifier, BandwidthMarket
- Token (SHM) deployment
- Frontend for relay operator registration

**Deliverables**:
- `contracts/RelayRegistry.sol` - Relay registration
- `contracts/AttestationVerifier.sol` - TPM verification
- `contracts/BandwidthMarket.sol` - Token economics
- Hardhat deployment scripts
- Web dashboard for relay operators

**Blockchain Choice**: Polygon (low fees) or Arbitrum (Ethereum L2)

### 5.4 Phase 4: Integration (Weeks 13-16)

**Objectives**:
- Integrate DHT with existing ShadowMesh client
- Bootstrap from blockchain state
- End-to-end P2P connections

**Deliverables**:
- Modified `cmd/client/main.go` with DHT initialization
- Bootstrap logic from smart contracts
- Peer discovery replacing centralized directory
- Full P2P tunnel establishment

**Testing**:
- 1000-node testnet
- Measure peer discovery time (<30s)
- Connection success rates
- Performance vs. centralized architecture

### 5.5 Phase 5: Relay Economics (Weeks 17-20)

**Objectives**:
- Relay operator staking flow
- Bandwidth payment system
- TPM attestation integration

**Deliverables**:
- `pkg/blockchain/relay_client.go` - Smart contract interaction
- `pkg/attestation/tpm_quote.go` - TPM 2.0 quote generation
- Relay operator CLI tools
- Payment channel implementation (off-chain micropayments)

### 5.6 Phase 6: Production Hardening (Weeks 21-24)

**Objectives**:
- Security audit
- Performance optimization
- Monitoring and observability

**Deliverables**:
- Third-party security audit report
- Prometheus metrics
- Grafana dashboards
- Incident response playbook
- Documentation

**Launch Criteria**:
- All tests passing
- Security audit complete
- 100+ relay nodes staked
- <5ms latency overhead vs. direct connection
- 99.9% connection success rate

---

## 6. Technology Stack

### 6.1 Core Implementation (Go)

**Language**: Go 1.21+

**Key Libraries**:
```go
// Cryptography
"github.com/cloudflare/circl/kem/kyber/kyber1024"
"github.com/cloudflare/circl/sign/dilithium/mode5"
"golang.org/x/crypto/chacha20poly1305"

// Networking
"github.com/libp2p/go-libp2p" // Only for multiaddr, not full libp2p
"github.com/gorilla/websocket"

// DHT (custom implementation)
"shadowmesh/pkg/dht"

// Blockchain interaction
"github.com/ethereum/go-ethereum/ethclient"
"github.com/ethereum/go-ethereum/accounts/abi/bind"

// Serialization
"google.golang.org/protobuf"

// NAT traversal
"github.com/pion/stun" // Only for comparison, not actual use
```

### 6.2 Smart Contracts (Solidity)

**Language**: Solidity 0.8.20+

**Framework**: Hardhat

**Libraries**:
```json
{
  "@openzeppelin/contracts": "^5.0.0",
  "@chainlink/contracts": "^0.8.0",
  "hardhat": "^2.19.0",
  "@nomicfoundation/hardhat-toolbox": "^4.0.0"
}
```

### 6.3 Blockchain Selection

**Mainnet Options**:
1. **Polygon** (Recommended for MVP)
   - Low gas fees (~$0.01 per tx)
   - Fast finality (~2s)
   - EVM-compatible
   - Large user base

2. **Arbitrum**
   - Ethereum L2 (more security)
   - Moderate fees (~$0.50 per tx)
   - Slower finality (~10s)

3. **Base** (Coinbase L2)
   - Growing ecosystem
   - Low fees
   - Good UX for non-crypto users

**Testnet**: Polygon Mumbai or Arbitrum Sepolia

### 6.4 Development Tools

**Testing**:
- Go test framework
- Testcontainers (for integration tests)
- Hardhat (smart contract testing)

**Monitoring**:
- Prometheus (metrics)
- Grafana (visualization)
- Jaeger (distributed tracing)

**CI/CD**:
- GitHub Actions
- Docker for reproducible builds

---

## 7. Performance Analysis

### 7.1 DHT Lookup Performance

**Theoretical Complexity**:
- Kademlia lookup: O(log N) messages
- For N=10,000 peers, log₂(10,000) ≈ 13 hops
- With parallel queries (α=3), reduced to ~5 rounds

**Expected Latency**:
```
Peer discovery time = (log N / α) × RTT
With N=10,000, α=3, RTT=50ms:
  = (13 / 3) × 50ms
  ≈ 217ms
```

**Optimization**:
- Cache frequently-accessed peers
- Prefetch during idle time
- Use UDP for low latency

### 7.2 NAT Traversal Success Rates

**Expected Success**:
- Easy NAT (Full Cone): 95%+ direct connection
- Moderate NAT (Restricted): 80%+ direct connection
- Hard NAT (Symmetric): 30% direct, 70% via relay

**Fallback Latency**:
- Direct: +0.5ms
- Single relay: +2-5ms
- Multi-hop (3 relays): +10-20ms

### 7.3 Throughput Comparison

```
Configuration          Throughput    Latency
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Direct P2P             6-7 Gbps      <1ms
Single Relay           3-4 Gbps      +2ms
Multi-hop (3 relays)   1-2 Gbps      +15ms
WireGuard (baseline)   5-6 Gbps      +1ms
```

**Bottlenecks**:
- Relay node CPU (ChaCha20 encryption)
- Network bandwidth
- DHT lookup time (first connection only)

**Mitigation**:
- Hardware acceleration (AES-NI for ChaCha20)
- Relay node load balancing
- Persistent peer connections

### 7.4 Blockchain Overhead

**On-Chain Operations** (rare):
- Relay registration: Once per operator
- Attestation: Every 2 hours (batched)
- Slashing: Only on misbehavior

**Off-Chain Operations** (common):
- Peer discovery: Via DHT
- Connection establishment: P2P
- Bandwidth payments: State channels (settle periodically)

**Cost Analysis**:
```
Operation               Frequency     Gas Cost     USD (Polygon)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Relay registration      Once          200k gas     $0.02
Attestation             Every 2h      50k gas      $0.005
Bandwidth payment       Per GB        30k gas      $0.003
Slash relay             On violation  100k gas     $0.01

Total per relay/month:  ~360 tx      ~18M gas     $18
```

**Optimization**:
- Batch attestations from multiple relays
- Use L2 rollups for cheaper transactions
- Implement state channels for micropayments

---

## 8. Competitive Positioning

### 8.1 Feature Comparison Matrix

```
Feature                 WireGuard  Tailscale  ZeroTier  ShadowMesh
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Post-Quantum Crypto     ❌         ❌         ❌        ✅
Decentralized           ❌         ❌         ❌        ✅
No STUN/TURN deps       ❌         ❌         ❌        ✅
Incentivized relays     ❌         ❌         ❌        ✅
Multi-hop routing       ❌         ❌         ❌        ✅
TPM attestation         ❌         ❌         ❌        ✅
Blockchain verified     ❌         ❌         ❌        ✅
NAT traversal           ✅         ✅         ✅        ✅
High throughput         ✅         ✅         ✅        ✅
```

### 8.2 Unique Value Propositions

**vs. WireGuard**:
- WireGuard: Simple protocol, no NAT traversal, manual peer management
- ShadowMesh: Automatic peer discovery, decentralized coordination, quantum-safe

**vs. Tailscale**:
- Tailscale: Centralized coordination server, closed-source control plane
- ShadowMesh: No central server, blockchain-verified relays, open-source

**vs. ZeroTier**:
- ZeroTier: Centralized root servers, proprietary protocol
- ShadowMesh: DHT-based, standardized crypto (NIST PQC), decentralized

**vs. Tor**:
- Tor: Slow (~1 Mbps), designed for anonymity not performance
- ShadowMesh: High-speed (1-6 Gbps), optional multi-hop, better UX

### 8.3 Target Market Segments

**Primary Markets**:
1. **Crypto/Web3 Users** - Natural fit, already have wallets
2. **Privacy Advocates** - Want decentralization, no trust in Big Tech
3. **Enterprise Security** - Quantum-resistance, attestation, compliance

**Go-To-Market Strategy**:
1. Launch on crypto-native platforms (Reddit /r/cryptocurrency, Twitter crypto)
2. Token incentives for early relay operators
3. Partner with Web3 projects (decentralized hosting, DeFi)
4. Enterprise pilot programs (fintech, healthcare)

### 8.4 Pricing Strategy

**Consumer Tier**:
- Free: 10 GB/month via community relays
- Premium: $15/month unlimited bandwidth
- Ultra: $30/month + multi-hop routing + priority relays

**Enterprise Tier**:
- Standard: $50/user/month
- Pro: $100/user/month + SLA + dedicated relays
- Custom: Contact sales (government, defense)

**Relay Operator Revenue**:
- Earn 0.01 SHM per GB relayed (~$0.10 at $10/SHM)
- Stake 1000 SHM ($10,000 at $10/SHM)
- ROI: ~20-30% APY if relay serves 100 GB/day

---

## 9. Migration Path from Current Architecture

### 9.1 Current State (Relay-Based)

```
Client → Centralized Directory Server → Relay Node → Internet
```

**Dependencies**:
- Directory server for relay discovery
- STUN server for NAT detection
- TURN server for symmetric NAT

### 9.2 Hybrid Transition (Months 1-3)

```
Client → {DHT OR Directory Server} → {Community Relay OR Paid Relay}
```

**Implementation**:
1. Deploy smart contracts on testnet
2. Add DHT client code (opt-in flag `--enable-dht`)
3. Bootstrap DHT from centralized directory
4. Gradual relay operator migration to staking

**Compatibility**:
- Old clients use centralized servers
- New clients use DHT
- Both share same protocol (ML-KEM-1024 handshake)

### 9.3 Full Decentralization (Months 4-6)

```
Client → DHT → Community Relays (blockchain-verified)
```

**Implementation**:
1. Deprecate centralized directory (read-only)
2. All new relays must stake on-chain
3. Remove STUN/TURN dependencies
4. DHT becomes default (remove `--enable-dht` flag)

**Cutover Plan**:
- Announce 3-month deprecation period
- Offer relay migration incentives (bonus tokens)
- Maintain backward compatibility for 6 months

---

## 10. Risk Analysis and Mitigation

### 10.1 Technical Risks

**Risk**: DHT fails to converge in large networks
**Likelihood**: Low
**Impact**: High
**Mitigation**: Extensive testing with 10,000+ node simulations, fallback to centralized bootstrap

**Risk**: NAT traversal success rate <60%
**Likelihood**: Medium
**Impact**: High
**Mitigation**: Multi-tier fallback (hole punch → single relay → multi-relay), pre-launch testing with diverse NAT configurations

**Risk**: Smart contract exploit drains staking pool
**Likelihood**: Low
**Impact**: Critical
**Mitigation**: Professional security audit, bug bounty program, multi-sig governance, gradual rollout

**Risk**: TPM attestation bypassed
**Likelihood**: Low
**Impact**: High
**Mitigation**: Regular PCR whitelist updates, community oversight, reputation system catches anomalies

### 10.2 Economic Risks

**Risk**: Token price crashes, relay operators quit
**Likelihood**: Medium
**Impact**: High
**Mitigation**: Hybrid model (free community relays + paid premium relays), long-term staking rewards, protocol revenue diversification

**Risk**: Insufficient relay nodes at launch
**Likelihood**: Medium
**Impact**: Medium
**Mitigation**: Pre-launch relay operator recruitment, bonus tokens for first 100 operators, easy setup scripts

**Risk**: Bandwidth costs exceed token revenue
**Likelihood**: Low
**Impact**: Medium
**Mitigation**: Dynamic pricing based on supply/demand, allow relay operators to set own prices, subsidize with protocol treasury initially

### 10.3 Regulatory Risks

**Risk**: Token classified as security
**Likelihood**: Medium (US), Low (EU/Asia)
**Impact**: High
**Mitigation**: Utility token design (not investment), legal opinion, possible dual-token model (utility + governance)

**Risk**: VPN services restricted in jurisdiction
**Likelihood**: Medium (China, Russia)
**Impact**: Medium
**Mitigation**: Decentralized architecture makes shutdown difficult, traffic obfuscation, community-run relays in free jurisdictions

**Risk**: Smart contract considered money transmission
**Likelihood**: Low
**Impact**: Medium
**Mitigation**: Peer-to-peer payments, no custody of funds, legal structure as protocol not service provider

### 10.4 Operational Risks

**Risk**: Relay operators collude to censor traffic
**Likelihood**: Low
**Impact**: Medium
**Mitigation**: Multi-hop routing, random relay selection, reputation system, easy relay deployment (anyone can run)

**Risk**: Sybil attack dominates DHT
**Likelihood**: Medium
**Impact**: High
**Mitigation**: Proof-of-work on DHT entries, relay staking requirements, reputation filtering, periodic routing table refresh

**Risk**: Blockchain network congestion
**Likelihood**: Medium
**Impact**: Low
**Mitigation**: Use L2 (Polygon/Arbitrum), batch transactions, off-chain state channels, multi-chain deployment

---

## 11. Success Metrics

### 11.1 Technical KPIs

**Performance**:
- Peer discovery time: <30s for 90th percentile
- NAT traversal success: >80% direct connection
- Throughput: >3 Gbps via relay, >6 Gbps direct
- Latency overhead: <5ms added vs. direct connection

**Reliability**:
- Network uptime: 99.9%
- DHT query success rate: >99%
- Connection establishment: >95% success
- Relay node uptime: >99% (enforced by slashing)

**Security**:
- Zero critical vulnerabilities post-audit
- No successful Sybil attacks in testnet
- <1% malicious relay detection rate
- All cryptography uses NIST-approved PQC

### 11.2 Business KPIs

**Adoption**:
- 1,000+ active users by Month 6
- 10,000+ active users by Month 12
- 100+ relay operators by Month 6
- 500+ relay operators by Month 12

**Revenue**:
- $50k MRR by Month 12
- $500k MRR by Month 24
- 30%+ gross margin (token revenue - infrastructure costs)

**Token Economics**:
- $10+ SHM price by Month 12 (assuming utility value)
- $100M+ total value locked in staking
- >80% of bandwidth paid in SHM tokens (vs. fiat)

### 11.3 Community KPIs

**Engagement**:
- 10,000+ Discord/Telegram members
- 5,000+ GitHub stars
- 100+ code contributors
- 50+ relay operator community meetups

**Decentralization**:
- No single relay operator >5% of network capacity
- Relays in 50+ countries
- <30% of traffic through top 10 relays
- Community governance (DAO) controls protocol upgrades

---

## 12. Open Questions and Future Research

### 12.1 Unresolved Technical Questions

**Q1**: Can we achieve <10ms peer discovery with DHT?
**Approach**: Implement predictive prefetching, cache frequently-contacted peers, use faster DHT (e.g., S/Kademlia)

**Q2**: How to handle relay nodes behind CGNAT?
**Approach**: Relay-to-relay tunneling, use public relays as bridges, investigate IPv6 deployment

**Q3**: Is TPM attestation sufficient or do we need SGX?
**Approach**: TPM for MVP (broader hardware support), add SGX for sensitive operations in Phase 2

**Q4**: Should we support IPv6-only networks?
**Approach**: Yes, dual-stack from Day 1, IPv6 may help with NAT traversal

### 12.2 Economic Model Questions

**Q1**: What's the optimal stake amount to prevent Sybil attacks?
**Approach**: Start with 1000 SHM, adjust based on token price and attack cost analysis

**Q2**: Should bandwidth pricing be fixed or auction-based?
**Approach**: Fixed pricing for MVP (predictable UX), auction model in Phase 2 for efficiency

**Q3**: How to handle relay operator churn?
**Approach**: Long-term staking rewards, reputation bonuses for uptime, easy migration scripts

**Q4**: Should we support fiat payments or SHM-only?
**Approach**: Hybrid (fiat for consumers, SHM for power users), on-ramp partnerships

### 12.3 Future Enhancements

**Phase 2 Features** (Months 7-12):
- Mobile apps (iOS, Android)
- Browser extension (VPN in browser)
- Site-to-site VPN mode
- Multi-device sync (like Tailscale)

**Phase 3 Features** (Months 13-18):
- Anonymous credentials (like Tor)
- Traffic analysis resistance (constant-rate padding)
- Decentralized exit node marketplace
- AI-powered relay selection

**Phase 4 Features** (Months 19-24):
- Hardware security modules (HSM) for enterprises
- Integration with decentralized storage (IPFS, Arweave)
- Mesh networking mode (peer-to-peer LANs)
- Quantum key distribution (QKD) for ultra-secure links

---

## 13. Conclusion

This architecture transforms ShadowMesh into a **fully decentralized, quantum-safe, incentivized VPN network** with no centralized dependencies. Key differentiators:

1. **First-mover advantage**: Only decentralized quantum-safe VPN
2. **Web3-native**: Blockchain-verified relays, token economics
3. **Censorship-resistant**: DHT-based, no single point of failure
4. **Financially sustainable**: Relay operators earn tokens, protocol charges fees

**Next Steps**:
1. Review and approve architecture with team
2. Begin Phase 1 implementation (DHT core)
3. Deploy testnet smart contracts
4. Recruit relay operator beta testers
5. Launch MVP in 6 months

**Long-term Vision**: By 2027, ShadowMesh will be the standard for decentralized private networking, with 100,000+ users and 1,000+ community-run relays across the globe.

---

## Appendix A: Protocol Message Formats

### A.1 DHT Messages (Protobuf)

```protobuf
syntax = "proto3";
package shadowmesh.dht;

message DHTMessage {
  enum MessageType {
    PING = 0;
    STORE = 1;
    FIND_NODE = 2;
    FIND_VALUE = 3;
  }

  MessageType type = 1;
  bytes message_id = 2;
  bytes sender_peer_id = 3;
  bytes key = 4;
  bytes value = 5;
  repeated PeerInfo closer_peers = 6;
  uint64 timestamp = 7;
  bytes signature = 8;
  bytes nonce = 9; // Proof-of-work nonce
}

message PeerInfo {
  bytes peer_id = 1;
  repeated string multiaddrs = 2;
  uint32 nat_type = 3;
  uint64 last_seen = 4;
}

message PeerRecord {
  bytes peer_id = 1;
  repeated string multiaddrs = 2;
  uint32 nat_type = 3;
  repeated bytes relay_preferences = 4;
  uint64 timestamp = 5;
  bytes signature = 6;
  uint32 protocol_version = 7;
}
```

### A.2 Hole Punch Coordination

```protobuf
message HolePunchRequest {
  bytes requester_peer_id = 1;
  bytes target_peer_id = 2;
  repeated string requester_addrs = 3;
  uint64 punch_start_time = 4; // Coordinated start time
  bytes signature = 5;
}

message HolePunchResponse {
  bytes target_peer_id = 1;
  repeated string target_addrs = 2;
  bool ready = 3;
  bytes signature = 4;
}
```

### A.3 Circuit Relay Messages

```protobuf
message CircuitRelayRequest {
  bytes source_peer_id = 1;
  bytes dest_peer_id = 2;
  bytes relay_peer_id = 3;
  uint64 bandwidth_quota = 4; // In bytes
  bytes payment_proof = 5;    // Signed ticket or state channel proof
  bytes signature = 6;
}

message CircuitRelayData {
  bytes source_peer_id = 1;
  bytes dest_peer_id = 2;
  bytes encrypted_payload = 3;
  uint64 sequence_num = 4;
  bytes relay_signature = 5;
}
```

---

## Appendix B: Implementation Checklist

### Phase 1: Core DHT (Weeks 1-4)
- [ ] PeerID generation from ML-DSA-87 public key
- [ ] k-bucket routing table implementation
- [ ] FIND_NODE operation
- [ ] STORE operation
- [ ] FIND_VALUE operation
- [ ] Proof-of-work validation for DHT entries
- [ ] Unit tests (90%+ coverage)
- [ ] 100-node local DHT integration test

### Phase 2: NAT Traversal (Weeks 5-8)
- [ ] AutoNAT protocol implementation
- [ ] NAT type detection (Easy, Moderate, Hard)
- [ ] DHT-coordinated hole punching
- [ ] UDP simultaneous open
- [ ] Circuit relay fallback protocol
- [ ] Multi-relay routing (2-3 hops)
- [ ] NAT traversal success rate testing

### Phase 3: Smart Contracts (Weeks 9-12)
- [ ] RelayRegistry.sol implementation
- [ ] AttestationVerifier.sol implementation
- [ ] BandwidthMarket.sol implementation
- [ ] SHM token (ERC-20) deployment
- [ ] Hardhat test suite
- [ ] Deploy to Polygon Mumbai testnet
- [ ] Web dashboard for relay operators

### Phase 4: Integration (Weeks 13-16)
- [ ] Integrate DHT with ShadowMesh client
- [ ] Bootstrap from blockchain state
- [ ] Replace centralized directory with DHT
- [ ] End-to-end P2P connection test
- [ ] 1000-node testnet deployment
- [ ] Performance benchmarking

### Phase 5: Relay Economics (Weeks 17-20)
- [ ] Smart contract interaction library
- [ ] Relay operator staking flow
- [ ] TPM attestation generation
- [ ] Bandwidth payment state channels
- [ ] Relay operator CLI tools
- [ ] Economic simulation (token price, demand)

### Phase 6: Production (Weeks 21-24)
- [ ] Security audit (3rd party)
- [ ] Bug bounty program launch
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Documentation (user + operator)
- [ ] Mainnet deployment (Polygon)
- [ ] Public beta launch

---

## Appendix C: References

### Academic Papers
1. **Kademlia DHT**: Maymounkov, P., & Mazières, D. (2002). "Kademlia: A Peer-to-Peer Information System Based on the XOR Metric"
2. **NAT Traversal**: Ford, B., Srisuresh, P., & Kegel, D. (2005). "Peer-to-Peer Communication Across Network Address Translators"
3. **Post-Quantum Cryptography**: NIST (2024). "Post-Quantum Cryptography Standardization"

### Protocol Specifications
- NIST FIPS 203 (ML-KEM)
- NIST FIPS 204 (ML-DSA)
- TPM 2.0 Library Specification
- ERC-20 Token Standard

### Open Source Projects
- libp2p (reference for DHT design)
- WireGuard (protocol inspiration)
- Ethereum (smart contract platform)
- Polygon (L2 scaling solution)

---

**Document End**

*This architecture is a living document and will be updated as implementation progresses and new requirements emerge.*
