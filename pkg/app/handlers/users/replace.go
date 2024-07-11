package users

import (
	"net/http"

	"github.com/elimity-com/scim"
)

func (u UsersResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u.logger.Trace().Str("user_id", id).Any("attributes", attributes).Msg("replacing user")

	err := u.Delete(r, id)
	if err != nil {
		return scim.Resource{}, err
	}

	resource, err := u.Create(r, attributes)
	if err != nil {
		return scim.Resource{}, err
	}

	return resource, nil
}
