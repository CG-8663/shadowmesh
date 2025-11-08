package layer3

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/songgao/water"
)

// packetBufferPool reduces memory allocations for packet reads (Solution 2: Buffer Pool)
var packetBufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 1500) // MTU size
		return &b
	},
}

// TUNDevice interface for TUN device operations
type TUNDevice interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	Name() string
}

// TUNInterface represents a TUN device for Layer 3 networking
type TUNInterface struct {
	iface      TUNDevice
	name       string
	ipAddr     string
	netmask    string
	isActive   bool
	writeQueue chan []byte  // Async write queue (Phase 3 Fix: prevent UDP blocking)
	mu         sync.RWMutex
	wg         sync.WaitGroup
}

// NewTUNInterface creates or attaches to an existing TUN device
func NewTUNInterface(name, ipAddr, netmask string) (*TUNInterface, error) {
	config := water.Config{
		DeviceType: water.TUN,
	}

	// Set device name if specified
	if name != "" {
		config.Name = name
	}

	// Create TUN device
	iface, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device: %w", err)
	}

	tun := &TUNInterface{
		iface:      iface,
		name:       iface.Name(),
		ipAddr:     ipAddr,
		netmask:    netmask,
		isActive:   true,
		writeQueue: make(chan []byte, 8192), // Buffer 8192 packets (~12MB at MTU)
	}

	// Start async write worker
	tun.wg.Add(1)
	go tun.writeWorker()
	log.Printf("[TUN-INIT] Async write queue initialized (buffer: 8192 packets)")

	// Configure IP address if provided
	if ipAddr != "" && netmask != "" {
		if err := tun.configureIP(); err != nil {
			tun.Close()
			return nil, fmt.Errorf("failed to configure IP: %w", err)
		}
	}

	log.Printf("TUN device created: %s (IP: %s/%s)", tun.name, ipAddr, netmask)
	return tun, nil
}

// AttachToExisting opens an existing TUN device for reading/writing
func AttachToExisting(name string) (*TUNInterface, error) {
	// Open the TUN device file directly
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/net/tun: %w", err)
	}

	// Create ifreq structure to attach to specific device
	var ifr struct {
		name  [16]byte
		flags uint16
		_     [22]byte // padding
	}

	// Set device name
	copy(ifr.name[:], name)

	// Set flags for TUN device (IFF_TUN | IFF_NO_PI)
	// IFF_TUN = 0x0001, IFF_NO_PI = 0x1000
	ifr.flags = 0x1001

	// Use syscall to attach to the device
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(0x400454ca), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		file.Close()
		return nil, fmt.Errorf("failed to attach to TUN device %s: %v", name, errno)
	}

	tun := &TUNInterface{
		iface:      &rawTUNDevice{file: file},
		name:       name,
		isActive:   true,
		writeQueue: make(chan []byte, 8192),
	}

	// Start async write worker
	tun.wg.Add(1)
	go tun.writeWorker()
	log.Printf("[TUN-INIT] Async write queue initialized (buffer: 8192 packets)")

	log.Printf("Attached to existing TUN device: %s", name)
	return tun, nil
}

// rawTUNDevice implements the water.Interface interface for raw file access
type rawTUNDevice struct {
	file *os.File
}

func (r *rawTUNDevice) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *rawTUNDevice) Write(p []byte) (int, error) {
	return r.file.Write(p)
}

func (r *rawTUNDevice) Close() error {
	return r.file.Close()
}

func (r *rawTUNDevice) Name() string {
	return "tun0"
}

// configureIP configures the IP address and netmask for the TUN device
func (t *TUNInterface) configureIP() error {
	if runtime.GOOS == "darwin" {
		// macOS: use ifconfig
		return t.configureIPDarwin()
	}

	// Linux: use ip command
	return t.configureIPLinux()
}

