package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u.logger.Trace().Any("attributes", attributes).Msg("creating user")
	user, err := common.ResourceAttributesToUser(attributes)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert attributes to user")
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
	object, err := converter.SCIMUserToObject(user)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}
	sourceUserResp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	userMap, err := common.ProtobufStructToMap(sourceUserResp.Result.Properties)
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
			return scim.Resource{}, serrors.ScimErrorUniqueness
		}
		return scim.Resource{}, err
	}

	transformResult, err := common.TransformResource(userMap, scimConfig, "user")
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sync := directory.NewSync(scimConfig, dirClient)
	meta, err := sync.UpdateUser(r.Context(), sourceUserResp.Result.Id, transformResult)
	if err != nil {
		return scim.Resource{}, err
	}

	// for _, object := range transformResult.Objects {
	// 	resp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
	// 		Object: object,
	// 	})
	// 	if err != nil {
	// 		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
	// 			return scim.Resource{}, serrors.ScimErrorUniqueness
	// 		}
	// 		return scim.Resource{}, err
	// 	}

	// 	_, err = dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
	// 		Relation: &dsc.Relation{
	// 			ObjectType:  resp.Result.Type,
	// 			ObjectId:    resp.Result.Id,
	// 			Relation:    u.cfg.SCIM.Transform.SourceRelation,
	// 			SubjectType: u.cfg.SCIM.Transform.SourceUserType,
	// 			SubjectId:   sourceUserResp.Result.Id,
	// 		},
	// 	})

	// 	if err != nil {
	// 		return scim.Resource{}, err
	// 	}

	// 	if object.Type == u.cfg.SCIM.Transform.UserObjectType {
	// 		err = u.setUserMappings(r.Context(), dirClient, resp.Result.Id)
	// 		if err != nil {
	// 			return scim.Resource{}, err
	// 		}
	// 	}
	// }

	// for _, relation := range transformResult.Relations {
	// 	_, err := dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
	// 		Relation: relation,
	// 	})
	// 	if err != nil {
	// 		return scim.Resource{}, err
	// 	}
	// }

	// err = u.setAllIdentities(r.Context(), dirClient, resp.Result.Id, user)
	// if err != nil {
	// 	return scim.Resource{}, err
	// }

	// err = u.setUserGroups(r.Context(), dirClient, resp.Result.Id, user.Groups)
	// if err != nil {
	// 	return scim.Resource{}, err
	// }

	result = converter.ObjectToResource(sourceUserResp.Result, meta)

	return result, nil
}
