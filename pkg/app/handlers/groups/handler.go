package groups

import (
	"github.com/aserto-dev/scim/directory"
	"github.com/aserto-dev/scim/pkg/config"
)

type GroupResourceHandler struct {
	dirClient *directory.DirectoryClient
	cfg       *config.Config
}

func NewGroupResourceHandler(cfg *config.Config) (*GroupResourceHandler, error) {
	dirClient, err := directory.GetDirectoryClient(&cfg.Directory)
	if err != nil {
		return nil, err
	}
	return &GroupResourceHandler{
		dirClient: dirClient,
		cfg:       cfg,
	}, nil
}
