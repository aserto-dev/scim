package groups

import (
	"context"

	"github.com/elimity-com/scim"
)

func (g GroupResourceHandler) Replace(ctx context.Context, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	logger := g.logger.With().Str("method", "Replace").Str("id", id).Logger()
	logger.Info().Msg("replace group")

	err := g.Delete(ctx, id)
	if err != nil {
		logger.Error().Err(err).Msg("failed to delete group")
		return scim.Resource{}, err
	}

	resource, err := g.Create(ctx, attributes)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create group")
		return scim.Resource{}, err
	}

	logger.Trace().Any("resource", resource).Msg("group replaced")

	return resource, nil
}
