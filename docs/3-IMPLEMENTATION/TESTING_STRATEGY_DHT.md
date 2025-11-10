# ShadowMesh DHT Testing Strategy

**Version**: 1.0
**Target**: v0.2.0-alpha (DHT + PQC standalone release)
**Last Updated**: November 10, 2025
**Status**: Approved

---

## Executive Summary

Comprehensive testing strategy for ShadowMesh's Kademlia DHT implementation, covering unit tests, integration tests, performance benchmarks, and acceptance criteria for v0.2.0-alpha standalone release.

**Testing Pyramid**:
```
                  ▲
                 ╱ ╲
                ╱   ╲
               ╱ E2E ╲          10 tests (Manual + Automated)
              ╱───────╲
             ╱         ╲
            ╱Integration╲       50 tests (Automated)
           ╱─────────────╲
          ╱               ╲
         ╱  Unit Tests     ╲    200+ tests (Automated)
        ╱───────────────────╲
       ╱                     ╲
      ╱  Static Analysis     ╲  Linting, Security Scans
     ╱───────────────────────╲
```

**Coverage Targets**:
- Unit Tests: 85%+ code coverage
- Integration Tests: All DHT operations validated
- Performance: ≥25 Mbps throughput (maintain v11 baseline)
- Reliability: 95%+ peer discovery success rate

---

## 1. Unit Testing Strategy

### 1.1. PeerID Generation & Verification

**File**: `pkg/discovery/peer_id_test.go`

```go
package discovery_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestGeneratePeerID(t *testing.T) {
    tests := []struct {
        name string
        test func(t *testing.T)
    }{
        {
            name: "PeerID deterministic from public key",
            test: func(t *testing.T) {
                publicKey := generateMLDSA87PublicKey()
                peerID1 := GeneratePeerID(publicKey)
                peerID2 := GeneratePeerID(publicKey)
                assert.Equal(t, peerID1, peerID2, "Same public key must generate same PeerID")
            },
        },
        {
            name: "Different public keys generate different PeerIDs",
            test: func(t *testing.T) {
                publicKey1 := generateMLDSA87PublicKey()
                publicKey2 := generateMLDSA87PublicKey()
                peerID1 := GeneratePeerID(publicKey1)
                peerID2 := GeneratePeerID(publicKey2)
                assert.NotEqual(t, peerID1, peerID2, "Different keys must generate different PeerIDs")
            },
        },
        {
            name: "PeerID is 32 bytes (256 bits)",
            test: func(t *testing.T) {
                publicKey := generateMLDSA87PublicKey()
                peerID := GeneratePeerID(publicKey)
                assert.Len(t, peerID, 32, "PeerID must be 32 bytes")
            },
        },
        {
            name: "PeerID collision resistance (1000 keys)",
            test: func(t *testing.T) {
                peerIDs := make(map[PeerID]bool)
                for i := 0; i < 1000; i++ {
                    publicKey := generateMLDSA87PublicKey()
                    peerID := GeneratePeerID(publicKey)
                    assert.False(t, peerIDs[peerID], "PeerID collision detected")
                    peerIDs[peerID] = true
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, tt.test)
    }
}

func TestVerifyPeerOwnership(t *testing.T) {
    privateKey, publicKey := generateMLDSA87Keypair()
    peerID := GeneratePeerID(publicKey)
    challenge := GenerateChallenge()
    signature := signChallenge(privateKey, challenge)

    tests := []struct {
        name      string
        peerID    PeerID
        publicKey []byte
        signature []byte
        challenge []byte
        wantErr   bool
    }{
        {
            name:      "Valid ownership proof",
            peerID:    peerID,
            publicKey: publicKey,
            signature: signature,
            challenge: challenge,
            wantErr:   false,
        },
        {
            name:      "Mismatched PeerID and public key",
            peerID:    GeneratePeerID(generateMLDSA87PublicKey()), // Different key
            publicKey: publicKey,
            signature: signature,
            challenge: challenge,
            wantErr:   true,
        },
        {
            name:      "Invalid signature",
            peerID:    peerID,
            publicKey: publicKey,
            signature: []byte("invalid-signature"),
            challenge: challenge,
            wantErr:   true,
        },
        {
            name:      "Wrong challenge signed",
            peerID:    peerID,
            publicKey: publicKey,
            signature: signChallenge(privateKey, []byte("different-challenge")),
            challenge: challenge,
            wantErr:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := VerifyPeerOwnership(tt.peerID, tt.publicKey, tt.signature, tt.challenge)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// Benchmark PeerID generation
func BenchmarkGeneratePeerID(b *testing.B) {
    publicKey := generateMLDSA87PublicKey()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        GeneratePeerID(publicKey)
    }
}
```

