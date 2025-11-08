package persistence

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/shadowmesh/shadowmesh/pkg/discovery"
)

// PostgresStore handles PostgreSQL persistence
type PostgresStore struct {
	db *sql.DB
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(config Config) (*PostgresStore, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &PostgresStore{db: db}

	// Initialize schema
	if err := store.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("PostgreSQL connection established")
	return store, nil
}

// InitSchema creates necessary tables if they don't exist
func (ps *PostgresStore) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS peers (
		peer_id VARCHAR(64) PRIMARY KEY,
		public_key TEXT NOT NULL,
		ip_address VARCHAR(45) NOT NULL,
		port INTEGER NOT NULL,
		is_public BOOLEAN DEFAULT false,
		last_seen TIMESTAMP NOT NULL,
		verified BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_peers_last_seen ON peers(last_seen);
	CREATE INDEX IF NOT EXISTS idx_peers_is_public ON peers(is_public);
	CREATE INDEX IF NOT EXISTS idx_peers_verified ON peers(verified);

	CREATE TABLE IF NOT EXISTS sessions (
		session_token VARCHAR(64) PRIMARY KEY,
		peer_id VARCHAR(64) NOT NULL,
		created_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_peer_id ON sessions(peer_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

	CREATE TABLE IF NOT EXISTS challenges (
		challenge VARCHAR(64) PRIMARY KEY,
		timestamp BIGINT NOT NULL,
		expires_at BIGINT NOT NULL,
		used BOOLEAN DEFAULT false
	);

	CREATE INDEX IF NOT EXISTS idx_challenges_expires_at ON challenges(expires_at);
	`

	_, err := ps.db.Exec(schema)
	return err
}

// SavePeer saves or updates a peer in the database
func (ps *PostgresStore) SavePeer(peer *discovery.PeerInfo) error {
	query := `
		INSERT INTO peers (peer_id, public_key, ip_address, port, is_public, last_seen, verified, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (peer_id)
		DO UPDATE SET
			ip_address = EXCLUDED.ip_address,
			port = EXCLUDED.port,
			is_public = EXCLUDED.is_public,
			last_seen = EXCLUDED.last_seen,
			verified = EXCLUDED.verified,
			updated_at = NOW()
	`

	_, err := ps.db.Exec(query,
		peer.PeerID,
		peer.PublicKey,
		peer.IPAddress,
		peer.Port,
		peer.IsPublic,
		peer.LastSeen,
		peer.Verified,
	)

	return err
}

// GetPeer retrieves a peer by ID
func (ps *PostgresStore) GetPeer(peerID string) (*discovery.PeerInfo, error) {
	query := `
		SELECT peer_id, public_key, ip_address, port, is_public, last_seen, verified
		FROM peers
		WHERE peer_id = $1
	`

	var peer discovery.PeerInfo
	err := ps.db.QueryRow(query, peerID).Scan(
		&peer.PeerID,
		&peer.PublicKey,
		&peer.IPAddress,
		&peer.Port,
		&peer.IsPublic,
		&peer.LastSeen,
		&peer.Verified,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("peer not found")
	}
	if err != nil {
		return nil, err
	}

	return &peer, nil
}

// GetAllPeers retrieves all peers from the database
func (ps *PostgresStore) GetAllPeers() ([]*discovery.PeerInfo, error) {
	query := `
		SELECT peer_id, public_key, ip_address, port, is_public, last_seen, verified
		FROM peers
		ORDER BY last_seen DESC
	`

	rows, err := ps.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	peers := make([]*discovery.PeerInfo, 0)
	for rows.Next() {
		var peer discovery.PeerInfo
		err := rows.Scan(
			&peer.PeerID,
			&peer.PublicKey,
			&peer.IPAddress,
			&peer.Port,
			&peer.IsPublic,
			&peer.LastSeen,
			&peer.Verified,
		)
		if err != nil {
			return nil, err
		}
		peers = append(peers, &peer)
	}

	return peers, nil
}

// GetPublicPeers retrieves only public peers
func (ps *PostgresStore) GetPublicPeers() ([]*discovery.PeerInfo, error) {
	query := `
		SELECT peer_id, public_key, ip_address, port, is_public, last_seen, verified
		FROM peers
		WHERE is_public = true
		ORDER BY last_seen DESC
	`

	rows, err := ps.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	peers := make([]*discovery.PeerInfo, 0)
	for rows.Next() {
		var peer discovery.PeerInfo
		err := rows.Scan(
			&peer.PeerID,
			&peer.PublicKey,
			&peer.IPAddress,
			&peer.Port,
			&peer.IsPublic,
			&peer.LastSeen,
			&peer.Verified,
		)
		if err != nil {
			return nil, err
		}
		peers = append(peers, &peer)
	}

	return peers, nil
}

// DeletePeer removes a peer from the database
func (ps *PostgresStore) DeletePeer(peerID string) error {
	query := `DELETE FROM peers WHERE peer_id = $1`
	_, err := ps.db.Exec(query, peerID)
	return err
}

// DeleteStalePeers removes peers not seen in the specified duration
func (ps *PostgresStore) DeleteStalePeers(duration time.Duration) (int, error) {
	query := `DELETE FROM peers WHERE last_seen < $1`
	threshold := time.Now().Add(-duration)

	result, err := ps.db.Exec(query, threshold)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}

// SaveSession saves a session to the database
func (ps *PostgresStore) SaveSession(token, peerID string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (session_token, peer_id, created_at, expires_at)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (session_token) DO NOTHING
	`

	_, err := ps.db.Exec(query, token, peerID, expiresAt)
	return err
}

// GetSession retrieves a session by token
func (ps *PostgresStore) GetSession(token string) (peerID string, expiresAt time.Time, err error) {
	query := `SELECT peer_id, expires_at FROM sessions WHERE session_token = $1`
	err = ps.db.QueryRow(query, token).Scan(&peerID, &expiresAt)
	return
}

// DeleteExpiredSessions removes expired sessions
func (ps *PostgresStore) DeleteExpiredSessions() (int, error) {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`
	result, err := ps.db.Exec(query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}

// SaveChallenge saves a challenge to the database
func (ps *PostgresStore) SaveChallenge(challenge string, expiresAt int64) error {
	query := `
		INSERT INTO challenges (challenge, timestamp, expires_at, used)
		VALUES ($1, $2, $3, false)
		ON CONFLICT (challenge) DO NOTHING
	`

	_, err := ps.db.Exec(query, challenge, time.Now().Unix(), expiresAt)
	return err
}

// MarkChallengeUsed marks a challenge as used
func (ps *PostgresStore) MarkChallengeUsed(challenge string) error {
	query := `UPDATE challenges SET used = true WHERE challenge = $1`
	_, err := ps.db.Exec(query, challenge)
	return err
}

// DeleteExpiredChallenges removes expired challenges
func (ps *PostgresStore) DeleteExpiredChallenges() (int, error) {
	query := `DELETE FROM challenges WHERE expires_at < $1`
	result, err := ps.db.Exec(query, time.Now().Unix())
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}

// GetStats returns database statistics
func (ps *PostgresStore) GetStats() (map[string]interface{}, error) {
	var totalPeers, publicPeers, verifiedPeers int
	var totalSessions, totalChallenges int

	// Count peers
	ps.db.QueryRow("SELECT COUNT(*) FROM peers").Scan(&totalPeers)
	ps.db.QueryRow("SELECT COUNT(*) FROM peers WHERE is_public = true").Scan(&publicPeers)
	ps.db.QueryRow("SELECT COUNT(*) FROM peers WHERE verified = true").Scan(&verifiedPeers)

	// Count sessions and challenges
	ps.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&totalSessions)
	ps.db.QueryRow("SELECT COUNT(*) FROM challenges").Scan(&totalChallenges)

	return map[string]interface{}{
		"total_peers":     totalPeers,
		"public_peers":    publicPeers,
		"verified_peers":  verifiedPeers,
		"active_sessions": totalSessions,
		"pending_challenges": totalChallenges,
	}, nil
}

// Close closes the database connection
func (ps *PostgresStore) Close() error {
	log.Println("Closing PostgreSQL connection")
	return ps.db.Close()
}

// LoadPeersIntoKademlia loads all peers from database into Kademlia table
func (ps *PostgresStore) LoadPeersIntoKademlia(kt *discovery.KademliaTable) error {
	peers, err := ps.GetAllPeers()
	if err != nil {
		return err
	}

	for _, peer := range peers {
		if err := kt.AddPeer(peer); err != nil {
			log.Printf("Warning: failed to add peer %s to Kademlia: %v", peer.PeerID, err)
		}
	}

	log.Printf("Loaded %d peers from database into Kademlia table", len(peers))
	return nil
}
