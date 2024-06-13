package users

import (
	"net/http"

	"github.com/aserto-dev/go-aserto/client"
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
	// dirClient *directory.DirectoryClient
	cfg    *config.Config
	logger *zerolog.Logger
	// converter *common.Converter
	// sync      *directory.Sync
}

func NewUsersResourceHandler(cfg *config.Config, logger *zerolog.Logger) *UsersResourceHandler {
	usersLogger := logger.With().Str("component", "users").Logger()

	return &UsersResourceHandler{
		// dirClient: dirClient,
		cfg:    cfg,
		logger: &usersLogger,
		// converter: common.NewConverter(*cfg),
		// sync: &directory.Sync{},
	}
}

func (u UsersResourceHandler) getDirectoryClient(r *http.Request) (*directory.DirectoryClient, error) {
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

// func (u UsersResourceHandler) setUserGroups(ctx context.Context, dirClient *directory.DirectoryClient, userID string, groups []common.UserGroup) error {
// 	relations, err := dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
// 		SubjectType: u.cfg.SCIM.UserObjectType,
// 		SubjectId:   userID,
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	for _, v := range relations.Results {
// 		if v.Relation == u.cfg.SCIM.GroupMemberRelation {
// 			_, err = dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
// 				SubjectType: v.SubjectType,
// 				SubjectId:   v.SubjectId,
// 				Relation:    v.Relation,
// 				ObjectType:  v.ObjectType,
// 				ObjectId:    v.ObjectId,
// 			})
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	for _, v := range groups {
// 		_, err = dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
// 			Relation: &dsc.Relation{
// 				SubjectId:   userID,
// 				SubjectType: u.cfg.SCIM.UserObjectType,
// 				Relation:    u.cfg.SCIM.GroupMemberRelation,
// 				ObjectType:  u.cfg.SCIM.GroupObjectType,
// 				ObjectId:    v.Value,
// 			}})
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func (u UsersResourceHandler) addUserToGroup(ctx context.Context, dirClient *directory.DirectoryClient, userID, group string) error {
// 	rel, err := dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
// 		SubjectType: u.cfg.SCIM.UserObjectType,
// 		SubjectId:   userID,
// 		ObjectType:  u.cfg.SCIM.GroupObjectType,
// 		ObjectId:    group,
// 		Relation:    u.cfg.SCIM.GroupMemberRelation,
// 	})
// 	if err != nil {
// 		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
// 			_, err = dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
// 				Relation: &dsc.Relation{
// 					SubjectId:   userID,
// 					SubjectType: u.cfg.SCIM.UserObjectType,
// 					Relation:    u.cfg.SCIM.GroupMemberRelation,
// 					ObjectType:  u.cfg.SCIM.GroupObjectType,
// 					ObjectId:    group,
// 				}})
// 			return err
// 		}
// 		return err
// 	}

// 	if rel != nil {
// 		return serrors.ScimErrorUniqueness
// 	}
// 	return nil
// }

// func (u UsersResourceHandler) removeUserFromGroup(ctx context.Context, dirClient *directory.DirectoryClient, userID, group string) error {
// 	_, err := dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
// 		SubjectType: u.cfg.SCIM.UserObjectType,
// 		SubjectId:   userID,
// 		ObjectType:  u.cfg.SCIM.GroupObjectType,
// 		ObjectId:    group,
// 		Relation:    u.cfg.SCIM.GroupMemberRelation,
// 	})
// 	if err != nil {
// 		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
// 			return serrors.ScimErrorMutability
// 		}
// 		return err
// 	}

// 	_, err = dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
// 		SubjectType: u.cfg.SCIM.UserObjectType,
// 		SubjectId:   userID,
// 		ObjectType:  u.cfg.SCIM.GroupObjectType,
// 		ObjectId:    group,
// 		Relation:    u.cfg.SCIM.GroupMemberRelation,
// 	})
// 	return err
// }

// func (u UsersResourceHandler) setIdentity(ctx context.Context, dirClient *directory.DirectoryClient, userID, identity string, propsMap map[string]interface{}) error {
// 	props, err := structpb.NewStruct(propsMap)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = dirClient.Writer.SetObject(ctx, &dsw.SetObjectRequest{
// 		Object: &dsc.Object{
// 			Type:       u.cfg.SCIM.IdentityObjectType,
// 			Id:         identity,
// 			Properties: props,
// 		},
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	_, err = dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
// 		Relation: &dsc.Relation{
// 			SubjectId:   userID,
// 			SubjectType: u.cfg.SCIM.UserObjectType,
// 			Relation:    u.cfg.SCIM.IdentityRelation,
// 			ObjectType:  u.cfg.SCIM.IdentityObjectType,
// 			ObjectId:    identity,
// 		}})
// 	return err
// }

// func (u UsersResourceHandler) removeIdentity(ctx context.Context, dirClient *directory.DirectoryClient, identity string) error {
// 	_, err := dirClient.Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
// 		ObjectType:    u.cfg.SCIM.IdentityObjectType,
// 		ObjectId:      identity,
// 		WithRelations: true,
// 	})

// 	return err
// }

// func (u UsersResourceHandler) setAllIdentities(ctx context.Context, dirClient *directory.DirectoryClient, userID string, user *common.User) error {
// 	if user.UserName != "" {
// 		err := u.setIdentity(ctx, dirClient, userID, user.UserName, map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_USERNAME"})
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	if u.cfg.SCIM.CreateEmailIdentities {
// 		for _, email := range user.Emails {
// 			if email.Value == user.UserName {
// 				continue
// 			}

// 			err := u.setIdentity(ctx, dirClient, userID, email.Value, map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_EMAIL"})
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	if user.ExternalID != "" {
// 		err := u.setIdentity(ctx, dirClient, userID, user.ExternalID, map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_PID"})
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func (u UsersResourceHandler) setUserMappings(ctx context.Context, dirClient *directory.DirectoryClient, userID string) error {
// 	for _, userMap := range u.cfg.SCIM.UserMappings {
// 		if userMap.SubjectID == userID {
// 			_, err := dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
// 				Relation: &dsc.Relation{
// 					SubjectType:     u.cfg.SCIM.Transform.UserObjectType,
// 					SubjectId:       userMap.SubjectID,
// 					Relation:        userMap.Relation,
// 					ObjectType:      userMap.ObjectType,
// 					ObjectId:        userMap.ObjectID,
// 					SubjectRelation: userMap.SubjectRelation,
// 				},
// 			})
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }
