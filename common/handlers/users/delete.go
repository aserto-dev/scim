package users

import (
	"context"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
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

	if err := u.deleteUserIdentities(ctx, id, logger); err != nil {
		return err
	}

	logger.Trace().Msg("deleting user")

	if err := u.deleteUserObjects(ctx, id, logger); err != nil {
		return err
	}

	logger.Trace().Msg("user deleted")

	return nil
}

func (u UsersResourceHandler) deleteUserIdentities(ctx context.Context, id string, logger zerolog.Logger) error {
	identityRelation, err := u.cfg.ParseIdentityRelation(id, "")
	if err != nil {
		logger.Err(err).Msg("failed to get identity relation")
		return err
	}

	relations, err := u.getUserIdentityRelations(ctx, identityRelation, id, logger)
	if err != nil {
		return err
	}

	return u.deleteIdentityObjects(ctx, relations, logger)
}

func (u UsersResourceHandler) getUserIdentityRelations(
	ctx context.Context,
	identityRelation *dsc.Relation,
	id string,
	logger zerolog.Logger,
) (*dsr.GetRelationsResponse, error) {
	resp, err := u.dirClient.DS().Reader.GetRelations(ctx, &dsr.GetRelationsRequest{
		SubjectType: identityRelation.GetSubjectType(),
		SubjectId:   identityRelation.GetSubjectId(),
		ObjectType:  identityRelation.GetObjectType(),
		ObjectId:    identityRelation.GetObjectId(),
		Relation:    identityRelation.GetRelation(),
	})
	if err != nil {
		logger.Err(err).Msg("failed to get relations")
		st, ok := status.FromError(err)

		if ok && st.Code() == codes.NotFound {
			return nil, serrors.ScimErrorResourceNotFound(id)
		}

		return nil, err
	}

	return resp, nil
}

func (u UsersResourceHandler) deleteIdentityObjects(
	ctx context.Context,
	resp *dsr.GetRelationsResponse,
	logger zerolog.Logger,
) error {
	for _, v := range resp.GetResults() {
		objectID, err := u.getIdentityObjectID(v, logger)
		if err != nil {
			return err
		}

		logger.Trace().Str("id", objectID).Msg("deleting identity")

		if err := u.deleteIdentityObject(ctx, objectID, logger); err != nil {
			return err
		}
	}

	return nil
}

func (u UsersResourceHandler) getIdentityObjectID(relation *dsc.Relation, logger zerolog.Logger) (string, error) {
	switch relation.GetObjectType() {
	case u.cfg.User.IdentityObjectType:
		return relation.GetObjectId(), nil
	case u.cfg.User.ObjectType:
		return relation.GetSubjectId(), nil
	default:
		logger.Error().Str("object_type", relation.GetObjectType()).Msg("unexpected object type")
		return "", serrors.ScimErrorBadRequest("unexpected object type in identity relation")
	}
}

func (u UsersResourceHandler) deleteIdentityObject(ctx context.Context, objectID string, logger zerolog.Logger) error {
	_, err := u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectId:      objectID,
		ObjectType:    u.cfg.User.IdentityObjectType,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Msg("failed to delete identity")
		return err
	}

	return nil
}

func (u UsersResourceHandler) deleteUserObjects(ctx context.Context, id string, logger zerolog.Logger) error {
	_, err := u.dirClient.DS().Writer.DeleteObject(ctx, &dsw.DeleteObjectRequest{
		ObjectType:    u.cfg.User.ObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		logger.Err(err).Msg("failed to delete user")
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
		logger.Err(err).Msg("failed to delete user source object")
		st, ok := status.FromError(err)

		if ok && st.Code() == codes.NotFound {
			return serrors.ScimErrorResourceNotFound(id)
		}

		return err
	}

	return nil
}
