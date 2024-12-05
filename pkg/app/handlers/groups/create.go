package groups

import (
	"net/http"

	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/pkg/convert"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (u GroupResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u.logger.Trace().Any("attributes", attributes).Msg("creating group")
	group, err := convert.ResourceAttributesToGroup(attributes)
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	var result scim.Resource
	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Resource{}, serrors.ScimErrorInternal
	}
	scimConfigMap, err := dirClient.GetTransformConfigMap(r.Context(), u.cfg.SCIM.SCIMConfigKey)
	if err != nil {
		return scim.Resource{}, err
	}
	scimConfig, err := convert.TransformConfigFromMap(&u.cfg.SCIM.TransformDefaults, scimConfigMap)
	if err != nil {
		return scim.Resource{}, err
	}

	converter := convert.NewConverter(scimConfig)
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

	transformResult, err := convert.TransformResource(attributes, scimConfig, "group")
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to transform group")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	sync := directory.NewSync(scimConfig, dirClient)
	meta, err := sync.UpdateGroup(r.Context(), sourceGroupResp.Result.Id, transformResult)
	if err != nil {
		return scim.Resource{}, err
	}

	result = u.converter.ObjectToResource(sourceGroupResp.Result, meta)

	return result, nil
}
