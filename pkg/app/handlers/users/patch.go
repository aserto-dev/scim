package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/convert"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
)

func (u UsersResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	u.logger.Trace().Str("user_id", id).Any("operations", operations).Msg("patching user")

	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Resource{}, serrors.ScimErrorInternal
	}

	scimConfigMap, err := dirClient.GetTransformConfigMap(r.Context())
	if err != nil {
		return scim.Resource{}, err
	}
	scimConfig, err := convert.TransformConfigFromMap(u.cfg.SCIM.TransformDefaults, scimConfigMap)
	if err != nil {
		return scim.Resource{}, err
	}

	converter := convert.NewConverter(scimConfig)

	getObjResp, err := dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    scimConfig.SourceUserType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	var attr scim.ResourceAttributes
	oldAttr := converter.ObjectToResourceAttributes(getObjResp.Result)

	for _, op := range operations {
		switch op.Op {
		case scim.PatchOperationAdd:
			attr, err = common.HandlePatchOPAdd(oldAttr, op)
			if err != nil {
				return scim.Resource{}, err
			}
		case scim.PatchOperationRemove:
			attr, err = common.HandlePatchOPRemove(oldAttr, op)
			if err != nil {
				return scim.Resource{}, err
			}
		case scim.PatchOperationReplace:
			attr, err = common.HandlePatchOPReplace(oldAttr, op)
			if err != nil {
				return scim.Resource{}, err
			}
		}
	}

	if err != nil {
		return scim.Resource{}, err
	}

	transformResult, err := convert.TransformResource(attr, scimConfig, "user")
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	userObj := getObjResp.Result
	props, err := structpb.NewStruct(attr)
	if err != nil {
		return scim.Resource{}, err
	}
	userObj.Properties = props
	sourceUserResp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: userObj,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	sync := directory.NewSync(scimConfig, dirClient)
	meta, err := sync.UpdateUser(r.Context(), getObjResp.Result.Id, transformResult, attr)
	if err != nil {
		return scim.Resource{}, err
	}

	resource := converter.ObjectToResource(sourceUserResp.Result, meta)

	return resource, nil
}
