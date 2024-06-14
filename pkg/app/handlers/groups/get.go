package groups

import (
	"net/http"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (u GroupResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Resource{}, serrors.ScimErrorInternal
	}

	scimConfig, err := dirClient.GetTransformConfig(r.Context())
	if err != nil {
		return scim.Resource{}, err
	}

	resp, err := dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    scimConfig.SourceGroupType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := u.converter.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	return resource, nil
}

func (u GroupResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	var (
		resources = make([]scim.Resource, 0)
	)

	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Page{}, serrors.ScimErrorInternal
	}

	scimConfig, err := dirClient.GetTransformConfig(r.Context())
	if err != nil {
		return scim.Page{}, err
	}

	resp, err := dirClient.Reader.GetObjects(r.Context(), &dsr.GetObjectsRequest{
		ObjectType: scimConfig.SourceGroupType,
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
		resource := u.converter.ObjectToResource(v, scim.Meta{
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
