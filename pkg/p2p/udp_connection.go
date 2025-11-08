package p2p

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/crypto"
)

// calculateIPv4Checksum computes the IPv4 header checksum
func calculateIPv4Checksum(header []byte) uint16 {
	// IP header length must be at least 20 bytes
	if len(header) < 20 {
		return 0
	}

	// Clear existing checksum (bytes 10-11)
	header[10] = 0
	header[11] = 0

	// Calculate checksum over the IP header
	var sum uint32
	for i := 0; i < len(header); i += 2 {
		// Read 16-bit word in big-endian
		sum += uint32(header[i])<<8 | uint32(header[i+1])
	}

	// Add carry bits
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// One's complement
	return ^uint16(sum)
}

// UDPConnection handles UDP data path for Layer 3 frame forwarding with encryption
type UDPConnection struct {
	conn           *net.UDPConn
	remoteAddr     *net.UDPAddr
	peerID         string
	isActive       bool
	mu             sync.RWMutex
	sendCount      uint64
	recvCount      uint64
	sequenceNum    uint64        // Frame sequence number for loss detection
	lastRecvSeq    uint64        // Last received sequence number
	frameHandler   func([]byte)
	cipher         *crypto.ChaCha20Cipher // Encryption cipher for frames
	// RTT measurement
	rttSamples     []time.Duration
	rttMu          sync.RWMutex
	lastRTT        time.Duration
	avgRTT         time.Duration
	echoRequestSeq uint64
}

// Frame types for UDP protocol
const (
	FrameTypeData        uint8 = 0x00 // Regular data frame
	FrameTypeEchoRequest uint8 = 0x01 // Echo request for RTT measurement
	FrameTypeEchoReply   uint8 = 0x02 // Echo reply for RTT measurement
)

// UDPFrame represents a single UDP frame with header
// Header format: [8 bytes seq][1 byte type][8 bytes timestamp][2 bytes size][N bytes payload]
type UDPFrame struct {
	SequenceNum uint64 // 8 bytes
	FrameType   uint8  // 1 byte (data, echo request, echo reply)
	Timestamp   int64  // 8 bytes (nanoseconds since epoch)
	FrameSize   uint16 // 2 bytes
	Frame       []byte // Variable length payload
}

// NewUDPConnection creates a new UDP connection for data path
func NewUDPConnection(localPort int, peerID string) (*UDPConnection, error) {
	addr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: localPort,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}

	// Set socket buffer sizes (128MB for burst tolerance - Phase 3 Fix)
	// Adaptive buffers can send 2604 packets × 1500 bytes = ~3.9MB bursts
	// 128MB provides 32× headroom for multiple concurrent peers
	if err := conn.SetReadBuffer(128 * 1024 * 1024); err != nil {
		log.Printf("[WARN] Failed to set UDP read buffer to 128MB: %v (will use system default)", err)
	} else {
		log.Printf("[UDP-INIT] UDP receive buffer set to 128MB")
	}

	if err := conn.SetWriteBuffer(128 * 1024 * 1024); err != nil {
		log.Printf("[WARN] Failed to set UDP write buffer to 128MB: %v (will use system default)", err)
	} else {
		log.Printf("[UDP-INIT] UDP send buffer set to 128MB")
	}

	return &UDPConnection{
		conn:     conn,
		peerID:   peerID,
		isActive: true,
	}, nil
}

// SetRemoteAddr sets the remote peer's UDP endpoint
func (u *UDPConnection) SetRemoteAddr(ip string, port int) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	u.mu.Lock()
	u.remoteAddr = addr
	u.mu.Unlock()

	log.Printf("UDP data path established to %s:%d", ip, port)
	return nil
}

