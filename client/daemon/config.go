package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the client daemon configuration
type Config struct {
	// Operating mode: "relay", "listener", or "connector"
	Mode string `yaml:"mode"`

	// Relay configuration (for relay mode)
	Relay RelayConfig `yaml:"relay,omitempty"`

	// P2P configuration (for listener/connector mode)
	P2P P2PConfig `yaml:"p2p,omitempty"`

	// TAP device configuration
	TAP TAPDeviceConfig `yaml:"tap"`

	// Crypto configuration
	Crypto CryptoConfig `yaml:"crypto"`

	// Client identity
	Identity IdentityConfig `yaml:"identity"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`
}

// RelayConfig contains relay server settings
type RelayConfig struct {
	URL               string        `yaml:"url"`                // WebSocket URL (ws://... or wss://...)
	TLSSkipVerify     bool          `yaml:"tls_skip_verify"`    // Skip TLS certificate verification (for self-signed certs)
	ReconnectInterval time.Duration `yaml:"reconnect_interval"` // Time between reconnection attempts
	MaxReconnects     int           `yaml:"max_reconnects"`     // Maximum reconnection attempts
	HandshakeTimeout  time.Duration `yaml:"handshake_timeout"`  // Handshake timeout
}

// P2PConfig contains direct peer-to-peer settings
type P2PConfig struct {
	// For listener mode
	ListenAddress string `yaml:"listen_address,omitempty"` // Address to listen on (e.g., "0.0.0.0:8443")

	// For connector mode
	PeerAddress string `yaml:"peer_address,omitempty"` // Peer address to connect to (e.g., "1.2.3.4:8443")

	// TLS settings
	TLSEnabled    bool   `yaml:"tls_enabled"`     // Enable TLS
	TLSSkipVerify bool   `yaml:"tls_skip_verify"` // Skip TLS certificate verification
	TLSCertFile   string `yaml:"tls_cert_file,omitempty"`   // Path to TLS certificate (for listener)
	TLSKeyFile    string `yaml:"tls_key_file,omitempty"`    // Path to TLS private key (for listener)

	// Connection settings
	HandshakeTimeout  time.Duration `yaml:"handshake_timeout,omitempty"`  // Handshake timeout
	ReconnectInterval time.Duration `yaml:"reconnect_interval,omitempty"` // Reconnect interval (connector only)
	MaxReconnects     int           `yaml:"max_reconnects,omitempty"`     // Max reconnection attempts (connector only)
}

// TAPDeviceConfig contains TAP device settings
type TAPDeviceConfig struct {
	Name    string `yaml:"name"`     // TAP device name (e.g., "tap0")
	MTU     int    `yaml:"mtu"`      // Maximum Transmission Unit
	IPAddr  string `yaml:"ip_addr"`  // IP address for TAP interface
	Netmask string `yaml:"netmask"`  // Network mask
}

// CryptoConfig contains cryptographic settings
type CryptoConfig struct {
	KeyRotationInterval time.Duration `yaml:"key_rotation_interval"` // How often to rotate keys
	EnableKeyRotation   bool          `yaml:"enable_key_rotation"`   // Enable automatic key rotation
}

