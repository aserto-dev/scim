package users

import (
	"context"

	cerr "github.com/aserto-dev/errors"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/common"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
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
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	attr := converter.ObjectToResourceAttributes(getObjResp.Result)

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

	if err != nil {
		logger.Error().Err(err).Msg("error handling patch operation")
		return scim.Resource{}, err
	}

	transformResult, err := converter.TransformResource(attr, "user")
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	userObj := getObjResp.Result
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

	meta, err := u.dirClient.SetUser(ctx, getObjResp.Result.Id, transformResult, attr)
	if err != nil {
		logger.Error().Err(err).Msg("failed to sync user")
		return scim.Resource{}, err
	}

	resource := converter.ObjectToResource(sourceUserResp.Result, meta)

	logger.Trace().Any("response", resource).Msg("user patched")

	return resource, nil
}
