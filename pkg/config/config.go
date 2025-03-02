package config

import (
	"os"
	"strings"

	client "github.com/aserto-dev/go-aserto"
	"github.com/aserto-dev/logger"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

var (
	DefaultTLSGenDir = os.ExpandEnv("$HOME/.config/aserto/scim/certs")
	ErrInvalidConfig = errors.New("invalid config")
)

type Config struct {
	Logging   logger.Config `json:"logging"`
	Directory client.Config `json:"directory"`
	Server    struct {
		ListenAddress string           `json:"listen_address"`
		Certs         client.TLSConfig `json:"certs"`
		Auth          AuthConfig       `json:"auth"`
	} `json:"server"`

	SCIM struct {
		CreateEmailIdentities bool            `json:"create_email_identities"`
		CreateRoleGroups      bool            `json:"create_role_groups"`
		GroupMappings         []ObjectMapping `json:"group_mappings"`
		UserMappings          []ObjectMapping `json:"user_mappings"`
		UserObjectType        string          `json:"user_object_type"`
		GroupMemberRelation   string          `json:"group_member_relation"`
		GroupObjectType       string          `json:"group_object_type"`
		IdentityObjectType    string          `json:"identity_object_type"`
		IdentityRelation      string          `json:"identity_relation"`
		Identity              struct {
			ObjectType string
			Relation   string
		} `json:"-"`
	} `json:"scim"`
}

type ObjectMapping struct {
	SubjectID       string `json:"subject_id"`
	ObjectType      string `json:"object_type"`
	ObjectID        string `json:"object_id"`
	Relation        string `json:"relation"`
	SubjectRelation string `json:"subject_relation"`
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

func NewConfig(configPath string) (*Config, error) { // nolint // function will contain repeating statements for defaults
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

	v.SetDefault("scim.create_email_identities", true)
	v.SetDefault("scim.user_object_type", "user")
	v.SetDefault("scim.identity_object_type", "identity")
	v.SetDefault("scim.identity_relation", "user#identifier")
	v.SetDefault("scim.group_object_type", "group")
	v.SetDefault("scim.group_member_relation", "member")

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
	if cfg.SCIM.UserObjectType == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.user_object_type is required")
	}
	if cfg.SCIM.IdentityObjectType == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.identity_object_type is required")
	}
	if cfg.SCIM.IdentityRelation == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.identity_relation is required")
	} else {
		object, relation, found := strings.Cut(cfg.SCIM.IdentityRelation, "#")
		if !found {
			return errors.Wrap(ErrInvalidConfig, "identity relation must be in the format object#relation")
		}
		if object != cfg.SCIM.IdentityObjectType && object != cfg.SCIM.UserObjectType {
			return errors.Wrapf(ErrInvalidConfig, "identity relation object type [%s] doesn't match user or identity type", object)
		}
		if relation == "" {
			return errors.Wrap(ErrInvalidConfig, "identity relation relation is required")
		}

		cfg.SCIM.Identity.ObjectType = object
		cfg.SCIM.Identity.Relation = relation
	}
	if cfg.SCIM.GroupObjectType == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.group_object_type is required")
	}
	if cfg.SCIM.GroupMemberRelation == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.group_member_relation is required")
	}

	return nil
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
