// / SPDX-LICENSE-IDENTIFIER: GPL2.0
// /
// / (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// / Created at 2025/11/14 星期五 20:17:28
// / Last modified at 2025/11/14 星期五 20:39:20
package main

import (
	// Standard library imports
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	// SQLite driver
	_ "github.com/mattn/go-sqlite3"
	// SSH library
	"golang.org/x/crypto/ssh"
)

// Config holds the server configuration parameters
type Config struct {
	ListenIP    string // IP address to listen on
	ListenPort  string // Port to listen on
	DatabaseLoc string // Path to SQLite database file
	LogFile     string // Path to log file (empty for stdout)
}

// SessionData represents all data collected about an SSH session
type SessionData struct {
	RemoteAddr           string // Client IP address
	Port                 string // Client port
	ClientVersion        string // SSH client version string
	SupportedAlgorithms  string // Supported encryption algorithms
	PublicKeyFingerprint string // Fingerprint of public key used
	Username             string // Username attempted
	Password             string // Password attempted
	Timestamp            string // Time of connection/attempt
	ConnectionError      string // Any connection error
	AuthAttempted        bool   // Whether authentication was attempted
	AuthMethod           string // Authentication method used
}

// Global variables
var (
	config       Config             // Server configuration
	db           *sql.DB            // Database connection
	hostKey      ssh.Signer         // SSH host key
	serverCtx    context.Context    // Server context
	serverCancel context.CancelFunc // Function to cancel server context
	wg           sync.WaitGroup     // Wait group for graceful shutdown
	listener     net.Listener       // Network listener
	listenerMtx  sync.Mutex         // Mutex for listener access
	logger       *log.Logger        // Logger instance
	logFile      *os.File           // Log file handle
	dbMutex      sync.Mutex         // Mutex for database access
)

// init initializes command-line flags
func init() {
	flag.StringVar(&config.ListenIP, "l", "0.0.0.0", "Server listen IP address")
	flag.StringVar(&config.ListenPort, "p", "22", "Server listen port")
	flag.StringVar(&config.DatabaseLoc, "d", "honeypot.db", "SQLite database file location")
	flag.StringVar(&config.LogFile, "log", "", "Path to log file (empty for stdout)")
}

// main is the entry point of the application
func Main() {
	flag.Parse()

	// Initialize logger
	if err := setupLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer closeLogger()

	// Create server context for graceful shutdown
	serverCtx, serverCancel = context.WithCancel(context.Background())
	defer serverCancel()

	// Verify configuration
	if err := verifyConfig(); err != nil {
		logger.Fatalf("Configuration error: %v", err)
	}

	// Initialize database
	if err := initDB(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Generate SSH host key
	var err error
	hostKey, err = generateHostKey()
	if err != nil {
		logger.Fatalf("Failed to generate host key: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Printf("Received signal: %v", sig)
		shutdownServer()
	}()

	// Start the SSH server
	if err := startServer(); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}

	// Wait for all connections to finish
	wg.Wait()
	logger.Println("Server shutdown complete")
}

// setupLogger initializes the logger based on configuration
func setupLogger() error {
	var output io.Writer = os.Stdout

	// If log file is specified, use it instead of stdout
	if config.LogFile != "" {
		f, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("error opening log file: %v", err)
		}
		logFile = f
		output = f
	}

	// Create logger with timestamp
	logger = log.New(output, "", log.LstdFlags)
	return nil
}

