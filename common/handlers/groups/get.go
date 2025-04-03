package groups

import (
	"context"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (g GroupResourceHandler) Get(ctx context.Context, id string) (scim.Resource, error) {
	logger := g.logger.With().Str("method", "Get").Str("id", id).Logger()
	logger.Info().Msg("get group")

	if !g.cfg.Groups() {
		logger.Error().Msg("groups not enabled")
		return scim.Resource{}, serrors.ScimErrorBadRequest("groups not enabled")
	}

	resp, err := g.dirClient.DS().Reader.GetObject(ctx, &dsr.GetObjectRequest{
		ObjectType:    g.cfg.Group.SourceObjectType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to get group")
		return scim.Resource{}, err
	}

	converter := convert.NewConverter(g.cfg)

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := converter.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	return resource, nil
}

func (g GroupResourceHandler) GetAll(ctx context.Context, params scim.ListRequestParams) (scim.Page, error) {
	logger := g.logger.With().Str("method", "GetAll").Logger()
	logger.Info().Msg("getting all groups")

	var (
		resources = make([]scim.Resource, 0)
	)

	if !g.cfg.Groups() {
		logger.Error().Msg("groups not enabled")
		return scim.Page{}, serrors.ScimErrorBadRequest("groups not enabled")
	}

	resp, err := g.dirClient.DS().Reader.GetObjects(ctx, &dsr.GetObjectsRequest{
		ObjectType: g.cfg.Group.SourceObjectType,
		Page: &dsc.PaginationRequest{
			Size: int32(params.Count), //nolint:gosec
		},
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to read groups")
		return scim.Page{}, err
	}

	converter := convert.NewConverter(g.cfg)

	for _, v := range resp.Results {
		createdAt := v.CreatedAt.AsTime()
		updatedAt := v.UpdatedAt.AsTime()
		resource := converter.ObjectToResource(v, scim.Meta{
			Created:      &createdAt,
			LastModified: &updatedAt,
			Version:      v.Etag,
		})
		resources = append(resources, resource)
	}

	logger.Trace().Int("total_results", len(resources)).Msg("groups read")

	return scim.Page{
		TotalResults: len(resources),
		Resources:    resources,
	}, nil
}
