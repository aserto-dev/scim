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
	u.logger.Trace().Str("user_id", id).Any("attributes", attributes).Msg("replacing user")
	getObjResp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    "user",
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	object, err := common.ResourceAttributesToObject(attributes, "user", id)
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}
	object.Id = id
	object.Etag = getObjResp.Result.Etag

	setResp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	if attributes["groups"] != nil {
		err = u.setUserGroups(r.Context(), id, attributes["groups"].([]string))
		if err != nil {
			return scim.Resource{}, err
		}
	}

	createdAt := setResp.Result.CreatedAt.AsTime()
	updatedAt := setResp.Result.UpdatedAt.AsTime()
	resource := common.ObjectToResource(setResp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      setResp.Result.Etag,
	})

	return resource, nil
}