// SendFrame sends a Layer 3 IP packet over UDP with sequencing
// Encrypts only the payload, preserving IP header for kernel validation
func (u *UDPConnection) SendFrame(frame []byte) error {
	startTotal := time.Now()

	u.mu.RLock()
	if !u.isActive {
		u.mu.RUnlock()
		return fmt.Errorf("UDP connection closed")
	}
	remoteAddr := u.remoteAddr
	u.mu.RUnlock()

	if remoteAddr == nil {
		return fmt.Errorf("remote address not set")
	}

	// Get sequence number (atomic increment)
	seq := atomic.AddUint64(&u.sequenceNum, 1)

	// Build UDP frame with header
	// [8 bytes seq][1 byte type][8 bytes timestamp][2 bytes size][1 byte ipHeaderLen][N bytes IP header][M bytes encrypted payload]
	startBuild := time.Now()

	// Parse IP header to encrypt only payload
	var frameData []byte
	u.mu.RLock()
	cipher := u.cipher
	u.mu.RUnlock()

	if cipher != nil && len(frame) >= 20 {
		// Parse IP header length (IPv4: first byte bits 0-3 * 4)
		version := frame[0] >> 4
		if version == 4 {
			// IPv4: IHL is in lower 4 bits of first byte
			ipHeaderLen := int(frame[0]&0x0F) * 4
			if ipHeaderLen >= 20 && ipHeaderLen <= len(frame) {
				// Split: IP header + payload
				ipHeader := frame[:ipHeaderLen]
				payload := frame[ipHeaderLen:]

				// Encrypt only the payload
				encryptedPayload, err := cipher.Encrypt(payload)
				if err != nil {
					return fmt.Errorf("encryption failed: %w", err)
				}

				log.Printf("[CRYPTO-TX] Encrypted payload: ipHeaderLen=%d, payloadSize=%d→%d bytes (overhead=%d)",
					ipHeaderLen, len(payload), len(encryptedPayload), len(encryptedPayload)-len(payload))

				// Log IP addresses from header (bytes 12-19: src IP at 12-15, dst IP at 16-19)
				if ipHeaderLen >= 20 {
					log.Printf("[IP-TX] Header bytes 0-19: %02x (src=%d.%d.%d.%d, dst=%d.%d.%d.%d)",
						ipHeader[:20], ipHeader[12], ipHeader[13], ipHeader[14], ipHeader[15],
						ipHeader[16], ipHeader[17], ipHeader[18], ipHeader[19])
				}

				// Build: [1 byte header length][IP header][encrypted payload]
				frameData = make([]byte, 1+ipHeaderLen+len(encryptedPayload))
				frameData[0] = byte(ipHeaderLen)
				copy(frameData[1:], ipHeader)
				copy(frameData[1+ipHeaderLen:], encryptedPayload)
			} else {
				// Invalid header length, send unencrypted
				log.Printf("[CRYPTO-WARN] Invalid IP header length %d, sending unencrypted", ipHeaderLen)
				frameData = frame
			}
		} else {
			// IPv6 or unknown version, send unencrypted for now
			log.Printf("[CRYPTO-WARN] Non-IPv4 packet (version %d), sending unencrypted", version)
			frameData = frame
		}
	} else {
		// No cipher or packet too small
		frameData = frame
	}

	frameSize := uint16(len(frameData))

	// Solution 3 (Phase 3): Use stack-allocated buffer for standard frames
	// Header (19 bytes) + MTU (1500 bytes) + Encryption overhead (28 bytes) = 1547 bytes
	const maxStackSize = 1600
	var packet []byte

	if len(frameData) <= (maxStackSize - 19) {
		// Stack allocation - zero heap allocation for standard-sized frames
		var stackBuf [maxStackSize]byte
		packet = stackBuf[:19+len(frameData)]
	} else {
		// Fallback to heap for oversized frames (rare)
		packet = make([]byte, 19+len(frameData))
		log.Printf("[UDP-SEND-WARNING] Oversized frame: %d bytes (exceeds buffer), using heap allocation", len(frameData))
	}

	binary.BigEndian.PutUint64(packet[0:8], seq)
	packet[8] = FrameTypeData                                           // Frame type
	binary.BigEndian.PutUint64(packet[9:17], uint64(time.Now().UnixNano())) // Timestamp
	binary.BigEndian.PutUint16(packet[17:19], frameSize)
	copy(packet[19:], frameData)
	buildDuration := time.Since(startBuild)

	// UDP write
	startWrite := time.Now()
	_, err := u.conn.WriteToUDP(packet, remoteAddr)
	if err != nil {
		return fmt.Errorf("UDP write failed: %w", err)
	}
	writeDuration := time.Since(startWrite)
	totalDuration := time.Since(startTotal)

	// Log timing every 100th frame
	count := atomic.AddUint64(&u.sendCount, 1)
	if count%100 == 0 {
		log.Printf("[PROFILE-UDP-SEND-%s] Total=%v Build=%v UDPWrite=%v Seq=%d FrameSize=%d",
			u.peerID, totalDuration, buildDuration, writeDuration, seq, frameSize)
	}

	return nil
}

// SetCipher sets the encryption cipher for this connection
func (u *UDPConnection) SetCipher(cipher *crypto.ChaCha20Cipher) {
	u.mu.Lock()
	u.cipher = cipher
	u.mu.Unlock()
	log.Printf("[UDP-CRYPTO] Encryption enabled for peer %s", u.peerID)
}

// SetFrameHandler sets the callback for received frames
func (u *UDPConnection) SetFrameHandler(handler func([]byte)) {
	u.mu.Lock()
	u.frameHandler = handler
	u.mu.Unlock()
}

// StartReceiving starts the UDP receive loop
func (u *UDPConnection) StartReceiving() {
	go u.receiveLoop()
}

