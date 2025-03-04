package users

import (
	"context"

	cerr "github.com/aserto-dev/errors"
	"github.com/aserto-dev/go-aserto/ds/v3"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/directory"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

const (
	Emails          = "emails"
	Groups          = "groups"
	IdentityKindKey = "kind"
)

type UsersResourceHandler struct {
	dirClient *ds.Client
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

func (u UsersResourceHandler) setUserGroups(ctx context.Context, userID string, groups []common.UserGroup) error {
	relations, err := u.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   userID,
	})
	if err != nil {
		return err
	}

	for _, v := range relations.Results {
		if v.Relation == u.cfg.SCIM.GroupMemberRelation {
			u.logger.Trace().Str("user_id", userID).Str("group", v.ObjectId).Msg("removing user from group")
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
		u.logger.Trace().Str("user_id", userID).Str("group", v.Value).Msg("setting user group")
		_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
			Relation: &dsc.Relation{
				SubjectId:   userID,
				SubjectType: u.cfg.SCIM.UserObjectType,
				Relation:    u.cfg.SCIM.GroupMemberRelation,
				ObjectType:  u.cfg.SCIM.GroupObjectType,
				ObjectId:    v.Value,
			}})
		if err != nil {
			return err
		}
	}

	return nil
}

func (u UsersResourceHandler) addUserToGroup(ctx context.Context, userID, group string) error {
	rel, err := u.dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   userID,
		ObjectType:  u.cfg.SCIM.GroupObjectType,
		ObjectId:    group,
		Relation:    u.cfg.SCIM.GroupMemberRelation,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
			u.logger.Trace().Str("user_id", userID).Str("group", group).Msg("adding user to group")
			_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectId:   userID,
					SubjectType: u.cfg.SCIM.UserObjectType,
					Relation:    u.cfg.SCIM.GroupMemberRelation,
					ObjectType:  u.cfg.SCIM.GroupObjectType,
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
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   userID,
		ObjectType:  u.cfg.SCIM.GroupObjectType,
		ObjectId:    group,
		Relation:    u.cfg.SCIM.GroupMemberRelation,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
			return serrors.ScimErrorMutability
		}
		return err
	}

	u.logger.Trace().Str("user_id", userID).Str("group", group).Msg("removing user from group")
	_, err = u.dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   userID,
		ObjectType:  u.cfg.SCIM.GroupObjectType,
		ObjectId:    group,
		Relation:    u.cfg.SCIM.GroupMemberRelation,
	})
	return err
}

func (u UsersResourceHandler) setIdentity(ctx context.Context, userID, identity string, propsMap map[string]interface{}) error {
	props, err := structpb.NewStruct(propsMap)
	if err != nil {
		return err
	}

	u.logger.Trace().Str("user_id", userID).Str("identity", identity).Any("props", props).Msg("setting identity")
	_, err = u.dirClient.Writer.SetObject(ctx, &dsw.SetObjectRequest{
		Object: &dsc.Object{
			Type:       u.cfg.SCIM.IdentityObjectType,
			Id:         identity,
			Properties: props,
		},
	})
	if err != nil {
		return err
	}

	rel, err := u.getIdentityRelation(userID, identity)
	if err != nil {
		u.logger.Err(err).Msg("failed to get identity relation")
		return err
	}

	u.logger.Trace().Str("user_id", userID).Str("identity", identity).Any("relation", rel).Msg("setting identity relation")
	_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{Relation: rel})
	return err
}

func (u UsersResourceHandler) removeIdentity(ctx context.Context, identity string) error {
	u.logger.Info().Str("identity", identity).Msg("removing identity")
	_, err := u.dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.SCIM.IdentityObjectType,
		ObjectId:      identity,
		WithRelations: true,
	})

	return err
}

func (u UsersResourceHandler) setAllIdentities(ctx context.Context, userID string, user *common.User) error {
	u.logger.Info().Str("user_id", userID).Msg("setting identities")
	if user.UserName != "" {
		u.logger.Debug().Str("user_id", userID).Str("username", user.UserName).Msg("setting username identity")
		err := u.setIdentity(ctx, userID, user.UserName, map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_USERNAME"})
		if err != nil {
			return err
		}
	}

	if u.cfg.SCIM.CreateEmailIdentities {
		for _, email := range user.Emails {
			if email.Value == user.UserName {
				continue
			}

			u.logger.Debug().Str("user_id", userID).Str("email", email.Value).Msg("setting email identity")
			err := u.setIdentity(ctx, userID, email.Value, map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_EMAIL"})
			if err != nil {
				return err
			}
		}
	}

	if user.ExternalID != "" {
		u.logger.Debug().Str("user_id", userID).Str("external_id", user.ExternalID).Msg("setting external_id identity")
		err := u.setIdentity(ctx, userID, user.ExternalID, map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_PID"})
		if err != nil {
			return err
		}
	}

	return nil
}

func (u UsersResourceHandler) setUserMappings(ctx context.Context, userID string) error {
	for _, userMap := range u.cfg.SCIM.UserMappings {
		if userMap.SubjectID == userID {
			u.logger.Trace().Str("user_id", userID).Str("object_id", userMap.ObjectID).Msg("setting user mapping")
			_, err := u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectType:     u.cfg.SCIM.UserObjectType,
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

func (u UsersResourceHandler) getIdentityRelation(userID, identity string) (*dsc.Relation, error) {
	switch u.cfg.SCIM.Identity.ObjectType {
	case u.cfg.SCIM.IdentityObjectType:
		return &dsc.Relation{
			SubjectId:   userID,
			SubjectType: u.cfg.SCIM.UserObjectType,
			Relation:    u.cfg.SCIM.Identity.Relation,
			ObjectType:  u.cfg.SCIM.IdentityObjectType,
			ObjectId:    identity,
		}, nil
	case u.cfg.SCIM.UserObjectType:
		return &dsc.Relation{
			SubjectId:   identity,
			SubjectType: u.cfg.SCIM.IdentityObjectType,
			Relation:    u.cfg.SCIM.Identity.Relation,
			ObjectType:  u.cfg.SCIM.UserObjectType,
			ObjectId:    userID,
		}, nil
	default:
		return nil, errors.New("invalid identity relation")
	}
}
