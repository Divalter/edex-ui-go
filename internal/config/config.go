package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Shell                string            `json:"shell"`
	ShellArgs            []string          `json:"shellArgs"`
	Cwd                  string            `json:"cwd"`
	Env                  map[string]string `json:"env"`
	Theme                string            `json:"theme"`
	Keyboard             string            `json:"keyboard"`
	TermFontSize         int               `json:"termFontSize"`
	Audio                bool              `json:"audio"`
	AudioVolume          float64           `json:"audioVolume"`
	DisableFeedbackAudio bool              `json:"disableFeedbackAudio"`
	PingAddr             string            `json:"pingAddr"`
	NoIntro              bool              `json:"noIntro"`
	NoCursor             bool              `json:"noCursor"`
	AllowWindowed        bool              `json:"allowWindowed"`
}

func GetConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "edex-ui-go")
}

func Load() (*Config, error) {
	configPath := filepath.Join(GetConfigDir(), "config.json")
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := Save(cfg); err != nil {
			return cfg, err // Return defaults even if save fails
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)
}
