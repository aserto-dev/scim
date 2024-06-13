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

		// _, err = dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
		// 	Relation: &dsc.Relation{
		// 		ObjectType:  resp.Result.Type,
		// 		ObjectId:    resp.Result.Id,
		// 		Relation:    transformConfig.SourceRelation,
		// 		SubjectType: transformConfig.SourceUserType,
		// 		SubjectId:   userID,
		// 	},
		// })

		// if err != nil {
		// 	return scim.Resource{}, err
		// }

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

func (s *Sync) Delete(ctx context.Context, dirClient *DirectoryClient, transformConfig config.TransformConfig, userID string) error {
	relations, err := dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		ObjectType:               transformConfig.UserObjectType,
		ObjectId:                 userID,
		Relation:                 transformConfig.IdentityRelation,
		WithObjects:              true,
		WithEmptySubjectRelation: true,
	})
	if err != nil {
		return err
	}

	for _, rel := range relations.Objects {
		_, err := dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
			ObjectType:    transformConfig.IdentityObjectType,
			ObjectId:      rel.Id,
			WithRelations: true,
		})
		if err != nil {
			return err
		}
	}

	_, err = dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    transformConfig.UserObjectType,
		ObjectId:      userID,
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
