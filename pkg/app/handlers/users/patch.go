package users

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
	"google.golang.org/protobuf/types/known/structpb"
)

func (u UsersResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	u.logger.Trace().Str("user_id", id).Any("operations", operations).Msg("patching user")

	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return scim.Resource{}, serrors.ScimErrorInternal
	}

	scimConfig, err := dirClient.GetTransformConfig(r.Context())
	if err != nil {
		return scim.Resource{}, err
	}

	converter := common.NewConverter(scimConfig)

	getObjResp, err := dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    scimConfig.SourceUserType,
		ObjectId:      id,
		WithRelations: false,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return scim.Resource{}, serrors.ScimErrorResourceNotFound(id)
		}
		return scim.Resource{}, err
	}

	var attr scim.ResourceAttributes
	oldAttr := converter.ObjectToResourceAttributes(getObjResp.Result)

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

	transformResult, err := common.TransformResource(attr, scimConfig, "user")
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to convert user to object")
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	userObj := getObjResp.Result
	props, err := structpb.NewStruct(attr)
	if err != nil {
		return scim.Resource{}, err
	}
	userObj.Properties = props
	sourceUserResp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: userObj,
	})
	if err != nil {
		return scim.Resource{}, err
	}

	sync := directory.NewSync(scimConfig, dirClient)
	meta, err := sync.UpdateUser(r.Context(), getObjResp.Result.Id, transformResult)
	if err != nil {
		return scim.Resource{}, err
	}
	// var resource scim.Resource
	// for _, object := range transformResult.Objects {
	// 	// if object.Type == u.cfg.SCIM.UserObjectType {
	// 	// 	if object.Properties == nil {
	// 	// 		object.Properties = &structpb.Struct{}
	// 	// 	}
	// 	// 	object.Properties = structpb.NewValue(attr)
	// 	// 	if err != nil {
	// 	// 		u.logger.Error().Err(err).Msg("failed to set user properties")
	// 	// 		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	// 	// 	}
	// 	// }
	// 	resp, err := dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
	// 		Object: object,
	// 	})
	// 	if err != nil {
	// 		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
	// 			return scim.Resource{}, serrors.ScimErrorUniqueness
	// 		}
	// 		return scim.Resource{}, err
	// 	}
	// 	// if object.Type == u.cfg.SCIM.UserObjectType {
	// 	// 	createdAt := resp.Result.CreatedAt.AsTime()
	// 	// 	updatedAt := resp.Result.UpdatedAt.AsTime()

	// 	// 	resource = u.converter.ObjectToResource(resp.Result, scim.Meta{
	// 	// 		Created:      &createdAt,
	// 	// 		LastModified: &updatedAt,
	// 	// 		Version:      resp.Result.Etag,
	// 	// 	})

	// 	// 	err = u.setUserMappings(r.Context(), dirClient, resp.Result.Id)
	// 	// 	if err != nil {
	// 	// 		return scim.Resource{}, err
	// 	// 	}
	// 	// }

	// 	_, err = dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
	// 		Relation: &dsc.Relation{
	// 			ObjectType:  resp.Result.Type,
	// 			ObjectId:    resp.Result.Id,
	// 			Relation:    u.cfg.SCIM.Transform.SourceRelation,
	// 			SubjectType: u.cfg.SCIM.Transform.SourceUserType,
	// 			SubjectId:   sourceUserResp.Result.Id,
	// 		},
	// 	})

	// 	if err != nil {
	// 		return scim.Resource{}, err
	// 	}
	// }

	// createdAt := sourceUserResp.Result.CreatedAt.AsTime()
	// updatedAt := sourceUserResp.Result.UpdatedAt.AsTime()
	resource := converter.ObjectToResource(sourceUserResp.Result, meta)

	return resource, nil
}

// func (u UsersResourceHandler) handlePatchOPAdd(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
// 	var err error

// 	if op.Path == nil || op.Path.ValueExpression == nil {
// 		// simple add property
// 		switch v := op.Value.(type) {
// 		case string:
// 			if objectProps[op.Path.AttributePath.AttributeName] != nil {
// 				return nil, serrors.ScimErrorUniqueness
// 			}
// 			objectProps[op.Path.AttributePath.AttributeName] = op.Value
// 		case map[string]interface{}:
// 			value := v
// 			for k, v := range value {
// 				if objectProps[k] != nil {
// 					return nil, serrors.ScimErrorUniqueness
// 				}
// 				objectProps[k] = v
// 			}
// 		}
// 	} else {
// 		fltr, err := filter.ParseAttrExp([]byte(op.Path.ValueExpression.(*filter.AttributeExpression).String()))
// 		if err != nil {
// 			return nil, err
// 		}

