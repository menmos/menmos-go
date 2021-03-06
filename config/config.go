package config

import (
	"fmt"
	"os"
	"path"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

const menmosConfigDirName = "menmos"
const menmosConfigFileName = "client.toml"

// A Config represents the on-disk configuration of a menmos client.
type Config struct {
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

func loadConfigFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open menmos configuration file")
	}

	decoder := toml.NewDecoder(file).SetTagName("json")

	var cfg Config
	err = decoder.Decode(&cfg)

	if err != nil {
		err = errors.Wrap(err, "failed to decode TOML config")
	}

	return &cfg, err
}

func getDefaultConfigPath() (string, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to get the user configuration directory")
	}

	menmosConfigDirPath := path.Join(configPath, menmosConfigDirName)

	menmosConfigPath := path.Join(menmosConfigDirPath, menmosConfigFileName)
	return menmosConfigPath, nil
}

// LoadDefault loads a config from the default path.
func LoadOrCreateDefault() (*Config, error) {
	configPath, err := getDefaultConfigPath()
	if err != nil {
		return nil, err
	}

	configDir := path.Dir(configPath)
	if err := os.MkdirAll(configDir, 0644); err != nil {
		return nil, errors.Wrap(err, "failed to create menmos config directory")
	}

	config, err := loadConfigFromFile(configPath)
	return config, err
}

// LoadProfileFromDefaultConfig is a utility method for loading a single profile from the default config location.
func LoadProfileFromDefaultConfig(profileName string) (*Profile, error) {
	config, err := LoadOrCreateDefault()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read profile from configuration")
	}

	if profile, ok := config.Profiles[profileName]; ok {
		return &profile, nil
	}

	return nil, errors.New(fmt.Sprintf("profile '%s' not found", profileName))
}
