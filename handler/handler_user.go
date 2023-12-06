package handler

import (
	"context"
	"net/http"

	cerr "github.com/aserto-dev/errors"
	"github.com/aserto-dev/go-directory/pkg/derr"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/directory"
	"github.com/aserto-dev/scim/pkg/config"
	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	filter "github.com/scim2/filter-parser/v2"
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

func (u UsersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	object, err := resourceAttrToObject(attributes, "user", attributes["userName"].(string))
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
			return scim.Resource{}, serrors.ScimErrorUniqueness
		}
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := objectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	if attributes["userName"] != nil {
		propsMap := make(map[string]interface{})
		propsMap["kind"] = "IDENTITY_KIND_USERNAME"
		props, err := structpb.NewStruct(propsMap)
		if err != nil {
			return scim.Resource{}, err
		}
		_, err = u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
			Object: &dsc.Object{
				Type:       "identity",
				Id:         attributes["userName"].(string),
				Properties: props,
			},
		})
		if err != nil {
			return scim.Resource{}, err
		}

		_, err = u.dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
			Relation: &dsc.Relation{
				SubjectId:   resp.Result.Id,
				SubjectType: "user",
				Relation:    "identifier",
				ObjectType:  "identity",
				ObjectId:    attributes["userName"].(string),
			}})
		if err != nil {
			return scim.Resource{}, err
		}
	}

	if attributes["emails"] != nil {
		for _, m := range attributes["emails"].([]interface{}) {
			email := m.(map[string]interface{})
			propsMap := make(map[string]interface{})
			propsMap["kind"] = "IDENTITY_KIND_EMAIL"
			props, err := structpb.NewStruct(propsMap)
			if err != nil {
				return scim.Resource{}, err
			}

			if email["value"].(string) == attributes["userName"].(string) {
				continue
			}

			_, err = u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
				Object: &dsc.Object{
					Type:       "identity",
					Id:         email["value"].(string),
					Properties: props,
				},
			})
			if err != nil {
				return scim.Resource{}, err
			}

			_, err = u.dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectId:   resp.Result.Id,
					SubjectType: "user",
					Relation:    "identifier",
					ObjectType:  "identity",
					ObjectId:    email["value"].(string),
				}})
			if err != nil {
				return scim.Resource{}, err
			}
		}
	}

	if attributes["groups"] != nil {
		err = u.setUserGroups(r.Context(), resp.Result.Id, attributes["groups"].([]string))
		if err != nil {
			return scim.Resource{}, err
		}
	}

	return resource, nil
}

func (u UsersResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	resp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    "user",
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := objectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	return resource, nil
}

func (u UsersResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	var (
		resources = make([]scim.Resource, 0)
	)

	resp, err := u.dirClient.Reader.GetObjects(r.Context(), &dsr.GetObjectsRequest{
		ObjectType: "user",
		Page: &dsc.PaginationRequest{
			Size: int32(params.Count),
		},
	})
	if err != nil {
		return scim.Page{}, err
	}

	var f filter.AttributeExpression

	if params.Filter != nil {
		f, err = filter.ParseAttrExp([]byte(params.Filter.(*filter.AttributeExpression).String()))
		if err != nil {
			return scim.Page{}, err
		}
	}

	for _, v := range resp.Results {
		createdAt := v.CreatedAt.AsTime()
		updatedAt := v.UpdatedAt.AsTime()
		resource := objectToResource(v, scim.Meta{
			Created:      &createdAt,
			LastModified: &updatedAt,
			Version:      v.Etag,
		})

		if params.Filter != nil {
			switch f.Operator {
			case filter.EQ:
				if resource.Attributes[f.AttributePath.AttributeName] == f.CompareValue {
					resources = append(resources, resource)
				}
			case filter.NE:
				if resource.Attributes[f.AttributePath.AttributeName] != f.CompareValue {
					resources = append(resources, resource)
				}
			}
		} else {
			resources = append(resources, resource)
		}
	}

	return scim.Page{
		TotalResults: len(resources),
		Resources:    resources,
	}, nil
}

