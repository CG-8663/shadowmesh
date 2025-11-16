package layer2

import (
	"bytes"
	"testing"
)

// TestParseFrameIPv4 tests parsing a valid IPv4 Ethernet frame
func TestParseFrameIPv4(t *testing.T) {
	// Valid IPv4 frame: dst MAC, src MAC, EtherType 0x0800, payload
	data := []byte{
		0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, // Destination MAC
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66, // Source MAC
		0x08, 0x00, // EtherType: IPv4
		0x45, 0x00, 0x00, 0x3C, // IPv4 payload start
	}

	frame, err := ParseFrame(data)
	if err != nil {
		t.Fatalf("ParseFrame() failed: %v", err)
	}

	// Verify destination MAC
	expectedDst := [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if frame.DestinationMAC != expectedDst {
		t.Errorf("DestinationMAC = %v, want %v", frame.DestinationMAC, expectedDst)
	}

	// Verify source MAC
	expectedSrc := [6]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}
	if frame.SourceMAC != expectedSrc {
		t.Errorf("SourceMAC = %v, want %v", frame.SourceMAC, expectedSrc)
	}

	// Verify EtherType
	if frame.EtherType != EtherTypeIPv4 {
		t.Errorf("EtherType = 0x%04X, want 0x%04X (IPv4)", frame.EtherType, EtherTypeIPv4)
	}

	// Verify payload
	expectedPayload := []byte{0x45, 0x00, 0x00, 0x3C}
	if !bytes.Equal(frame.Payload, expectedPayload) {
		t.Errorf("Payload = %v, want %v", frame.Payload, expectedPayload)
	}
}

// TestParseFrameARP tests parsing a valid ARP Ethernet frame
func TestParseFrameARP(t *testing.T) {
	// Valid ARP frame
	data := []byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // Broadcast destination
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Source MAC
		0x08, 0x06, // EtherType: ARP
		0x00, 0x01, 0x08, 0x00, // ARP payload
	}

	frame, err := ParseFrame(data)
	if err != nil {
		t.Fatalf("ParseFrame() failed: %v", err)
	}

	if frame.EtherType != EtherTypeARP {
		t.Errorf("EtherType = 0x%04X, want 0x%04X (ARP)", frame.EtherType, EtherTypeARP)
	}

	expectedPayload := []byte{0x00, 0x01, 0x08, 0x00}
	if !bytes.Equal(frame.Payload, expectedPayload) {
		t.Errorf("Payload = %v, want %v", frame.Payload, expectedPayload)
	}
}

// TestParseFrameIPv6 tests parsing a valid IPv6 Ethernet frame
func TestParseFrameIPv6(t *testing.T) {
	// Valid IPv6 frame
	data := []byte{
		0x33, 0x33, 0x00, 0x00, 0x00, 0x01, // IPv6 multicast MAC
		0xFE, 0x80, 0x00, 0x00, 0x00, 0x01, // Source MAC
		0x86, 0xDD, // EtherType: IPv6
		0x60, 0x00, 0x00, 0x00, // IPv6 payload
	}

	frame, err := ParseFrame(data)
	if err != nil {
		t.Fatalf("ParseFrame() failed: %v", err)
	}

	if frame.EtherType != EtherTypeIPv6 {
		t.Errorf("EtherType = 0x%04X, want 0x%04X (IPv6)", frame.EtherType, EtherTypeIPv6)
	}
}

// TestParseFrameCustomProtocol tests parsing a frame with custom EtherType
func TestParseFrameCustomProtocol(t *testing.T) {
	// Custom protocol frame (e.g., 0x88CC for LLDP)
	data := []byte{
		0x01, 0x80, 0xC2, 0x00, 0x00, 0x0E, // Destination MAC
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Source MAC
		0x88, 0xCC, // EtherType: LLDP (custom protocol)
		0x01, 0x02, 0x03, 0x04, // Custom payload
	}

	frame, err := ParseFrame(data)
	if err != nil {
		t.Fatalf("ParseFrame() failed: %v", err)
	}

	expectedEtherType := uint16(0x88CC)
	if frame.EtherType != expectedEtherType {
		t.Errorf("EtherType = 0x%04X, want 0x%04X", frame.EtherType, expectedEtherType)
	}

	expectedPayload := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(frame.Payload, expectedPayload) {
		t.Errorf("Payload = %v, want %v", frame.Payload, expectedPayload)
	}
}

// TestParseFrameMinimumSize tests parsing a minimum-sized frame (header only)
func TestParseFrameMinimumSize(t *testing.T) {
	// Minimum frame: 14 bytes (header only, no payload)
	data := []byte{
		0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, // Destination MAC
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66, // Source MAC
		0x08, 0x00, // EtherType: IPv4
	}

	frame, err := ParseFrame(data)
	if err != nil {
		t.Fatalf("ParseFrame() failed: %v", err)
	}

	if len(frame.Payload) != 0 {
		t.Errorf("Payload length = %d, want 0 (no payload for minimum frame)", len(frame.Payload))
	}
}

