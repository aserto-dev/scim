package groups

import (
	"context"
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	"github.com/scim2/filter-parser/v2"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func (u GroupResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	u.logger.Trace().Str("group_id", id).Any("operations", operations).Msg("patching group")
	getObjResp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    u.cfg.SCIM.GroupObjectType,
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	object := getObjResp.Result

	for _, op := range operations {
		switch op.Op {
		case scim.PatchOperationAdd:
			err := u.handlePatchOPAdd(r.Context(), object, op)
			if err != nil {
				return scim.Resource{}, err
			}
		case scim.PatchOperationRemove:
			err := u.handlePatchOPRemove(r.Context(), object, op)
			if err != nil {
				return scim.Resource{}, err
			}
		case scim.PatchOperationReplace:
			err := u.handlePatchOPReplace(object, op)
			if err != nil {
				return scim.Resource{}, err
			}
		}
	}

	if err != nil {
		return scim.Resource{}, err
	}
	object.Etag = getObjResp.Result.Etag
	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		u.logger.Err(err).Msg("error setting object")
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

func (u GroupResourceHandler) handlePatchOPAdd(ctx context.Context, object *dsc.Object, op scim.PatchOperation) error {
	var err error
	objectProps := object.Properties.AsMap()
	if op.Path == nil || op.Path.ValueExpression == nil {
		// simple add property
		switch value := op.Value.(type) {
		case string:
			if objectProps[op.Path.AttributePath.AttributeName] != nil {
				return serrors.ScimErrorUniqueness
			}
			objectProps[op.Path.AttributePath.AttributeName] = op.Value
		case map[string]interface{}:
			for k, v := range value {
				if objectProps[k] != nil {
					return serrors.ScimErrorUniqueness
				}
				objectProps[k] = v
			}
		case []interface{}:
			for _, v := range value {
				switch val := v.(type) {
				case string:
					objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), v)
				case map[string]interface{}:
					properties := val
					objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), properties)
					if op.Path.AttributePath.AttributeName == GroupMembers {
						err = u.addUserToGroup(ctx, properties["value"].(string), object.Id)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	object.Properties, err = structpb.NewStruct(objectProps)
	return err
}

func (u GroupResourceHandler) handlePatchOPRemove(ctx context.Context, object *dsc.Object, op scim.PatchOperation) error {
	var err error
	objectProps := object.Properties.AsMap()
	var oldValue interface{}

	switch value := objectProps[op.Path.AttributePath.AttributeName].(type) {
	case string:
		oldValue = objectProps[op.Path.AttributePath.AttributeName]
		delete(objectProps, op.Path.AttributePath.AttributeName)
	case []interface{}:
		ftr, err := filter.ParseAttrExp([]byte(op.Path.ValueExpression.(*filter.AttributeExpression).String()))
		if err != nil {
			return err
		}

		index := -1
		if ftr.Operator == filter.EQ {
			for i, v := range value {
				originalValue := v.(map[string]interface{})
				if originalValue[ftr.AttributePath.AttributeName].(string) == ftr.CompareValue {
					oldValue = originalValue
					index = i
				}
			}
			if index == -1 {
				return serrors.ScimErrorMutability
			}
			objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{})[:index], objectProps[op.Path.AttributePath.AttributeName].([]interface{})[index+1:]...)
		}
	}

	if op.Path.AttributePath.AttributeName == GroupMembers {
		user := oldValue.(map[string]interface{})["value"].(string)
		err = u.removeUserFromGroup(ctx, user, object.Id)
		if err != nil {
			return err
		}
	}

	object.Properties, err = structpb.NewStruct(objectProps)
	return err
}

func (u GroupResourceHandler) handlePatchOPReplace(object *dsc.Object, op scim.PatchOperation) error {
	var err error
	objectProps := object.Properties.AsMap()

	switch value := op.Value.(type) {
	case string:
		objectProps[op.Path.AttributePath.AttributeName] = op.Value
	case map[string]interface{}:
		for k, v := range value {
			objectProps[k] = v
		}
	}

	object.Properties, err = structpb.NewStruct(objectProps)
	return err
}

func (u GroupResourceHandler) addUserToGroup(ctx context.Context, userID, group string) error {
	rel, err := u.dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   userID,
		ObjectType:  u.cfg.SCIM.GroupObjectType,
		ObjectId:    group,
		Relation:    u.cfg.SCIM.GroupMemberRelation,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
			_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
				Relation: &dsc.Relation{
					SubjectId:   userID,
					SubjectType: u.cfg.SCIM.UserObjectType,
					Relation:    u.cfg.SCIM.GroupMemberRelation,
					ObjectType:  u.cfg.SCIM.GroupObjectType,
					ObjectId:    group,
				}})
			return err
		}
		return err
	}

	if rel != nil {
		return serrors.ScimErrorUniqueness
	}
	return nil
}

func (u GroupResourceHandler) removeUserFromGroup(ctx context.Context, userID, group string) error {
	_, err := u.dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   userID,
		ObjectType:  u.cfg.SCIM.GroupObjectType,
		ObjectId:    group,
		Relation:    u.cfg.SCIM.GroupMemberRelation,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
			return serrors.ScimErrorMutability
		}
		return err
	}

	_, err = u.dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
		SubjectType: u.cfg.SCIM.UserObjectType,
		SubjectId:   userID,
		ObjectType:  u.cfg.SCIM.GroupObjectType,
		ObjectId:    group,
		Relation:    u.cfg.SCIM.GroupMemberRelation,
	})
	return err
}
