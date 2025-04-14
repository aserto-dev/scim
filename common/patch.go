package common

import (
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/scim2/filter-parser/v2"
)

func HandlePatchOPAdd(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error

	if op.Path == nil || op.Path.ValueExpression == nil {
		return AddProperty(objectProps, op)
	}

	valueExpression, ok := op.Path.ValueExpression.(*filter.AttributeExpression)
	if !ok {
		return nil, serrors.ScimErrorInvalidPath
	}

	fltr, err := filter.ParseAttrExp([]byte(valueExpression.String()))
	if err != nil {
		return nil, err
	}

	properties := make(map[string]interface{})
	if op.Path.ValueExpression == nil {
		properties[*op.Path.SubAttribute] = op.Value

		return objectProps, nil
	}

	if objectProps[op.Path.AttributePath.AttributeName] != nil {
		attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]interface{})
		if !ok {
			return nil, serrors.ScimErrorInvalidPath
		}
		for _, v := range attrProps {
			originalValue, ok := v.(map[string]interface{})
			if !ok {
				return nil, serrors.ScimErrorInvalidPath
			}
			switch fltr.Operator {
			case filter.EQ:
				value, ok := originalValue[fltr.AttributePath.AttributeName].(string)
				if ok && value == fltr.CompareValue {
					if originalValue[*op.Path.SubAttribute] != nil {
						return nil, serrors.ScimErrorUniqueness
					}
					properties = originalValue
				}
			case filter.PR, filter.NE, filter.CO, filter.SW, filter.EW, filter.GT, filter.GE, filter.LT, filter.LE:
				return nil, serrors.ScimErrorBadRequest("operand not supported")
			}
		}
	} else {
		objectProps[op.Path.AttributePath.AttributeName] = make([]interface{}, 0)
	}
	if len(properties) == 0 {
		properties[fltr.AttributePath.AttributeName] = fltr.CompareValue
		properties[*op.Path.SubAttribute] = op.Value
		attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]interface{})
		if !ok {
			return nil, serrors.ScimErrorInvalidPath
		}
		objectProps[op.Path.AttributePath.AttributeName] = append(attrProps, properties)
	}

	return objectProps, err
}

func HandlePatchOPRemove(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error

	switch value := objectProps[op.Path.AttributePath.AttributeName].(type) {
	case string:
		delete(objectProps, op.Path.AttributePath.AttributeName)
	case []interface{}:
		attrExpr, ok := op.Path.ValueExpression.(*filter.AttributeExpression)
		if !ok {
			return nil, serrors.ScimErrorInvalidPath
		}
		ftr, err := filter.ParseAttrExp([]byte(attrExpr.String()))
		if err != nil {
			return nil, err
		}

		index := -1
		if ftr.Operator == filter.EQ {
			for i, v := range value {
				originalValue, ok := v.(map[string]interface{})
				if !ok {
					return nil, serrors.ScimErrorInvalidPath
				}
				value, ok := originalValue[ftr.AttributePath.AttributeName].(string)
				if ok && value == ftr.CompareValue {
					index = i
				}
			}
			if index == -1 {
				return nil, serrors.ScimErrorMutability
			}
			attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]interface{})
			if !ok {
				return nil, serrors.ScimErrorInvalidPath
			}
			objectProps[op.Path.AttributePath.AttributeName] = append(attrProps[:index], attrProps[index+1:]...)
		}
	}

	return objectProps, err
}

func HandlePatchOPReplace(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error

	if op.Path == nil {
		value, ok := op.Value.(map[string]interface{})
		if ok {
			for k, v := range value {
				objectProps[k] = v
			}
		}

		return objectProps, nil
	}

	switch value := objectProps[op.Path.AttributePath.AttributeName].(type) {
	case string:
		objectProps[op.Path.AttributePath.AttributeName] = op.Value
	case map[string]interface{}:
		if op.Path.AttributePath.SubAttribute != nil {
			value[*op.Path.AttributePath.SubAttribute] = op.Value
		} else {
			objectProps[op.Path.AttributePath.AttributeName] = op.Value
		}
	case []interface{}:
		if op.Path.ValueExpression == nil {
			objectProps[op.Path.AttributePath.AttributeName] = op.Value
			break
		}

		value, err := ReplaceInInterfaceArray(value, op)
		if err != nil {
			return nil, err
		}
		objectProps[op.Path.AttributePath.AttributeName] = value
	}

	return objectProps, err
}

func ReplaceInInterfaceArray(value []interface{}, op scim.PatchOperation) ([]interface{}, error) {
	attrExpr, ok := op.Path.ValueExpression.(*filter.AttributeExpression)
	if !ok {
		return nil, serrors.ScimErrorInvalidPath
	}
	ftr, err := filter.ParseAttrExp([]byte(attrExpr.String()))
	if err != nil {
		return nil, err
	}

	index := -1
	switch ftr.Operator {
	case filter.EQ:
		for i, v := range value {
			originalValue, ok := v.(map[string]interface{})
			if !ok {
				return nil, serrors.ScimErrorInvalidPath
			}
			value, ok := originalValue[ftr.AttributePath.AttributeName].(string)
			if ok && value == ftr.CompareValue {
				index = i
			}
		}
		if index == -1 {
			return nil, serrors.ScimErrorMutability
		}

		if originalValue, ok := value[index].(map[string]interface{}); ok {
			originalValue[*op.Path.SubAttribute] = op.Value
			value[index] = originalValue

			return value, nil
		} else {
			return nil, serrors.ScimErrorInvalidPath
		}
	case filter.PR, filter.NE, filter.CO, filter.SW, filter.EW, filter.GT, filter.GE, filter.LT, filter.LE:
		return nil, serrors.ScimErrorBadRequest("operand not supported")
	}

	return value, nil
}

func AddProperty(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
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
				if attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]interface{}); ok {
					objectProps[op.Path.AttributePath.AttributeName] = append(attrProps, v)
				} else {
					return nil, serrors.ScimErrorInvalidPath
				}
			case map[string]interface{}:
				if objectProps[op.Path.AttributePath.AttributeName] == nil {
					objectProps[op.Path.AttributePath.AttributeName] = make([]interface{}, 0)
				}

				properties := val
				attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]interface{})
				if !ok {
					return nil, serrors.ScimErrorInvalidPath
				}

				objectProps[op.Path.AttributePath.AttributeName] = append(attrProps, properties)
			}
		}
	}

	return objectProps, nil
}
