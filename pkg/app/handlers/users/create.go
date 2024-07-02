package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/convert"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u.logger.Trace().Any("attributes", attributes).Msg("creating user")
	user, err := convert.ResourceAttributesToUser(attributes)
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

	scimConfigMap, err := dirClient.GetTransformConfigMap(r.Context())
	if err != nil {
		return scim.Resource{}, err
	}
	scimConfig, err := convert.TransformConfigFromMap(u.cfg.SCIM.TransformDefaults, scimConfigMap)
	if err != nil {
		return scim.Resource{}, err
	}

	converter := convert.NewConverter(scimConfig)
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

	userMap, err := convert.ProtobufStructToMap(sourceUserResp.Result.Properties)
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
			return scim.Resource{}, serrors.ScimErrorUniqueness
		}
		return scim.Resource{}, err
	}

	transformResult, err := convert.TransformResource(userMap, scimConfig, "user")
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sync := directory.NewSync(scimConfig, dirClient)
	meta, err := sync.UpdateUser(r.Context(), sourceUserResp.Result.Id, transformResult, attributes)
	if err != nil {
		return scim.Resource{}, err
	}

	result = converter.ObjectToResource(sourceUserResp.Result, meta)

	return result, nil
}
