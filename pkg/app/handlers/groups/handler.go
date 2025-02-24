package groups

import (
	"context"

	"github.com/aserto-dev/go-aserto/ds/v3"
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
	dirClient *ds.Client
	cfg       *config.Config
	logger    *zerolog.Logger
}

func NewGroupResourceHandler(cfg *config.Config, logger *zerolog.Logger) (*GroupResourceHandler, error) {
	dirClient, err := directory.GetDirectoryClient(&cfg.Directory)
	if err != nil {
		return nil, err
	}

	groupLogger := logger.With().Str("component", "groups").Logger()

	return &GroupResourceHandler{
		dirClient: dirClient,
		cfg:       cfg,
		logger:    &groupLogger,
	}, nil
}

func (u GroupResourceHandler) setGroupMappings(ctx context.Context, groupID string) error {
	for _, groupMap := range u.cfg.SCIM.GroupMappings {
		if groupMap.SubjectID == groupID {
			u.logger.Trace().Str("groupID", groupID).Str("relation", groupMap.Relation).Str("objectID", groupMap.ObjectID).Msg("setting group mapping")
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
				u.logger.Error().Err(err).Str("groupID", groupID).Str("relation", groupMap.Relation).Str("objectID", groupMap.ObjectID).Msg("failed to set group mapping")
				return err
			}
		}
	}
	return nil
}
