package directory

import (
	"context"
	"errors"
	"slices"

	"github.com/aserto-dev/ds-load/sdk/common/msg"
	cerr "github.com/aserto-dev/errors"
	"github.com/aserto-dev/go-aserto/ds/v3"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Client struct {
	cfg    *convert.TransformConfig
	client *ds.Client
	logger *zerolog.Logger
}

func NewDirectoryClient(transformCfg *convert.TransformConfig, logger *zerolog.Logger, dirClient *ds.Client) *Client {
	return &Client{
		cfg:    transformCfg,
		logger: logger,
		client: dirClient,
	}
}

func (s *Client) DS() *ds.Client {
	return s.client
}

func (s *Client) SetUser(ctx context.Context, userID string, data *msg.Transform, userAttributes scim.ResourceAttributes) (scim.Meta, error) {
	logger := s.logger.With().Str("method", "SetUser").Str("id", userID).Logger()
	logger.Trace().Msg("set user")

	idRelation, err := s.cfg.ParseIdentityRelation(userID, "")
	if err != nil {
		return scim.Meta{}, err
	}

	relations, err := s.client.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               idRelation.GetObjectType(),
		ObjectId:                 idRelation.GetObjectId(),
		Relation:                 idRelation.GetRelation(),
		SubjectType:              idRelation.GetSubjectType(),
		SubjectId:                idRelation.GetSubjectId(),
		WithObjects:              false,
		WithEmptySubjectRelation: true,
	})
	if err != nil {
		st, ok := status.FromError(err)

		if ok && st.Code() != codes.NotFound {
			return scim.Meta{}, err
		}
	}

	result, addedIdentities, err := s.importObjects(ctx, data.GetObjects(), userAttributes)
	if err != nil {
		return result, err
	}

	logger.Trace().Any("identities", addedIdentities).Msg("added identities")

	for _, relation := range data.GetRelations() {
		logger.Trace().Any("relation", relation).Msg("setting relation")

		_, err := s.client.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: relation,
		})
		if err != nil {
			return result, err
		}
	}

	mErr := &multierror.Error{}

	for _, rel := range relations.GetResults() {
		if !slices.Contains(addedIdentities, rel.GetObjectId()) {
			logger.Trace().Str("identity", rel.GetObjectId()).Msg("deleting identity")

			_, err := s.client.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
				ObjectType:    s.cfg.User.IdentityObjectType,
				ObjectId:      rel.GetObjectId(),
				WithRelations: true,
			})
			if err != nil {
				mErr = multierror.Append(mErr, err)
				logger.Err(err).Str("identity", rel.GetObjectId()).Msg("failed to delete identity")
			}
		}
	}

	return result, mErr.ErrorOrNil()
}

func (s *Client) importObjects(ctx context.Context, objects []*dsc.Object, userAttributes scim.ResourceAttributes) (scim.Meta, []string, error) {
	var err error

	result := scim.Meta{}
	addedIdentities := make([]string, 0)

	for _, object := range objects {
		if object.GetType() == s.cfg.User.ObjectType {
			var userProperties map[string]any
			if object.GetProperties() == nil {
				userProperties = make(map[string]any)
			} else {
				userProperties = object.GetProperties().AsMap()
			}

			for key, value := range s.cfg.User.PropertyMapping {
				userProperties[key] = userAttributes[value]
			}

			object.Properties, err = structpb.NewStruct(userProperties)
			if err != nil {
				return result, addedIdentities, err
			}
		}

		resp, err := s.client.Writer.SetObject(ctx, &dsw.SetObjectRequest{
			Object: object,
		})
		if err != nil {
			if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
				return result, addedIdentities, serrors.ScimErrorUniqueness
			}

			return result, addedIdentities, err
		}

		if resp.GetResult().GetType() == s.cfg.User.IdentityObjectType {
			addedIdentities = append(addedIdentities, resp.GetResult().GetId())
		}

		if object.GetType() == s.cfg.User.ObjectType {
			err = s.setRelations(ctx, resp.GetResult().GetId(), resp.GetResult().GetType())
			if err != nil {
				return result, addedIdentities, err
			}

			createdAt := resp.GetResult().GetCreatedAt().AsTime()
			updatedAt := resp.GetResult().GetUpdatedAt().AsTime()
			result.Created = &createdAt
			result.LastModified = &updatedAt
			result.Version = resp.GetResult().GetEtag()
		}
	}

	return result, addedIdentities, nil
}

func (s *Client) DeleteUser(ctx context.Context, userID string) error {
	logger := s.logger.With().Str("method", "DeleteUser").Str("id", userID).Logger()
	logger.Trace().Msg("delete user")

	identityRelation, err := s.cfg.ParseIdentityRelation(userID, "")
	if err != nil {
		return err
	}

	relations, err := s.client.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: identityRelation.GetSubjectType(),
		SubjectId:   identityRelation.GetSubjectId(),
		ObjectType:  identityRelation.GetObjectType(),
		ObjectId:    identityRelation.GetObjectId(),
		Relation:    identityRelation.GetRelation(),
	})
	if err != nil {
		st, ok := status.FromError(err)

		if ok && st.Code() == codes.NotFound {
			return serrors.ScimErrorResourceNotFound(userID)
		}
	}

	for _, v := range relations.GetResults() {
		var objectID string

		switch v.GetObjectType() {
		case s.cfg.User.IdentityObjectType:
			objectID = v.GetObjectId()
		case s.cfg.User.ObjectType:
			objectID = v.GetSubjectId()
		default:
			return serrors.ScimErrorBadRequest("unexpected object type in identity relation")
		}

		logger.Trace().Str("id", v.GetObjectId()).Msg("deleting identity")

		_, err = s.client.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
			ObjectId:      objectID,
			ObjectType:    s.cfg.User.IdentityObjectType,
			WithRelations: true,
		})
		if err != nil {
			return err
		}
	}

	logger.Trace().Msg("deleting user")

	_, err = s.client.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    s.cfg.User.ObjectType,
		ObjectId:      userID,
		WithRelations: true,
	})

	return err
}

