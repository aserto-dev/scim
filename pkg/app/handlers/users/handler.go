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
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

const (
	Emails = "emails"
	Groups = "groups"
)

type UsersResourceHandler struct {
	dirClient *directory.DirectoryClient
	cfg       *config.Config
}

func NewUsersResourceHandler(cfg *config.Config) (*UsersResourceHandler, error) {
	dirClient, err := directory.GetDirectoryClient(&cfg.Directory)
	if err != nil {
		return nil, err
	}
	return &UsersResourceHandler{
		dirClient: dirClient,
		cfg:       cfg,
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

func (u UsersResourceHandler) setIdentity(ctx context.Context, userID, identity, kind string) error {
	propsMap := make(map[string]interface{})
	propsMap["kind"] = kind
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
