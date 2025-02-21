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
	"github.com/samber/lo"
)

func (u UsersResourceHandler) Delete(r *http.Request, id string) error {
	if id == "" {
		return serrors.ScimErrorBadRequest("missing id")
	}

	logger := u.logger.With().Str("method", "Delete").Str("id", id).Logger()
	logger.Info().Msg("delete user")
	relations, err := u.dirClient.Reader.GetRelations(r.Context(), &dsr.GetRelationsRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   id,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
		return err
	}

	var identities []*dsc.Relation
	if u.cfg.SCIM.InvertIdentityRelation {
		resp, err := u.dirClient.Reader.GetRelations(r.Context(), &dsr.GetRelationsRequest{
			SubjectType: u.cfg.SCIM.IdentityObjectType,
			Relation:    u.cfg.SCIM.IdentityRelation,
			ObjectId:    id,
		})
		if err != nil {
			if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
				return serrors.ScimErrorResourceNotFound(id)
			}
			return err
		}
		identities = resp.Results
	} else {
		identities = lo.Filter(relations.Results, func(rel *dsc.Relation, i int) bool {
			return rel.Relation == u.cfg.SCIM.IdentityRelation
		})
	}

	for _, v := range identities {
		logger.Trace().Str("id", v.ObjectId).Msg("deleting identity")
		_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
			ObjectId:      v.ObjectId,
			ObjectType:    v.ObjectType,
			WithRelations: true,
		})
		if err != nil {
			return err
		}
	}

	_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
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
