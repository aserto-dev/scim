package users

import (
	"context"
	"log"
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

func (u UsersResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	log.Println("PATCH", id, operations)
	getObjResp, err := u.dirClient.Reader.GetObject(r.Context(), &dsr.GetObjectRequest{
		ObjectType:    "user",
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
		log.Println(err)
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

func (u UsersResourceHandler) handlePatchOPAdd(ctx context.Context, object *dsc.Object, op scim.PatchOperation) error {
	var err error
	objectProps := object.Properties.AsMap()
	if op.Path == nil || op.Path.ValueExpression == nil {
		// simple add property
		switch v := op.Value.(type) {
		case string:
			if objectProps[op.Path.AttributePath.AttributeName] != nil {
				return serrors.ScimErrorUniqueness
			}
			objectProps[op.Path.AttributePath.AttributeName] = op.Value
		case map[string]interface{}:
			value := v
			for k, v := range value {
				if objectProps[k] != nil {
					return serrors.ScimErrorUniqueness
				}
				objectProps[k] = v
			}
		}
	} else {
		fltr, err := filter.ParseAttrExp([]byte(op.Path.ValueExpression.(*filter.AttributeExpression).String()))
		if err != nil {
			return err
		}

		switch op.Path.AttributePath.AttributeName {
		case Emails, Groups:
			properties := make(map[string]interface{})
			if op.Path.ValueExpression != nil {
				if objectProps[op.Path.AttributePath.AttributeName] != nil {
					for _, v := range objectProps[op.Path.AttributePath.AttributeName].([]interface{}) {
						originalValue := v.(map[string]interface{})
						if fltr.Operator == filter.EQ {
							if originalValue[fltr.AttributePath.AttributeName].(string) == fltr.CompareValue {
								if originalValue[*op.Path.SubAttribute] != nil {
									return serrors.ScimErrorUniqueness
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

			if op.Path.AttributePath.AttributeName == Emails && u.cfg.SCIM.CreateEmailIdentities {
				err = u.setIdentity(ctx, object.Id, op.Value.(string), "IDENTITY_KIND_EMAIL")
				if err != nil {
					return err
				}
			} else if op.Path.AttributePath.AttributeName == Groups {
				err = u.addUserToGroup(ctx, object.Id, op.Value.(string))
				if err != nil {
					return err
				}
			}
		}
	}

	object.Properties, err = structpb.NewStruct(objectProps)
	return err
}

func (u UsersResourceHandler) handlePatchOPRemove(ctx context.Context, object *dsc.Object, op scim.PatchOperation) error {
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

	if op.Path.AttributePath.AttributeName == Emails && u.cfg.SCIM.CreateEmailIdentities {
		email := oldValue.(map[string]interface{})["value"].(string)
		err = u.removeIdentity(ctx, email)
		if err != nil {
			return err
		}
	} else if op.Path.AttributePath.AttributeName == Groups {
		group := oldValue.(map[string]interface{})["value"].(string)
		err = u.removeUserFromGroup(ctx, object.Id, group)
		if err != nil {
			return err
		}
	}

	object.Properties, err = structpb.NewStruct(objectProps)
	return err
}

func (u UsersResourceHandler) handlePatchOPReplace(object *dsc.Object, op scim.PatchOperation) error {
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
