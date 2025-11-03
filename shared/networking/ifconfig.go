package networking

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// InterfaceConfigurator configures network interfaces
type InterfaceConfigurator struct {
	platform string
}

// NewInterfaceConfigurator creates a new interface configurator
func NewInterfaceConfigurator() *InterfaceConfigurator {
	return &InterfaceConfigurator{
		platform: runtime.GOOS,
	}
}

// ConfigureInterface configures a TAP interface with IP address and brings it up
func (ic *InterfaceConfigurator) ConfigureInterface(ifaceName, ipAddr, netmask string) error {
	switch ic.platform {
	case "linux":
		return ic.configureLinux(ifaceName, ipAddr, netmask)
	case "darwin":
		return ic.configureDarwin(ifaceName, ipAddr, netmask)
	default:
		return fmt.Errorf("unsupported platform: %s", ic.platform)
	}
}

// DeleteInterface removes interface configuration and brings it down
func (ic *InterfaceConfigurator) DeleteInterface(ifaceName string) error {
	switch ic.platform {
	case "linux":
		return ic.deleteLinux(ifaceName)
	case "darwin":
		return ic.deleteDarwin(ifaceName)
	default:
		return fmt.Errorf("unsupported platform: %s", ic.platform)
	}
}

// configureLinux configures interface on Linux using ip command
func (ic *InterfaceConfigurator) configureLinux(ifaceName, ipAddr, netmask string) error {
	// Convert netmask to CIDR notation
	cidr, err := netmaskToCIDR(netmask)
	if err != nil {
		return fmt.Errorf("invalid netmask: %w", err)
	}

	// Add IP address: ip addr add <ip>/<cidr> dev <iface>
	cmd := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/%d", ipAddr, cidr), "dev", ifaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add IP address: %w, output: %s", err, string(output))
	}

	// Bring interface up: ip link set <iface> up
	cmd = exec.Command("ip", "link", "set", ifaceName, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w, output: %s", err, string(output))
	}

	return nil
}

// deleteLinux removes interface configuration on Linux
func (ic *InterfaceConfigurator) deleteLinux(ifaceName string) error {
	// Bring interface down: ip link set <iface> down
	cmd := exec.Command("ip", "link", "set", ifaceName, "down")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Interface may already be down, not a fatal error
		_ = fmt.Errorf("warning: failed to bring interface down: %w, output: %s", err, string(output))
	}

	// Flush all addresses: ip addr flush dev <iface>
	cmd = exec.Command("ip", "addr", "flush", "dev", ifaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to flush addresses: %w, output: %s", err, string(output))
	}

	return nil
}

// configureDarwin configures interface on macOS using ifconfig
func (ic *InterfaceConfigurator) configureDarwin(ifaceName, ipAddr, netmask string) error {
	// Configure interface: ifconfig <iface> <ip> netmask <netmask> up
	cmd := exec.Command("ifconfig", ifaceName, ipAddr, "netmask", netmask, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure interface: %w, output: %s", err, string(output))
	}

	return nil
}

// deleteDarwin removes interface configuration on macOS
func (ic *InterfaceConfigurator) deleteDarwin(ifaceName string) error {
	// Bring interface down: ifconfig <iface> down
	cmd := exec.Command("ifconfig", ifaceName, "down")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface down: %w, output: %s", err, string(output))
	}

	// Delete address: ifconfig <iface> delete (optional, interface removal handles this)
	cmd = exec.Command("ifconfig", ifaceName, "delete")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Address may not exist, not a fatal error
		_ = fmt.Errorf("warning: failed to delete address: %w, output: %s", err, string(output))
	}

	return nil
}

// AddRoute adds a route to the routing table
func (ic *InterfaceConfigurator) AddRoute(destination, gateway, ifaceName string) error {
	switch ic.platform {
	case "linux":
		return ic.addRouteLinux(destination, gateway, ifaceName)
	case "darwin":
		return ic.addRouteDarwin(destination, gateway, ifaceName)
	default:
		return fmt.Errorf("unsupported platform: %s", ic.platform)
	}
}