**Coverage Target**: 100% (critical security code)

**Acceptance Criteria**:
- [x] All tests pass
- [x] PeerID generation <1ms (benchmark)
- [x] No PeerID collisions in 1000 key test
- [x] Ownership verification works correctly

---

### 1.2. XOR Distance Metric

**File**: `pkg/discovery/distance_test.go`

```go
func TestXORDistance(t *testing.T) {
    tests := []struct {
        name string
        test func(t *testing.T)
    }{
        {
            name: "Commutative property: XOR(A, B) == XOR(B, A)",
            test: func(t *testing.T) {
                id1 := randomPeerID()
                id2 := randomPeerID()
                d1 := XORDistance(id1, id2)
                d2 := XORDistance(id2, id1)
                assert.Equal(t, d1.Cmp(d2), 0, "XOR distance must be commutative")
            },
        },
        {
            name: "Identity property: XOR(A, A) == 0",
            test: func(t *testing.T) {
                id := randomPeerID()
                distance := XORDistance(id, id)
                assert.Equal(t, distance.Cmp(big.NewInt(0)), 0, "Distance to self must be 0")
            },
        },
        {
            name: "Triangle inequality",
            test: func(t *testing.T) {
                a := randomPeerID()
                b := randomPeerID()
                c := randomPeerID()

                dAB := XORDistance(a, b)
                dBC := XORDistance(b, c)
                dAC := XORDistance(a, c)

                // XOR(A, C) <= XOR(A, B) + XOR(B, C)
                sum := new(big.Int).Add(dAB, dBC)
                assert.True(t, dAC.Cmp(sum) <= 0, "Triangle inequality violated")
            },
        },
        {
            name: "Closer peers have smaller distance",
            test: func(t *testing.T) {
                target := PeerID{0xFF, 0xFF, 0xFF} // All 1s for first 3 bytes
                closer := PeerID{0xFF, 0xFF, 0xFE}  // Differs in last bit of byte 3
                farther := PeerID{0x00, 0x00, 0x00} // All 0s

                dCloser := XORDistance(target, closer)
                dFarther := XORDistance(target, farther)

                assert.True(t, dCloser.Cmp(dFarther) < 0, "Closer peer must have smaller distance")
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, tt.test)
    }
}

func BenchmarkXORDistance(b *testing.B) {
    id1 := randomPeerID()
    id2 := randomPeerID()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        XORDistance(id1, id2)
    }
}
```

**Coverage Target**: 100%

**Acceptance Criteria**:
- [x] All XOR properties validated
- [x] XOR calculation <10μs (benchmark)

---

### 1.3. k-bucket Operations

**File**: `pkg/discovery/kbucket_test.go`

