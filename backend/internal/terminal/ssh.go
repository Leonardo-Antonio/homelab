package terminal

import (
	"context"
	"fmt"
	"net"
	"os"

	"homelab/backend/internal/config"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// dial opens an SSH connection to the configured target using the request
// context for cancellation. Credentials come only from server-side config.
func dial(ctx context.Context, cfg config.SSHConfig) (*ssh.Client, error) {
	authMethods, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := hostKeyCallback(cfg)
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.ConnectTimeout,
	}

	address := net.JoinHostPort(cfg.Host, cfg.Port)

	dialer := net.Dialer{Timeout: cfg.ConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial ssh host: %w", err)
	}

	sshConn, channels, requests, err := ssh.NewClientConn(conn, address, clientConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake: %w", err)
	}

	return ssh.NewClient(sshConn, channels, requests), nil
}

func authMethods(cfg config.SSHConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if cfg.PrivateKeyPath != "" {
		key, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key: %w", err)
		}

		signer, err := parsePrivateKey(key, cfg.PrivateKeyPassphrase)
		if err != nil {
			return nil, err
		}

		methods = append(methods, ssh.PublicKeys(signer))
	}

	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no ssh credentials configured: set SSH_PASSWORD or SSH_PRIVATE_KEY_PATH")
	}

	return methods, nil
}

func parsePrivateKey(key []byte, passphrase string) (ssh.Signer, error) {
	if passphrase != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse encrypted private key: %w", err)
		}

		return signer, nil
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return signer, nil
}

func hostKeyCallback(cfg config.SSHConfig) (ssh.HostKeyCallback, error) {
	if cfg.KnownHostsPath != "" {
		callback, err := knownhosts.New(cfg.KnownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("load known_hosts: %w", err)
		}

		return callback, nil
	}

	if cfg.InsecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	return nil, fmt.Errorf("host key verification required: set SSH_KNOWN_HOSTS_PATH or SSH_INSECURE_IGNORE_HOST_KEY")
}
