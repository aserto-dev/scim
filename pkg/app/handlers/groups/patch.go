package groups

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/directory"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func (u GroupResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	u.logger.Trace().Str("group_id", id).Any("operations", operations).Msg("patching group")

	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Resource{}, serrors.ScimErrorInternal
	}

	scimConfig, err := dirClient.GetTransformConfig(r.Context())
	if err != nil {
		return scim.Resource{}, err
	}

	getObjResp, err := dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    scimConfig.SourceGroupType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	converter := common.NewConverter(scimConfig)
	var attr scim.ResourceAttributes
	oldAttr := converter.ObjectToResourceAttributes(getObjResp.Result)

	// object := getObjResp.Result

	for _, op := range operations {
		switch op.Op {
		case scim.PatchOperationAdd:
			attr, err = common.HandlePatchOPAdd(oldAttr, op)
			if err != nil {
				return scim.Resource{}, err
			}
		case scim.PatchOperationRemove:
			attr, err = common.HandlePatchOPRemove(oldAttr, op)
			if err != nil {
				return scim.Resource{}, err
			}
		case scim.PatchOperationReplace:
			attr, err = common.HandlePatchOPReplace(oldAttr, op)
			if err != nil {
				return scim.Resource{}, err
			}
		}
	}

	if err != nil {
		return scim.Resource{}, err
	}
	// object.Etag = getObjResp.Result.Etag
	// resp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
	// 	Object: object,
	// })
	// if err != nil {
	// 	u.logger.Err(err).Msg("error setting object")
	// 	return scim.Resource{}, err
	// }

	transformResult, err := common.TransformResource(attr, scimConfig, "group")
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert group to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	groupObj := getObjResp.Result
	props, err := structpb.NewStruct(attr)
	if err != nil {
		return scim.Resource{}, err
	}
	groupObj.Properties = props
	sourceGroupResp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: groupObj,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	sync := directory.NewSync(scimConfig, dirClient)
	meta, err := sync.UpdateGroup(r.Context(), getObjResp.Result.Id, transformResult)
	if err != nil {
		return scim.Resource{}, err
	}

	resource := u.converter.ObjectToResource(sourceGroupResp.Result, meta)

	return resource, nil
}

// func (u GroupResourceHandler) handlePatchOPAdd(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
// 	var err error
// 	// objectProps := object.Properties.AsMap()
// 	if op.Path == nil || op.Path.ValueExpression == nil {
// 		// simple add property
// 		switch value := op.Value.(type) {
// 		case string:
// 			if objectProps[op.Path.AttributePath.AttributeName] != nil {
// 				return nil, serrors.ScimErrorUniqueness
// 			}
// 			objectProps[op.Path.AttributePath.AttributeName] = op.Value
// 		case map[string]interface{}:
// 			for k, v := range value {
// 				if objectProps[k] != nil {
// 					return nil, serrors.ScimErrorUniqueness
// 				}
// 				objectProps[k] = v
// 			}
// 		case []interface{}:
// 			for _, v := range value {
// 				switch val := v.(type) {
// 				case string:
// 					objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), v)
// 				case map[string]interface{}:
// 					properties := val
// 					objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), properties)
// 					// if op.Path.AttributePath.AttributeName == GroupMembers {
// 					// 	err = u.addUserToGroup(ctx, properties["value"].(string), object.Id)
// 					// 	if err != nil {
// 					// 		return err
// 					// 	}
// 					// }
// 				}
// 			}
// 		}
// 	}

// 	// object.Properties, err = structpb.NewStruct(objectProps)
// 	return objectProps, err
// }

// func (u GroupResourceHandler) handlePatchOPRemove(ctx context.Context, object *dsc.Object, op scim.PatchOperation) error {
// 	var err error
// 	objectProps := object.Properties.AsMap()
// 	// var oldValue interface{}