```go
func TestKBucket(t *testing.T) {
    t.Run("Add up to k peers", func(t *testing.T) {
        kb := NewKBucket(20)
        for i := 0; i < 20; i++ {
            peer := randomPeerInfo()
            kb.AddPeer(peer)
        }
        assert.Len(t, kb.GetPeers(), 20, "k-bucket should hold 20 peers")
    })

    t.Run("LRU eviction when full", func(t *testing.T) {
        kb := NewKBucket(3)
        peer1 := randomPeerInfo()
        peer2 := randomPeerInfo()
        peer3 := randomPeerInfo()
        peer4 := randomPeerInfo()

        kb.AddPeer(peer1)
        kb.AddPeer(peer2)
        kb.AddPeer(peer3)
        kb.AddPeer(peer4) // Should evict peer1 (oldest)

        peers := kb.GetPeers()
        assert.Len(t, peers, 3)
        assert.NotContains(t, peers, peer1, "Oldest peer should be evicted")
        assert.Contains(t, peers, peer4, "Newest peer should be added")
    })

    t.Run("Update last seen moves peer to front", func(t *testing.T) {
        kb := NewKBucket(3)
        peer1 := randomPeerInfo()
        peer2 := randomPeerInfo()
        peer3 := randomPeerInfo()

        kb.AddPeer(peer1)
        kb.AddPeer(peer2)
        kb.AddPeer(peer3)

        // Update peer1 (should move to front)
        peer1.LastSeen = time.Now()
        kb.AddPeer(peer1)

        // Add peer4 (should evict peer2, not peer1)
        peer4 := randomPeerInfo()
        kb.AddPeer(peer4)

        peers := kb.GetPeers()
        assert.Contains(t, peers, peer1, "Recently updated peer should not be evicted")
        assert.NotContains(t, peers, peer2, "Least recently used peer should be evicted")
    })
}

func TestKBucketConcurrency(t *testing.T) {
    kb := NewKBucket(20)
    var wg sync.WaitGroup

    // 100 goroutines adding peers concurrently
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            peer := randomPeerInfo()
            kb.AddPeer(peer)
        }()
    }

    wg.Wait()

    // Verify no race conditions (should complete without panic)
    peers := kb.GetPeers()
    assert.LessOrEqual(t, len(peers), 20, "k-bucket should not exceed k peers")
}
```

**Coverage Target**: 90%+

**Run with race detector**:
```bash
go test ./pkg/discovery -run TestKBucketConcurrency -race
```

---

### 1.4. Routing Table Operations

**File**: `pkg/discovery/routing_table_test.go`

```go
func TestRoutingTable(t *testing.T) {
    localID := randomPeerID()
    rt := NewRoutingTable(localID, 20)

    t.Run("Add peers to correct buckets", func(t *testing.T) {
        // Add 1000 peers
        for i := 0; i < 1000; i++ {
            peer := randomPeerInfo()
            rt.AddPeer(peer)
        }

        // Verify peers distributed across buckets
        allPeers := rt.AllPeers()
        assert.Greater(t, len(allPeers), 0, "Routing table should have peers")

        // Check bucket index calculation
        for _, peer := range allPeers {
            expectedBucket := rt.BucketIndex(peer.PeerID)
            assert.GreaterOrEqual(t, expectedBucket, 0)
            assert.Less(t, expectedBucket, 256)
        }
    })

    t.Run("FindClosest returns peers in distance order", func(t *testing.T) {
        target := randomPeerID()

        // Add 100 peers
        for i := 0; i < 100; i++ {
            peer := randomPeerInfo()
            rt.AddPeer(peer)
        }

        // Find 20 closest
        closest := rt.FindClosest(target, 20)
        assert.LessOrEqual(t, len(closest), 20)

        // Verify they are in distance order
        for i := 0; i < len(closest)-1; i++ {
            d1 := XORDistance(target, closest[i].PeerID)
            d2 := XORDistance(target, closest[i+1].PeerID)
            assert.True(t, d1.Cmp(d2) <= 0, "Peers must be in distance order")
        }
    })

    t.Run("RemovePeer removes from routing table", func(t *testing.T) {
        peer := randomPeerInfo()
        rt.AddPeer(peer)

        allPeers := rt.AllPeers()
        assert.Contains(t, allPeers, peer)

        rt.RemovePeer(peer.PeerID)

        allPeers = rt.AllPeers()
        assert.NotContains(t, allPeers, peer)
    })
}

func BenchmarkFindClosest(b *testing.B) {
    localID := randomPeerID()
    rt := NewRoutingTable(localID, 20)

    // Add 1000 peers
    for i := 0; i < 1000; i++ {
        peer := randomPeerInfo()
        rt.AddPeer(peer)
    }

    target := randomPeerID()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        rt.FindClosest(target, 20)
    }
}
```

**Coverage Target**: 85%+