// IdentityConfig contains client identity settings
type IdentityConfig struct {
	KeysDir       string `yaml:"keys_dir"`        // Directory to store keys
	PrivateKeyFile string `yaml:"private_key_file"` // Path to private signing key
	ClientIDFile   string `yaml:"client_id_file"`   // Path to client ID file
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`  // Log level (debug, info, warn, error)
	Format string `yaml:"format"` // Log format (json, text)
	File   string `yaml:"file"`   // Log file path (empty for stdout)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	keysDir := filepath.Join(homeDir, ".shadowmesh", "keys")

	return &Config{
		Mode: "relay", // Default to relay mode for backwards compatibility
		Relay: RelayConfig{
			URL:               "wss://relay.shadowmesh.network:443",
			ReconnectInterval: 5 * time.Second,
			MaxReconnects:     10,
			HandshakeTimeout:  30 * time.Second,
		},
		P2P: P2PConfig{
			TLSEnabled:        true,
			TLSSkipVerify:     false,
			HandshakeTimeout:  30 * time.Second,
			ReconnectInterval: 5 * time.Second,
			MaxReconnects:     10,
		},
		TAP: TAPDeviceConfig{
			Name:    "tap0",
			MTU:     1500,
			IPAddr:  "10.42.0.2",
			Netmask: "255.255.255.0",
		},
		Crypto: CryptoConfig{
			KeyRotationInterval: 1 * time.Hour,
			EnableKeyRotation:   true,
		},
		Identity: IdentityConfig{
			KeysDir:        keysDir,
			PrivateKeyFile: filepath.Join(keysDir, "signing_key.json"),
			ClientIDFile:   filepath.Join(keysDir, "client_id.txt"),
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			File:   "",
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a YAML file
func (c *Config) SaveConfig(path string) error {
	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate mode
	validModes := map[string]bool{"relay": true, "listener": true, "connector": true}
	if !validModes[c.Mode] {
		return fmt.Errorf("invalid mode: %s (must be 'relay', 'listener', or 'connector')", c.Mode)
	}

	// Validate mode-specific settings
	switch c.Mode {
	case "relay":
		if c.Relay.URL == "" {
			return fmt.Errorf("relay URL cannot be empty in relay mode")
		}
		if c.Relay.ReconnectInterval < time.Second {
			return fmt.Errorf("reconnect interval too short: %v", c.Relay.ReconnectInterval)
		}
		if c.Relay.MaxReconnects < 1 {
			return fmt.Errorf("max reconnects must be at least 1")
		}

	case "listener":
		if c.P2P.ListenAddress == "" {
			return fmt.Errorf("listen_address cannot be empty in listener mode")
		}
		if c.P2P.TLSEnabled && (c.P2P.TLSCertFile == "" || c.P2P.TLSKeyFile == "") {
			return fmt.Errorf("TLS cert and key files required when TLS is enabled")
		}

	case "connector":
		if c.P2P.PeerAddress == "" {
			return fmt.Errorf("peer_address cannot be empty in connector mode")
		}
		if c.P2P.ReconnectInterval < time.Second {
			return fmt.Errorf("reconnect interval too short: %v", c.P2P.ReconnectInterval)
		}
		if c.P2P.MaxReconnects < 1 {
			return fmt.Errorf("max reconnects must be at least 1")
		}
	}

	// Validate TAP configuration
	if c.TAP.Name == "" {
		return fmt.Errorf("TAP device name cannot be empty")
	}
	if c.TAP.MTU < 576 || c.TAP.MTU > 9000 {
		return fmt.Errorf("invalid MTU: %d (must be between 576 and 9000)", c.TAP.MTU)
	}

	// Validate key rotation interval
	if c.Crypto.EnableKeyRotation && c.Crypto.KeyRotationInterval < time.Minute {
		return fmt.Errorf("key rotation interval too short: %v", c.Crypto.KeyRotationInterval)
	}

	// Validate identity paths
	if c.Identity.KeysDir == "" {
		return fmt.Errorf("keys directory cannot be empty")
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	return nil
}

// EnsureDirectories creates necessary directories
func (c *Config) EnsureDirectories() error {
	// Create keys directory
	if err := os.MkdirAll(c.Identity.KeysDir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Create log directory if logging to file
	if c.Logging.File != "" {
		logDir := filepath.Dir(c.Logging.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".shadowmesh", "config.yaml")
}

// LoadOrCreateConfig loads existing config or creates a new one with defaults
func LoadOrCreateConfig(path string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config
		config := DefaultConfig()

		// Ensure directories exist
		if err := config.EnsureDirectories(); err != nil {
			return nil, err
		}

		// Save default config
		if err := config.SaveConfig(path); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}

		return config, nil
	}

	// Load existing config
	return LoadConfig(path)
}
