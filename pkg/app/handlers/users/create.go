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
	u.logger.Trace().Any("attributes", attributes).Msg("creating user")
	user, err := common.ResourceAttributesToUser(attributes)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert attributes to user")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	object, err := common.UserToObject(user)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Resource{}, serrors.ScimErrorInternal
	}

	resp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
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

	err = u.setAllIdentities(r.Context(), dirClient, resp.Result.Id, user)
	if err != nil {
		return scim.Resource{}, err
	}

	err = u.setUserGroups(r.Context(), dirClient, resp.Result.Id, user.Groups)
	if err != nil {
		return scim.Resource{}, err
	}

	err = u.setUserMappings(r.Context(), dirClient, resp.Result.Id)
	if err != nil {
		return scim.Resource{}, err
	}

	return resource, nil
}
