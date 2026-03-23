package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ConfigData struct {
	GfgUserName  string `json:"gfguserName,omitempty"`
	SessionID    string `json:"sessionid,omitempty"`
	CookieString string `json:"cookie_string,omitempty"`
	Language     string `json:"language,omitempty"`
}

var (
	Cfg        ConfigData
	configPath string
)

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = os.TempDir()
	}
	appDir := filepath.Join(userConfigDir, "gfg-cli")
	os.MkdirAll(appDir, 0o755)
	configPath = filepath.Join(appDir, "config.json")
	Load()
}

func Load() {
	Cfg = ConfigData{Language: "cpp"} // default
	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &Cfg)
	}
	if Cfg.Language == "" {
		Cfg.Language = "cpp"
	}
}

func Save() error {
	data, err := json.MarshalIndent(Cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0o644)
}

func GetConfigPath() string {
	return configPath
}
