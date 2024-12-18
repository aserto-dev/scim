package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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
)

type TransformConfigMap map[string]interface{}

type Config struct {
	Logging   logger.Config `json:"logging"`
	Directory client.Config `json:"directory"`
	RootDS    client.Config `json:"root_ds"`
	Server    struct {
		ListenAddress string           `json:"listen_address"`
		Certs         client.TLSConfig `json:"certs"`
		Auth          AuthConfig       `json:"auth"`
	} `json:"server"`

	SCIM struct {
		SCIMConfigKey     string          `json:"scim_config_key"`
		TransformDefaults TransformConfig `json:"transform_defaults"`
	} `json:"scim"`
}

type TransformConfig struct {
	CreateEmailIdentities bool              `json:"create_email_identities"`
	CreateRoleGroups      bool              `json:"create_role_groups"`
	Template              string            `json:"template"`
	UserObjectType        string            `json:"user_object_type"`
	GroupMemberRelation   string            `json:"group_member_relation"`
	GroupObjectType       string            `json:"group_object_type"`
	IdentityObjectType    string            `json:"identity_object_type"`
	IdentityRelation      string            `json:"identity_relation"`
	RoleObjectType        string            `json:"role_object_type"`
	RoleRelation          string            `json:"role_relation"`
	SourceUserType        string            `json:"source_user_type"`
	SourceGroupType       string            `json:"source_group_type"`
	GroupMappings         []ObjectMapping   `json:"group_mappings"`
	UserMappings          []ObjectMapping   `json:"user_mappings"`
	ManagerRelation       string            `json:"manager_relation"`
	UserPropertiesMapping map[string]string `json:"user_properties_mapping"`
}

func (t *TransformConfig) ToMap() (map[string]interface{}, error) {
	var result map[string]interface{}
	cfg, err := json.Marshal(t)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransformConfig to json")
	}
	err = json.Unmarshal(cfg, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal TransformConfig to map")
	}

	return result, nil
}

type ObjectMapping struct {
	SubjectID       string `json:"subject_id"`
	ObjectType      string `json:"object_type"`
	ObjectID        string `json:"object_id"`
	Relation        string `json:"relation"`
	SubjectRelation string `json:"subject_relation"`
}

type AuthConfig struct {
	Anonymous bool `json:"anonymous"`
	Basic     struct {
		Enabled     bool   `json:"enabled"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		Passthrough bool   `json:"passthrough"`
	} `json:"basic"`
	Bearer struct {
		Enabled     bool   `json:"enabled"`
		Token       string `json:"token"`
		Passthrough bool   `json:"passthrough"`
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
	v.SetDefault("server.certs.tls_key_path", filepath.Join(DefaultTLSGenDir, "grpc.key"))
	v.SetDefault("server.certs.tls_cert_path", filepath.Join(DefaultTLSGenDir, "grpc.crt"))
	v.SetDefault("server.certs.tls_ca_cert_path", filepath.Join(DefaultTLSGenDir, "grpc-ca.crt"))

	v.SetDefault("scim.scim_config_key", "scim_config")

	v.SetDefault("scim.transform_defaults.create_email_identities", true)
	v.SetDefault("scim.transform_defaults.group_object_type", "group")
	v.SetDefault("scim.transform_defaults.group_member_relation", "member")
	v.SetDefault("scim.transform_defaults.identity_object_type", "identity")
	v.SetDefault("scim.transform_defaults.identity_relation", "identifier")
	v.SetDefault("scim.transform_defaults.manager_relation", "manager")
	v.SetDefault("scim.transform_defaults.source_group_type", "scim.2.0.group")
	v.SetDefault("scim.transform_defaults.source_user_type", "scim.2.0.user")
	v.SetDefault("scim.transform_defaults.template", "users-groups-roles-v1")
	v.SetDefault("scim.transform_defaults.user_object_type", "user")

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
