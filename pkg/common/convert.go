package common

import (
	"encoding/json"

	"github.com/aserto-dev/ds-load/sdk/common/msg"
	"github.com/aserto-dev/ds-load/sdk/transform"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type Converter struct {
	cfg *config.TransformConfig
}

func NewConverter(cfg *config.TransformConfig) *Converter {
	return &Converter{cfg: cfg}
}

func (c *Converter) ObjectToResource(object *dsc.Object, meta scim.Meta) scim.Resource {
	eID := optional.String{}
	attr := c.ObjectToResourceAttributes(object)

	return scim.Resource{
		ID:         object.Id,
		ExternalID: eID,
		Attributes: attr,
		Meta:       meta,
	}
}

func (c *Converter) ObjectToResourceAttributes(object *dsc.Object) scim.ResourceAttributes {
	attr := object.Properties.AsMap()
	delete(attr, "password")

	return attr
}

// func (c *Converter)ResourceAttributesToObject(resourceAttributes scim.ResourceAttributes, objectType, id string) (*dsc.Object, error) {
// 	var propKey string
// 	switch objectType {
// 	case c.cfg.SCIM.UserObjectType:
// 		propKey = c.cfg.SCIM.SCIMUserPropertyKey
// 	case c.cfg.SCIM.GroupObjectType:
// 		propKey = c.cfg.SCIM.SCIMGroupPropertyKey
// 	}

// 	props, err := structpb.NewStruct(resourceAttributes)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// var displayName string
// 	// if resourceAttributes["displayName"] != nil {
// 	// 	displayName = resourceAttributes["displayName"].(string)
// 	// } else {
// 	// 	displayName = id
// 	// }

// 	object := &dsc.Object{
// 		Type:       objectType,
// 		Properties: props,
// 		Id:         id,
// 		// DisplayName: displayName,
// 	}
// 	return object, nil
// }

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

func ResourceAttributesToGroup(attributes scim.ResourceAttributes) (*Group, error) {
	var group Group
	data, err := json.Marshal(attributes)
	if err != nil {
		return &Group{}, err
	}

	if err := json.Unmarshal(data, &group); err != nil {
		return &Group{}, err
	}
	return &group, nil
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

func (c *Converter) SCIMUserToObject(user *User) (*dsc.Object, error) {
	attributes, err := ToResourceAttributes(&user)
	if err != nil {
		return nil, err
	}
	delete(attributes, "password")
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

	// props.Fields["enabled"] = structpb.NewBoolValue(user.Active)

	object := &dsc.Object{
		Type:        c.cfg.SourceUserType,
		Properties:  props,
		Id:          userID,
		DisplayName: displayName,
	}
	return object, nil
}

func (c *Converter) SCIMGroupToObject(group *Group) (*dsc.Object, error) {
	attributes, err := ToResourceAttributes(&group)
	if err != nil {
		return nil, err
	}
	props, err := structpb.NewStruct(attributes)
	if err != nil {
		return nil, err
	}

	objID := group.ID
	if objID == "" {
		objID = group.DisplayName
	}

	displayName := group.DisplayName
	if displayName == "" {
		displayName = objID
	}

	object := &dsc.Object{
		Type:        c.cfg.SourceGroupType,
		Properties:  props,
		Id:          objID,
		DisplayName: displayName,
	}
	return object, nil
}

func TransformResource(userMap map[string]interface{}, cfg *config.TransformConfig, objType string) (*msg.Transform, error) {
	template, err := getTemplateContent(cfg.Template)
	if err != nil {
		return nil, err
	}

	transformInput := make(map[string]interface{})
	transformInput["input"] = userMap
	vars, err := cfg.ToMap()
	if err != nil {
		return nil, err
	}

	transformInput["vars"] = vars
	transformInput["objectType"] = objType
	transformer := transform.NewGoTemplateTransform(template)
	return transformer.TransformObject(transformInput)
}

func ProtobufStructToMap(s *structpb.Struct) (map[string]interface{}, error) {
	b, err := protojson.Marshal(s)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// func TransformConfigFromMap(t map[string]interface{}) (*config.TransformConfig, error) {
// 	cfg := &config.TransformConfig{}

// 	err := mapstructure.Decode(t, cfg)
// 	if err != nil {
// 		return &config.TransformConfig{}, errors.Wrap(err, "failed to decode transform config")
// 	}

// 	return cfg, nil
// }
