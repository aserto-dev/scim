package groups

import "context"

func (g GroupResourceHandler) Delete(ctx context.Context, id string) error {
	logger := g.logger.With().Str("method", "Delete").Str("id", id).Logger()
	logger.Info().Msg("delete group")

	err := g.dirClient.DeleteGroup(ctx, id)
	if err != nil {
		logger.Error().Err(err).Msg("failed to delete group")
		return err
	}

	return nil
}
