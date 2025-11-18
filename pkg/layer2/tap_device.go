package layer2

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/songgao/water"
)

// TAPDevice manages a TAP (Layer 2) network interface
type TAPDevice struct {
	iface     *water.Interface
	name      string
	mtu       int
	readChan  chan *EthernetFrame // Parsed frames read from TAP (to be encrypted and sent)
	writeChan chan []byte         // Frames to write to TAP (decrypted frames)
	errorChan chan error
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// TAPConfig contains configuration for the TAP device
type TAPConfig struct {
	Name string // TAP device name (e.g., "shadowmesh0" on Linux, system-assigned on macOS)
	MTU  int    // Maximum Transmission Unit (default 1500)
}

// NewTAPDevice creates and configures a new TAP device
//
// Platform-specific naming:
//   - Linux: Supports custom names (e.g., "shadowmesh0"). If empty, kernel assigns "tap0", "tap1", etc.
//   - macOS: Ignores custom names. System assigns "utunX" or "tapX" automatically.
//     The water library will return the actual assigned name via iface.Name().
//     Applications should use config.Name as an internal identifier only.
func NewTAPDevice(config TAPConfig) (*TAPDevice, error) {
	// Set defaults
	if config.MTU == 0 {
		config.MTU = 1500
	}

	// Configure TAP interface
	tapConfig := water.Config{
		DeviceType: water.TAP,
	}

	// On Linux, this sets the desired interface name
	// On macOS, this is ignored and the system assigns utunX or tapX
	if config.Name != "" {
		tapConfig.Name = config.Name
	}

	// Create TAP interface (requires root/admin privileges)
	iface, err := water.New(tapConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TAP device: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	tap := &TAPDevice{
		iface:     iface,
		name:      iface.Name(), // Actual OS-assigned name (may differ from config.Name on macOS)
		mtu:       config.MTU,
		readChan:  make(chan *EthernetFrame, 2000), // Increased from 100 to handle burst traffic
		writeChan: make(chan []byte, 2000),         // Increased from 100 to prevent "TAP write channel full" errors
		errorChan: make(chan error, 10),
		ctx:       ctx,
		cancel:    cancel,
	}

	return tap, nil
}

// Start begins reading and writing frames
func (tap *TAPDevice) Start() {
	tap.wg.Add(2)
	go tap.readLoop()
	go tap.writeLoop()
}

// Stop gracefully stops the TAP device
func (tap *TAPDevice) Stop() error {
	tap.cancel()
	tap.wg.Wait()

	if err := tap.iface.Close(); err != nil {
		return fmt.Errorf("failed to close TAP device: %w", err)
	}

	close(tap.readChan)
	close(tap.writeChan)
	close(tap.errorChan)

	return nil
}

// readLoop continuously reads Ethernet frames from the TAP device
func (tap *TAPDevice) readLoop() {
	defer tap.wg.Done()

	buffer := make([]byte, tap.mtu+14) // MTU + Ethernet header (14 bytes)

	for {
		select {
		case <-tap.ctx.Done():
			return

		default:
			// Read frame from TAP device
			n, err := tap.iface.Read(buffer)
			if err != nil {
				if err == io.EOF {
					return
				}
				select {
				case tap.errorChan <- fmt.Errorf("TAP read error: %w", err):
				default:
				}
				continue
			}

			// Parse Ethernet frame (validates size and extracts header fields)
			parsedFrame, err := ParseFrame(buffer[:n])
			if err != nil {
				// Malformed frame - log and drop
				select {
				case tap.errorChan <- fmt.Errorf("malformed frame dropped: %w", err):
				default:
				}
				continue
			}

			// Send parsed frame to read channel for encryption
			select {
			case tap.readChan <- parsedFrame:
			case <-tap.ctx.Done():
				return
			default:
				// Channel full, drop frame
				select {
				case tap.errorChan <- fmt.Errorf("read channel full, dropping frame"):
				default:
				}
			}
		}
	}
}

// writeLoop continuously writes frames to the TAP device
func (tap *TAPDevice) writeLoop() {
	defer tap.wg.Done()

	for {
		select {
		case <-tap.ctx.Done():
			return

		case frame := <-tap.writeChan:
			// Validate frame
			if len(frame) < 14 {
				select {
				case tap.errorChan <- fmt.Errorf("dropping invalid frame: too short (%d bytes)", len(frame)):
				default:
				}
				continue
			}

			if len(frame) > tap.mtu+14 {
				select {
				case tap.errorChan <- fmt.Errorf("dropping invalid frame: too large (%d bytes)", len(frame)):
				default:
				}
				continue
			}

			// Write frame to TAP device
			_, err := tap.iface.Write(frame)
			if err != nil {
				select {
				case tap.errorChan <- fmt.Errorf("TAP write error: %w", err):
				default:
				}
			}
		}
	}
}

// ReadChannel returns the channel for parsed frames read from TAP
func (tap *TAPDevice) ReadChannel() <-chan *EthernetFrame {
	return tap.readChan
}

// WriteChannel returns the channel for frames to write to TAP
func (tap *TAPDevice) WriteChannel() chan<- []byte {
	return tap.writeChan
}

// ErrorChannel returns the channel for errors
func (tap *TAPDevice) ErrorChannel() <-chan error {
	return tap.errorChan
}

// Name returns the TAP device name
func (tap *TAPDevice) Name() string {
	return tap.name
}

// MTU returns the configured MTU
func (tap *TAPDevice) MTU() int {
	return tap.mtu
}

// ConfigureInterface configures the TAP interface with IP address and routing
// This requires CAP_NET_ADMIN capability or root privileges
func (tap *TAPDevice) ConfigureInterface(ipAddr, netmask string) error {
	// Bring interface up
	cmdUp := exec.Command("ip", "link", "set", "dev", tap.name, "up")
	if output, err := cmdUp.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring up interface %s: %w (output: %s)", tap.name, err, string(output))
	}

	// Set IP address and netmask
	cidr := ipAddr + "/" + netmask
	cmdAddr := exec.Command("ip", "addr", "add", cidr, "dev", tap.name)
	if output, err := cmdAddr.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address %s on %s: %w (output: %s)", cidr, tap.name, err, string(output))
	}

	return nil
}
