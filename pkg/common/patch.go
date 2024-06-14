package common

import (
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/scim2/filter-parser/v2"
)

func HandlePatchOPAdd(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error

	if op.Path == nil || op.Path.ValueExpression == nil {
		// simple add property
		switch value := op.Value.(type) {
		case string:
			if objectProps[op.Path.AttributePath.AttributeName] != nil {
				return nil, serrors.ScimErrorUniqueness
			}
			objectProps[op.Path.AttributePath.AttributeName] = op.Value
		case map[string]interface{}:
			for k, v := range value {
				if objectProps[k] != nil {
					return nil, serrors.ScimErrorUniqueness
				}
				objectProps[k] = v
			}
		case []interface{}:
			for _, v := range value {
				switch val := v.(type) {
				case string:
					if objectProps[op.Path.AttributePath.AttributeName] == nil {
						objectProps[op.Path.AttributePath.AttributeName] = make([]string, 0)
					}
					objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), v)
				case map[string]interface{}:
					if objectProps[op.Path.AttributePath.AttributeName] == nil {
						objectProps[op.Path.AttributePath.AttributeName] = make([]interface{}, 0)
					}
					properties := val
					objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), properties)
				}
			}
		}
	} else {
		fltr, err := filter.ParseAttrExp([]byte(op.Path.ValueExpression.(*filter.AttributeExpression).String()))
		if err != nil {
			return nil, err
		}

		// switch op.Path.AttributePath.AttributeName {
		// case Emails, Groups:
		properties := make(map[string]interface{})
		if op.Path.ValueExpression != nil {
			if objectProps[op.Path.AttributePath.AttributeName] != nil {
				for _, v := range objectProps[op.Path.AttributePath.AttributeName].([]interface{}) {
					originalValue := v.(map[string]interface{})
					if fltr.Operator == filter.EQ {
						if originalValue[fltr.AttributePath.AttributeName].(string) == fltr.CompareValue {
							if originalValue[*op.Path.SubAttribute] != nil {
								return nil, serrors.ScimErrorUniqueness
							}
							properties = originalValue
						}
					}
				}
			} else {
				objectProps[op.Path.AttributePath.AttributeName] = make([]interface{}, 0)
			}
			if len(properties) == 0 {
				properties[fltr.AttributePath.AttributeName] = fltr.CompareValue
				properties[*op.Path.SubAttribute] = op.Value
				objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{}), properties)
			}
		} else {
			properties[*op.Path.SubAttribute] = op.Value
		}

		// if op.Path.AttributePath.AttributeName == Emails && u.cfg.SCIM.CreateEmailIdentities {
		// 	err = u.setIdentity(ctx, dirClient, object.Id, op.Value.(string), map[string]interface{}{IdentityKindKey: "IDENTITY_KIND_EMAIL"})
		// 	if err != nil {
		// 		return err
		// 	}
		// } else if op.Path.AttributePath.AttributeName == Groups {
		// 	err = u.addUserToGroup(ctx, dirClient, object.Id, op.Value.(string))
		// 	if err != nil {
		// 		return err
		// 	}
		// }
		// }
	}

	// object.Properties, err = structpb.NewStruct(objectProps)
	return objectProps, err
}

func HandlePatchOPRemove(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error
	// objectProps, ok := object.Properties.AsMap()[u.cfg.SCIM.SCIMUserPropertyKey].(map[string]interface{})
	// if !ok {
	// 	return errors.New("failed to get user properties")
	// }
	// var oldValue interface{}

	switch value := objectProps[op.Path.AttributePath.AttributeName].(type) {
	case string:
		// oldValue = objectProps[op.Path.AttributePath.AttributeName]
		delete(objectProps, op.Path.AttributePath.AttributeName)
	case []interface{}:
		ftr, err := filter.ParseAttrExp([]byte(op.Path.ValueExpression.(*filter.AttributeExpression).String()))
		if err != nil {
			return nil, err
		}

		index := -1
		if ftr.Operator == filter.EQ {
			for i, v := range value {
				originalValue := v.(map[string]interface{})
				if originalValue[ftr.AttributePath.AttributeName].(string) == ftr.CompareValue {
					// oldValue = originalValue
					index = i
				}
			}
			if index == -1 {
				return nil, serrors.ScimErrorMutability
			}
			objectProps[op.Path.AttributePath.AttributeName] = append(objectProps[op.Path.AttributePath.AttributeName].([]interface{})[:index], objectProps[op.Path.AttributePath.AttributeName].([]interface{})[index+1:]...)
		}
	}

	// if op.Path.AttributePath.AttributeName == Emails && u.cfg.SCIM.CreateEmailIdentities {
	// 	email := oldValue.(map[string]interface{})["value"].(string)
	// 	err = u.removeIdentity(ctx, dirClient, email)
	// 	if err != nil {
	// 		return err
	// 	}
	// } else if op.Path.AttributePath.AttributeName == Groups {
	// 	group := oldValue.(map[string]interface{})["value"].(string)
	// 	err = u.removeUserFromGroup(ctx, dirClient, object.Id, group)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// object.Properties, err = structpb.NewStruct(objectProps)
	return objectProps, err
}

func HandlePatchOPReplace(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error
	// objectProps, ok := object.Properties.AsMap()[u.cfg.SCIM.SCIMUserPropertyKey].(map[string]interface{})
	// if !ok {
	// 	return errors.New("failed to get user properties")
	// }

	switch value := op.Value.(type) {
	case string:
		objectProps[op.Path.AttributePath.AttributeName] = op.Value
	case map[string]interface{}:
		for k, v := range value {
			// if k == "active" {
			// 	objectProps["enabled"] = v
			// }
			objectProps[k] = v
		}
	}

	// object.Properties, err = structpb.NewStruct(objectProps)
	return objectProps, err
}
