package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/api"
	"github.com/shadowmesh/shadowmesh/pkg/authentication"
	"github.com/shadowmesh/shadowmesh/pkg/config"
	"github.com/shadowmesh/shadowmesh/pkg/discovery"
	"github.com/shadowmesh/shadowmesh/pkg/persistence"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "/etc/shadowmesh/discovery.yaml", "Path to configuration file")
	generateConfig := flag.Bool("generate-config", false, "Generate default config file")
	flag.Parse()

	// Generate default config if requested
	if *generateConfig {
		cfg := config.GenerateDefaultConfig("north_america")
		if err := config.WriteConfigFile(cfg, "discovery.yaml"); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		log.Println("Generated default config: discovery.yaml")
		return
	}

	// Load configuration
	log.Printf("Loading configuration from: %s", *configPath)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting ShadowMesh Discovery Node (Region: %s)", cfg.Server.Region)

	// Initialize PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	postgresConfig := persistence.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}
	postgres, err := persistence.NewPostgresStore(postgresConfig)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgres.Close()

	// Initialize Redis cache
	log.Println("Connecting to Redis...")
	redisConfig := persistence.RedisCacheConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		TTL:      cfg.Redis.TTL,
	}
	redis, err := persistence.NewRedisCache(redisConfig)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Initialize Kademlia routing table
	log.Printf("Initializing Kademlia DHT (Peer ID: %s)", cfg.Discovery.PeerID)
	kademlia := discovery.NewKademliaTable(cfg.Discovery.PeerID)

	// Load existing peers from database into Kademlia
	log.Println("Loading peers from database...")
	if err := postgres.LoadPeersIntoKademlia(kademlia); err != nil {
		log.Printf("Warning: failed to load peers from database: %v", err)
	}
	log.Printf("Loaded %d peers into Kademlia table", kademlia.Size())

	// Initialize authentication server
	log.Println("Initializing authentication server...")
	authServer := authentication.NewAuthServer()

	// Initialize HTTP API server
	log.Printf("Initializing HTTP API server (port %d)...", cfg.Server.HTTPPort)
	apiServer := api.NewAPIServer(cfg.Server.HTTPPort, authServer, kademlia)

	// Start background cleanup tasks
	go startCleanupTasks(cfg, postgres, redis, kademlia, authServer)

	// Start periodic peer persistence
	go startPeerPersistence(cfg, postgres, kademlia)

	// Start API server in goroutine
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	log.Println("Discovery node started successfully")
	log.Printf("HTTP API:    http://0.0.0.0:%d", cfg.Server.HTTPPort)
	log.Printf("Health check: http://0.0.0.0:%d/health", cfg.Server.HTTPPort)
	log.Printf("Stats:        http://0.0.0.0:%d/stats", cfg.Server.HTTPPort)

	// Wait for shutdown signal
	waitForShutdown(apiServer, postgres, redis)
}

// startCleanupTasks runs periodic cleanup for stale peers, sessions, and challenges
func startCleanupTasks(cfg *config.Config, postgres *persistence.PostgresStore, redis *persistence.RedisCache, kademlia *discovery.KademliaTable, authServer *authentication.AuthServer) {
	ticker := time.NewTicker(cfg.Discovery.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Running cleanup tasks...")

		// Cleanup stale peers (not seen in 24 hours)
		stalePeers := kademlia.CleanupStale()
		if stalePeers > 0 {
			log.Printf("Removed %d stale peers from Kademlia", stalePeers)
		}

		// Cleanup database
		deletedPeers, err := postgres.DeleteStalePeers(24 * time.Hour)
		if err != nil {
			log.Printf("Error deleting stale peers from database: %v", err)
		} else if deletedPeers > 0 {
			log.Printf("Deleted %d stale peers from database", deletedPeers)
		}

		// Cleanup expired sessions
		deletedSessions, err := postgres.DeleteExpiredSessions()
		if err != nil {
			log.Printf("Error deleting expired sessions: %v", err)
		} else if deletedSessions > 0 {
			log.Printf("Deleted %d expired sessions", deletedSessions)
		}

		// Cleanup expired challenges
		deletedChallenges, err := postgres.DeleteExpiredChallenges()
		if err != nil {
			log.Printf("Error deleting expired challenges: %v", err)
		} else if deletedChallenges > 0 {
			log.Printf("Deleted %d expired challenges", deletedChallenges)
		}

		// Cleanup in-memory auth server
		authServer.CleanupExpired()

		log.Printf("Cleanup complete. Active peers: %d", kademlia.Size())
	}
}

// startPeerPersistence periodically persists Kademlia peers to database
func startPeerPersistence(cfg *config.Config, postgres *persistence.PostgresStore, kademlia *discovery.KademliaTable) {
	ticker := time.NewTicker(5 * time.Minute) // Persist every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Persisting peers to database...")

		peers := kademlia.GetAllPeers()
		saved := 0
		for _, peer := range peers {
			if err := postgres.SavePeer(peer); err != nil {
				log.Printf("Error saving peer %s: %v", peer.PeerID, err)
			} else {
				saved++
			}
		}

		log.Printf("Persisted %d/%d peers to database", saved, len(peers))
	}
}

// waitForShutdown waits for shutdown signal and gracefully stops services
func waitForShutdown(apiServer *api.APIServer, postgres *persistence.PostgresStore, redis *persistence.RedisCache) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down gracefully...", sig)

	// Stop API server
	if err := apiServer.Stop(); err != nil {
		log.Printf("Error stopping API server: %v", err)
	}

	// Close database connections
	if err := postgres.Close(); err != nil {
		log.Printf("Error closing PostgreSQL: %v", err)
	}

	if err := redis.Close(); err != nil {
		log.Printf("Error closing Redis: %v", err)
	}

	log.Println("Shutdown complete")
	os.Exit(0)
}
