package config

import (
	"os"
	"strings"
	"time"

	client "github.com/aserto-dev/go-aserto"
	"github.com/aserto-dev/logger"
	config "github.com/aserto-dev/scim/common/config"
	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

const (
	DefaultReadTimeout       = 5 * time.Second
	DefaultReadHeaderTimeout = 2 * time.Second
	DefaultWriteTimeout      = 10 * time.Second
	DefaultIdleTimeout       = 30 * time.Second
)

var (
	DefaultTLSGenDir = os.ExpandEnv("$HOME/.config/aserto/scim/certs")
	ErrInvalidConfig = errors.New("invalid config")
)

type Config struct {
	Logging   logger.Config `json:"logging"`
	Directory client.Config `json:"directory"`
	Server    struct {
		ListenAddress     string           `json:"listen_address"`
		Certs             client.TLSConfig `json:"certs"`
		Auth              AuthConfig       `json:"auth"`
		ReadTimeout       time.Duration    `json:"read_timeout"`
		ReadHeaderTimeout time.Duration    `json:"read_header_timeout"`
		WriteTimeout      time.Duration    `json:"write_timeout"`
		IdleTimeout       time.Duration    `json:"idle_timeout"`
	} `json:"server"`

	SCIM         config.Config `json:"scim"`
	TemplateFile string        `json:"template_file"`
}

type AuthConfig struct {
	Basic struct {
		Enabled  bool   `json:"enabled"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"basic"`
	Bearer struct {
		Enabled bool   `json:"enabled"`
		Token   string `json:"token"`
	} `json:"bearer"`
}

func NewConfig(configPath string) (*Config, error) {
	file := "config.yaml"
	v := viper.New()

	if configPath != "" {
		exists, err := fileExists(configPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to determine if config file '%s' exists", configPath)
		}

		if !exists {
			return nil, errors.Errorf("config file '%s' doesn't exist", configPath)
		}

		file = configPath
	}

	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.SetConfigFile(file)
	v.SetEnvPrefix("ASERTO_SCIM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults.
	v.SetDefault("server.listen_address", ":8080")
	v.SetDefault("server.auth.basic.enabled", "false")
	v.SetDefault("server.auth.bearer.enabled", "false")

	v.SetDefault("server.read_timeout", DefaultReadTimeout)
	v.SetDefault("server.read_header_timeout", DefaultReadHeaderTimeout)
	v.SetDefault("server.write_timeout", DefaultWriteTimeout)
	v.SetDefault("server.idle_timeout", DefaultIdleTimeout)

	v.SetDefault("scim.user.object_type", "user")
	v.SetDefault("scim.user.identity_object_type", "identity")
	v.SetDefault("scim.user.identity_relation", "user#identifier")
	v.SetDefault("scim.user.source_object_type", "scim-user")
	v.SetDefault("scim.group.object_type", "group")
	v.SetDefault("scim.group.group_member_relation", "member")
	v.SetDefault("scim.group.source_object_type", "scim-group")

	// Allow setting via env vars.
	v.SetDefault("directory.api_key", "")
	v.SetDefault("server.auth.basic.password", "")
	v.SetDefault("server.auth.bearer.token", "")

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

	if cfg.Logging.LogLevel == "" {
		cfg.Logging.LogLevelParsed = zerolog.InfoLevel
	} else {
		cfg.Logging.LogLevelParsed, err = zerolog.ParseLevel(cfg.Logging.LogLevel)
		if err != nil {
			return nil, errors.Wrapf(err, "logging.log_level failed to parse")
		}
	}

	err = cfg.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "config validation failed")
	}

	return cfg, nil
}

func (cfg *Config) Validate() error {
	return cfg.SCIM.Validate()
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
