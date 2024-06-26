package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aserto-dev/certs"
	"github.com/aserto-dev/go-aserto/client"
	"github.com/aserto-dev/logger"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

var (
	DefaultTLSGenDir = os.ExpandEnv("$HOME/.config/aserto/scim/certs")
)

type Config struct {
	Logging   logger.Config `json:"logging"`
	Directory client.Config `json:"directory"`
	Server    struct {
		ListenAddress string               `json:"listen_address"`
		Certs         certs.TLSCredsConfig `json:"certs"`
		Auth          AuthConfig           `json:"auth"`
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

func NewConfig(configPath string, log *zerolog.Logger, certsGenerator *certs.Generator) (*Config, error) { // nolint // function will contain repeating statements for defaults
	configLogger := log.With().Str("component", "config").Logger()
	log = &configLogger

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

	v.SetDefault("scim.create_email_identities", true)
	v.SetDefault("scim.user_object_type", "user")
	v.SetDefault("scim.identity_object_type", "identity")
	v.SetDefault("scim.identity_relation", "identifier")
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

	if certsGenerator != nil {
		err = cfg.setupCerts(log, certsGenerator)
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup certs")
		}
	}

	return cfg, nil
}

func NewLoggerConfig(configPath string) (*logger.Config, error) {
	discardLogger := zerolog.New(io.Discard)
	cfg, err := NewConfig(configPath, &discardLogger, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new config")
	}

	return &cfg.Logging, nil
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

func (c *Config) setupCerts(log *zerolog.Logger, certsGenerator *certs.Generator) error {
	existingFiles := []string{}
	for _, file := range []string{
		c.Server.Certs.TLSCACertPath,
		c.Server.Certs.TLSCertPath,
		c.Server.Certs.TLSKeyPath,
	} {
		exists, err := fileExists(file)
		if err != nil {
			return errors.Wrapf(err, "failed to determine if file '%s' exists", file)
		}

		if !exists {
			continue
		}

		existingFiles = append(existingFiles, file)
	}

	if len(existingFiles) == 0 {
		err := certsGenerator.MakeDevCert(&certs.CertGenConfig{
			CommonName:       "aserto-scim",
			CertKeyPath:      c.Server.Certs.TLSKeyPath,
			CertPath:         c.Server.Certs.TLSCertPath,
			CACertPath:       c.Server.Certs.TLSCACertPath,
			DefaultTLSGenDir: DefaultTLSGenDir,
		})
		if err != nil {
			return errors.Wrap(err, "failed to generate gateway certs")
		}
	} else {
		msg := zerolog.Arr()
		for _, f := range existingFiles {
			msg.Str(f)
		}
		log.Info().Array("existing-files", msg).Msg("some cert files already exist, skipping generation")
	}

	return nil
}
