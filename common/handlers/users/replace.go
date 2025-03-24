package users

import (
	"context"

	"github.com/elimity-com/scim"
)

func (u UsersResourceHandler) Replace(ctx context.Context, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Replace").Str("id", id).Logger()
	logger.Info().Msg("replace user")
	u.logger.Trace().Any("attributes", attributes).Msg("replacing user")

	err := u.Delete(ctx, id)
	if err != nil {
		logger.Err(err).Msg("failed to delete user")
		return scim.Resource{}, err
	}

	resource, err := u.Create(ctx, attributes)
	if err != nil {
		logger.Err(err).Msg("failed to create user")
		return scim.Resource{}, err
	}

	logger.Trace().Any("resource", resource).Msg("user replaced")

	return resource, nil
}
