package users

import (
	"net/http"

	"github.com/aserto-dev/go-aserto/client"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/rs/zerolog"
)

const (
	Emails          = "emails"
	Groups          = "groups"
	IdentityKindKey = "kind"
)

type UsersResourceHandler struct {
	cfg    *config.Config
	logger *zerolog.Logger
}

func NewUsersResourceHandler(cfg *config.Config, logger *zerolog.Logger) *UsersResourceHandler {
	usersLogger := logger.With().Str("component", "users").Logger()

	return &UsersResourceHandler{
		cfg:    cfg,
		logger: &usersLogger,
	}
}

func (u UsersResourceHandler) getDirectoryClient(r *http.Request) (*directory.DirectoryClient, error) {
	tenantID := r.Context().Value(common.ContextKeyTenantID)
	apiKey := r.Context().Value(common.ContextKeyAPIKey)
	if tenantID == nil {
		tenantID = u.cfg.Directory.TenantID
	}

	if apiKey == nil {
		apiKey = u.cfg.Directory.APIKey
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
