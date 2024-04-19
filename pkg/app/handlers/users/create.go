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
	object, err := common.ResourceAttributesToObject(attributes, "user", attributes["userName"].(string))
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
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

	err = u.setAllIdentities(r.Context(), resp.Result.Id, attributes)
	if err != nil {
		return scim.Resource{}, err
	}

	if attributes["groups"] != nil {
		err = u.setUserGroups(r.Context(), resp.Result.Id, attributes["groups"].([]string))
		if err != nil {
			return scim.Resource{}, err
		}
	}

	return resource, nil
}
