package groups

import (
	"net/http"

	"github.com/aserto-dev/go-aserto/client"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/rs/zerolog"
)

const (
	GroupMembers = "members"
)

type GroupResourceHandler struct {
	cfg       *config.Config
	logger    *zerolog.Logger
	converter *common.Converter
}

func NewGroupResourceHandler(cfg *config.Config, logger *zerolog.Logger) *GroupResourceHandler {
	groupLogger := logger.With().Str("component", "groups").Logger()

	return &GroupResourceHandler{
		cfg:    cfg,
		logger: &groupLogger,
		// converter: common.NewConverter(*cfg),
	}
}

// func (u GroupResourceHandler) setGroupMappings(ctx context.Context, dirClient *directory.DirectoryClient, groupID string) error {
// 	for _, groupMap := range u.cfg.SCIM.GroupMappings {
// 		if groupMap.SubjectID == groupID {
// 			_, err := dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
// 				Relation: &dsc.Relation{
// 					SubjectType:     u.cfg.SCIM.Transform.GroupObjectType,
// 					SubjectId:       groupID,
// 					Relation:        groupMap.Relation,
// 					ObjectType:      groupMap.ObjectType,
// 					ObjectId:        groupMap.ObjectID,
// 					SubjectRelation: groupMap.SubjectRelation,
// 				},
// 			})
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

func (u GroupResourceHandler) getDirectoryClient(r *http.Request) (*directory.DirectoryClient, error) {
	tenantID := r.Context().Value("aserto-tenant-id")
	apiKey := r.Context().Value("aserto-api-key")
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
