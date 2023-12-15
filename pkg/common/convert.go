package common

import (
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"google.golang.org/protobuf/types/known/structpb"
)

func ObjectToResource(object *dsc.Object, meta scim.Meta) scim.Resource {
	eID := optional.String{}
	attr := object.Properties.AsMap()
	delete(attr, "password")

	return scim.Resource{
		ID:         object.Id,
		ExternalID: eID,
		Attributes: attr,
		Meta:       meta,
	}
}

func ResourceAttributesToObject(resourceAttributes scim.ResourceAttributes, objectType, id string) (*dsc.Object, error) {
	props, err := structpb.NewStruct(resourceAttributes)
	if err != nil {
		return nil, err
	}

	var displayName string
	if resourceAttributes["displayName"] != nil {
		displayName = resourceAttributes["displayName"].(string)
	} else {
		displayName = id
	}
	object := &dsc.Object{
		Type:        objectType,
		Properties:  props,
		Id:          id,
		DisplayName: displayName,
	}
	return object, nil
}
