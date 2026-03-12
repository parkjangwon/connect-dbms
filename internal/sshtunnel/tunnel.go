package sshtunnel

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	gossh "golang.org/x/crypto/ssh"

	"oslo/internal/db"
)

type Tunnel struct {
	listener net.Listener
	client   *gossh.Client
}

func (t *Tunnel) Close() error {
	if t == nil {
		return nil
	}
	if t.listener != nil {
		t.listener.Close()
	}
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

func PrepareConnConfig(cfg db.ConnConfig) (db.ConnConfig, io.Closer, error) {
	if cfg.SSH == nil {
		return cfg, nil, nil
	}
	if cfg.Host == "" || cfg.Port == 0 {
		return cfg, nil, fmt.Errorf("ssh tunnel requires target host and port in the DB config")
	}

	auth, err := authMethod(*cfg.SSH)
	if err != nil {
		return cfg, nil, err
	}

	client, err := gossh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.SSH.Host, cfg.SSH.Port), &gossh.ClientConfig{
		User:            cfg.SSH.User,
		Auth:            []gossh.AuthMethod{auth},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		return cfg, nil, fmt.Errorf("open ssh client: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return cfg, nil, fmt.Errorf("open local tunnel listener: %w", err)
	}

	tunnel := &Tunnel{listener: listener, client: client}
	go acceptLoop(tunnel, cfg.Host, cfg.Port)

	localPort := listener.Addr().(*net.TCPAddr).Port
	cfg.Host = "127.0.0.1"
	cfg.Port = localPort
	return cfg, tunnel, nil
}

func authMethod(cfg db.SSHConfig) (gossh.AuthMethod, error) {
	if cfg.KeyPath != "" {
		key, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read ssh key: %w", err)
		}
		signer, err := gossh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse ssh key: %w", err)
		}
		return gossh.PublicKeys(signer), nil
	}
	if cfg.Password != "" {
		return gossh.Password(cfg.Password), nil
	}
	return nil, fmt.Errorf("ssh tunnel requires ssh_password or ssh_key_path")
}

func acceptLoop(tunnel *Tunnel, remoteHost string, remotePort int) {
	for {
		localConn, err := tunnel.listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer localConn.Close()
			remoteConn, err := tunnel.client.Dial("tcp", fmt.Sprintf("%s:%d", remoteHost, remotePort))
			if err != nil {
				return
			}
			defer remoteConn.Close()

			done := make(chan struct{}, 2)
			go proxyCopy(localConn, remoteConn, done)
			go proxyCopy(remoteConn, localConn, done)
			<-done
		}()
	}
}

func proxyCopy(dst net.Conn, src net.Conn, done chan<- struct{}) {
	_, _ = io.Copy(dst, src)
	done <- struct{}{}
}