// closeLogger closes the log file if it's open
func closeLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// initDB initializes the SQLite database
func initDB() error {
	var err error
	// Open database with performance optimizations
	db, err = sql.Open("sqlite3", fmt.Sprintf("%s?_journal_mode=WAL&_cache_size=-10000", config.DatabaseLoc))
	if err != nil {
		return err
	}

	// Verify database connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Create tables and indexes if they don't exist
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS ssh_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		remote_addr TEXT NOT NULL,
		port INTEGER NOT NULL,
		client_version TEXT,
		supported_algorithms TEXT,
		public_key_fingerprint TEXT,
		username TEXT,
		password TEXT,
		auth_method TEXT,
		connection_error TEXT,
		auth_attempted BOOLEAN DEFAULT FALSE
	);

	CREATE INDEX IF NOT EXISTS idx_ssh_sessions_timestamp ON ssh_sessions(timestamp);
	CREATE INDEX IF NOT EXISTS idx_ssh_sessions_remote_addr ON ssh_sessions(remote_addr);
	CREATE INDEX IF NOT EXISTS idx_ssh_sessions_username ON ssh_sessions(username);
	CREATE INDEX IF NOT EXISTS idx_ssh_sessions_auth_attempted ON ssh_sessions(auth_attempted);

	PRAGMA journal_mode = WAL;
	PRAGMA page_size = 4096;
	PRAGMA cache_size = -10000;
	PRAGMA mmap_size = 268435456;
	`)

	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	// Configure database connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return nil
}

// formatTimestamp formats a time value for consistent database storage
func formatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

// isCancelled checks if the server is shutting down
func isCancelled() bool {
	select {
	case <-serverCtx.Done():
		return true
	default:
		return false
	}
}

// shutdownServer gracefully shuts down the server
func shutdownServer() {
	// Cancel the server context to signal shutdown
	serverCancel()

	// Close the listener
	listenerMtx.Lock()
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Printf("Error closing listener: %v", err)
		}
	}
	listenerMtx.Unlock()

	// Set a timeout for graceful shutdown
	timeout := 5 * time.Second
	logger.Printf("Waiting up to %v for connections to close...", timeout)

	shutdownDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownDone)
	}()

	// Wait for connections to close or timeout
	select {
	case <-shutdownDone:
		logger.Println("All connections closed gracefully")
	case <-time.After(timeout):
		logger.Println("Timeout reached, shutting down with active connections")
	}
}

// verifyConfig validates the server configuration
func verifyConfig() error {
	// Validate IP address
	if config.ListenIP != "0.0.0.0" {
		if net.ParseIP(config.ListenIP) == nil {
			return fmt.Errorf("invalid IP address: %s", config.ListenIP)
		}
	}

	// Validate port number
	if port, err := strconv.Atoi(config.ListenPort); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %s", config.ListenPort)
	}

	// Check for root privileges if needed
	if port, _ := strconv.Atoi(config.ListenPort); port <= 1024 && os.Geteuid() != 0 {
		return fmt.Errorf("ports below 1024 require root privileges")
	}

	return nil
}

// splitRemoteAddr splits a remote address into IP and port
func splitRemoteAddr(remoteAddr string) (string, string) {
	host, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If splitting fails, assume it's just the IP and use port 0
		return remoteAddr, "0"
	}
	return host, port
}

// startServer starts the SSH honeypot server
func startServer() error {
	// Configure SSH server
	sshConfig := &ssh.ServerConfig{
		// Password authentication callback
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			remoteHost, remotePort := splitRemoteAddr(c.RemoteAddr().String())

			// Save password attempt to database
			saveSessionData(SessionData{
				RemoteAddr:          remoteHost,
				Port:                remotePort,
				ClientVersion:       string(c.ClientVersion()),
				Username:            c.User(),
				Password:            string(pass),
				Timestamp:           formatTimestamp(time.Now()),
				AuthAttempted:       true,
				AuthMethod:          "password",
				SupportedAlgorithms: getSupportedAlgorithms(),
			})
			logAttempt(c, "password")
			// Always deny access
			return nil, fmt.Errorf("access denied")
		},
		// Public key authentication callback
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			remoteHost, remotePort := splitRemoteAddr(c.RemoteAddr().String())
			fp := fingerprintKey(key)

			// Save public key attempt to database
			saveSessionData(SessionData{
				RemoteAddr:           remoteHost,
				Port:                 remotePort,
				ClientVersion:        string(c.ClientVersion()),
				Username:             c.User(),
				PublicKeyFingerprint: fp,
				Timestamp:            formatTimestamp(time.Now()),
				AuthAttempted:        true,
				AuthMethod:           "public-key",
				SupportedAlgorithms:  getSupportedAlgorithms(),
			})
			logAttempt(c, "public-key ("+fp+")")
			// Always deny access
			return nil, fmt.Errorf("access denied")
		},
		NoClientAuth:    false,                                       // Require authentication
		ServerVersion:   "SSH-2.0-OpenSSH_9.6p1 Ubuntu-3ubuntu13.12", // Masquerade as OpenSSH
		MaxAuthTries:    3,                                           // Maximum authentication attempts
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {},
	}
	sshConfig.AddHostKey(hostKey)

	// Start TCP listener
	var err error
	listenerMtx.Lock()
	listener, err = net.Listen("tcp", net.JoinHostPort(config.ListenIP, config.ListenPort))
	listenerMtx.Unlock()

	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	logger.Printf("Go SSH Honeypot running on %s:%s (as OpenSSH 9.6p1)", config.ListenIP, config.ListenPort)
	logger.Printf("Press Ctrl+C to shutdown")

	// Start accepting connections in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			listenerMtx.Lock()
			if listener != nil {
				listener.Close()
			}
			listenerMtx.Unlock()
		}()

		for {
			// Accept new connections
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) || isCancelled() {
					return
				}
				logger.Printf("Error accepting connection: %v", err)
				continue
			}

			// Handle each connection in a separate goroutine
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				handleConnection(c, sshConfig)
			}(conn)
		}
	}()

	return nil
}

// handleConnection processes an incoming SSH connection
func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	remoteAddr := conn.RemoteAddr().String()
	remoteHost, remotePort := splitRemoteAddr(remoteAddr)

	// Save basic connection information
	saveSessionData(SessionData{
		RemoteAddr:    remoteHost,
		Port:          remotePort,
		Timestamp:     formatTimestamp(time.Now()),
		AuthAttempted: false,
	})

	defer func() {
		// Close connection when done
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			if !isCancelled() {
				logger.Printf("Error closing connection from %s: %v", remoteAddr, err)
			}
		}
	}()

	logger.Printf("New connection from %s:%s", remoteHost, remotePort)

	// Set connection timeout
	conn.SetDeadline(time.Now().Add(5 * time.Minute))

	// Channel to track when handling is complete
	done := make(chan struct{})
	go func() {
		defer close(done)

		// Perform SSH handshake
		sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
			// Save connection error if it wasn't just a disconnect
			saveSessionData(SessionData{
				RemoteAddr:      remoteHost,
				Port:            remotePort,
				Timestamp:       formatTimestamp(time.Now()),
				ConnectionError: err.Error(),
				AuthAttempted:   false,
			})

			if !isCancelled() && !strings.Contains(err.Error(), "ssh: disconnected") {
				logger.Printf("SSH handshake failed from %s:%s: %v", remoteHost, remotePort, err)
			}
			return
		}
		defer sshConn.Close()

		// Discard global requests
		go ssh.DiscardRequests(reqs)

		// Reject all channel requests
		for newChannel := range chans {
			newChannel.Reject(ssh.Prohibited, "access denied")
		}
	}()

	// Wait for handling to complete or server shutdown
	select {
	case <-done:
	case <-serverCtx.Done():
		conn.SetDeadline(time.Now())
		<-done
	}
}

// saveSessionData saves session information to the database
func saveSessionData(data SessionData) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Ensure thread-safe database access
		dbMutex.Lock()
		defer dbMutex.Unlock()

		// Check database connection
		if err := db.Ping(); err != nil {
			logger.Printf("Database connection error: %v. Reconnecting...", err)
			if err = initDB(); err != nil {
				logger.Printf("Failed to reconnect to database: %v", err)
				return
			}
		}

		// Convert port to integer
		portInt, err := strconv.Atoi(data.Port)
		if err != nil {
			portInt = 0
			logger.Printf("Invalid port number '%s', using 0 instead", data.Port)
		}

		// Insert session data
		_, err = db.Exec(`
		INSERT INTO ssh_sessions (
			timestamp, remote_addr, port, client_version,
			supported_algorithms, public_key_fingerprint,
			username, password, auth_method, connection_error, auth_attempted
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			data.Timestamp,
			data.RemoteAddr,
			portInt,
			data.ClientVersion,
			data.SupportedAlgorithms,
			data.PublicKeyFingerprint,
			data.Username,
			data.Password,
			data.AuthMethod,
			data.ConnectionError,
			data.AuthAttempted,
		)

		// Log errors (unless server is shutting down)
		if err != nil && !isCancelled() {
			logger.Printf("Error saving session to database: %v", err)
			logger.Printf("Failed data: %+v", data)
		} else {
			logger.Printf("Successfully saved session data for %s:%s", data.RemoteAddr, data.Port)
		}
	}()
}

// logAttempt logs an authentication attempt
func logAttempt(c ssh.ConnMetadata, authMethod string) {
	remoteHost, remotePort := splitRemoteAddr(c.RemoteAddr().String())
	logger.Printf("Login attempt from %s:%s - user: %s, method: %s", remoteHost, remotePort, c.User(), authMethod)
}

// getSupportedAlgorithms returns a string of supported algorithms for the fake SSH server
func getSupportedAlgorithms() string {
	return "kex:curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp256,ecdh-sha2-nistp384,ecdh-sha2-nistp521,diffie-hellman-group-exchange-sha256,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512,diffie-hellman-group14-sha256;" +
		"ciphers:chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-gcm@openssh.com,aes256-gcm@openssh.com;" +
		"macs:umac-64-etm@openssh.com,umac-128-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com,hmac-sha1-etm@openssh.com;" +
		"compression:none,zlib@openssh.com"
}

// fingerprintKey generates an MD5 fingerprint for a public key
func fingerprintKey(key ssh.PublicKey) string {
	hash := md5.Sum(key.Marshal())
	return hex.EncodeToString(hash[:])
}

// generateHostKey generates a new RSA host key for the SSH server
func generateHostKey() (ssh.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(privateKey)
}
