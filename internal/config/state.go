package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type State struct {
	Server  string `yaml:"server,omitempty"`
	Session string `yaml:"session,omitempty"`
}

func statePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "state.yaml")
}

func LoadState(configPath string) State {
	data, err := os.ReadFile(statePath(configPath))
	if err != nil {
		return State{}
	}
	var s State
	yaml.Unmarshal(data, &s)
	return s
}

func SaveState(configPath string, s State) {
	data, _ := yaml.Marshal(s)
	os.MkdirAll(filepath.Dir(statePath(configPath)), 0o755)
	os.WriteFile(statePath(configPath), data, 0o644)
}
