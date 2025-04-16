package users

import (
	"context"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/common"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func (u UsersResourceHandler) Patch(ctx context.Context, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Patch").Str("id", id).Logger()
	logger.Info().Msg("patch user")
	logger.Trace().Any("operations", operations).Msg("patching user")

	converter := convert.NewConverter(u.cfg)

	getObjResp, err := u.dirClient.DS().Reader.GetObject(ctx, &dsr.GetObjectRequest{
		ObjectType:    u.cfg.User.SourceObjectType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to get user")
		st, ok := status.FromError(err)

		if ok && st.Code() == codes.NotFound {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}

		return scim.Resource{}, err
	}

	attr := converter.ObjectToResourceAttributes(getObjResp.GetResult())

	for _, op := range operations {
		switch op.Op {
		case scim.PatchOperationAdd:
			attr, err = common.HandlePatchOPAdd(attr, op)
			if err != nil {
				logger.Error().Err(err).Msg("error adding property")
				return scim.Resource{}, err
			}
		case scim.PatchOperationRemove:
			attr, err = common.HandlePatchOPRemove(attr, op)
			if err != nil {
				logger.Error().Err(err).Msg("error removing property")
				return scim.Resource{}, err
			}
		case scim.PatchOperationReplace:
			attr, err = common.HandlePatchOPReplace(attr, op)
			if err != nil {
				logger.Error().Err(err).Msg("error replacing property")
				return scim.Resource{}, err
			}
		}
	}

	resource, err := u.updateUser(ctx, attr, getObjResp.GetResult(), converter, logger)
	if err != nil {
		return scim.Resource{}, err
	}

	logger.Trace().Any("response", resource).Msg("user patched")

	return resource, nil
}

func (u UsersResourceHandler) updateUser(
	ctx context.Context,
	attr map[string]interface{},
	userObj *dsc.Object,
	converter *convert.Converter,
	logger zerolog.Logger,
) (scim.Resource, error) {
	transformResult, err := converter.TransformResource(attr, "user")
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	props, err := structpb.NewStruct(attr)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert resource attributes to struct")
		return scim.Resource{}, err
	}

	userObj.Properties = props

	sourceUserResp, err := u.dirClient.DS().Writer.SetObject(ctx, &dsw.SetObjectRequest{
		Object: userObj,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to replace user")
		return scim.Resource{}, err
	}

	meta, err := u.dirClient.SetUser(ctx, userObj.GetId(), transformResult, attr)
	if err != nil {
		logger.Error().Err(err).Msg("failed to sync user")
		return scim.Resource{}, err
	}

	return converter.ObjectToResource(sourceUserResp.GetResult(), meta), nil
}
