package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"oslo/internal/db"
)

type Profile struct {
	Name     string            `json:"name"`
	Driver   string            `json:"driver"`
	Host     string            `json:"host,omitempty"`
	Port     int               `json:"port,omitempty"`
	User     string            `json:"user,omitempty"`
	Password string            `json:"password,omitempty"`
	Database string            `json:"database,omitempty"`
	DSN      string            `json:"dsn,omitempty"`
	Options  map[string]string `json:"options,omitempty"`

	MaxOpenConns           int `json:"max_open_conns,omitempty"`
	MaxIdleConns           int `json:"max_idle_conns,omitempty"`
	ConnMaxLifetimeSeconds int `json:"conn_max_lifetime_seconds,omitempty"`

	SSHHost     string `json:"ssh_host,omitempty"`
	SSHPort     int    `json:"ssh_port,omitempty"`
	SSHUser     string `json:"ssh_user,omitempty"`
	SSHPassword string `json:"ssh_password,omitempty"`
	SSHKeyPath  string `json:"ssh_key_path,omitempty"`
}

func (p *Profile) ToConnConfig() db.ConnConfig {
	return db.ConnConfig{
		Driver:   p.Driver,
		Host:     p.Host,
		Port:     p.Port,
		User:     p.User,
		Password: p.Password,
		Database: p.Database,
		DSN:      p.DSN,
		Options:  p.Options,

		MaxOpenConns:           p.MaxOpenConns,
		MaxIdleConns:           p.MaxIdleConns,
		ConnMaxLifetimeSeconds: p.ConnMaxLifetimeSeconds,
		SSH:                    p.SSHConfig(),
	}
}

func (p *Profile) SSHConfig() *db.SSHConfig {
	if p.SSHHost == "" || p.SSHUser == "" {
		return nil
	}

	port := p.SSHPort
	if port == 0 {
		port = 22
	}

	return &db.SSHConfig{
		Host:     p.SSHHost,
		Port:     port,
		User:     p.SSHUser,
		Password: p.SSHPassword,
		KeyPath:  p.SSHKeyPath,
	}
}

type Config struct {
	Profiles []Profile `json:"profiles"`
}

type Store struct {
	Path   string `json:"-"`
	Config Config
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "connect-dbms", "config.json")
}

func Load(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath()
	}
	s := &Store{Path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := s.Save(); err != nil {
				return nil, err
			}
			return s, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &s.Config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return s, nil
}

func (s *Store) Save() error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(s.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(s.Path, data, 0600)
}

func (s *Store) Get(name string) (*Profile, error) {
	for i := range s.Config.Profiles {
		if s.Config.Profiles[i].Name == name {
			return &s.Config.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("session not found: %s", name)
}

func (s *Store) Add(p Profile) error {
	for _, existing := range s.Config.Profiles {
		if existing.Name == p.Name {
			return fmt.Errorf("session already exists: %s", p.Name)
		}
	}
	s.Config.Profiles = append(s.Config.Profiles, p)
	return s.Save()
}

func (s *Store) Update(name string, p Profile) error {
	for i := range s.Config.Profiles {
		if s.Config.Profiles[i].Name == name {
			p.Name = name
			s.Config.Profiles[i] = p
			return s.Save()
		}
	}
	return fmt.Errorf("session not found: %s", name)
}

func (s *Store) Remove(name string) error {
	for i, p := range s.Config.Profiles {
		if p.Name == name {
			s.Config.Profiles = append(s.Config.Profiles[:i], s.Config.Profiles[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("session not found: %s", name)
}

func (s *Store) List() []Profile {
	return s.Config.Profiles
}
