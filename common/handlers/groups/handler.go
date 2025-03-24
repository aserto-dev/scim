package groups

import (
	"github.com/aserto-dev/go-aserto/ds/v3"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/aserto-dev/scim/common/directory"
	"github.com/rs/zerolog"
)

type GroupResourceHandler struct {
	cfg       *convert.TransformConfig
	logger    *zerolog.Logger
	dirClient *directory.Client
}

func NewGroupResourceHandler(logger *zerolog.Logger,
	cfg *convert.TransformConfig,
	dsClient *ds.Client,
) (*GroupResourceHandler, error) {
	groupLogger := logger.With().Str("component", "groups-handler").Logger()
	dirClient := directory.NewDirectoryClient(cfg, &groupLogger, dsClient)

	return &GroupResourceHandler{
		cfg:       cfg,
		logger:    &groupLogger,
		dirClient: dirClient,
	}, nil
}
