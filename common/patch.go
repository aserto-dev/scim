package common

import (
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/scim2/filter-parser/v2"
)

func HandlePatchOPAdd(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	if op.Path == nil || op.Path.ValueExpression == nil {
		return AddProperty(objectProps, op)
	}

	fltr, err := parseValueExpression(op.Path.ValueExpression)
	if err != nil {
		return nil, err
	}

	if op.Path.ValueExpression == nil {
		return handleSimpleAdd(objectProps, op)
	}

	return handleComplexAdd(objectProps, op, fltr)
}

func parseValueExpression(expr any) (*filter.AttributeExpression, error) {
	valueExpression, ok := expr.(*filter.AttributeExpression)
	if !ok {
		return nil, serrors.ScimErrorInvalidPath
	}

	fltr, err := filter.ParseAttrExp([]byte(valueExpression.String()))
	if err != nil {
		return nil, err
	}

	return &fltr, nil
}

func handleSimpleAdd(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	properties := make(map[string]any)
	properties[*op.Path.SubAttribute] = op.Value

	return objectProps, nil
}

func handleComplexAdd(objectProps scim.ResourceAttributes,
	op scim.PatchOperation,
	fltr *filter.AttributeExpression,
) (scim.ResourceAttributes, error) {
	attrName := op.Path.AttributePath.AttributeName

	if objectProps[attrName] == nil {
		objectProps[attrName] = make([]any, 0)
	}

	properties := make(map[string]any)

	if objectProps[attrName] != nil {
		var err error

		properties, err = processExistingAttributes(objectProps[attrName], op, fltr)
		if err != nil {
			return nil, err
		}
	}

	if len(properties) == 0 {
		return appendNewProperty(objectProps, op, fltr)
	}

	return objectProps, nil
}

func processExistingAttributes(attr any, op scim.PatchOperation, fltr *filter.AttributeExpression) (map[string]any, error) {
	attrProps, ok := attr.([]any)
	if !ok {
		return nil, serrors.ScimErrorInvalidPath
	}

	for _, v := range attrProps {
		originalValue, ok := v.(map[string]any)
		if !ok {
			return nil, serrors.ScimErrorInvalidPath
		}

		if result, err := processAttribute(originalValue, op, fltr); err != nil {
			return nil, err
		} else if result != nil {
			return result, nil
		}
	}

	return make(map[string]any), nil
}

func processAttribute(value map[string]any, op scim.PatchOperation, fltr *filter.AttributeExpression) (map[string]any, error) {
	switch fltr.Operator {
	case filter.EQ:
		return processEqualityOperator(value, op, fltr)
	case filter.PR, filter.NE, filter.CO, filter.SW, filter.EW, filter.GT, filter.GE, filter.LT, filter.LE:
		return nil, serrors.ScimErrorBadRequest("operand not supported")
	default:
		return nil, nil
	}
}

func processEqualityOperator(value map[string]any, op scim.PatchOperation, fltr *filter.AttributeExpression) (map[string]any, error) {
	attrValue, ok := value[fltr.AttributePath.AttributeName].(string)
	if !ok || attrValue != fltr.CompareValue {
		return nil, nil
	}

	if value[*op.Path.SubAttribute] != nil {
		return nil, serrors.ScimErrorUniqueness
	}

	return value, nil
}

func appendNewProperty(objectProps scim.ResourceAttributes,
	op scim.PatchOperation,
	fltr *filter.AttributeExpression,
) (scim.ResourceAttributes, error) {
	properties := map[string]any{
		fltr.AttributePath.AttributeName: fltr.CompareValue,
		*op.Path.SubAttribute:            op.Value,
	}

	attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]any)
	if !ok {
		return nil, serrors.ScimErrorInvalidPath
	}

	objectProps[op.Path.AttributePath.AttributeName] = append(attrProps, properties)

	return objectProps, nil
}

func HandlePatchOPRemove(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error

	switch value := objectProps[op.Path.AttributePath.AttributeName].(type) {
	case string:
		delete(objectProps, op.Path.AttributePath.AttributeName)
	case []any:
		objectProps, err = patchOrRemoveSlice(value, op, objectProps)
		if err != nil {
			return nil, err
		}
	}

	return objectProps, err
}

func HandlePatchOPReplace(objectProps scim.ResourceAttributes, op scim.PatchOperation) (scim.ResourceAttributes, error) {
	var err error

	if op.Path == nil {
		value, ok := op.Value.(map[string]any)
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
	case map[string]any:
		if op.Path.AttributePath.SubAttribute != nil {
			value[*op.Path.AttributePath.SubAttribute] = op.Value
		} else {
			objectProps[op.Path.AttributePath.AttributeName] = op.Value
		}
	case []any:
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

func ReplaceInInterfaceArray(value []any, op scim.PatchOperation) ([]any, error) {
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
			originalValue, ok := v.(map[string]any)

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

		if originalValue, ok := value[index].(map[string]any); ok {
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
	case map[string]any:
		for k, v := range value {
			if objectProps[k] != nil {
				return nil, serrors.ScimErrorUniqueness
			}

			objectProps[k] = v
		}
	case []any:
		for _, v := range value {
			switch val := v.(type) {
			case string:
				if objectProps[op.Path.AttributePath.AttributeName] == nil {
					objectProps[op.Path.AttributePath.AttributeName] = make([]string, 0)
				}

				if attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]any); ok {
					objectProps[op.Path.AttributePath.AttributeName] = append(attrProps, v)
				} else {
					return nil, serrors.ScimErrorInvalidPath
				}
			case map[string]any:
				if objectProps[op.Path.AttributePath.AttributeName] == nil {
					objectProps[op.Path.AttributePath.AttributeName] = make([]any, 0)
				}

				properties := val
				attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]any)

				if !ok {
					return nil, serrors.ScimErrorInvalidPath
				}

				objectProps[op.Path.AttributePath.AttributeName] = append(attrProps, properties)
			}
		}
	}

	return objectProps, nil
}

func patchOrRemoveSlice(value []any, op scim.PatchOperation, objectProps scim.ResourceAttributes) (scim.ResourceAttributes, error) {
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
			originalValue, ok := v.(map[string]any)
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

		attrProps, ok := objectProps[op.Path.AttributePath.AttributeName].([]any)

		if !ok {
			return nil, serrors.ScimErrorInvalidPath
		}

		objectProps[op.Path.AttributePath.AttributeName] = append(attrProps[:index], attrProps[index+1:]...)
	}

	return objectProps, nil
}
