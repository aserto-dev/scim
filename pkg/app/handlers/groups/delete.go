package groups

import (
	"net/http"

	"github.com/aserto-dev/scim/pkg/directory"
	serrors "github.com/elimity-com/scim/errors"
)

func (u GroupResourceHandler) Delete(r *http.Request, id string) error {
	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return serrors.ScimErrorInternal
	}

	scimConfig, err := dirClient.GetTransformConfig(r.Context())
	if err != nil {
		return err
	}

	sync := directory.NewSync(scimConfig, dirClient)
	err = sync.DeleteGroup(r.Context(), id)

	// relations, err := dirClient.Reader.GetRelations(r.Context(), &dsr.GetRelationsRequest{
	// 	SubjectType: scimConfig.GroupObjectType,
	// 	SubjectId:   id,
	// 	Relation:    scimConfig.SourceRelation,
	// })
	// if err != nil {
	// 	if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
	// 		return serrors.ScimErrorResourceNotFound(id)
	// 	}
	// 	return err
	// }

	// for _, v := range relations.Results {
	// 	_, err = dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
	// 		ObjectId:      v.ObjectId,
	// 		ObjectType:    v.ObjectType,
	// 		WithRelations: true,
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// _, err = dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
	// 	ObjectType:    scimConfig.GroupObjectType,
	// 	ObjectId:      id,
	// 	WithRelations: true,
	// })
	// if err != nil {
	// 	if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
	// 		return serrors.ScimErrorResourceNotFound(id)
	// 	}
	// }

	return err
}