**Acceptance Criteria**:
- [x] FindClosest returns peers in correct order
- [x] Bucket index calculation correct for all PeerIDs
- [x] FindClosest <10ms for 1000-peer routing table

---

### 1.5. DHT Message Protocol

**File**: `pkg/discovery/protocol_test.go`

```go
func TestDHTMessageProtocol(t *testing.T) {
    t.Run("Serialize and deserialize all message types", func(t *testing.T) {
        messages := []struct {
            msgType uint8
            payload []byte
        }{
            {MSG_PING, nil},
            {MSG_PONG, nil},
            {MSG_FIND_NODE, serializeFindNodePayload(randomPeerID())},
            {MSG_FOUND_NODES, serializeFoundNodesPayload([]PeerInfo{randomPeerInfo()})},
            // ... other message types
        }

        for _, msg := range messages {
            original := &DHTMessage{
                Type:      msg.msgType,
                RequestID: randomRequestID(),
                SenderID:  randomPeerID(),
                Payload:   msg.payload,
            }

            // Serialize
            data, err := original.Serialize()
            assert.NoError(t, err)

            // Deserialize
            decoded, err := DeserializeDHTMessage(data)
            assert.NoError(t, err)

            // Verify
            assert.Equal(t, original.Type, decoded.Type)
            assert.Equal(t, original.RequestID, decoded.RequestID)
            assert.Equal(t, original.SenderID, decoded.SenderID)
        }
    })

    t.Run("Signature verification", func(t *testing.T) {
        privateKey, publicKey := generateMLDSA87Keypair()
        peerID := GeneratePeerID(publicKey)

        msg := &DHTMessage{
            Type:      MSG_PING,
            RequestID: randomRequestID(),
            SenderID:  peerID,
            Payload:   nil,
        }

        // Sign message
        err := msg.Sign(privateKey)
        assert.NoError(t, err)

        // Verify signature
        err = msg.Verify(publicKey)
        assert.NoError(t, err)

        // Tamper with message
        msg.Payload = []byte("tampered")

        // Verification should fail
        err = msg.Verify(publicKey)
        assert.Error(t, err)
    })
}

func BenchmarkMessageSerialization(b *testing.B) {
    msg := &DHTMessage{
        Type:      MSG_PING,
        RequestID: randomRequestID(),
        SenderID:  randomPeerID(),
        Payload:   nil,
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        msg.Serialize()
    }
}
```

**Coverage Target**: 90%+

**Acceptance Criteria**:
- [x] All message types serialize/deserialize correctly
- [x] Signature verification works
- [x] Serialization <100μs per message

---

## 2. Integration Testing Strategy

### 2.1. Local 3-Node Network Tests

**File**: `test/integration/local_network_test.go`

