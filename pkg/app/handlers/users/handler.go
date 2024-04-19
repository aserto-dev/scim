package users

import (
	"context"

	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

const (
	Emails = "emails"
	Groups = "groups"
)

type UsersResourceHandler struct {
	dirClient *directory.DirectoryClient
	cfg       *config.Config
	logger    *zerolog.Logger
}

func NewUsersResourceHandler(cfg *config.Config, logger *zerolog.Logger) (*UsersResourceHandler, error) {
	usersLogger := logger.With().Str("component", "users").Logger()
	dirClient, err := directory.GetDirectoryClient(&cfg.Directory)
	if err != nil {
		return nil, err
	}
	return &UsersResourceHandler{
		dirClient: dirClient,
		cfg:       cfg,
		logger:    &usersLogger,
	}, nil
}

func (u UsersResourceHandler) setUserGroups(ctx context.Context, userID string, groups []string) error {
	relations, err := u.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: "user",
		SubjectId:   userID,
	})
	if err != nil {
		return err
	}

	for _, v := range relations.Results {
		if v.Relation == "member" {
			_, err = u.dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
				SubjectType: v.SubjectType,
				SubjectId:   v.SubjectId,
				Relation:    v.Relation,
				ObjectType:  v.ObjectType,
				ObjectId:    v.ObjectId,
			})
			if err != nil {
				return err
			}
		}
	}

	for _, v := range groups {
		_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: &dsc.Relation{
				SubjectId:   userID,
				SubjectType: "user",
				Relation:    "member",
				ObjectType:  "group",
				ObjectId:    v,
			}})
		if err != nil {
			return err
		}
	}

	return nil
}

func (u UsersResourceHandler) addUserToGroup(ctx context.Context, userID, group string) error {
	rel, err := u.dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
		SubjectType: "user",
		SubjectId:   userID,
		ObjectType:  "group",
		ObjectId:    group,
		Relation:    "member",
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
			_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectId:   userID,
					SubjectType: "user",
					Relation:    "member",
					ObjectType:  "group",
					ObjectId:    group,
				}})
			return err
		}
		return err
	}

	if rel != nil {
		return serrors.ScimErrorUniqueness
	}
	return nil
}

func (u UsersResourceHandler) removeUserFromGroup(ctx context.Context, userID, group string) error {
	_, err := u.dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
		SubjectType: "user",
		SubjectId:   userID,
		ObjectType:  "group",
		ObjectId:    group,
		Relation:    "member",
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
			return serrors.ScimErrorMutability
		}
		return err
	}

	_, err = u.dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
		SubjectType: "user",
		SubjectId:   userID,
		ObjectType:  "group",
		ObjectId:    group,
		Relation:    "member",
	})
	return err
}

func (u UsersResourceHandler) setIdentity(ctx context.Context, userID, identity string, propsMap map[string]interface{}) error {
	props, err := structpb.NewStruct(propsMap)
	if err != nil {
		return err
	}

	_, err = u.dirClient.Writer.SetObject(ctx, &dsw.SetObjectRequest{
		Object: &dsc.Object{
			Type:       "identity",
			Id:         identity,
			Properties: props,
		},
	})
	if err != nil {
		return err
	}

	_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
		Relation: &dsc.Relation{
			SubjectId:   userID,
			SubjectType: "user",
			Relation:    "identifier",
			ObjectType:  "identity",
			ObjectId:    identity,
		}})
	return err
}

func (u UsersResourceHandler) removeIdentity(ctx context.Context, identity string) error {
	_, err := u.dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    "identity",
		ObjectId:      identity,
		WithRelations: true,
	})

	return err
}

func (u UsersResourceHandler) setAllIdentities(ctx context.Context, userId string, attributes scim.ResourceAttributes) error {
	if attributes["userName"] != nil {
		err := u.setIdentity(ctx, userId, attributes["userName"].(string), nil)
		if err != nil {
			return err
		}
	}

	if attributes["emails"] != nil && u.cfg.SCIM.CreateEmailIdentities {
		for _, m := range attributes["emails"].([]interface{}) {
			email, ok := m.(map[string]interface{})
			if !ok {
				return errors.New("emails attribute is not a list of maps")
			}
			if email["value"].(string) == attributes["userName"].(string) {
				continue
			}

			err := u.setIdentity(ctx, userId, email["value"].(string), nil)
			if err != nil {
				return err
			}
		}
	}

	if attributes["externalId"] != nil {
		externalID := attributes["externalId"]
		externalIDStr, ok := externalID.(string)
		if !ok {
			return errors.New("externalId attribute is not a string")
		}
		err := u.setIdentity(ctx, userId, externalIDStr, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u UsersResourceHandler) setRoles(ctx context.Context, userID string, roles []interface{}) error {
	for _, role := range roles {
		r := role.(map[string]interface{})
		roleType := r["type"]
		if roleType == nil {
			u.logger.Debug().Msg("role type not found")
			continue
		}

		roleValue, ok := r["value"].(string)
		if !ok {
			continue
		}

		roleMap := u.cfg.SCIM.Roles.RoleMappings[roleType.(string)]
		if roleMap == nil {
			u.logger.Debug().Str("role", r["type"].(string)).Msg("role not found in role mappings")
			continue
		}

		objectType := roleMap.ObjectType
		if objectType == "" {
			objectType = roleType.(string)
		}

		u.logger.Trace().Str("role", r["value"].(string)).Str("relation", roleMap.Relation).Str("objectType", objectType).Str("subject", userID).Msg("setting role")
		_, err := u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: &dsc.Relation{
				SubjectId:   userID,
				SubjectType: "user",
				Relation:    roleMap.Relation,
				ObjectType:  objectType,
				ObjectId:    roleValue,
			}})
		if err != nil {
			return err
		}
	}

	return nil
}
