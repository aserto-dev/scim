package config

import (
	"os"
	"strings"

	"github.com/aserto-dev/go-aserto/client"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Directory client.Config `json:"directory"`
	Server    struct {
		ListenAddress string `json:"listen_address"`
		Auth          struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Token    string `json:"token"`
		} `json:"auth"`
	} `json:"server"`

	SCIM struct {
		CreateEmailIdentities bool `json:"create_email_identities"`
		CreateRoleGroups      bool `json:"create_role_groups"`
	} `json:"scim"`
}

func NewConfig(configPath string) (*Config, error) { // nolint // function will contain repeating statements for defaults
	file := "config.yaml"
	v := viper.New()

	if configPath != "" {
		exists, err := fileExists(string(configPath))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to determine if config file '%s' exists", configPath)
		}

		if !exists {
			return nil, errors.Errorf("config file '%s' doesn't exist", configPath)
		}

		file = string(configPath)
	}

	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.SetConfigFile(file)
	v.SetEnvPrefix("ASERTO_SCIM")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Set defaults.
	v.SetDefault("server.listen_address", ":8080")

	configExists, err := fileExists(file)
	if err != nil {
		return nil, errors.Wrapf(err, "filesystem error")
	}

	if configExists {
		if err = v.ReadInConfig(); err != nil {
			return nil, errors.Wrapf(err, "failed to read config file '%s'", file)
		}
	}
	v.AutomaticEnv()

	cfg := new(Config)

	err = v.UnmarshalExact(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "json"
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config file")
	}

	return cfg, nil
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, errors.Wrapf(err, "failed to stat file '%s'", path)
	}
}