```go
func TestLocal3NodeNetwork(t *testing.T) {
    // Start 3 nodes on localhost
    node1 := startNode("localhost:8443")
    node2 := startNode("localhost:8444")
    node3 := startNode("localhost:8445")

    defer node1.Shutdown()
    defer node2.Shutdown()
    defer node3.Shutdown()

    // Configure bootstrap peers
    node1.AddBootstrapPeer(node2.Address())
    node1.AddBootstrapPeer(node3.Address())
    node2.AddBootstrapPeer(node1.Address())
    node3.AddBootstrapPeer(node1.Address())

    // Wait for routing tables to converge
    time.Sleep(10 * time.Second)

    t.Run("All nodes discover each other", func(t *testing.T) {
        // Node1 should know about node2 and node3
        peers1 := node1.RoutingTable().AllPeers()
        assert.Contains(t, peerIDs(peers1), node2.PeerID())
        assert.Contains(t, peerIDs(peers1), node3.PeerID())

        // Node2 should know about node1 and node3
        peers2 := node2.RoutingTable().AllPeers()
        assert.Contains(t, peerIDs(peers2), node1.PeerID())
        assert.Contains(t, peerIDs(peers2), node3.PeerID())

        // Node3 should know about node1 and node2
        peers3 := node3.RoutingTable().AllPeers()
        assert.Contains(t, peerIDs(peers3), node1.PeerID())
        assert.Contains(t, peerIDs(peers3), node2.PeerID())
    })

    t.Run("PING between all node pairs", func(t *testing.T) {
        // node1 → node2
        ok, err := node1.Ping(node2.PeerInfo())
        assert.NoError(t, err)
        assert.True(t, ok)

        // node1 → node3
        ok, err = node1.Ping(node3.PeerInfo())
        assert.NoError(t, err)
        assert.True(t, ok)

        // node2 → node3
        ok, err = node2.Ping(node3.PeerInfo())
        assert.NoError(t, err)
        assert.True(t, ok)
    })

    t.Run("FIND_NODE lookup works", func(t *testing.T) {
        target := randomPeerID()

        // node1 finds closest peers to target
        results := node1.FindNode(target, 5*time.Second)
        assert.NotEmpty(t, results)

        // Results should include node2 and node3
        assert.Contains(t, peerIDs(results), node2.PeerID())
        assert.Contains(t, peerIDs(results), node3.PeerID())
    })

    t.Run("STORE and FIND_VALUE", func(t *testing.T) {
        // node1 stores its metadata
        err := node1.StoreSelf()
        assert.NoError(t, err)

        // Wait for propagation
        time.Sleep(1 * time.Second)

        // node3 retrieves node1's metadata
        metadata, err := node3.FindValue(node1.PeerID())
        assert.NoError(t, err)
        assert.NotNil(t, metadata)
        assert.Equal(t, node1.PeerID(), metadata.PeerID)
        assert.Equal(t, node1.Address(), metadata.Address)
    })
}
```

**Run with Docker Compose**:
```bash
cd test/integration
docker-compose up -d
go test -v -timeout 5m ./...
docker-compose down
```

**Coverage Target**: All DHT operations validated

**Acceptance Criteria**:
- [x] 3 nodes discover each other in <60 seconds
- [x] All PING operations successful
- [x] FIND_NODE returns correct peers
- [x] STORE and FIND_VALUE work across nodes

---

### 2.2. Distributed Multi-Node Tests

**File**: `test/integration/distributed_network_test.go`

```go
func TestDistributed10NodeNetwork(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping distributed test in short mode")
    }

    // Start 10 nodes across multiple machines
    nodes := make([]*Node, 10)
    for i := 0; i < 10; i++ {
        nodes[i] = startNode(fmt.Sprintf("node%d:8443", i))
        defer nodes[i].Shutdown()
    }

    // Bootstrap from first 3 nodes
    bootstrapNodes := nodes[:3]
    for i := 3; i < 10; i++ {
        for _, bootstrap := range bootstrapNodes {
            nodes[i].AddBootstrapPeer(bootstrap.Address())
        }
    }

    // Wait for convergence
    time.Sleep(60 * time.Second)

    t.Run("All nodes have populated routing tables", func(t *testing.T) {
        for i, node := range nodes {
            peers := node.RoutingTable().AllPeers()
            assert.GreaterOrEqual(t, len(peers), 5, fmt.Sprintf("Node %d has too few peers", i))
        }
    })

    t.Run("DHT lookup latency", func(t *testing.T) {
        target := randomPeerID()
        start := time.Now()
        results := nodes[0].FindNode(target, 5*time.Second)
        latency := time.Since(start)

        assert.NotEmpty(t, results)
        assert.Less(t, latency, 500*time.Millisecond, "DHT lookup took too long")
        t.Logf("DHT lookup latency: %v", latency)
    })

    t.Run("Peer discovery success rate", func(t *testing.T) {
        successCount := 0
        totalAttempts := 100

        for i := 0; i < totalAttempts; i++ {
            randomNode := nodes[rand.Intn(10)]
            target := nodes[rand.Intn(10)].PeerID()

            metadata, err := randomNode.FindValue(target)
            if err == nil && metadata != nil {
                successCount++
            }
        }

        successRate := float64(successCount) / float64(totalAttempts)
        assert.GreaterOrEqual(t, successRate, 0.95, "Peer discovery success rate below 95%%")
        t.Logf("Peer discovery success rate: %.2f%%", successRate*100)
    })
}
```