// 		// switch op.Path.AttributePath.AttributeName {
// 		// case Emails, Groups:
// 		properties := make(map[string]interface{})
// 		if op.Path.ValueExpression != nil {
// 			if objectProps[op.Path.AttributePath.AttributeName] != nil {
// 				for _, v := range objectProps[op.Path.AttributePath.AttributeName].([]interface{}) {
// 					originalValue := v.(map[string]interface{})
// 					if fltr.Operator == filter.EQ {
// 						if originalValue[fltr.AttributePath.AttributeName].(string) == fltr.CompareValue {
// 							if originalValue[*op.Path.SubAttribute] != nil {
// 								return nil, serrors.ScimErrorUniqueness
// 							}
// 							properties = originalValue
// 						}
// 					}
// 				}
// 			} else {
// 				objectProps[op.Path.AttributePath.AttributeName] = make([]interface{}, 0)
// 			}
// 			if len(properties) == 0 {
// 				properties[fltr.AttributePath.AttributeName] = fltr.CompareValue
// 				properties[*op.Path.SubAttribute] = op.Value
// 				objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), properties)
// 			}
// 		} else {
// 			properties[*op.Path.SubAttribute] = op.Value
// 		}

// 		// if op.Path.AttributePath.AttributeName == Emails && u.cfg.SCIM.CreateEmailIdentities {
// 		// 	err = u.setIdentity(ctx, dirClient, object.Id, op.Value.(string), map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_EMAIL"})
// 		// 	if err != nil {
// 		// 		return err
// 		// 	}
// 		// } else if op.Path.AttributePath.AttributeName == Groups {
// 		// 	err = u.addUserToGroup(ctx, dirClient, object.Id, op.Value.(string))
// 		// 	if err != nil {
// 		// 		return err
// 		// 	}
// 		// }
// 		// }
// 	}

// 	// object.Properties, err = structpb.NewStruct(objectProps)
// 	return objectProps, err
// }

// func (u UsersResourceHandler) handlePatchOPRemove(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
// 	var err error
// 	// objectProps, ok := object.Properties.AsMap()[u.cfg.SCIM.SCIMUserPropertyKey].(map[string]interface{})
// 	// if !ok {
// 	// 	return errors.New("failed to get user properties")
// 	// }
// 	// var oldValue interface{}

// 	switch value := objectProps[op.Path.AttributePath.AttributeName].(type) {
// 	case string:
// 		// oldValue = objectProps[op.Path.AttributePath.AttributeName]
// 		delete(objectProps, op.Path.AttributePath.AttributeName)
// 	case []interface{}:
// 		ftr, err := filter.ParseAttrExp([]byte(op.Path.ValueExpression.(*filter.AttributeExpression).String()))
// 		if err != nil {
// 			return nil, err
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
// 				return nil, serrors.ScimErrorMutability
// 			}
// 			objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{})[:index], objectProps[op.Path.AttributePath.AttributeName].([]interface{})[index+1:]...)
// 		}
// 	}

// 	// if op.Path.AttributePath.AttributeName == Emails && u.cfg.SCIM.CreateEmailIdentities {
// 	// 	email := oldValue.(map[string]interface{})["value"].(string)
// 	// 	err = u.removeIdentity(ctx, dirClient, email)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}
// 	// } else if op.Path.AttributePath.AttributeName == Groups {
// 	// 	group := oldValue.(map[string]interface{})["value"].(string)
// 	// 	err = u.removeUserFromGroup(ctx, dirClient, object.Id, group)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}
// 	// }

// 	// object.Properties, err = structpb.NewStruct(objectProps)
// 	return objectProps, err
// }

// func (u UsersResourceHandler) handlePatchOPReplace(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
// 	var err error
// 	// objectProps, ok := object.Properties.AsMap()[u.cfg.SCIM.SCIMUserPropertyKey].(map[string]interface{})
// 	// if !ok {
// 	// 	return errors.New("failed to get user properties")
// 	// }

// 	switch value := op.Value.(type) {
// 	case string:
// 		objectProps[op.Path.AttributePath.AttributeName] = op.Value
// 	case map[string]interface{}:
// 		for k, v := range value {
// 			// if k == "active" {
// 			// 	objectProps["enabled"] = v
// 			// }
// 			objectProps[k] = v
// 		}
// 	}

// 	// object.Properties, err = structpb.NewStruct(objectProps)
// 	return objectProps, err
// }