// TestParseFrameTooSmall tests that frames smaller than 14 bytes are rejected
func TestParseFrameTooSmall(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"empty frame", []byte{}},
		{"1 byte", []byte{0x01}},
		{"13 bytes (incomplete header)", []byte{
			0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF,
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66,
			0x08, // Missing second byte of EtherType
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseFrame(tc.data)
			if err == nil {
				t.Errorf("ParseFrame() succeeded, want error for %d-byte frame", len(tc.data))
			}
		})
	}
}

// TestParseFrameTooLarge tests that frames larger than 1514 bytes are rejected
func TestParseFrameTooLarge(t *testing.T) {
	// Create a frame that's 1515 bytes (1 byte over maximum)
	data := make([]byte, MaxFrameSize+1)
	// Fill with valid header
	copy(data[0:6], []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})  // Dst MAC
	copy(data[6:12], []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}) // Src MAC
	copy(data[12:14], []byte{0x08, 0x00})                        // EtherType
	// Remaining bytes are payload (too large)

	_, err := ParseFrame(data)
	if err == nil {
		t.Errorf("ParseFrame() succeeded, want error for %d-byte frame (max: %d)", len(data), MaxFrameSize)
	}
}

// TestParseFrameMaximumSize tests parsing a maximum-sized valid frame
func TestParseFrameMaximumSize(t *testing.T) {
	// Create a frame that's exactly 1514 bytes (maximum allowed)
	data := make([]byte, MaxFrameSize)
	// Fill with valid header
	copy(data[0:6], []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})  // Dst MAC
	copy(data[6:12], []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}) // Src MAC
	copy(data[12:14], []byte{0x08, 0x00})                        // EtherType: IPv4
	// Remaining 1500 bytes are payload (maximum MTU)

	frame, err := ParseFrame(data)
	if err != nil {
		t.Fatalf("ParseFrame() failed for max-size frame: %v", err)
	}

	// Verify payload is 1500 bytes (MTU)
	expectedPayloadSize := 1500
	if len(frame.Payload) != expectedPayloadSize {
		t.Errorf("Payload length = %d, want %d", len(frame.Payload), expectedPayloadSize)
	}
}

// TestEthernetFrameString tests the String() method for human-readable output
func TestEthernetFrameString(t *testing.T) {
	data := []byte{
		0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF,
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66,
		0x08, 0x00,
		0x45, 0x00,
	}

	frame, err := ParseFrame(data)
	if err != nil {
		t.Fatalf("ParseFrame() failed: %v", err)
	}

	str := frame.String()
	// Should contain MAC addresses and EtherType
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Should mention IPv4 for EtherType 0x0800
	if !bytes.Contains([]byte(str), []byte("IPv4")) {
		t.Errorf("String() = %q, want IPv4 mentioned for EtherType 0x0800", str)
	}
}

// TestSerializeRoundTrip tests that parsing and serializing yields the same data
func TestSerializeRoundTrip(t *testing.T) {
	// Original frame data
	original := []byte{
		0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, // Destination MAC
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66, // Source MAC
		0x08, 0x00, // EtherType: IPv4
		0x45, 0x00, 0x00, 0x3C, 0x1C, 0x46, 0x40, 0x00, // IPv4 payload
		0x40, 0x06, 0xB1, 0xE6, 0xAC, 0x10, 0x0A, 0x63,
		0xAC, 0x10, 0x0A, 0x0C,
	}

	// Parse the frame
	frame, err := ParseFrame(original)
	if err != nil {
		t.Fatalf("ParseFrame() failed: %v", err)
	}

	// Serialize it back
	serialized := frame.Serialize()

	// Should be identical to original
	if !bytes.Equal(serialized, original) {
		t.Errorf("Serialize() round-trip failed:\noriginal:   %v\nserialized: %v", original, serialized)
	}
}

// BenchmarkParseFrame benchmarks frame parsing performance
func BenchmarkParseFrame(b *testing.B) {
	// Typical IPv4 frame
	data := []byte{
		0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF,
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66,
		0x08, 0x00,
		0x45, 0x00, 0x00, 0x3C, 0x1C, 0x46, 0x40, 0x00,
		0x40, 0x06, 0xB1, 0xE6, 0xAC, 0x10, 0x0A, 0x63,
		0xAC, 0x10, 0x0A, 0x0C,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseFrame(data)
		if err != nil {
			b.Fatalf("ParseFrame() failed: %v", err)
		}
	}
}

// BenchmarkParseFrameMaxSize benchmarks parsing maximum-sized frames
func BenchmarkParseFrameMaxSize(b *testing.B) {
	data := make([]byte, MaxFrameSize)
	copy(data[0:6], []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})
	copy(data[6:12], []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66})
	copy(data[12:14], []byte{0x08, 0x00})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseFrame(data)
		if err != nil {
			b.Fatalf("ParseFrame() failed: %v", err)
		}
	}
}
