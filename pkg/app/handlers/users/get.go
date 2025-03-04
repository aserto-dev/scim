package users

import (
	"context"
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Get").Str("id", id).Logger()
	logger.Info().Msg("get user")
	resp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    u.cfg.SCIM.UserObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Str("id", id).Msg("failed to get user")
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

	logger.Trace().Any("user", resource).Msg("user retrieved")

	return resource, nil
}

func (u UsersResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	logger := u.logger.With().Str("method", "GetAll").Logger()
	logger.Info().Msg("getall users")

	var (
		resources = make([]scim.Resource, 0)
		pageToken = ""
		pageSize  = 100
		skipIndex = 1 // start index is 1-based
	)

	if params.Count != 0 && params.Count < pageSize {
		pageSize = params.Count
	}

	for {
		resp, err := u.getUsers(r.Context(), pageSize, pageToken)
		if err != nil {
			logger.Err(err).Msg("failed to get users")
			return scim.Page{}, err
		}

		pageToken = resp.Page.NextToken

		for _, v := range resp.Results {
			createdAt := v.CreatedAt.AsTime()
			updatedAt := v.UpdatedAt.AsTime()
			resource := common.ObjectToResource(v, scim.Meta{
				Created:      &createdAt,
				LastModified: &updatedAt,
				Version:      v.Etag,
			})

			if params.FilterValidator == nil || params.FilterValidator.PassesFilter(resource.Attributes) == nil {
				if skipIndex < params.StartIndex {
					skipIndex++
					continue
				}
				resources = append(resources, resource)
			}

			if len(resources) == params.Count {
				break
			}
		}

		if len(resources) >= pageSize || pageToken == "" {
			break
		}
	}

	logger.Trace().Int("total_results", len(resources)).Msg("users read")

	return scim.Page{
		TotalResults: len(resources),
		Resources:    resources,
	}, nil
}

func (u UsersResourceHandler) getUsers(ctx context.Context, count int, pageToken string) (*dsr.GetObjectsResponse, error) {
	return u.dirClient.Reader.GetObjects(ctx, &dsr.GetObjectsRequest{
		ObjectType: u.cfg.SCIM.UserObjectType,
		Page: &dsc.PaginationRequest{
			Size:  int32(count), //nolint:gosec
			Token: pageToken,
		},
	})
}
