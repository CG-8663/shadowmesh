package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete discovery node configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Security  SecurityConfig  `yaml:"security"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	HTTPPort  int    `yaml:"http_port"`
	HTTPSPort int    `yaml:"https_port"`
	TLSCert   string `yaml:"tls_cert"`
	TLSKey    string `yaml:"tls_key"`
	Region    string `yaml:"region"` // e.g., "north_america", "europe"
}

// DatabaseConfig holds PostgreSQL settings
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// RedisConfig holds Redis cache settings
type RedisConfig struct {
	Host     string        `yaml:"host"`
	Port     int           `yaml:"port"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	TTL      time.Duration `yaml:"ttl"` // Cache TTL in seconds
}

// DiscoveryConfig holds Kademlia and discovery settings
type DiscoveryConfig struct {
	PeerID            string        `yaml:"peer_id"`             // This node's peer ID
	KBucketSize       int           `yaml:"k_bucket_size"`       // K parameter (default: 20)
	RefreshInterval   time.Duration `yaml:"refresh_interval"`    // Bucket refresh (default: 3600s)
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`    // Stale peer cleanup (default: 3600s)
	SessionExpiry     time.Duration `yaml:"session_expiry"`      // Session TTL (default: 24h)
	ChallengeExpiry   time.Duration `yaml:"challenge_expiry"`    // Challenge TTL (default: 30s)
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	RequireAuth       bool     `yaml:"require_auth"`        // Require authentication for all requests
	AllowedOrigins    []string `yaml:"allowed_origins"`     // CORS origins
	RateLimitPerMin   int      `yaml:"rate_limit_per_min"`  // Requests per minute per IP
	MaxPeersPerIP     int      `yaml:"max_peers_per_ip"`    // Max peers from single IP
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`       // debug, info, warn, error
	OutputFile string `yaml:"output_file"` // Log file path (empty = stdout)
	MaxSizeMB  int    `yaml:"max_size_mb"` // Max log file size before rotation
	MaxBackups int    `yaml:"max_backups"` // Max old log files to keep
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.setDefaults()

	// Validate
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for optional config fields
func (c *Config) setDefaults() {
	// Server defaults
	if c.Server.HTTPPort == 0 {
		c.Server.HTTPPort = 8080
	}
	if c.Server.HTTPSPort == 0 {
		c.Server.HTTPSPort = 8443
	}
	if c.Server.Region == "" {
		c.Server.Region = "unknown"
	}

	// Database defaults
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}

	// Redis defaults
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379
	}
	if c.Redis.TTL == 0 {
		c.Redis.TTL = 5 * time.Minute
	}

	// Discovery defaults
	if c.Discovery.KBucketSize == 0 {
		c.Discovery.KBucketSize = 20
	}
	if c.Discovery.RefreshInterval == 0 {
		c.Discovery.RefreshInterval = 1 * time.Hour
	}
	if c.Discovery.CleanupInterval == 0 {
		c.Discovery.CleanupInterval = 1 * time.Hour
	}
	if c.Discovery.SessionExpiry == 0 {
		c.Discovery.SessionExpiry = 24 * time.Hour
	}
	if c.Discovery.ChallengeExpiry == 0 {
		c.Discovery.ChallengeExpiry = 30 * time.Second
	}

	// Security defaults
	if c.Security.RateLimitPerMin == 0 {
		c.Security.RateLimitPerMin = 60
	}
	if c.Security.MaxPeersPerIP == 0 {
		c.Security.MaxPeersPerIP = 10
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.MaxSizeMB == 0 {
		c.Logging.MaxSizeMB = 100
	}
	if c.Logging.MaxBackups == 0 {
		c.Logging.MaxBackups = 3
	}
}

// validate checks if configuration is valid
func (c *Config) validate() error {
	// Validate server config
	if c.Server.HTTPPort < 1 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}
	if c.Server.HTTPSPort < 1 || c.Server.HTTPSPort > 65535 {
		return fmt.Errorf("invalid HTTPS port: %d", c.Server.HTTPSPort)
	}

	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate Redis config
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	// Validate discovery config
	if c.Discovery.PeerID == "" {
		return fmt.Errorf("peer ID is required")
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", c.Logging.Level)
	}

	return nil
}

// GenerateDefaultConfig creates a default config file
func GenerateDefaultConfig(region string) *Config {
	return &Config{
		Server: ServerConfig{
			HTTPPort:  8080,
			HTTPSPort: 8443,
			TLSCert:   "/etc/shadowmesh/tls/cert.pem",
			TLSKey:    "/etc/shadowmesh/tls/key.pem",
			Region:    region,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "shadowmesh",
			Password: "changeme",
			DBName:   "shadowmesh",
			SSLMode:  "disable",
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      5 * time.Minute,
		},
		Discovery: DiscoveryConfig{
			PeerID:          "generate-random-peer-id",
			KBucketSize:     20,
			RefreshInterval: 1 * time.Hour,
			CleanupInterval: 1 * time.Hour,
			SessionExpiry:   24 * time.Hour,
			ChallengeExpiry: 30 * time.Second,
		},
		Security: SecurityConfig{
			RequireAuth:     true,
			AllowedOrigins:  []string{"*"},
			RateLimitPerMin: 60,
			MaxPeersPerIP:   10,
		},
		Logging: LoggingConfig{
			Level:      "info",
			OutputFile: "/var/log/shadowmesh/discovery.log",
			MaxSizeMB:  100,
			MaxBackups: 3,
		},
	}
}

// WriteConfigFile writes a config struct to a YAML file
func WriteConfigFile(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
