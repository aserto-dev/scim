package groups

import (
	"net/http"

	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (u GroupResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u.logger.Trace().Any("attributes", attributes).Msg("creating group")
	group, err := common.ResourceAttributesToGroup(attributes)
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	var result scim.Resource
	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Resource{}, serrors.ScimErrorInternal
	}
	scimConfig, err := dirClient.GetTransformConfig(r.Context())
	if err != nil {
		return scim.Resource{}, err
	}

	converter := common.NewConverter(scimConfig)
	object, err := converter.SCIMGroupToObject(group)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert group to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sourceGroupResp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	// groupMap, err := common.ProtobufStructToMap(sourceGroupResp.Result.Properties)
	// if err != nil {
	// 	if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
	// 		return scim.Resource{}, serrors.ScimErrorUniqueness
	// 	}
	// 	return scim.Resource{}, err
	// }

	transformResult, err := common.TransformResource(attributes, scimConfig, "group")
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to transform group")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sync := directory.NewSync(scimConfig, dirClient)
	meta, err := sync.UpdateGroup(r.Context(), sourceGroupResp.Result.Id, transformResult)
	if err != nil {
		return scim.Resource{}, err
	}

	// for _, object := range transformResult.Objects {
	// 	_, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
	// 		Object: object,
	// 	})
	// 	if err != nil {
	// 		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
	// 			return scim.Resource{}, serrors.ScimErrorUniqueness
	// 		}
	// 		return scim.Resource{}, err
	// 	}

	// 	// _, err = dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
	// 	// 	Relation: &dsc.Relation{
	// 	// 		ObjectType:  resp.Result.Type,
	// 	// 		ObjectId:    resp.Result.Id,
	// 	// 		Relation:    u.cfg.SCIM.Transform.SourceRelation,
	// 	// 		SubjectType: u.cfg.SCIM.Transform.SourceGroupType,
	// 	// 		SubjectId:   sourceGroupResp.Result.Id,
	// 	// 	},
	// 	// })

	// 	// if err != nil {
	// 	// 	return scim.Resource{}, err
	// 	// }

	// 	// if object.Type == u.cfg.SCIM.Transform.GroupObjectType {
	// 	// 	err = u.setGroupMappings(r.Context(), dirClient, resp.Result.Id)
	// 	// 	if err != nil {
	// 	// 		return scim.Resource{}, err
	// 	// 	}
	// 	// }
	// }

	// for _, relation := range transformResult.Relations {
	// 	_, err := dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
	// 		Relation: relation,
	// 	})
	// 	if err != nil {
	// 		return scim.Resource{}, err
	// 	}
	// }

	result = u.converter.ObjectToResource(sourceGroupResp.Result, meta)

	return result, nil
}
