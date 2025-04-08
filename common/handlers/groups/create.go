package groups

import (
	"context"

	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/aserto-dev/scim/common/model"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (g GroupResourceHandler) Create(ctx context.Context, attributes scim.ResourceAttributes) (scim.Resource, error) {
	groupName, ok := attributes["displayName"].(string)
	if !ok {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}
	logger := g.logger.With().Str("method", "Create").Str("name", groupName).Logger()
	logger.Info().Msg("create group")
	logger.Trace().Any("attributes", attributes).Msg("creating group")

	var group *model.Group
	err := convert.Unmarshal(attributes, group)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert attributes to group")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	var result scim.Resource

	converter := convert.NewConverter(g.cfg)
	object, err := converter.SCIMGroupToObject(group)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert group to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sourceGroupResp, err := g.dirClient.DS().Writer.SetObject(ctx, &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to create group")
		return scim.Resource{}, err
	}

	transformResult, err := converter.TransformResource(attributes, "group")
	if err != nil {
		logger.Error().Err(err).Msg("failed to transform group")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	meta, err := g.dirClient.SetGroup(ctx, sourceGroupResp.Result.Id, transformResult)
	if err != nil {
		logger.Error().Err(err).Msg("failed to sync group")
		return scim.Resource{}, err
	}

	result = converter.ObjectToResource(sourceGroupResp.Result, meta)

	logger.Trace().Any("response", result).Msg("group created")

	return result, nil
}
