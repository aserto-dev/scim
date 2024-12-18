package convert

import (
	"encoding/json"

	"github.com/aserto-dev/ds-load/sdk/common/msg"
	"github.com/aserto-dev/ds-load/sdk/transform"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/model"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/pkg/errors"
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

func ResourceAttributesToUser(attributes scim.ResourceAttributes) (*model.User, error) {
	var user model.User
	data, err := json.Marshal(attributes)
	if err != nil {
		return &model.User{}, err
	}

	if err := json.Unmarshal(data, &user); err != nil {
		return &model.User{}, err
	}
	return &user, nil
}

func ResourceAttributesToGroup(attributes scim.ResourceAttributes) (*model.Group, error) {
	var group model.Group
	data, err := json.Marshal(attributes)
	if err != nil {
		return &model.Group{}, err
	}

	if err := json.Unmarshal(data, &group); err != nil {
		return &model.Group{}, err
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

func UserToResource(meta scim.Meta, user *model.User) (scim.Resource, error) {
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

func (c *Converter) SCIMUserToObject(user *model.User) (*dsc.Object, error) {
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

	object := &dsc.Object{
		Type:        c.cfg.SourceUserType,
		Properties:  props,
		Id:          userID,
		DisplayName: displayName,
	}
	return object, nil
}

func (c *Converter) SCIMGroupToObject(group *model.Group) (*dsc.Object, error) {
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
	template, err := common.GetTemplateContent(cfg.Template)
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

func TransformConfigFromMap(defaults *config.TransformConfig, t map[string]interface{}) (*config.TransformConfig, error) {
	cfg := &config.TransformConfig{
		CreateEmailIdentities: defaults.CreateEmailIdentities,
		CreateRoleGroups:      defaults.CreateRoleGroups,
		Template:              defaults.Template,
		UserObjectType:        defaults.UserObjectType,
		GroupMemberRelation:   defaults.GroupMemberRelation,
		GroupObjectType:       defaults.GroupObjectType,
		IdentityObjectType:    defaults.IdentityObjectType,
		IdentityRelation:      defaults.IdentityRelation,
		RoleObjectType:        defaults.RoleObjectType,
		RoleRelation:          defaults.RoleRelation,
		SourceUserType:        defaults.SourceUserType,
		SourceGroupType:       defaults.SourceGroupType,
		GroupMappings:         defaults.GroupMappings,
		UserMappings:          defaults.UserMappings,
		ManagerRelation:       defaults.ManagerRelation,
		UserPropertiesMapping: defaults.UserPropertiesMapping,
	}
	jsonData, err := json.Marshal(t)
	if err != nil {
		return &config.TransformConfig{}, errors.Wrap(err, "failed to marshal transform config")
	}

	if err := json.Unmarshal(jsonData, cfg); err != nil {
		return &config.TransformConfig{}, errors.Wrap(err, "failed to unmarshal transform config")
	}

	return cfg, nil
}
