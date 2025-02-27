package groups

import (
	"net/http"

	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (u GroupResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Create").Str("displayName", attributes["displayName"].(string)).Logger()
	logger.Info().Msg("create group")
	logger.Trace().Any("attributes", attributes).Msg("creating group")

	object, err := common.ResourceAttributesToObject(attributes, u.cfg.SCIM.GroupObjectType, attributes["displayName"].(string))
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert attributes to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to create group")
		return scim.Resource{}, err
	}

	logger.Trace().Any("response", resp.Result).Msg("group object created")

	err = u.setGroupMappings(r.Context(), resp.Result.Id)
	if err != nil {
		logger.Err(err).Msg("failed to set group mappings")
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := common.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	logger.Trace().Any("resource", resource).Msg("group created")

	return resource, nil
}
