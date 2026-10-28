package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// config is what Zip Transform remembers between runs: the MCreator folder
// and mods folders, when it had to be told where those are.
type config struct {
	MCreator    string   `json:"mcreator,omitempty"`
	ModsFolders []string `json:"modsFolders,omitempty"`
}

func configPath() string {
	if p := os.Getenv("ZIPTRANSFORM_CONFIG"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ziptransform", "config.json")
}

// loadConfig reads the configuration. A missing or unreadable file is the
// same as an empty one: nothing in it is worth refusing to start over.
func loadConfig() config {
	var c config
	if p := configPath(); p != "" {
		if data, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(data, &c)
		}
	}
	return c
}

func (c config) save() error {
	p := configPath()
	if p == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}