func (s *Client) SetGroup(ctx context.Context, groupID string, data *msg.Transform) (scim.Meta, error) {
	logger := s.logger.With().Str("method", "SetGroup").Str("id", groupID).Logger()
	logger.Trace().Msg("set group")

	if s.cfg.Group == nil {
		logger.Warn().Msg("groups not enabled")
		return scim.Meta{}, nil
	}

	existingRelations, err := s.getGroupRelations(ctx, groupID)
	if err != nil {
		return scim.Meta{}, err
	}

	result, err := s.processGroupObjects(ctx, data.GetObjects(), logger)
	if err != nil {
		return result, err
	}

	addedMembers, err := s.processGroupRelations(ctx, data.GetRelations(), logger)
	if err != nil {
		return result, err
	}

	if err := s.removeStaleRelations(ctx, existingRelations, addedMembers, groupID, logger); err != nil {
		return result, err
	}

	return result, nil
}

func (s *Client) getGroupRelations(ctx context.Context, groupID string) (*dsr.GetRelationsResponse, error) {
	relations, err := s.client.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               s.cfg.Group.ObjectType,
		ObjectId:                 groupID,
		Relation:                 s.cfg.Group.GroupMemberRelation,
		WithObjects:              false,
		WithEmptySubjectRelation: true,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() != codes.NotFound {
			return nil, err
		}
	}

	return relations, nil
}

func (s *Client) processGroupObjects(ctx context.Context, objects []*dsc.Object, logger zerolog.Logger) (scim.Meta, error) {
	var result scim.Meta

	for _, object := range objects {
		logger.Trace().Any("object", object).Msg("setting object")

		resp, err := s.client.Writer.SetObject(ctx, &dsw.SetObjectRequest{
			Object: object,
		})
		if err != nil {
			if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
				return result, serrors.ScimErrorUniqueness
			}

			return result, err
		}

		if object.GetType() == s.cfg.Group.ObjectType {
			if err := s.setRelations(ctx, resp.GetResult().GetId(), resp.GetResult().GetType()); err != nil {
				return result, err
			}

			result = s.updateMetaFromResponse(resp.GetResult())
		}
	}

	return result, nil
}

func (s *Client) processGroupRelations(ctx context.Context, relations []*dsc.Relation, logger zerolog.Logger) ([]string, error) {
	addedMembers := make([]string, 0)

	for _, relation := range relations {
		if relation.GetRelation() == s.cfg.Group.GroupMemberRelation {
			addedMembers = append(addedMembers, relation.GetSubjectId())
		}

		logger.Trace().Any("relation", relation).Msg("setting relation")

		if _, err := s.client.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: relation,
		}); err != nil {
			return nil, err
		}
	}

	return addedMembers, nil
}

func (s *Client) removeStaleRelations(ctx context.Context,
	relations *dsr.GetRelationsResponse,
	addedMembers []string,
	groupID string,
	logger zerolog.Logger,
) error {
	if relations == nil {
		return nil
	}

	for _, rel := range relations.GetResults() {
		if !slices.Contains(addedMembers, rel.GetSubjectId()) {
			logger.Trace().Str("id", rel.GetSubjectId()).Msg("deleting relation")

			if err := s.deleteGroupRelation(ctx, groupID, rel); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Client) deleteGroupRelation(ctx context.Context, groupID string, rel *dsc.Relation) error {
	_, err := s.client.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
		ObjectType:  s.cfg.Group.ObjectType,
		ObjectId:    groupID,
		Relation:    s.cfg.Group.GroupMemberRelation,
		SubjectId:   rel.GetSubjectId(),
		SubjectType: rel.GetSubjectType(),
	})

	return err
}

func (s *Client) updateMetaFromResponse(result *dsc.Object) scim.Meta {
	createdAt := result.GetCreatedAt().AsTime()
	updatedAt := result.GetUpdatedAt().AsTime()

	return scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      result.GetEtag(),
	}
}

func (s *Client) DeleteGroup(ctx context.Context, groupID string) error {
	logger := s.logger.With().Str("method", "DeleteGroup").Str("id", groupID).Logger()
	logger.Trace().Msg("delete group")

	_, err := s.client.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    s.cfg.Group.SourceObjectType,
		ObjectId:      groupID,
		WithRelations: true,
	})
	if err != nil {
		return err
	}

	_, err = s.client.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    s.cfg.Group.ObjectType,
		ObjectId:      groupID,
		WithRelations: true,
	})

	return err
}

func (s *Client) setRelations(ctx context.Context, subjID, subjType string) error {
	for _, userMap := range s.cfg.Relations {
		if userMap.SubjectID == subjID && userMap.SubjectType == subjType {
			_, err := s.client.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectType:     userMap.SubjectType,
					SubjectId:       userMap.SubjectID,
					Relation:        userMap.Relation,
					ObjectType:      userMap.ObjectType,
					ObjectId:        userMap.ObjectID,
					SubjectRelation: userMap.SubjectRelation,
				},
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