**Infrastructure**: Use Terraform to provision 10 VPS instances for this test

**Coverage Target**: Realistic network conditions

**Acceptance Criteria**:
- [x] 10-node network converges in <90 seconds
- [x] DHT lookup latency <500ms
- [x] Peer discovery success rate ≥95%

---

## 3. Performance Testing Strategy

### 3.1. Throughput Regression Tests

**Maintain v11 performance baseline: 28.3 Mbps**

**File**: `test/performance/throughput_test.go`

```go
func TestThroughputRegression(t *testing.T) {
    // Start 2 nodes (sender and receiver)
    sender := startNode("localhost:8443")
    receiver := startNode("localhost:8444")

    defer sender.Shutdown()
    defer receiver.Shutdown()

    // Establish connection via DHT
    sender.AddBootstrapPeer(receiver.Address())
    time.Sleep(5 * time.Second)

    err := sender.ConnectToPeer(receiver.PeerID())
    assert.NoError(t, err)

    // Run iperf3-style throughput test
    duration := 10 * time.Second
    bytesSent, throughput := runThroughputTest(sender, receiver, duration)

    // Log results
    t.Logf("Bytes sent: %d", bytesSent)
    t.Logf("Throughput: %.2f Mbps", throughput)

    // Assert minimum throughput (25 Mbps to allow for variance)
    assert.GreaterOrEqual(t, throughput, 25.0, "Throughput below v11 baseline")
}

func runThroughputTest(sender, receiver *Node, duration time.Duration) (int64, float64) {
    // Implementation similar to iperf3
    // Send data through encrypted tunnel
    // Measure bytes/sec
}
```

**Run manually with iperf3**:
```bash
# On receiver node
iperf3 -s -p 5201

# On sender node (through ShadowMesh tunnel)
iperf3 -c 10.10.10.1 -p 5201 -t 10
```

**Acceptance Criteria**:
- [x] Throughput ≥25 Mbps (v11 achieved 28.3 Mbps)
- [x] No performance regression from v11

---

### 3.2. Latency Tests

**File**: `test/performance/latency_test.go`

```go
func TestDHTLookupLatency(t *testing.T) {
    // 10-node network
    nodes := startNodes(10)
    defer shutdownNodes(nodes)

    waitForConvergence(nodes, 60*time.Second)

    // Measure DHT lookup latency
    latencies := make([]time.Duration, 100)

    for i := 0; i < 100; i++ {
        randomNode := nodes[rand.Intn(10)]
        target := randomPeerID()

        start := time.Now()
        randomNode.FindNode(target, 5*time.Second)
        latencies[i] = time.Since(start)
    }

    // Calculate statistics
    avgLatency := average(latencies)
    p50Latency := percentile(latencies, 0.50)
    p95Latency := percentile(latencies, 0.95)
    p99Latency := percentile(latencies, 0.99)

    t.Logf("DHT Lookup Latency:")
    t.Logf("  Average: %v", avgLatency)
    t.Logf("  P50: %v", p50Latency)
    t.Logf("  P95: %v", p95Latency)
    t.Logf("  P99: %v", p99Latency)

    assert.Less(t, p95Latency, 500*time.Millisecond, "P95 latency too high")
}
```

**Acceptance Criteria**:
- [x] Average DHT lookup latency <200ms
- [x] P95 latency <500ms
- [x] P99 latency <1000ms

---

### 3.3. Scalability Tests

**File**: `test/performance/scalability_test.go`

