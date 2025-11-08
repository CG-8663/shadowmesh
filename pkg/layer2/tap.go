package layer2

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/songgao/water"
)

// TAPDevice interface for TAP device operations
type TAPDevice interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	Name() string
}

// TAPInterface represents a TAP device for Layer 2 networking
type TAPInterface struct {
	iface    TAPDevice
	name     string
	ipAddr   string
	netmask  string
	isActive bool
}

// NewTAPInterface creates or attaches to an existing TAP device
func NewTAPInterface(name, ipAddr, netmask string) (*TAPInterface, error) {
	config := water.Config{
		DeviceType: water.TAP,
	}

	// Set device name if specified
	if name != "" {
		config.Name = name
	}

	// Create TAP device
	iface, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TAP device: %w", err)
	}

	tap := &TAPInterface{
		iface:    iface,
		name:     iface.Name(),
		ipAddr:   ipAddr,
		netmask:  netmask,
		isActive: true,
	}

	// Configure IP address if provided
	if ipAddr != "" && netmask != "" {
		if err := tap.configureIP(); err != nil {
			tap.Close()
			return nil, fmt.Errorf("failed to configure IP: %w", err)
		}
	}

	log.Printf("TAP device created: %s (IP: %s/%s)", tap.name, ipAddr, netmask)
	return tap, nil
}

// AttachToExisting opens an existing TAP device (like chr001) for reading/writing
func AttachToExisting(name string) (*TAPInterface, error) {
	// Open the TAP device file directly
	// chr001 is typically at /dev/net/tun with the name chr001
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

	// Set device name (chr001)
	copy(ifr.name[:], name)

	// Set flags for TAP device (IFF_TAP | IFF_NO_PI)
	// IFF_TAP = 0x0002, IFF_NO_PI = 0x1000
	ifr.flags = 0x1002

	// Use syscall to attach to the device
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(0x400454ca), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		file.Close()
		return nil, fmt.Errorf("failed to attach to TAP device %s: %v", name, errno)
	}

	tap := &TAPInterface{
		iface:    &rawTAPDevice{file: file},
		name:     name,
		isActive: true,
	}

	log.Printf("Attached to existing TAP device: %s", name)
	return tap, nil
}

// rawTAPDevice implements the water.Interface interface for raw file access
type rawTAPDevice struct {
	file *os.File
}

func (r *rawTAPDevice) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *rawTAPDevice) Write(p []byte) (int, error) {
	return r.file.Write(p)
}

func (r *rawTAPDevice) Close() error {
	return r.file.Close()
}

func (r *rawTAPDevice) Name() string {
	return "chr001"
}

// configureIP configures the IP address and netmask for the TAP device
func (t *TAPInterface) configureIP() error {
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

	log.Printf("Configured TAP device %s with IP %s", t.name, cidr)
	return nil
}

// ReadFrame reads an Ethernet frame from the TAP device
func (t *TAPInterface) ReadFrame() ([]byte, error) {
	if !t.isActive {
		return nil, fmt.Errorf("TAP device is closed")
	}

	// Allocate buffer for maximum Ethernet frame size (1500 MTU + 14 byte header)
	frame := make([]byte, 1514)

	// Read frame from TAP device
	n, err := t.iface.Read(frame)
	if err != nil {
		return nil, fmt.Errorf("failed to read frame: %w", err)
	}

	return frame[:n], nil
}

// WriteFrame writes an Ethernet frame to the TAP device
func (t *TAPInterface) WriteFrame(frame []byte) error {
	if !t.isActive {
		return fmt.Errorf("TAP device is closed")
	}

	// Write frame to TAP device
	_, err := t.iface.Write(frame)
	if err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}

	return nil
}

// GetName returns the TAP device name
func (t *TAPInterface) GetName() string {
	return t.name
}

// IsActive returns whether the TAP device is active
func (t *TAPInterface) IsActive() bool {
	return t.isActive
}

// Close closes the TAP device
func (t *TAPInterface) Close() error {
	t.isActive = false
	return t.iface.Close()
}

// ParseEthernetFrame parses an Ethernet frame and extracts header info
func ParseEthernetFrame(frame []byte) (*EthernetFrame, error) {
	if len(frame) < 14 {
		return nil, fmt.Errorf("frame too short: %d bytes", len(frame))
	}

	return &EthernetFrame{
		DstMAC:  frame[0:6],
		SrcMAC:  frame[6:12],
		EtherType: uint16(frame[12])<<8 | uint16(frame[13]),
		Payload: frame[14:],
		Raw:     frame,
	}, nil
}

// EthernetFrame represents a parsed Ethernet frame
type EthernetFrame struct {
	DstMAC    []byte
	SrcMAC    []byte
	EtherType uint16 // 0x0800 = IPv4, 0x0806 = ARP, 0x86DD = IPv6
	Payload   []byte
	Raw       []byte
}

// IsIPv4 returns true if the frame contains an IPv4 packet
func (f *EthernetFrame) IsIPv4() bool {
	return f.EtherType == 0x0800
}

// IsARP returns true if the frame is an ARP packet
func (f *EthernetFrame) IsARP() bool {
	return f.EtherType == 0x0806
}

// String returns a string representation of the frame
func (f *EthernetFrame) String() string {
	return fmt.Sprintf("EthernetFrame[dst=%x, src=%x, type=%04x, len=%d]",
		f.DstMAC, f.SrcMAC, f.EtherType, len(f.Raw))
}
