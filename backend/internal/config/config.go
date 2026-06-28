package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DatabasePath    string
	PhotoStorageDir string
	StorageDir      string
	AllowedOrigin   string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	SSH             SSHConfig
	Network         NetworkConfig
	Cinema          CinemaConfig
}

// CinemaConfig controls the server-side catalog scraper. Sources is an
// optional comma separated allowlist of source ids; empty enables all.
type CinemaConfig struct {
	Sources        string
	RequestTimeout time.Duration
	UserAgent      string
}

// SSHConfig describes the single remote host the web terminal connects to.
// Credentials live only on the server; clients never choose the target.
type SSHConfig struct {
	Enabled               bool
	Host                  string
	Port                  string
	User                  string
	Password              string
	PrivateKeyPath        string
	PrivateKeyPassphrase  string
	KnownHostsPath        string
	InsecureIgnoreHostKey bool
	ConnectTimeout        time.Duration
}

type NetworkConfig struct {
	ARPPath       string
	DHCPLeasePath string
	DNSLogPath    string
	BlocklistPath string
}

func Load() Config {
	return Config{
		Addr:            getEnv("HTTP_ADDR", ":8080"),
		DatabasePath:    getEnv("DATABASE_PATH", "data/homelab.db"),
		PhotoStorageDir: getEnv("PHOTO_STORAGE_DIR", "data/photos"),
		StorageDir:      getEnv("STORAGE_DIR", "data/storage"),
		AllowedOrigin:   getEnv("ALLOWED_ORIGIN", "http://localhost:5173,http://localhost:5174"),
		ReadTimeout:     getDurationSeconds("HTTP_READ_TIMEOUT_SECONDS", 5),
		WriteTimeout:    getDurationSeconds("HTTP_WRITE_TIMEOUT_SECONDS", 10),
		IdleTimeout:     getDurationSeconds("HTTP_IDLE_TIMEOUT_SECONDS", 60),
		ShutdownTimeout: getDurationSeconds("HTTP_SHUTDOWN_TIMEOUT_SECONDS", 10),
		SSH: SSHConfig{
			Enabled:               getBool("SSH_ENABLED", false),
			Host:                  getEnv("SSH_HOST", "127.0.0.1"),
			Port:                  getEnv("SSH_PORT", "22"),
			User:                  getEnv("SSH_USER", ""),
			Password:              getEnv("SSH_PASSWORD", ""),
			PrivateKeyPath:        getEnv("SSH_PRIVATE_KEY_PATH", ""),
			PrivateKeyPassphrase:  getEnv("SSH_PRIVATE_KEY_PASSPHRASE", ""),
			KnownHostsPath:        getEnv("SSH_KNOWN_HOSTS_PATH", ""),
			InsecureIgnoreHostKey: getBool("SSH_INSECURE_IGNORE_HOST_KEY", false),
			ConnectTimeout:        getDurationSeconds("SSH_CONNECT_TIMEOUT_SECONDS", 10),
		},
		Network: NetworkConfig{
			ARPPath:       getEnv("NETWORK_ARP_PATH", "/proc/net/arp"),
			DHCPLeasePath: getEnv("NETWORK_DHCP_LEASE_PATH", ""),
			DNSLogPath:    getEnv("NETWORK_DNS_LOG_PATH", ""),
			BlocklistPath: getEnv("NETWORK_BLOCKLIST_PATH", "data/network-blocklist.txt"),
		},
		Cinema: CinemaConfig{
			Sources:        getEnv("CINEMA_SOURCES", ""),
			RequestTimeout: getDurationSeconds("CINEMA_REQUEST_TIMEOUT_SECONDS", 12),
			UserAgent:      getEnv("CINEMA_USER_AGENT", ""),
		},
	}
}

func getBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}

	return value
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getDurationSeconds(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		value = fallback
	}

	return time.Duration(value) * time.Second
}