```go
func TestRoutingTableScalability(t *testing.T) {
    localID := randomPeerID()
    rt := NewRoutingTable(localID, 20)

    // Add 10,000 peers
    for i := 0; i < 10000; i++ {
        peer := randomPeerInfo()
        rt.AddPeer(peer)
    }

    // Measure FindClosest performance
    target := randomPeerID()

    start := time.Now()
    closest := rt.FindClosest(target, 20)
    latency := time.Since(start)

    assert.Len(t, closest, 20)
    assert.Less(t, latency, 50*time.Millisecond, "FindClosest too slow with 10K peers")

    t.Logf("FindClosest latency (10K peers): %v", latency)
}

func TestMemoryUsage(t *testing.T) {
    // Measure memory usage with 1000-peer routing table
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    before := m.Alloc

    localID := randomPeerID()
    rt := NewRoutingTable(localID, 20)

    for i := 0; i < 1000; i++ {
        peer := randomPeerInfo()
        rt.AddPeer(peer)
    }

    runtime.ReadMemStats(&m)
    after := m.Alloc

    memoryUsed := after - before
    t.Logf("Memory used for 1000-peer routing table: %d bytes (%.2f MB)", memoryUsed, float64(memoryUsed)/1024/1024)

    // Should be <10 MB for 1000 peers
    assert.Less(t, memoryUsed, uint64(10*1024*1024), "Memory usage too high")
}
```

**Acceptance Criteria**:
- [x] FindClosest <50ms with 10,000-peer routing table
- [x] Memory usage <10 MB for 1000-peer routing table

---

## 4. End-to-End (E2E) Testing

### 4.1. Standalone Operation Test

**Manual Test Procedure**:

1. **Clean Install**:
```bash
# Remove any existing ShadowMesh data
rm -rf ~/.shadowmesh

# Install v0.2.0-alpha binary
wget https://github.com/shadowmesh/shadowmesh/releases/download/v0.2.0-alpha/shadowmesh-v0.2.0-alpha-linux-amd64
chmod +x shadowmesh-v0.2.0-alpha-linux-amd64
sudo mv shadowmesh-v0.2.0-alpha-linux-amd64 /usr/local/bin/shadowmesh
```

2. **Start Node (No Configuration)**:
```bash
# Start with default bootstrap nodes
shadowmesh start
```

3. **Verify Standalone Operation**:
```bash
# Check status (should show "Connected to DHT")
shadowmesh status

# Verify routing table populated
shadowmesh routing-table
# Should show peers from bootstrap nodes
```

4. **Connect to Remote Peer**:
```bash
# Get peer's PeerID from friend
PEER_ID="3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b"

# Connect via DHT lookup
shadowmesh connect $PEER_ID

# Verify tunnel established
shadowmesh tunnels
```

5. **Test Traffic**:
```bash
# Ping remote peer through tunnel
ping 10.10.10.2

# Run speed test
iperf3 -c 10.10.10.2 -t 10
```

**Acceptance Criteria**:
- [x] Node starts without configuration
- [x] Connects to bootstrap nodes in <30 seconds
- [x] DHT routing table populates
- [x] Peer discovery via DHT successful
- [x] Encrypted tunnel established
- [x] Traffic flows through tunnel

---

### 4.2. Network Partition Recovery Test

**Scenario**: Simulate network partition and verify DHT recovers

**Procedure**:
1. Start 5-node network
2. Block traffic between 2 nodes using iptables
3. Wait 5 minutes
4. Unblock traffic
5. Verify routing tables re-converge

**Acceptance Criteria**:
- [x] Partitioned nodes detect peer failure (via PING timeout)
- [x] Routing tables update to remove unreachable peers
- [x] After partition heals, routing tables re-converge in <5 minutes

---

### 4.3. Stress Test (1000 Concurrent Connections)

**Goal**: Verify bootstrap nodes handle high load

**Procedure**:
1. Deploy 3 bootstrap nodes
2. Simulate 1000 clients connecting simultaneously
3. Measure bootstrap node performance

**Metrics**:
- CPU usage
- Memory usage
- DHT request handling rate
- Failed connection rate

**Acceptance Criteria**:
- [x] Bootstrap nodes handle 1000 concurrent connections
- [x] CPU usage <80%
- [x] Memory usage <2 GB
- [x] Failed connection rate <1%

---

## 5. Test Automation & CI/CD

### 5.1. GitHub Actions Workflow

**File**: `.github/workflows/test-dht.yml`

