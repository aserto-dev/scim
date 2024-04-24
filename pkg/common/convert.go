package common

import (
	"encoding/json"

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

func ResourceAttributesToUser(attributes scim.ResourceAttributes) (*User, error) {
	var user User
	data, err := json.Marshal(attributes)
	if err != nil {
		return &User{}, err
	}

	if err := json.Unmarshal(data, &user); err != nil {
		return &User{}, err
	}
	return &user, nil
}

func ToResourceAttributes(value interface{}) (result scim.ResourceAttributes, err error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &result)
	return
}

func UserToResource(meta scim.Meta, user *User) (scim.Resource, error) {
	attributes, err := ToResourceAttributes(&user)
	if err != nil {
		return scim.Resource{}, err
	}
	eID := optional.String{}
	if user.ExternalID != "" {
		eID = optional.NewString(user.ExternalID)
	}
	return scim.Resource{
		ID:         user.ID,
		ExternalID: eID,
		Attributes: attributes,
		Meta:       meta,
	}, nil
}

func UserToObject(user *User) (*dsc.Object, error) {
	attributes, err := ToResourceAttributes(&user)
	if err != nil {
		return nil, err
	}
	props, err := structpb.NewStruct(attributes)
	if err != nil {
		return nil, err
	}

	userID := user.ID
	if userID == "" {
		userID = user.UserName
	}

	displayName := user.DisplayName
	if displayName == "" {
		displayName = userID
	}
	object := &dsc.Object{
		Type:        "user",
		Properties:  props,
		Id:          userID,
		DisplayName: displayName,
	}
	return object, nil
}