// receiveLoop continuously receives UDP frames
func (u *UDPConnection) receiveLoop() {
	buffer := make([]byte, 65535) // Max UDP packet size

	for {
		u.mu.RLock()
		if !u.isActive {
			u.mu.RUnlock()
			break
		}
		u.mu.RUnlock()

		startTotal := time.Now()

		// UDP read
		startRead := time.Now()
		n, remoteAddr, err := u.conn.ReadFromUDP(buffer)
		if err != nil {
			if u.IsActive() {
				log.Printf("UDP read error: %v", err)
			}
			continue
		}
		readDuration := time.Since(startRead)

		// Update remote address if not set (for listening side)
		u.mu.Lock()
		if u.remoteAddr == nil {
			u.remoteAddr = remoteAddr
			log.Printf("UDP remote address learned: %s", remoteAddr)
		}
		u.mu.Unlock()

		// Parse frame header (19 bytes: [8 seq][1 type][8 timestamp][2 size][N payload])
		startParse := time.Now()
		if n < 19 {
			log.Printf("Received undersized UDP packet: %d bytes", n)
			continue
		}

		seq := binary.BigEndian.Uint64(buffer[0:8])
		frameType := buffer[8]
		timestamp := int64(binary.BigEndian.Uint64(buffer[9:17]))
		frameSize := binary.BigEndian.Uint16(buffer[17:19])

		if int(frameSize)+19 != n {
			log.Printf("Frame size mismatch: expected %d, got %d", int(frameSize)+19, n)
			continue
		}

		parseDuration := time.Since(startParse)

		// Handle different frame types
		switch frameType {
		case FrameTypeEchoRequest:
			// Echo request: send echo reply with original timestamp
			u.sendEchoReply(seq, timestamp)
			continue

		case FrameTypeEchoReply:
			// Echo reply: calculate RTT
			rtt := time.Since(time.Unix(0, timestamp))
			u.rttMu.Lock()
			u.lastRTT = rtt
			u.rttSamples = append(u.rttSamples, rtt)
			if len(u.rttSamples) > 10 {
				u.rttSamples = u.rttSamples[1:] // Keep last 10 samples
			}
			// Calculate average RTT
			var totalRTT time.Duration
			for _, sample := range u.rttSamples {
				totalRTT += sample
			}
			u.avgRTT = totalRTT / time.Duration(len(u.rttSamples))
			u.rttMu.Unlock()
			log.Printf("[RTT] Peer %s: %v (avg %v)", u.peerID, rtt, u.avgRTT)
			continue

		case FrameTypeData:
			// Regular data frame - extract payload and decrypt if needed
			encrypted := buffer[19:19+frameSize]
			var frame []byte

			u.mu.RLock()
			cipher := u.cipher
			u.mu.RUnlock()

			if cipher != nil && len(encrypted) >= 1 {
				// New format: [1 byte ipHeaderLen][IP header][encrypted payload]
				ipHeaderLen := int(encrypted[0])

				if ipHeaderLen >= 20 && ipHeaderLen < len(encrypted) {
					// Extract IP header and encrypted payload
					ipHeader := encrypted[1:1+ipHeaderLen]
					encryptedPayload := encrypted[1+ipHeaderLen:]

					// Decrypt only the payload
					decryptedPayload, err := cipher.Decrypt(encryptedPayload)
					if err != nil {
						log.Printf("[CRYPTO-ERROR] Payload decryption failed from %s (seq %d): %v", u.peerID, seq, err)
						continue
					}

					log.Printf("[CRYPTO-RX] Decrypted payload: ipHeaderLen=%d, encryptedSize=%d→%d bytes (seq %d)",
						ipHeaderLen, len(encryptedPayload), len(decryptedPayload), seq)

					// Reconstruct: [IP header][decrypted payload]
					frame = make([]byte, ipHeaderLen+len(decryptedPayload))
					copy(frame, ipHeader)
					copy(frame[ipHeaderLen:], decryptedPayload)

					// Log IP addresses from reconstructed frame
					if len(frame) >= 20 {
						log.Printf("[IP-RX] Reconstructed frame bytes 0-19: %02x (src=%d.%d.%d.%d, dst=%d.%d.%d.%d)",
							frame[:20], frame[12], frame[13], frame[14], frame[15],
							frame[16], frame[17], frame[18], frame[19])
					}

					// Recalculate IP header checksum
					if len(frame) >= ipHeaderLen && ipHeaderLen >= 20 {
					// Update Total Length (bytes 2-3) to reflect actual packet size
					totalLength := uint16(len(frame))
					binary.BigEndian.PutUint16(frame[2:4], totalLength)
					log.Printf("[IP-RX] Updated Total Length: %d bytes", totalLength)

					// Recalculate checksum over updated header
					checksum := calculateIPv4Checksum(frame[:ipHeaderLen])
					binary.BigEndian.PutUint16(frame[10:12], checksum)
					log.Printf("[IP-RX] Recalculated IP checksum: 0x%04x", checksum)
					}
				} else {
					// Old format or invalid header length - try full decryption (backward compat)
					decrypted, err := cipher.Decrypt(encrypted)
					if err != nil {
						log.Printf("[CRYPTO-ERROR] Decryption failed from %s (seq %d): %v", u.peerID, seq, err)
						continue
					}
					frame = decrypted
				}
			} else {
				// Unencrypted (backward compat)
				frame = make([]byte, frameSize)
				copy(frame, encrypted)
			}

			// Detect packet loss
			lastSeq := atomic.LoadUint64(&u.lastRecvSeq)
			if lastSeq > 0 && seq > lastSeq+1 {
				lost := seq - lastSeq - 1
				log.Printf("Detected %d lost frames (last=%d, current=%d)", lost, lastSeq, seq)
			}
			atomic.StoreUint64(&u.lastRecvSeq, seq)

			// Call frame handler
			startHandler := time.Now()
			u.mu.RLock()
			handler := u.frameHandler
			u.mu.RUnlock()

			if handler != nil {
				handler(frame)
			}
			handlerDuration := time.Since(startHandler)

			totalDuration := time.Since(startTotal)

			// Log timing every 100th frame
			count := atomic.AddUint64(&u.recvCount, 1)
			if count%100 == 0 {
				log.Printf("[PROFILE-UDP-RECV-%s] Total=%v UDPRead=%v Parse=%v Handler=%v Seq=%d FrameSize=%d",
					u.peerID, totalDuration, readDuration, parseDuration, handlerDuration, seq, frameSize)
			}

		default:
			log.Printf("Unknown frame type: %d", frameType)
		}
	}
}

