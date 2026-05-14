package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port      int      `toml:"port"`
	Dir       string   `toml:"dir"`
	WatchExts []string `toml:"watch_exts"`
}

func loadConfig() Config {
	cfg := Config{Port: 5999, Dir: "."}

	exe, err := os.Executable()
	if err != nil {
		return cfg
	}

	configPath := filepath.Join(filepath.Dir(exe), "webs.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		log.Printf("Warning: Could not parse %s: %v", configPath, err)
	}

	return cfg
}
