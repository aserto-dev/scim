package groups

import (
	"net/http"

	"github.com/aserto-dev/go-aserto/client"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/convert"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/rs/zerolog"
)

const (
	GroupMembers = "members"
)

type GroupResourceHandler struct {
	cfg       *config.Config
	logger    *zerolog.Logger
	converter *convert.Converter
}

func NewGroupResourceHandler(cfg *config.Config, logger *zerolog.Logger) *GroupResourceHandler {
	groupLogger := logger.With().Str("component", "groups").Logger()

	return &GroupResourceHandler{
		cfg:    cfg,
		logger: &groupLogger,
	}
}

func (u GroupResourceHandler) getDirectoryClient(r *http.Request) (*directory.DirectoryClient, error) {
	tenantID := r.Context().Value(common.ContextKeyTenantID)
	apiKey := r.Context().Value(common.ContextKeyAPIKey)
	if tenantID == nil || apiKey == nil {
		return directory.GetDirectoryClient(r.Context(), &u.cfg.Directory)
	}

	dirCfg := &client.Config{
		Address:          u.cfg.Directory.Address,
		TenantID:         tenantID.(string),
		Insecure:         u.cfg.Directory.Insecure,
		APIKey:           apiKey.(string),
		TimeoutInSeconds: u.cfg.Directory.TimeoutInSeconds,
	}
	return directory.GetDirectoryClient(r.Context(), dirCfg)
}
