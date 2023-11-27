package handler

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	"github.com/aserto-dev/go-directory/pkg/derr"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/aserto-dev/go-aserto/client"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/directory"
	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
)

type UsersResourceHandler struct {
	dirClient *directory.DirectoryClient
}

func NewUsersResourceHandler(cfg *client.Config) (*UsersResourceHandler, error) {
	dirClient, err := directory.GetDirectoryClient(cfg)
	if err != nil {
		return nil, err
	}
	return &UsersResourceHandler{
		dirClient: dirClient,
	}, nil
}

func (u UsersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return scim.Resource{}, err
	}
	object, err := resourceAttrToObject(attributes, "user", uuid.String())
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
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

	for _, m := range attributes["emails"].([]interface{}) {
		email := m.(map[string]interface{})
		props, err := structpb.NewStruct(email)
		if err != nil {
			return scim.Resource{}, err
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
				ObjectType:  "identity",
				ObjectId:    email["value"].(string),
				Relation:    "identifier",
				SubjectType: "user",
				SubjectId:   uuid.String(),
			},
		})
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
	for _, v := range resp.Results {
		createdAt := v.CreatedAt.AsTime()
		updatedAt := v.UpdatedAt.AsTime()
		resource := objectToResource(v, scim.Meta{
			Created:      &createdAt,
			LastModified: &updatedAt,
			Version:      v.Etag,
		})
		resources = append(resources, resource)
	}

	return scim.Page{
		TotalResults: len(resources),
		Resources:    resources,
	}, nil
}

func (u UsersResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	object, err := resourceAttrToObject(attributes, "user", id)
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}
	object.Id = id

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

func (u UsersResourceHandler) Delete(r *http.Request, id string) error {
	_, err := u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
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
	return scim.Resource{}, &serrors.ScimError{
		Status: http.StatusNotImplemented,
	}
}

func objectToResource(object *dsc.Object, meta scim.Meta) scim.Resource {
	// use pid as external id?
	eID := optional.String{}

	return scim.Resource{
		ID:         object.Id,
		ExternalID: eID,
		Attributes: object.Properties.AsMap(),
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
