package groups

import (
	"net/http"

	"github.com/elimity-com/scim"
)

func (u GroupResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	err := u.Delete(r, id)
	if err != nil {
		return scim.Resource{}, err
	}
	return u.Create(r, attributes)
	// dirClient, err := u.getDirectoryClient(r)
	// if err != nil {
	// 	u.logger.Error().Err(err).Msg("failed to get directory client")
	// 	return scim.Resource{}, serrors.ScimErrorInternal
	// }

	// getObjResp, err := dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
	// 	ObjectType:    u.cfg.SCIM.GroupObjectType,
	// 	ObjectId:      id,
	// 	WithRelations: true,
	// })
	// if err != nil {
	// 	if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
	// 		return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
	// 	}
	// 	return scim.Resource{}, err
	// }

	// object, err := common.ResourceAttributesToGroup(attributes)
	// if err != nil {
	// 	return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	// }
	// object.Id = id
	// object.Etag = getObjResp.Result.Etag

	// setResp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
	// 	Object: object,
	// })
	// if err != nil {
	// 	return scim.Resource{}, err
	// }

	// createdAt := setResp.Result.CreatedAt.AsTime()
	// updatedAt := setResp.Result.UpdatedAt.AsTime()
	// resource := common.ObjectToResource(setResp.Result, scim.Meta{
	// 	Created:      &createdAt,
	// 	LastModified: &updatedAt,
	// 	Version:      setResp.Result.Etag,
	// })

	// return resource, nil
}
