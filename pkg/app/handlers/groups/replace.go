package groups

import (
	"net/http"

	"github.com/elimity-com/scim"
)

func (u GroupResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u.logger.Trace().Str("id", id).Any("attributes", attributes).Msg("replacing group")
	err := u.Delete(r, id)
	if err != nil {
		return scim.Resource{}, err
	}
	return u.Create(r, attributes)
}
