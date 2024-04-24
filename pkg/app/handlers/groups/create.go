package groups

import (
	"net/http"

	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
)

func (u GroupResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	object, err := common.ResourceAttributesToObject(attributes, u.cfg.SCIM.GroupObjectType, attributes["displayName"].(string))
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	err = u.setGroupMappings(r.Context(), resp.Result.Id)
	if err != nil {
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := common.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	return resource, nil
}
