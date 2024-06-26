package directory

import (
	"context"
	"errors"
	"slices"

	"github.com/aserto-dev/ds-load/sdk/common/msg"
	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

type Sync struct {
	cfg       *config.TransformConfig
	dirClient *DirectoryClient
}

func NewSync(cfg *config.TransformConfig, dirClient *DirectoryClient) *Sync {
	return &Sync{
		cfg:       cfg,
		dirClient: dirClient,
	}
}

func (s *Sync) UpdateUser(ctx context.Context, userID string, data *msg.Transform) (scim.Meta, error) {
	relations, err := s.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               s.cfg.UserObjectType,
		ObjectId:                 userID,
		Relation:                 s.cfg.IdentityRelation,
		WithObjects:              true,
		WithEmptySubjectRelation: true,
	})
	if err != nil && !errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
		return scim.Meta{}, err
	}

	addedIdentities := make([]string, 0)

	result := scim.Meta{}
	for _, object := range data.Objects {
		resp, err := s.dirClient.Writer.SetObject(ctx, &dsw.SetObjectRequest{
			Object: object,
		})
		if err != nil {
			if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
				return result, serrors.ScimErrorUniqueness
			}
			return result, err
		}

		if resp.Result.Type == s.cfg.IdentityObjectType {
			addedIdentities = append(addedIdentities, resp.Result.Id)
		}

		if object.Type == s.cfg.UserObjectType {
			err = s.setUserMappings(ctx, resp.Result.Id)
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
		_, err := s.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: relation,
		})
		if err != nil {
			return result, err
		}
	}

	if relations != nil {
		for _, rel := range relations.Objects {
			if !slices.Contains(addedIdentities, rel.Id) {
				_, err := s.dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
					ObjectType:    s.cfg.IdentityObjectType,
					ObjectId:      rel.Id,
					WithRelations: true,
				})
				if err != nil {
					return result, err
				}
			}
		}
	}

	return result, nil
}

func (s *Sync) DeleteUser(ctx context.Context, userID string) error {
	relations, err := s.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               s.cfg.UserObjectType,
		ObjectId:                 userID,
		Relation:                 s.cfg.IdentityRelation,
		WithObjects:              true,
		WithEmptySubjectRelation: true,
	})
	if err != nil {
		return err
	}

	for _, rel := range relations.Objects {
		_, err := s.dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
			ObjectType:    s.cfg.IdentityObjectType,
			ObjectId:      rel.Id,
			WithRelations: true,
		})
		if err != nil {
			return err
		}
	}

	_, err = s.dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    s.cfg.UserObjectType,
		ObjectId:      userID,
		WithRelations: true,
	})

	return err
}

func (s *Sync) UpdateGroup(ctx context.Context, groupID string, data *msg.Transform) (scim.Meta, error) {
	relations, err := s.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               s.cfg.GroupObjectType,
		ObjectId:                 groupID,
		Relation:                 s.cfg.GroupMemberRelation,
		WithObjects:              true,
		WithEmptySubjectRelation: true,
	})
	if err != nil && !errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
		return scim.Meta{}, err
	}

	addedMembers := make([]string, 0)

	result := scim.Meta{}
	for _, object := range data.Objects {
		resp, err := s.dirClient.Writer.SetObject(ctx, &dsw.SetObjectRequest{
			Object: object,
		})
		if err != nil {
			if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
				return result, serrors.ScimErrorUniqueness
			}
			return result, err
		}

		if object.Type == s.cfg.GroupObjectType {
			err = s.setGroupMappings(ctx, resp.Result.Id)
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
		if relation.Relation == s.cfg.GroupMemberRelation {
			addedMembers = append(addedMembers, relation.ObjectId)
		}
		_, err := s.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: relation,
		})
		if err != nil {
			return result, err
		}
	}

	if relations != nil {
		for _, obj := range relations.Objects {
			if !slices.Contains(addedMembers, obj.Id) {
				_, err := s.dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
					ObjectType:  obj.Type,
					ObjectId:    obj.Id,
					Relation:    s.cfg.GroupMemberRelation,
					SubjectId:   groupID,
					SubjectType: s.cfg.GroupObjectType,
				})
				if err != nil {
					return result, err
				}
			}
		}
	}

	return result, nil
}

func (s *Sync) DeleteGroup(ctx context.Context, groupID string) error {
	_, err := s.dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    s.cfg.SourceGroupType,
		ObjectId:      groupID,
		WithRelations: true,
	})

	if err != nil {
		return err
	}

	_, err = s.dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    s.cfg.GroupObjectType,
		ObjectId:      groupID,
		WithRelations: true,
	})

	return err
}

func (s *Sync) setUserMappings(ctx context.Context, userID string) error {
	for _, userMap := range s.cfg.UserMappings {
		if userMap.SubjectID == userID {
			_, err := s.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectType:     s.cfg.UserObjectType,
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

func (s *Sync) setGroupMappings(ctx context.Context, groupID string) error {
	for _, groupMap := range s.cfg.GroupMappings {
		if groupMap.SubjectID == groupID {
			_, err := s.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectType:     s.cfg.GroupObjectType,
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
