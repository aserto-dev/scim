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
	idRelation, err := s.cfg.GetIdentityRelation(userID, "")
	if err != nil {
		return scim.Meta{}, err
	}

	relations, err := s.client.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               idRelation.ObjectType,
		ObjectId:                 idRelation.ObjectId,
		Relation:                 idRelation.Relation,
		SubjectType:              idRelation.SubjectType,
		SubjectId:                idRelation.SubjectId,
		WithObjects:              false,
		WithEmptySubjectRelation: true,
	})
	if err != nil && !errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
		return scim.Meta{}, err
	}

	result, addedIdentities, err := s.importObjects(ctx, data.Objects, userAttributes)
	if err != nil {
		return result, err
	}

	logger.Trace().Any("identities", addedIdentities).Msg("added identities")

	for _, relation := range data.Relations {
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
		if !slices.Contains(addedIdentities, rel.ObjectId) {
			logger.Trace().Str("identity", rel.ObjectId).Msg("deleting identity")
			_, err := s.client.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
				ObjectType:    s.cfg.User.IdentityObjectType,
				ObjectId:      rel.ObjectId,
				WithRelations: true,
			})
			if err != nil {
				mErr = multierror.Append(mErr, err)
				logger.Error().Err(err).Str("identity", rel.ObjectId).Msg("failed to delete identity")
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
		if object.Type == s.cfg.User.ObjectType {
			var userProperties map[string]any
			if object.Properties == nil {
				userProperties = make(map[string]any)
			} else {
				userProperties = object.Properties.AsMap()
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

		if resp.Result.Type == s.cfg.User.IdentityObjectType {
			addedIdentities = append(addedIdentities, resp.Result.Id)
		}

		if object.Type == s.cfg.User.ObjectType {
			err = s.setRelations(ctx, resp.Result.Id, resp.Result.Type)
			if err != nil {
				return result, addedIdentities, err
			}

			createdAt := resp.Result.CreatedAt.AsTime()
			updatedAt := resp.Result.UpdatedAt.AsTime()
			result.Created = &createdAt
			result.LastModified = &updatedAt
			result.Version = resp.Result.Etag
		}
	}

	return result, addedIdentities, nil
}

func (s *Client) DeleteUser(ctx context.Context, userID string) error {
	logger := s.logger.With().Str("method", "DeleteUser").Str("id", userID).Logger()
	logger.Trace().Msg("delete user")
	identityRelation, err := s.cfg.GetIdentityRelation(userID, "")
	if err != nil {
		return err
	}

	relations, err := s.client.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: identityRelation.SubjectType,
		SubjectId:   identityRelation.SubjectId,
		ObjectType:  identityRelation.ObjectType,
		ObjectId:    identityRelation.ObjectId,
		Relation:    identityRelation.Relation,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrNotFound) {
			return serrors.ScimErrorResourceNotFound(userID)
		}
	}

	for _, v := range relations.Results {
		var objectID string
		switch v.ObjectType {
		case s.cfg.User.IdentityObjectType:
			objectID = v.ObjectId
		case s.cfg.User.ObjectType:
			objectID = v.SubjectId
		default:
			return serrors.ScimErrorBadRequest("unexpected object type in identity relation")
		}

		logger.Trace().Str("id", v.ObjectId).Msg("deleting identity")
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

	relations, err := s.client.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               s.cfg.Group.ObjectType,
		ObjectId:                 groupID,
		Relation:                 s.cfg.Group.GroupMemberRelation,
		WithObjects:              false,
		WithEmptySubjectRelation: true,
	})
	if err != nil && !errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
		return scim.Meta{}, err
	}

	addedMembers := make([]string, 0)

	result := scim.Meta{}
	for _, object := range data.Objects {
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

		if object.Type == s.cfg.Group.ObjectType {
			err = s.setRelations(ctx, resp.Result.Id, resp.Result.Type)
			if err != nil {
				return result, err
			}

			createdAt := resp.Result.CreatedAt.AsTime()
			updatedAt := resp.Result.UpdatedAt.AsTime()
			result.Created = &createdAt
			result.LastModified = &updatedAt
			result.Version = resp.Result.Etag
		}
	}

	for _, relation := range data.Relations {
		if relation.Relation == s.cfg.Group.GroupMemberRelation {
			addedMembers = append(addedMembers, relation.SubjectId)
		}
		logger.Trace().Any("relation", relation).Msg("setting relation")
		_, err := s.client.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: relation,
		})
		if err != nil {
			return result, err
		}
	}

	if relations != nil {
		for _, rel := range relations.Results {
			if !slices.Contains(addedMembers, rel.SubjectId) {
				logger.Trace().Str("id", rel.SubjectId).Msg("deleting relation")
				_, err := s.client.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
					ObjectType:  s.cfg.Group.ObjectType,
					ObjectId:    groupID,
					Relation:    s.cfg.Group.GroupMemberRelation,
					SubjectId:   rel.SubjectId,
					SubjectType: rel.SubjectType,
				})
				if err != nil {
					return result, err
				}
			}
		}
	}

	return result, nil
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
