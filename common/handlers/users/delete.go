package users

import (
	"context"

	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (u UsersResourceHandler) Delete(ctx context.Context, id string) error {
	logger := u.logger.With().Str("method", "Delete").Str("id", id).Logger()
	logger.Info().Msg("delete user")

	identityRelation, err := u.cfg.GetIdentityRelation(id, "")
	if err != nil {
		logger.Error().Err(err).Msg("failed to get identity relation")
	}

	resp, err := u.dirClient.DS().Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: identityRelation.GetSubjectType(),
		SubjectId:   identityRelation.GetSubjectId(),
		ObjectType:  identityRelation.GetObjectType(),
		ObjectId:    identityRelation.GetObjectId(),
		Relation:    identityRelation.GetRelation(),
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to get relations")
		st, ok := status.FromError(err)

		if ok && st.Code() == codes.NotFound {
			return serrors.ScimErrorResourceNotFound(id)
		}

		return err
	}

	for _, v := range resp.GetResults() {
		var objectID string

		switch v.GetObjectType() {
		case u.cfg.User.IdentityObjectType:
			objectID = v.GetObjectId()
		case u.cfg.User.ObjectType:
			objectID = v.GetSubjectId()
		default:
			logger.Error().Str("object_type", v.GetObjectType()).Msg("unexpected object type")
			return serrors.ScimErrorBadRequest("unexpected object type in identity relation")
		}

		logger.Trace().Str("id", v.GetObjectId()).Msg("deleting identity")

		_, err = u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
			ObjectId:      objectID,
			ObjectType:    u.cfg.User.IdentityObjectType,
			WithRelations: true,
		})
		if err != nil {
			logger.Error().Err(err).Msg("failed to delete identity")
			return err
		}
	}

	logger.Trace().Msg("deleting user")

	if err := u.deleteUserObjects(ctx, id, logger); err != nil {
		return err
	}

	logger.Trace().Msg("user deleted")

	return nil
}

func (u UsersResourceHandler) deleteUserObjects(ctx context.Context, id string, logger zerolog.Logger) error {
	_, err := u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.User.ObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to delete user")
		st, ok := status.FromError(err)

		if ok && st.Code() == codes.NotFound {
			return serrors.ScimErrorResourceNotFound(id)
		}

		return err
	}

	logger.Trace().Msg("deleting user source object")

	_, err = u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.User.SourceObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to delete user source object")
		st, ok := status.FromError(err)

		if ok && st.Code() == codes.NotFound {
			return serrors.ScimErrorResourceNotFound(id)
		}

		return err
	}

	return nil
}
