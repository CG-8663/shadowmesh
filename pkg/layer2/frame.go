package layer2

import (
	"encoding/binary"
	"fmt"
)

// EthernetFrame represents a parsed Layer 2 Ethernet frame
type EthernetFrame struct {
	DestinationMAC [6]byte // Bytes 0-5: Destination MAC address
	SourceMAC      [6]byte // Bytes 6-11: Source MAC address
	EtherType      uint16  // Bytes 12-13: EtherType (network byte order)
	Payload        []byte  // Bytes 14-end: Frame payload
}

// Common EtherType values
const (
	EtherTypeIPv4 = 0x0800 // Internet Protocol version 4 (IPv4)
	EtherTypeARP  = 0x0806 // Address Resolution Protocol (ARP)
	EtherTypeIPv6 = 0x86DD // Internet Protocol version 6 (IPv6)
)

// Frame size constraints
const (
	EthernetHeaderSize = 14   // Minimum Ethernet header size (6 + 6 + 2 bytes)
	MinFrameSize       = 14   // Minimum valid frame size (header only)
	MaxFrameSize       = 1514 // Maximum frame size (1500 MTU + 14 header)
)

// ParseFrame parses raw Ethernet frame data into an EthernetFrame struct
//
// The frame must be at least 14 bytes (Ethernet header) and at most 1514 bytes
// (1500 byte MTU + 14 byte header). Returns an error if the frame is malformed.
//
// Ethernet frame structure:
//   - Bytes 0-5:   Destination MAC address
//   - Bytes 6-11:  Source MAC address
//   - Bytes 12-13: EtherType (big-endian/network byte order)
//   - Bytes 14+:   Payload
func ParseFrame(data []byte) (*EthernetFrame, error) {
	// Validate minimum frame size
	if len(data) < MinFrameSize {
		return nil, fmt.Errorf("frame too small: got %d bytes, minimum %d bytes required", len(data), MinFrameSize)
	}

	// Validate maximum frame size
	if len(data) > MaxFrameSize {
		return nil, fmt.Errorf("frame too large: got %d bytes, maximum %d bytes allowed", len(data), MaxFrameSize)
	}

	frame := &EthernetFrame{}

	// Extract destination MAC (bytes 0-5)
	copy(frame.DestinationMAC[:], data[0:6])

	// Extract source MAC (bytes 6-11)
	copy(frame.SourceMAC[:], data[6:12])

	// Extract EtherType (bytes 12-13, big-endian)
	frame.EtherType = binary.BigEndian.Uint16(data[12:14])

	// Extract payload (bytes 14 to end)
	if len(data) > EthernetHeaderSize {
		frame.Payload = make([]byte, len(data)-EthernetHeaderSize)
		copy(frame.Payload, data[EthernetHeaderSize:])
	}

	return frame, nil
}

// Serialize converts the EthernetFrame back to raw bytes
//
// Returns a byte slice containing the full Ethernet frame:
//   - Bytes 0-5:   Destination MAC
//   - Bytes 6-11:  Source MAC
//   - Bytes 12-13: EtherType (big-endian)
//   - Bytes 14+:   Payload
func (f *EthernetFrame) Serialize() []byte {
	size := EthernetHeaderSize + len(f.Payload)
	data := make([]byte, size)

	// Copy destination MAC (bytes 0-5)
	copy(data[0:6], f.DestinationMAC[:])

	// Copy source MAC (bytes 6-11)
	copy(data[6:12], f.SourceMAC[:])

	// Write EtherType (bytes 12-13, big-endian)
	binary.BigEndian.PutUint16(data[12:14], f.EtherType)

	// Copy payload (bytes 14+)
	if len(f.Payload) > 0 {
		copy(data[EthernetHeaderSize:], f.Payload)
	}

	return data
}

// String returns a human-readable representation of the Ethernet frame
func (f *EthernetFrame) String() string {
	etherTypeStr := fmt.Sprintf("0x%04X", f.EtherType)
	switch f.EtherType {
	case EtherTypeIPv4:
		etherTypeStr = "IPv4"
	case EtherTypeARP:
		etherTypeStr = "ARP"
	case EtherTypeIPv6:
		etherTypeStr = "IPv6"
	}

	return fmt.Sprintf("Frame[dst=%02x:%02x:%02x:%02x:%02x:%02x, src=%02x:%02x:%02x:%02x:%02x:%02x, type=%s, payload=%d bytes]",
		f.DestinationMAC[0], f.DestinationMAC[1], f.DestinationMAC[2],
		f.DestinationMAC[3], f.DestinationMAC[4], f.DestinationMAC[5],
		f.SourceMAC[0], f.SourceMAC[1], f.SourceMAC[2],
		f.SourceMAC[3], f.SourceMAC[4], f.SourceMAC[5],
		etherTypeStr, len(f.Payload))
}