// 	switch value := objectProps[op.Path.AttributePath.AttributeName].(type) {
// 	case string:
// 		// oldValue = objectProps[op.Path.AttributePath.AttributeName]
// 		delete(objectProps, op.Path.AttributePath.AttributeName)
// 	case []interface{}:
// 		ftr, err := filter.ParseAttrExp([]byte(op.Path.ValueExpression.(*filter.AttributeExpression).String()))
// 		if err != nil {
// 			return err
// 		}

// 		index := -1
// 		if ftr.Operator == filter.EQ {
// 			for i, v := range value {
// 				originalValue := v.(map[string]interface{})
// 				if originalValue[ftr.AttributePath.AttributeName].(string) == ftr.CompareValue {
// 					// oldValue = originalValue
// 					index = i
// 				}
// 			}
// 			if index == -1 {
// 				return serrors.ScimErrorMutability
// 			}
// 			objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{})[:index], objectProps[op.Path.AttributePath.AttributeName].([]interface{})[index+1:]...)
// 		}
// 	}

// 	// if op.Path.AttributePath.AttributeName == GroupMembers {
// 	// 	user := oldValue.(map[string]interface{})["value"].(string)
// 	// 	err = u.removeUserFromGroup(ctx, user, object.Id)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}
// 	// }

// 	object.Properties, err = structpb.NewStruct(objectProps)
// 	return err
// }

// func (u GroupResourceHandler) handlePatchOPReplace(object *dsc.Object, op scim.PatchOperation) error {
// 	var err error
// 	objectProps := object.Properties.AsMap()

// 	switch value := op.Value.(type) {
// 	case string:
// 		objectProps[op.Path.AttributePath.AttributeName] = op.Value
// 	case map[string]interface{}:
// 		for k, v := range value {
// 			objectProps[k] = v
// 		}
// 	}

// 	object.Properties, err = structpb.NewStruct(objectProps)
// 	return err
// }

// func (u GroupResourceHandler) addUserToGroup(ctx context.Context, userID, group string) error {
// 	rel, err := u.dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
// 		SubjectType: u.cfg.SCIM.Transform.UserObjectType,
// 		SubjectId:   userID,
// 		ObjectType:  u.cfg.SCIM.Transform.GroupObjectType,
// 		ObjectId:    group,
// 		Relation:    u.cfg.SCIM.Transform.GroupMemberRelation,
// 	})
// 	if err != nil {
// 		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
// 			_, err = u.dirClient.Writer.SetRelation(ctx, &dsw.SetRelationRequest{
// 				Relation: &dsc.Relation{
// 					SubjectId:   userID,
// 					SubjectType: u.cfg.SCIM.Transform.UserObjectType,
// 					Relation:    u.cfg.SCIM.Transform.GroupMemberRelation,
// 					ObjectType:  u.cfg.SCIM.Transform.GroupObjectType,
// 					ObjectId:    group,
// 				}})
// 			return err
// 		}
// 		return err
// 	}

// 	if rel != nil {
// 		return serrors.ScimErrorUniqueness
// 	}
// 	return nil
// }

// func (u GroupResourceHandler) removeUserFromGroup(ctx context.Context, userID, group string) error {
// 	_, err := u.dirClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
// 		SubjectType: u.cfg.SCIM.Transform.UserObjectType,
// 		SubjectId:   userID,
// 		ObjectType:  u.cfg.SCIM.Transform.GroupObjectType,
// 		ObjectId:    group,
// 		Relation:    u.cfg.SCIM.Transform.GroupMemberRelation,
// 	})
// 	if err != nil {
// 		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrRelationNotFound) {
// 			return serrors.ScimErrorMutability
// 		}
// 		return err
// 	}

// 	_, err = u.dirClient.Writer.DeleteRelation(ctx, &dsw.DeleteRelationRequest{
// 		SubjectType: u.cfg.SCIM.Transform.UserObjectType,
// 		SubjectId:   userID,
// 		ObjectType:  u.cfg.SCIM.Transform.GroupObjectType,
// 		ObjectId:    group,
// 		Relation:    u.cfg.SCIM.Transform.GroupMemberRelation,
// 	})
// 	return err
// }
