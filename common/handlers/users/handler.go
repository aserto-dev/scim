package users

import (
	"github.com/aserto-dev/go-aserto/ds/v3"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/aserto-dev/scim/common/directory"

	"github.com/rs/zerolog"
)

type UsersResourceHandler struct {
	cfg       *convert.TransformConfig
	logger    *zerolog.Logger
	dirClient *directory.Client
}

func NewUsersResourceHandler(logger *zerolog.Logger,
	cfg *convert.TransformConfig,
	dsClient *ds.Client,
) (*UsersResourceHandler, error) {
	usersLogger := logger.With().Str("component", "users-handler").Logger()

	dirClient := directory.NewDirectoryClient(cfg, &usersLogger, dsClient)

	return &UsersResourceHandler{
		cfg:       cfg,
		logger:    &usersLogger,
		dirClient: dirClient,
	}, nil
}
