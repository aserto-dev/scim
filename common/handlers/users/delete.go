package users

import (
	"context"

	cerr "github.com/aserto-dev/errors"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Delete(ctx context.Context, id string) error {
	logger := u.logger.With().Str("method", "Delete").Str("id", id).Logger()
	logger.Info().Msg("delete user")

	identityRelation, err := u.cfg.GetIdentityRelation(id, "")
	if err != nil {
		logger.Err(err).Msg("failed to get identity relation")
	}

	resp, err := u.dirClient.DS().Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: identityRelation.SubjectType,
		SubjectId:   identityRelation.SubjectId,
		ObjectType:  identityRelation.ObjectType,
		ObjectId:    identityRelation.ObjectId,
		Relation:    identityRelation.Relation,
	})
	if err != nil {
		logger.Err(err).Msg("failed to get relations")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
		return err
	}

	identities := resp.Results

	for _, v := range identities {
		var objectID string
		switch v.ObjectType {
		case u.cfg.User.IdentityObjectType:
			objectID = v.ObjectId
		case u.cfg.User.ObjectType:
			objectID = v.SubjectId
		default:
			logger.Error().Str("object_type", v.ObjectType).Msg("unexpected object type")
			return serrors.ScimErrorBadRequest("unexpected object type in identity relation")
		}

		logger.Trace().Str("id", v.ObjectId).Msg("deleting identity")
		_, err = u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
			ObjectId:      objectID,
			ObjectType:    u.cfg.User.IdentityObjectType,
			WithRelations: true,
		})
		if err != nil {
			logger.Err(err).Msg("failed to delete identity")
			return err
		}
	}

	logger.Trace().Msg("deleting user")
	_, err = u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.User.ObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Msg("failed to delete user")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
	}

	logger.Trace().Msg("deleting user source object")
	_, err = u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.User.SourceObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Msg("failed to delete user source object")
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
	}

	logger.Trace().Msg("user deleted")
	return err
}
