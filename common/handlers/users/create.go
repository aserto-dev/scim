package users

import (
	"context"

	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/aserto-dev/scim/common/model"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/rs/zerolog"
)

func (u UsersResourceHandler) Create(ctx context.Context, attributes scim.ResourceAttributes) (scim.Resource, error) {
	userName, ok := attributes["userName"].(string)
	if !ok {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	logger := u.logger.With().Str("method", "Create").Str("userName", userName).Logger()
	logger.Info().Msg("create user")
	logger.Trace().Any("attributes", attributes).Msg("creating user")

	user, err := u.convertAttributesToUser(attributes, logger)
	if err != nil {
		return scim.Resource{}, err
	}

	converter := convert.NewConverter(u.cfg)

	result, err := u.createUserObject(ctx, user, attributes, converter, logger)
	if err != nil {
		return scim.Resource{}, err
	}

	logger.Trace().Any("response", result).Msg("user created")

	return result, nil
}

func (u UsersResourceHandler) convertAttributesToUser(attributes scim.ResourceAttributes, logger zerolog.Logger) (*model.User, error) {
	user := &model.User{}
	if err := convert.Unmarshal(attributes, user); err != nil {
		logger.Err(err).Msg("failed to convert attributes to user")
		return nil, serrors.ScimErrorInvalidSyntax
	}

	return user, nil
}

func (u UsersResourceHandler) createUserObject(
	ctx context.Context,
	user *model.User,
	attributes scim.ResourceAttributes,
	converter *convert.Converter,
	logger zerolog.Logger,
) (scim.Resource, error) {
	object, err := converter.SCIMUserToObject(user)
	if err != nil {
		logger.Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sourceUserResp, err := u.dirClient.DS().Writer.SetObject(ctx, &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		logger.Err(err).Msg("failed to create user")
		return scim.Resource{}, err
	}

	return u.processUserResponse(ctx, sourceUserResp, attributes, converter, logger)
}

func (u UsersResourceHandler) processUserResponse(
	ctx context.Context,
	sourceUserResp *dsw.SetObjectResponse,
	attributes scim.ResourceAttributes,
	converter *convert.Converter,
	logger zerolog.Logger,
) (scim.Resource, error) {
	userMap, err := convert.ProtobufStructToMap(sourceUserResp.GetResult().GetProperties())
	if err != nil {
		logger.Err(err).Msg("failed to convert user to map")
		return scim.Resource{}, err
	}

	transformResult, err := converter.TransformResource(userMap, "user")
	if err != nil {
		logger.Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	meta, err := u.dirClient.SetUser(ctx, sourceUserResp.GetResult().GetId(), transformResult, attributes)
	if err != nil {
		logger.Err(err).Msg("failed to sync user")
		return scim.Resource{}, err
	}

	return converter.ObjectToResource(sourceUserResp.GetResult(), meta), nil
}