// DeleteRoute removes a route from the routing table
func (ic *InterfaceConfigurator) DeleteRoute(destination string) error {
	switch ic.platform {
	case "linux":
		return ic.deleteRouteLinux(destination)
	case "darwin":
		return ic.deleteRouteDarwin(destination)
	default:
		return fmt.Errorf("unsupported platform: %s", ic.platform)
	}
}

// addRouteLinux adds route on Linux
func (ic *InterfaceConfigurator) addRouteLinux(destination, gateway, ifaceName string) error {
	// ip route add <destination> via <gateway> dev <iface>
	var cmd *exec.Cmd
	if gateway != "" {
		cmd = exec.Command("ip", "route", "add", destination, "via", gateway, "dev", ifaceName)
	} else {
		cmd = exec.Command("ip", "route", "add", destination, "dev", ifaceName)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add route: %w, output: %s", err, string(output))
	}

	return nil
}

// deleteRouteLinux removes route on Linux
func (ic *InterfaceConfigurator) deleteRouteLinux(destination string) error {
	// ip route del <destination>
	cmd := exec.Command("ip", "route", "del", destination)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete route: %w, output: %s", err, string(output))
	}

	return nil
}

// addRouteDarwin adds route on macOS
func (ic *InterfaceConfigurator) addRouteDarwin(destination, gateway, ifaceName string) error {
	// route add -net <destination> <gateway>
	var cmd *exec.Cmd
	if gateway != "" {
		cmd = exec.Command("route", "add", "-net", destination, gateway)
	} else {
		cmd = exec.Command("route", "add", "-net", destination, "-interface", ifaceName)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add route: %w, output: %s", err, string(output))
	}

	return nil
}

// deleteRouteDarwin removes route on macOS
func (ic *InterfaceConfigurator) deleteRouteDarwin(destination string) error {
	// route delete -net <destination>
	cmd := exec.Command("route", "delete", "-net", destination)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete route: %w, output: %s", err, string(output))
	}

	return nil
}

// netmaskToCIDR converts dotted decimal netmask to CIDR notation
func netmaskToCIDR(netmask string) (int, error) {
	parts := strings.Split(netmask, ".")
	if len(parts) != 4 {
		return 0, fmt.Errorf("invalid netmask format")
	}

	var cidr int
	for _, part := range parts {
		var octet int
		_, err := fmt.Sscanf(part, "%d", &octet)
		if err != nil {
			return 0, fmt.Errorf("invalid octet in netmask: %w", err)
		}

		// Count bits in octet
		for i := 7; i >= 0; i-- {
			if (octet & (1 << i)) != 0 {
				cidr++
			} else {
				// Once we hit a zero bit, rest should be zero
				if octet != 0 {
					return 0, fmt.Errorf("invalid netmask: non-contiguous bits")
				}
				break
			}
		}
	}

	return cidr, nil
}

// EnableIPForwarding enables IP forwarding (required for routing between interfaces)
func (ic *InterfaceConfigurator) EnableIPForwarding() error {
	switch ic.platform {
	case "linux":
		// echo 1 > /proc/sys/net/ipv4/ip_forward
		cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to enable IP forwarding: %w, output: %s", err, string(output))
		}
		return nil

	case "darwin":
		// sysctl -w net.inet.ip.forwarding=1
		cmd := exec.Command("sysctl", "-w", "net.inet.ip.forwarding=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to enable IP forwarding: %w, output: %s", err, string(output))
		}
		return nil

	default:
		return fmt.Errorf("unsupported platform: %s", ic.platform)
	}
}

// DisableIPForwarding disables IP forwarding
func (ic *InterfaceConfigurator) DisableIPForwarding() error {
	switch ic.platform {
	case "linux":
		cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=0")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to disable IP forwarding: %w, output: %s", err, string(output))
		}
		return nil

	case "darwin":
		cmd := exec.Command("sysctl", "-w", "net.inet.ip.forwarding=0")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to disable IP forwarding: %w, output: %s", err, string(output))
		}
		return nil

	default:
		return fmt.Errorf("unsupported platform: %s", ic.platform)
	}
}
