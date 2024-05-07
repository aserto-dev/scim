package groups

import (
	"context"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
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

func NewGroupResourceHandler(cfg *config.Config, logger *zerolog.Logger, dirClient *directory.DirectoryClient) *GroupResourceHandler {
	groupLogger := logger.With().Str("component", "groups").Logger()

	return &GroupResourceHandler{
		dirClient: dirClient,
		cfg:       cfg,
		logger:    &groupLogger,
	}
}

func (u GroupResourceHandler) setGroupMappings(ctx context.Context, groupID string) error {
	for _, groupMap := range u.cfg.SCIM.GroupMappings {
		if groupMap.SubjectID == groupID {
			_, err := u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectType:     u.cfg.SCIM.GroupObjectType,
					SubjectId:       groupID,
					Relation:        groupMap.Relation,
					ObjectType:      groupMap.ObjectType,
					ObjectId:        groupMap.ObjectID,
					SubjectRelation: groupMap.SubjectRelation,
				},
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
