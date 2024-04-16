package groups

import (
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/rs/zerolog"
)

const (
	GroupMembers = "members"
)

type GroupResourceHandler struct {
	dirClient *directory.DirectoryClient
	cfg       *config.Config
	logger    *zerolog.Logger
}

func NewGroupResourceHandler(cfg *config.Config, logger *zerolog.Logger) (*GroupResourceHandler, error) {
	groupLogger := logger.With().Str("component", "groups").Logger()
	dirClient, err := directory.GetDirectoryClient(&cfg.Directory)
	if err != nil {
		return nil, err
	}
	return &GroupResourceHandler{
		dirClient: dirClient,
		cfg:       cfg,
		logger:    &groupLogger,
	}, nil
}
