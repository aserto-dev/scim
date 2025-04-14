package groups

import (
	"context"

	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/common"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func (g GroupResourceHandler) Patch(ctx context.Context, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	logger := g.logger.With().Str("method", "Patch").Str("id", id).Logger()
	logger.Info().Msg("patch group")

	if !g.cfg.HasGroups() {
		logger.Error().Msg("groups not enabled")
		return scim.Resource{}, serrors.ScimErrorBadRequest("groups not enabled")
	}

	getObjResp, err := g.dirClient.DS().Reader.GetObject(ctx, &dsr.GetObjectRequest{
		ObjectType:    g.cfg.Group.SourceObjectType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to get group")
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	converter := convert.NewConverter(g.cfg)
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

	transformResult, err := converter.TransformResource(attr, "group")
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert group to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	groupObj := getObjResp.Result
	props, err := structpb.NewStruct(attr)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert attributes to struct")
		return scim.Resource{}, err
	}
	groupObj.Properties = props
	sourceGroupResp, err := g.dirClient.DS().Writer.SetObject(ctx, &dsw.SetObjectRequest{
		Object: groupObj,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to replace group")
		return scim.Resource{}, err
	}

	meta, err := g.dirClient.SetGroup(ctx, getObjResp.Result.Id, transformResult)
	if err != nil {
		logger.Error().Err(err).Msg("failed to sync group")
		return scim.Resource{}, err
	}

	resource := converter.ObjectToResource(sourceGroupResp.Result, meta)

	logger.Trace().Any("response", resource).Msg("group patched")

	return resource, nil
}
