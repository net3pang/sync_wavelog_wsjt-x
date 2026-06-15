package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	WavelogURL string `json:"wavelog_url"`
	APIKey     string `json:"api_key"`
	StationID  string `json:"station_id"`
	LogPath    string `json:"wsjtx_log_path"`
}

func Default() *Config {
	return &Config{
		WavelogURL: "",
		APIKey:     "",
		StationID:  "",
		LogPath:    defaultLogPath(),
	}
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "wavelog_sync_config.json")
}

func defaultLogPath() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("USERPROFILE"),
			"AppData", "Local", "WSJT-X", "wsjtx_log.adi")
	}
	return filepath.Join(home,
		"Library", "Application Support", "WSJT-X", "wsjtx_log.adi")
}

func Load() *Config {
	cfg := Default()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, cfg)
	return cfg
}

func Save(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}