// configureIPLinux configures IP on Linux using 'ip' command
func (t *TUNInterface) configureIPLinux() error {
	// Bring interface up
	cmd := exec.Command("ip", "link", "set", "dev", t.name, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up interface: %w", err)
	}

	// Set IP address
	cidr := fmt.Sprintf("%s/%s", t.ipAddr, t.netmask)
	cmd = exec.Command("ip", "addr", "add", cidr, "dev", t.name)
	if err := cmd.Run(); err != nil {
		// Ignore error if address already exists
		log.Printf("Warning: failed to set IP (may already exist): %v", err)
	}

	log.Printf("Configured TUN device %s with IP %s", t.name, cidr)
	return nil
}

// configureIPDarwin configures IP on macOS using 'ifconfig' command
func (t *TUNInterface) configureIPDarwin() error {
	// macOS ifconfig syntax: ifconfig utun9 10.10.10.8 10.10.10.8 netmask 255.255.255.0 up
	// For point-to-point interfaces, src and dst IPs are the same

	// Calculate netmask from CIDR (e.g., "24" -> "255.255.255.0")
	netmask := cidrToNetmask(t.netmask)

	// Configure interface: ifconfig <name> <local> <remote> netmask <mask> up
	cmd := exec.Command("ifconfig", t.name, t.ipAddr, t.ipAddr, "netmask", netmask, "up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to configure interface: %w (output: %s)", err, string(output))
	}

	log.Printf("Configured TUN device %s with IP %s/%s (macOS)", t.name, t.ipAddr, t.netmask)
	return nil
}

// cidrToNetmask converts CIDR prefix length to dotted decimal netmask
func cidrToNetmask(cidr string) string {
	switch cidr {
	case "8":
		return "255.0.0.0"
	case "16":
		return "255.255.0.0"
	case "24":
		return "255.255.255.0"
	case "32":
		return "255.255.255.255"
	default:
		// Default to /24 if unknown
		return "255.255.255.0"
	}
}

// ReadPacket reads an IP packet from the TUN device
// Uses buffer pool to reduce memory allocations and GC pressure
func (t *TUNInterface) ReadPacket() ([]byte, error) {
	if !t.isActive {
		return nil, fmt.Errorf("TUN device is closed")
	}

	// Get buffer from pool (Solution 2: Buffer Pool)
	bufPtr := packetBufferPool.Get().(*[]byte)
	buf := *bufPtr

	// Read packet from TUN device
	n, err := t.iface.Read(buf)
	if err != nil {
		packetBufferPool.Put(bufPtr) // Return buffer on error
		return nil, fmt.Errorf("failed to read packet: %w", err)
	}

	// Allocate result buffer (caller owns)
	result := make([]byte, n)
	copy(result, buf[:n])

	packetBufferPool.Put(bufPtr) // Return buffer to pool
	return result, nil
}

// writeWorker handles async writes to TUN device
// Runs in separate goroutine to prevent UDP receive blocking
func (t *TUNInterface) writeWorker() {
	defer t.wg.Done()

	packetsWritten := uint64(0)
	packetsDropped := uint64(0)

	for packet := range t.writeQueue {
		t.mu.RLock()
		active := t.isActive
		t.mu.RUnlock()

		if !active {
			break
		}

		// Blocking write to TUN (kernel may still drop if TX queue full)
		_, err := t.iface.Write(packet)
		if err != nil {
			packetsDropped++
			if packetsDropped%1000 == 0 {
				log.Printf("[TUN-WRITE-ERROR] Failed to write packet (%d errors so far): %v", packetsDropped, err)
			}
		} else {
			packetsWritten++
			if packetsWritten%10000 == 0 {
				queueLen := len(t.writeQueue)
				queueCap := cap(t.writeQueue)
				queuePct := float64(queueLen) / float64(queueCap) * 100
				log.Printf("[TUN-WRITE-STATS] Packets written: %d, Errors: %d, Queue: %d/%d (%.1f%%)",
					packetsWritten, packetsDropped, queueLen, queueCap, queuePct)
			}
		}
	}

	log.Printf("[TUN-WRITE-WORKER] Shutdown complete. Total written: %d, Errors: %d", packetsWritten, packetsDropped)
}

