package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Delete(r *http.Request, id string) error {
	u.logger.Trace().Str("user_id", id).Msg("deleting user")

	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return serrors.ScimErrorInternal
	}

	relations, err := dirClient.Reader.GetRelations(r.Context(), &dsr.GetRelationsRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   id,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
		return err
	}

	for _, v := range relations.Results {
		if v.Relation == u.cfg.SCIM.IdentityRelation {
			_, err = dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
				ObjectId:      v.ObjectId,
				ObjectType:    v.ObjectType,
				WithRelations: true,
			})
			if err != nil {
				return err
			}
		}
	}

	_, err = dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.SCIM.UserObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
	}

	return err
}
