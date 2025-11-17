package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the relay server configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Identity IdentityConfig `yaml:"identity"`
	Limits   LimitsConfig   `yaml:"limits"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig contains server-specific settings
type ServerConfig struct {
	ListenAddr string    `yaml:"listen_addr"` // e.g., "0.0.0.0:8443"
	TLS        TLSConfig `yaml:"tls"`
}

// TLSConfig contains TLS/SSL settings
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// IdentityConfig contains relay identity settings
type IdentityConfig struct {
	RelayID      string `yaml:"relay_id"`      // Hex-encoded relay ID
	SigningKey   string `yaml:"signing_key"`   // Path to signing key file
	KeysDir      string `yaml:"keys_dir"`      // Directory for key storage
	AutoGenerate bool   `yaml:"auto_generate"` // Auto-generate keys if missing
}

// LimitsConfig contains connection and resource limits
type LimitsConfig struct {
	MaxClients        int `yaml:"max_clients"`        // Maximum simultaneous clients
	HandshakeTimeout  int `yaml:"handshake_timeout"`  // Seconds to complete handshake
	HeartbeatInterval int `yaml:"heartbeat_interval"` // Seconds between heartbeats
	HeartbeatTimeout  int `yaml:"heartbeat_timeout"`  // Seconds before client timeout
	MaxFrameSize      int `yaml:"max_frame_size"`     // Maximum frame size in bytes
	ReadBufferSize    int `yaml:"read_buffer_size"`   // WebSocket read buffer
	WriteBufferSize   int `yaml:"write_buffer_size"`  // WebSocket write buffer
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`       // debug, info, warn, error
	Format     string `yaml:"format"`      // text, json
	OutputFile string `yaml:"output_file"` // Log file path (empty = stdout)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	keysDir := filepath.Join(homeDir, ".shadowmesh-relay", "keys")

	return &Config{
		Server: ServerConfig{
			ListenAddr: "0.0.0.0:8443",
			TLS: TLSConfig{
				Enabled:  true,
				CertFile: "/etc/shadowmesh/relay.crt",
				KeyFile:  "/etc/shadowmesh/relay.key",
			},
		},
		Identity: IdentityConfig{
			RelayID:      "", // Generated on first run
			SigningKey:   filepath.Join(keysDir, "signing_key.json"),
			KeysDir:      keysDir,
			AutoGenerate: true,
		},
		Limits: LimitsConfig{
			MaxClients:        1000,
			HandshakeTimeout:  30,
			HeartbeatInterval: 30,
			HeartbeatTimeout:  90,
			MaxFrameSize:      65536,
			ReadBufferSize:    2 * 1024 * 1024, // 2MB (increased from 4KB for burst traffic)
			WriteBufferSize:   2 * 1024 * 1024, // 2MB (prevents buffer full errors)
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			OutputFile: "",
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// LoadOrCreateConfig loads config from file or creates default
func LoadOrCreateConfig(path string) (*Config, error) {
	// Try to load existing config
	if _, err := os.Stat(path); err == nil {
		return LoadConfig(path)
	}

	// Create default config
	config := DefaultConfig()

	// Ensure config directory exists
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write default config
	if err := config.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save default config: %w", err)
	}

	fmt.Printf("Created default config at: %s\n", path)
	return config, nil
}

// Save writes the configuration to a YAML file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server settings
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("server.listen_addr is required")
	}

	// Validate TLS settings
	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" {
			return fmt.Errorf("server.tls.cert_file is required when TLS is enabled")
		}
		if c.Server.TLS.KeyFile == "" {
			return fmt.Errorf("server.tls.key_file is required when TLS is enabled")
		}
	}

	// Validate identity settings
	if c.Identity.KeysDir == "" {
		return fmt.Errorf("identity.keys_dir is required")
	}
	if c.Identity.SigningKey == "" {
		return fmt.Errorf("identity.signing_key is required")
	}

	// Validate limits
	if c.Limits.MaxClients < 1 {
		return fmt.Errorf("limits.max_clients must be at least 1")
	}
	if c.Limits.HandshakeTimeout < 5 {
		return fmt.Errorf("limits.handshake_timeout must be at least 5 seconds")
	}
	if c.Limits.HeartbeatInterval < 10 {
		return fmt.Errorf("limits.heartbeat_interval must be at least 10 seconds")
	}
	if c.Limits.HeartbeatTimeout <= c.Limits.HeartbeatInterval {
		return fmt.Errorf("limits.heartbeat_timeout must be greater than heartbeat_interval")
	}
	if c.Limits.MaxFrameSize < 1500 || c.Limits.MaxFrameSize > 65536 {
		return fmt.Errorf("limits.max_frame_size must be between 1500 and 65536")
	}

	// Validate logging settings
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}

	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("logging.format must be one of: text, json")
	}

	return nil
}

// GetKeysDir returns the keys directory, creating it if necessary
func (c *Config) GetKeysDir() (string, error) {
	keysDir := c.Identity.KeysDir
	if keysDir == "" {
		return "", fmt.Errorf("keys directory not configured")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create keys directory: %w", err)
	}

	return keysDir, nil
}

// GetSigningKeyPath returns the full path to the signing key file
func (c *Config) GetSigningKeyPath() string {
	return c.Identity.SigningKey
}

// GetTLSFiles returns the paths to TLS certificate and key files
func (c *Config) GetTLSFiles() (certFile, keyFile string, err error) {
	if !c.Server.TLS.Enabled {
		return "", "", fmt.Errorf("TLS is not enabled")
	}

	certFile = c.Server.TLS.CertFile
	keyFile = c.Server.TLS.KeyFile

	// Check if files exist
	if _, err := os.Stat(certFile); err != nil {
		return "", "", fmt.Errorf("TLS cert file not found: %s", certFile)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return "", "", fmt.Errorf("TLS key file not found: %s", keyFile)
	}

	return certFile, keyFile, nil
}