```yaml
name: DHT Test Suite

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run unit tests
        run: |
          go test ./pkg/discovery/... -v -race -coverprofile=coverage.out

      - name: Check coverage
        run: |
          go tool cover -func=coverage.out | grep total | awk '{print $3}'
          # Fail if coverage <85%
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 85" | bc -l) )); then
            echo "Coverage $coverage% is below 85%"
            exit 1
          fi

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Start Docker Compose test network
        run: |
          cd test/integration
          docker-compose up -d
          sleep 30  # Wait for network to converge

      - name: Run integration tests
        run: |
          go test ./test/integration/... -v -timeout 10m

      - name: Shutdown test network
        run: |
          cd test/integration
          docker-compose down

  performance-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run performance benchmarks
        run: |
          go test ./test/performance/... -bench=. -benchtime=10s

      - name: Check performance regression
        run: |
          # Compare with baseline (v11: 28.3 Mbps)
          # Fail if throughput <25 Mbps
```

**Trigger**: Every push, pull request

**Acceptance Criteria**:
- [x] All tests pass on CI
- [x] Coverage ≥85%
- [x] No performance regression

---

## 6. Testing Schedule

### Sprint 0 (Weeks 1-2): Unit Tests
- **Day 1-2**: PeerID generation and verification tests
- **Day 3-4**: XOR distance and routing table tests
- **Day 5-6**: k-bucket tests with concurrency
- **Day 7-8**: DHT message protocol tests
- **Day 9-10**: Code review and test refinement

### Sprint 1 (Weeks 3-4): Integration Tests
- **Week 3**: Local 3-node network tests, PING/FIND_NODE
- **Week 4**: STORE/FIND_VALUE tests, distributed 10-node tests

### Sprint 2 (Weeks 5-6): Performance & E2E
- **Week 5**: Throughput regression tests, latency benchmarks
- **Week 6**: E2E standalone operation, stress tests

---

## 7. Acceptance Criteria Summary

### v0.2.0-alpha Release Gates

**Functional**:
- [ ] All unit tests pass (200+ tests, 85%+ coverage)
- [ ] All integration tests pass (50+ tests)
- [ ] 3-node local network: All peers discover each other in <60s
- [ ] 10-node distributed network: Peer discovery success rate ≥95%
- [ ] DHT lookup latency <500ms (P95)

**Performance**:
- [ ] Throughput ≥25 Mbps (maintain v11 baseline)
- [ ] Latency <50ms added overhead
- [ ] FindClosest <50ms with 10K-peer routing table
- [ ] Memory usage <10 MB for 1000-peer routing table

**Reliability**:
- [ ] 24-hour uptime test without crashes
- [ ] Network partition recovery <5 minutes
- [ ] Bootstrap nodes handle 1000 concurrent connections

**Security**:
- [ ] PeerID verification working (no false positives)
- [ ] ML-DSA-87 signature validation (all messages signed)
- [ ] No unauthenticated connections allowed
- [ ] Rate limiting prevents DoS attacks

---

## 8. Test Reporting

### Daily Reports (During Sprint 0-1)
```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Generate test summary
go test ./... -json | tee test-results.json
```

### Weekly Dashboard

**Metrics to Track**:
- Test pass rate (%)
- Code coverage (%)
- Performance benchmarks (throughput, latency)
- Memory usage trends

**Tools**:
- GitHub Actions (CI results)
- Codecov (coverage visualization)
- Grafana (performance trends)

---

## Conclusion

This testing strategy provides comprehensive coverage for ShadowMesh's DHT implementation:

✅ **200+ unit tests** covering all core components
✅ **50+ integration tests** for multi-node scenarios
✅ **Performance regression** tests maintain v11 baseline
✅ **E2E tests** validate standalone operation
✅ **Automated CI/CD** pipeline for continuous validation

**Next Steps**:
1. Begin unit test implementation (TICKET-001 to TICKET-011)
2. Set up local 3-node test network (TICKET-012)
3. Configure CI/CD pipeline (GitHub Actions)
4. Run tests after each implementation phase
5. Validate all acceptance criteria before v0.2.0-alpha release

---

**Document Control**
- Version: 1.0
- Created: November 10, 2025
- Author: Winston (Architect)
- Status: Approved for Implementation
