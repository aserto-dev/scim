package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Delete(r *http.Request, id string) error {
	logger := u.logger.With().Str("method", "Delete").Str("id", id).Logger()
	logger.Info().Msg("delete user")

	identityRelation, err := u.getIdentityRelation(id, "")
	if err != nil {
		u.logger.Err(err).Msg("failed to get identity relation")
		return err
	}

	var identities []*dsc.Relation

	resp, err := u.dirClient.Reader.GetRelations(r.Context(), &dsr.GetRelationsRequest{
		SubjectType: identityRelation.SubjectType,
		SubjectId:   identityRelation.SubjectId,
		Relation:    identityRelation.Relation,
		ObjectId:    identityRelation.ObjectId,
		ObjectType:  identityRelation.ObjectType,
	})
	if err != nil {
		logger.Err(err).Msg("failed to get identities")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
		return err
	}
	identities = resp.Results

	for _, v := range identities {
		var objectID string
		switch v.ObjectType {
		case u.cfg.SCIM.IdentityObjectType:
			objectID = v.ObjectId
		case u.cfg.SCIM.UserObjectType:
			objectID = v.SubjectId
		default:
			logger.Error().Str("object_type", v.ObjectType).Msg("unexpected object type")
			return serrors.ScimErrorBadRequest("unexpected object type in identity relation")
		}

		logger.Trace().Str("identity", objectID).Msg("deleting identity")
		_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
			ObjectId:      objectID,
			ObjectType:    u.cfg.SCIM.IdentityObjectType,
			WithRelations: true,
		})
		if err != nil {
			logger.Err(err).Msg("failed to delete identity")
			return err
		}
	}

	_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.SCIM.UserObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Msg("failed to delete user")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
	}

	logger.Trace().Msg("user deleted")

	return err
}