// WritePacket writes an IP packet to the TUN device (async, non-blocking)
func (t *TUNInterface) WritePacket(packet []byte) error {
	t.mu.RLock()
	if !t.isActive {
		t.mu.RUnlock()
		return fmt.Errorf("TUN device is closed")
	}
	t.mu.RUnlock()

	// Make a copy of the packet (caller may reuse buffer)
	packetCopy := make([]byte, len(packet))
	copy(packetCopy, packet)

	// Non-blocking send to write queue
	select {
	case t.writeQueue <- packetCopy:
		return nil
	default:
		// Queue full - drop packet (will be retransmitted at higher layer)
		return fmt.Errorf("TUN write queue full (packet dropped)")
	}
}

// GetName returns the TUN device name
func (t *TUNInterface) GetName() string {
	return t.name
}

// IsActive returns whether the TUN device is active
func (t *TUNInterface) IsActive() bool {
	return t.isActive
}

// Close closes the TUN device
func (t *TUNInterface) Close() error {
	t.mu.Lock()
	t.isActive = false
	t.mu.Unlock()

	// Close write queue to signal worker shutdown
	close(t.writeQueue)

	// Wait for write worker to finish
	log.Printf("[TUN-CLOSE] Waiting for write worker to finish...")
	t.wg.Wait()

	return t.iface.Close()
}

// ParseIPPacket parses an IP packet and extracts header info
func ParseIPPacket(packet []byte) (*IPPacket, error) {
	if len(packet) < 20 {
		return nil, fmt.Errorf("packet too short: %d bytes", len(packet))
	}

	// IP version and header length
	version := packet[0] >> 4
	if version != 4 && version != 6 {
		return nil, fmt.Errorf("invalid IP version: %d", version)
	}

	ip := &IPPacket{
		Version: version,
		Raw:     packet,
	}

	if version == 4 {
		// IPv4 packet
		headerLen := int(packet[0]&0x0F) * 4
		if len(packet) < headerLen {
			return nil, fmt.Errorf("IPv4 packet truncated")
		}

		ip.HeaderLength = headerLen
		ip.Protocol = packet[9]
		ip.SrcIP = packet[12:16]
		ip.DstIP = packet[16:20]
		ip.Payload = packet[headerLen:]
	} else {
		// IPv6 packet
		ip.HeaderLength = 40
		ip.Protocol = packet[6]
		ip.SrcIP = packet[8:24]
		ip.DstIP = packet[24:40]
		ip.Payload = packet[40:]
	}

	return ip, nil
}

// IPPacket represents a parsed IP packet
type IPPacket struct {
	Version      uint8
	HeaderLength int
	Protocol     uint8  // 1 = ICMP, 6 = TCP, 17 = UDP
	SrcIP        []byte
	DstIP        []byte
	Payload      []byte
	Raw          []byte
}

// IsIPv4 returns true if the packet is IPv4
func (p *IPPacket) IsIPv4() bool {
	return p.Version == 4
}

// IsIPv6 returns true if the packet is IPv6
func (p *IPPacket) IsIPv6() bool {
	return p.Version == 6
}

// IsICMP returns true if the packet is ICMP
func (p *IPPacket) IsICMP() bool {
	return p.Protocol == 1
}

// IsTCP returns true if the packet is TCP
func (p *IPPacket) IsTCP() bool {
	return p.Protocol == 6
}

// IsUDP returns true if the packet is UDP
func (p *IPPacket) IsUDP() bool {
	return p.Protocol == 17
}

// String returns a string representation of the packet
func (p *IPPacket) String() string {
	return fmt.Sprintf("IPPacket[v%d, proto=%d, src=%v, dst=%v, len=%d]",
		p.Version, p.Protocol, p.SrcIP, p.DstIP, len(p.Raw))
}
