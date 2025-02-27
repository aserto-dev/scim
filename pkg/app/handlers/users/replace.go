package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Replace").Str("id", id).Logger()
	logger.Info().Msg("replace user")
	logger.Trace().Any("attributes", attributes).Msg("replacing user")
	getObjResp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    u.cfg.SCIM.UserObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Msg("failed to get user")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	user, err := common.ResourceAttributesToUser(attributes)
	if err != nil {
		logger.Err(err).Msg("failed to convert attributes to user")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	object, err := common.UserToObject(user)
	if err != nil {
		logger.Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}
	object.Id = id
	object.Etag = getObjResp.Result.Etag

	setResp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		logger.Err(err).Msg("failed to replace user")
		return scim.Resource{}, err
	}

	err = u.setAllIdentities(r.Context(), id, user)
	if err != nil {
		logger.Err(err).Msg("failed to set identities")
		return scim.Resource{}, err
	}

	err = u.setUserGroups(r.Context(), id, user.Groups)
	if err != nil {
		logger.Err(err).Msg("failed to set groups")
		return scim.Resource{}, err
	}

	err = u.setUserMappings(r.Context(), id)
	if err != nil {
		logger.Err(err).Msg("failed to set mappings")
		return scim.Resource{}, err
	}

	createdAt := setResp.Result.CreatedAt.AsTime()
	updatedAt := setResp.Result.UpdatedAt.AsTime()
	resource := common.ObjectToResource(setResp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      setResp.Result.Etag,
	})

	logger.Trace().Any("resource", resource).Msg("user replaced")

	return resource, nil
}
