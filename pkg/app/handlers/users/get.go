package users

import (
	"log"
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	"github.com/scim2/filter-parser/v2"
)

func (u UsersResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	log.Println("GET", id)
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
	resource := common.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	return resource, nil
}

func (u UsersResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	log.Println("GETALL", params)
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
		resource := common.ObjectToResource(v, scim.Meta{
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