func (u UsersResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	getObjResp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    "user",
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	object, err := resourceAttrToObject(attributes, "user", id)
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}
	object.Id = id
	object.Etag = getObjResp.Result.Etag

	setResp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	if attributes["groups"] != nil {
		err = u.setUserGroups(r.Context(), id, attributes["groups"].([]string))
		if err != nil {
			return scim.Resource{}, err
		}
	}

	createdAt := setResp.Result.CreatedAt.AsTime()
	updatedAt := setResp.Result.UpdatedAt.AsTime()
	resource := objectToResource(setResp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      setResp.Result.Etag,
	})

	return resource, nil
}

func (u UsersResourceHandler) Delete(r *http.Request, id string) error {
	relations, err := u.dirClient.Reader.GetRelations(r.Context(), &dsr.GetRelationsRequest{
		SubjectType: "user",
		SubjectId:   id,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
		return err
	}

	for _, v := range relations.Results {
		if v.Relation == "identifier" {
			_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
				ObjectId:      v.ObjectId,
				ObjectType:    v.ObjectType,
				WithRelations: true,
			})
			if err != nil {
				return err
			}
		}
	}

	_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
		ObjectType:    "user",
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
	}

	return err
}

func (u UsersResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	getObjResp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    "user",
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	object, err := resourceAttrToObject(getObjResp.Result.Properties.AsMap(), "user", id)
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	for _, op := range operations {
		switch op.Op {
		case scim.PatchOperationAdd:
			if op.Path != nil && op.Path.AttributePath.AttributeName == "groups" {
				err = u.addUserToGroup(r.Context(), id, op.Value.(string))
				if err != nil {
					return scim.Resource{}, err
				}
			}
		case scim.PatchOperationRemove:
			if op.Path != nil && op.Path.AttributePath.AttributeName == "groups" {
				err = u.removeUserFromGroup(r.Context(), id, op.Value.(string))
				if err != nil {
					return scim.Resource{}, err
				}
			}
		case scim.PatchOperationReplace:
			if op.Path != nil && op.Path.AttributePath.AttributeName == "groups" {
				err = u.removeUserFromGroup(r.Context(), id, op.Value.(string))
				if err != nil {
					return scim.Resource{}, err
				}
				err = u.addUserToGroup(r.Context(), id, op.Value.(string))
				if err != nil {
					return scim.Resource{}, err
				}
			}
		}
	}

	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := objectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	return resource, nil
}

func (u UsersResourceHandler) setUserGroups(ctx context.Context, userId string, groups []string) error {
	relations, err := u.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: "user",
		SubjectId:   userId,
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
				SubjectId:   userId,
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

func (u UsersResourceHandler) addUserToGroup(ctx context.Context, userId, group string) error {
	rel, err := u.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: "user",
		SubjectId:   userId,
		ObjectType:  "group",
		ObjectId:    group,
		Relation:    "member",
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
			_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectId:   userId,
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

func (u UsersResourceHandler) removeUserFromGroup(ctx context.Context, userId, group string) error {
	rel, err := u.dirClient.Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: "user",
		SubjectId:   userId,
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

	if rel != nil {
		return serrors.ScimErrorUniqueness
	}
	return nil
}

func objectToResource(object *dsc.Object, meta scim.Meta) scim.Resource {
	// use pid as external id?
	eID := optional.String{}
	attr := object.Properties.AsMap()
	delete(attr, "password")

	return scim.Resource{
		ID:         object.Id,
		ExternalID: eID,
		Attributes: attr,
		Meta:       meta,
	}
}

func resourceAttrToObject(resourceAttributes scim.ResourceAttributes, objectType, id string) (*dsc.Object, error) {
	props, err := structpb.NewStruct(resourceAttributes)
	if err != nil {
		return nil, err
	}

	var userName string
	if resourceAttributes["userName"] != nil {
		userName = resourceAttributes["userName"].(string)
	}
	object := &dsc.Object{
		Type:        objectType,
		Properties:  props,
		Id:          id,
		DisplayName: userName,
	}
	return object, nil
}
