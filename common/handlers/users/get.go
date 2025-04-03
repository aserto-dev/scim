package users

import (
	"context"

	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Get(ctx context.Context, id string) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Get").Str("id", id).Logger()
	logger.Info().Msg("get user")

	converter := convert.NewConverter(u.cfg)

	resp, err := u.dirClient.DS().Reader.GetObject(ctx, &dsr.GetObjectRequest{
		ObjectType:    u.cfg.User.SourceObjectType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to get user")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := converter.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	logger.Trace().Any("user", resource).Msg("user retrieved")

	return resource, nil
}

func (u UsersResourceHandler) GetAll(ctx context.Context, params scim.ListRequestParams) (scim.Page, error) {
	logger := u.logger.With().Str("method", "GetAll").Logger()
	logger.Info().Msg("getting all users")

	var (
		resources = make([]scim.Resource, 0)
		pageToken = ""
		pageSize  = 100
		skipIndex = 1 // start index is 1-based
	)

	if params.Count != 0 && params.Count < pageSize {
		pageSize = params.Count
	}

	converter := convert.NewConverter(u.cfg)

	for {
		resp, err := u.getUsers(ctx, pageSize, pageToken)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get users")
			return scim.Page{}, err
		}

		pageToken = resp.Page.NextToken

		for _, v := range resp.Results {
			createdAt := v.CreatedAt.AsTime()
			updatedAt := v.UpdatedAt.AsTime()
			resource := converter.ObjectToResource(v, scim.Meta{
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
	return u.dirClient.DS().Reader.GetObjects(ctx, &dsr.GetObjectsRequest{
		ObjectType: u.cfg.User.SourceObjectType,
		Page: &dsc.PaginationRequest{
			Size:  int32(count), //nolint:gosec
			Token: pageToken,
		},
	})
}
