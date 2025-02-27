package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	logger := u.logger.With().Str("method", "Create").Str("userName", attributes["userName"].(string)).Logger()
	logger.Info().Msg("create user")
	logger.Trace().Any("attributes", attributes).Msg("creating user")
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

	logger.Trace().Any("object", object).Msg("creating user object")
	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		logger.Err(err).Msg("failed to create user")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
			return scim.Resource{}, serrors.ScimErrorUniqueness
		}
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := common.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	err = u.setAllIdentities(r.Context(), resp.Result.Id, user)
	if err != nil {
		logger.Err(err).Msg("failed to set identities")
		return scim.Resource{}, err
	}

	err = u.setUserGroups(r.Context(), resp.Result.Id, user.Groups)
	if err != nil {
		logger.Err(err).Msg("failed to set groups")
		return scim.Resource{}, err
	}

	err = u.setUserMappings(r.Context(), resp.Result.Id)
	if err != nil {
		logger.Err(err).Msg("failed to set mappings")
		return scim.Resource{}, err
	}

	logger.Trace().Any("resource", resource).Msg("user created")

	return resource, nil
}
