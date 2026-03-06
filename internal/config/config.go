package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Server struct {
	Name    string `yaml:"name"`
	Host    string `yaml:"host"`
	KeyFile string `yaml:"key_file,omitempty"`
}

func (s Server) IsLocal() bool {
	return s.Host == "local" || s.Host == "localhost"
}

type Config struct {
	Servers []Server `yaml:"servers"`
}

const defaultConfigDir = ".config/tmux-manager"
const defaultConfigFile = "config.yaml"

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultConfigDir, defaultConfigFile)
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("no servers defined in %s", path)
	}

	for i, s := range cfg.Servers {
		if s.Name == "" {
			return nil, fmt.Errorf("server %d: missing name", i)
		}
		if s.Host == "" {
			return nil, fmt.Errorf("server %q: missing host", s.Name)
		}
	}


	ensureLocal(&cfg)
	return &cfg, nil
}

var localServer = Server{Name: "local", Host: "local"}

func ensureLocal(cfg *Config) {
	for _, s := range cfg.Servers {
		if s.IsLocal() {
			return
		}
	}
	// Prepend local as the first entry
	cfg.Servers = append([]Server{localServer}, cfg.Servers...)
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