// GetStats returns connection statistics
func (u *UDPConnection) GetStats() (sent uint64, recv uint64, lastSeq uint64) {
	return atomic.LoadUint64(&u.sendCount),
		atomic.LoadUint64(&u.recvCount),
		atomic.LoadUint64(&u.lastRecvSeq)
}

// Close closes the UDP connection
func (u *UDPConnection) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.isActive = false
	return u.conn.Close()
}

// IsActive returns whether the connection is active
func (u *UDPConnection) IsActive() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.isActive
}

// GetPeerID returns the peer ID
func (u *UDPConnection) GetPeerID() string {
	return u.peerID
}

// GetLocalPort returns the local UDP port
func (u *UDPConnection) GetLocalPort() int {
	if u.conn == nil {
		return 0
	}
	return u.conn.LocalAddr().(*net.UDPAddr).Port
}

// sendEchoReply sends an echo reply frame with the original timestamp
func (u *UDPConnection) sendEchoReply(seq uint64, originalTimestamp int64) error {
	u.mu.RLock()
	if !u.isActive {
		u.mu.RUnlock()
		return fmt.Errorf("UDP connection closed")
	}
	remoteAddr := u.remoteAddr
	u.mu.RUnlock()

	if remoteAddr == nil {
		return fmt.Errorf("remote address not set")
	}

	// Build echo reply packet with original timestamp
	// [8 bytes seq][1 byte type][8 bytes timestamp][2 bytes size=0][no payload]
	packet := make([]byte, 19)
	binary.BigEndian.PutUint64(packet[0:8], seq)
	packet[8] = FrameTypeEchoReply
	binary.BigEndian.PutUint64(packet[9:17], uint64(originalTimestamp))
	binary.BigEndian.PutUint16(packet[17:19], 0) // No payload

	_, err := u.conn.WriteToUDP(packet, remoteAddr)
	return err
}

// SendEchoRequest sends an echo request to measure RTT
func (u *UDPConnection) SendEchoRequest() error {
	u.mu.RLock()
	if !u.isActive {
		u.mu.RUnlock()
		return fmt.Errorf("UDP connection closed")
	}
	remoteAddr := u.remoteAddr
	u.mu.RUnlock()

	if remoteAddr == nil {
		return fmt.Errorf("remote address not set")
	}

	// Get sequence number
	seq := atomic.AddUint64(&u.echoRequestSeq, 1)

	// Build echo request packet with current timestamp
	// [8 bytes seq][1 byte type][8 bytes timestamp][2 bytes size=0][no payload]
	packet := make([]byte, 19)
	binary.BigEndian.PutUint64(packet[0:8], seq)
	packet[8] = FrameTypeEchoRequest
	binary.BigEndian.PutUint64(packet[9:17], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint16(packet[17:19], 0) // No payload

	_, err := u.conn.WriteToUDP(packet, remoteAddr)
	return err
}

// GetRTT returns the last measured RTT and average RTT
func (u *UDPConnection) GetRTT() (last time.Duration, avg time.Duration) {
	u.rttMu.RLock()
	defer u.rttMu.RUnlock()
	return u.lastRTT, u.avgRTT
}
