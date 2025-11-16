package nat

import (
	"fmt"
	"net"
	"time"
)

// STUNClient handles STUN requests to discover public IP:port
type STUNClient struct {
	servers []string
}

// NewSTUNClient creates a new STUN client
func NewSTUNClient() *STUNClient {
	return &STUNClient{
		servers: []string{
			"stun.l.google.com:19302",
			"stun1.l.google.com:19302",
			"stun2.l.google.com:19302",
		},
	}
}

// DiscoverPublicAddress discovers the public IP and port using STUN
func (s *STUNClient) DiscoverPublicAddress(localPort int) (*net.UDPAddr, error) {
	// Create UDP socket on specified port
	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve local address: %w", err)
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}
	defer conn.Close()

	// Try each STUN server
	for _, server := range s.servers {
		addr, err := s.queryServer(conn, server)
		if err == nil {
			return addr, nil
		}
	}

	return nil, fmt.Errorf("all STUN servers failed")
}

// queryServer sends STUN binding request to a server
func (s *STUNClient) queryServer(conn *net.UDPConn, server string) (*net.UDPAddr, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, err
	}

	// STUN Binding Request
	// Message Type: Binding Request (0x0001)
	// Magic Cookie: 0x2112A442
	// Transaction ID: 96 random bits
	txID := make([]byte, 12)
	for i := range txID {
		txID[i] = byte(time.Now().UnixNano())
	}

	request := make([]byte, 20)
	// Message Type: Binding Request
	request[0] = 0x00
	request[1] = 0x01
	// Message Length: 0
	request[2] = 0x00
	request[3] = 0x00
	// Magic Cookie
	request[4] = 0x21
	request[5] = 0x12
	request[6] = 0xA4
	request[7] = 0x42
	// Transaction ID
	copy(request[8:], txID)

	// Send request
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, err = conn.WriteToUDP(request, serverAddr)
	if err != nil {
		return nil, err
	}

	// Receive response
	response := make([]byte, 1500)
	n, _, err := conn.ReadFromUDP(response)
	if err != nil {
		return nil, err
	}

	// Parse STUN response
	return s.parseResponse(response[:n])
}

// parseResponse parses STUN binding response
func (s *STUNClient) parseResponse(data []byte) (*net.UDPAddr, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("response too short")
	}

	// Verify message type: Binding Success Response (0x0101)
	if data[0] != 0x01 || data[1] != 0x01 {
		return nil, fmt.Errorf("not a binding success response")
	}

	// Parse attributes
	offset := 20
	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		attrType := uint16(data[offset])<<8 | uint16(data[offset+1])
		attrLen := uint16(data[offset+2])<<8 | uint16(data[offset+3])
		offset += 4

		if offset+int(attrLen) > len(data) {
			break
		}

		// XOR-MAPPED-ADDRESS (0x0020) or MAPPED-ADDRESS (0x0001)
		if attrType == 0x0020 || attrType == 0x0001 {
			return s.parseXORMappedAddress(data[offset:offset+int(attrLen)], attrType == 0x0020)
		}

		// Align to 4-byte boundary
		offset += int(attrLen)
		if attrLen%4 != 0 {
			offset += 4 - int(attrLen%4)
		}
	}

	return nil, fmt.Errorf("no mapped address in response")
}

// parseXORMappedAddress parses XOR-MAPPED-ADDRESS or MAPPED-ADDRESS attribute
func (s *STUNClient) parseXORMappedAddress(data []byte, isXOR bool) (*net.UDPAddr, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("address attribute too short")
	}

	family := data[1]
	port := uint16(data[2])<<8 | uint16(data[3])

	if isXOR {
		// XOR with magic cookie (0x2112A442)
		port ^= 0x2112
	}

	var ip net.IP
	if family == 0x01 { // IPv4
		if len(data) < 8 {
			return nil, fmt.Errorf("IPv4 address too short")
		}
		ip = net.IPv4(data[4], data[5], data[6], data[7])

		if isXOR {
			// XOR with magic cookie bytes
			ip[0] ^= 0x21
			ip[1] ^= 0x12
			ip[2] ^= 0xA4
			ip[3] ^= 0x42
		}
	} else {
		return nil, fmt.Errorf("unsupported address family: %d", family)
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}, nil
}

// Candidate represents a connection candidate (local or public)
type Candidate struct {
	Type string // "host", "srflx" (server reflexive), "relay"
	IP   string
	Port int
}

// GatherCandidates gathers all connection candidates
func (s *STUNClient) GatherCandidates(localPort int) ([]Candidate, error) {
	candidates := []Candidate{}

	// Get local addresses
	localAddrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range localAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					candidates = append(candidates, Candidate{
						Type: "host",
						IP:   ipnet.IP.String(),
						Port: localPort,
					})
				}
			}
		}
	}

	// Get public address via STUN
	publicAddr, err := s.DiscoverPublicAddress(localPort)
	if err == nil {
		candidates = append(candidates, Candidate{
			Type: "srflx",
			IP:   publicAddr.IP.String(),
			Port: publicAddr.Port,
		})
	}

	return candidates, nil
}
