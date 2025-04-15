package users

import (
	"context"

	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/aserto-dev/scim/common/model"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (u UsersResourceHandler) Create(ctx context.Context, attributes scim.ResourceAttributes) (scim.Resource, error) {
	userName, ok := attributes["userName"].(string)
	if !ok {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	logger := u.logger.With().Str("method", "Create").Str("userName", userName).Logger()
	logger.Info().Msg("create user")
	logger.Trace().Any("attributes", attributes).Msg("creating user")

	user := &model.User{}
	err := convert.Unmarshal(attributes, user)

	if err != nil {
		logger.Error().Err(err).Msg("failed to convert attributes to user")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	var result scim.Resource

	converter := convert.NewConverter(u.cfg)
	object, err := converter.SCIMUserToObject(user)

	if err != nil {
		logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sourceUserResp, err := u.dirClient.DS().Writer.SetObject(ctx, &dsw.SetObjectRequest{
		Object: object,
	})

	if err != nil {
		logger.Error().Err(err).Msg("failed to create user")
		return scim.Resource{}, err
	}

	userMap, err := convert.ProtobufStructToMap(sourceUserResp.GetResult().GetProperties())
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert user to map")
		return scim.Resource{}, err
	}

	transformResult, err := converter.TransformResource(userMap, "user")
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	meta, err := u.dirClient.SetUser(ctx, sourceUserResp.GetResult().GetId(), transformResult, attributes)
	if err != nil {
		logger.Error().Err(err).Msg("failed to sync user")
		return scim.Resource{}, err
	}

	result = converter.ObjectToResource(sourceUserResp.GetResult(), meta)

	logger.Trace().Any("response", result).Msg("user created")

	return result, nil
}
