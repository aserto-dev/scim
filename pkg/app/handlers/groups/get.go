package groups

import (
	"net/http"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
)

func (u GroupResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Get").Str("id", id).Logger()
	logger.Info().Msg("get group")

	resp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    u.cfg.SCIM.GroupObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Str("id", id).Msg("failed to get group")
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := common.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	logger.Trace().Any("group", resource).Msg("group retrieved")

	return resource, nil
}

func (u GroupResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	u.logger.Info().Msg("getall groups")

	var (
		resources = make([]scim.Resource, 0)
	)

	resp, err := u.dirClient.Reader.GetObjects(r.Context(), &dsr.GetObjectsRequest{
		ObjectType: u.cfg.SCIM.GroupObjectType,
		Page: &dsc.PaginationRequest{
			Size: int32(params.Count), //nolint:gosec
		},
	})
	if err != nil {
		u.logger.Err(err).Msg("failed to read groups")
		return scim.Page{}, err
	}

	for _, v := range resp.Results {
		createdAt := v.CreatedAt.AsTime()
		updatedAt := v.UpdatedAt.AsTime()
		resource := common.ObjectToResource(v, scim.Meta{
			Created:      &createdAt,
			LastModified: &updatedAt,
			Version:      v.Etag,
		})
		resources = append(resources, resource)
	}

	u.logger.Trace().Int("total_results", len(resources)).Msg("groups read")

	return scim.Page{
		TotalResults: len(resources),
		Resources:    resources,
	}, nil
}
