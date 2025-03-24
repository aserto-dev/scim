package convert

import (
	"encoding/json"

	"github.com/aserto-dev/ds-load/sdk/common/msg"
	"github.com/aserto-dev/ds-load/sdk/transform"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	"github.com/aserto-dev/scim/common/model"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrSourceUserTypeNotSet  = errors.New("source user type not set")
	ErrSourceGroupTypeNotSet = errors.New("source group type not set")
	ErrGroupsNotEnabled      = errors.New("groups not enabled")
)

type Converter struct {
	cfg *TransformConfig
}

func NewConverter(cfg *TransformConfig) *Converter {
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

func ToResourceAttributes(value interface{}) (scim.ResourceAttributes, error) {
	var result scim.ResourceAttributes
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &result)
	return result, err
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

	sourceUserType := c.cfg.User.SourceObjectType
	if sourceUserType == "" {
		return nil, ErrSourceUserTypeNotSet
	}

	object := &dsc.Object{
		Type:        sourceUserType,
		Properties:  props,
		Id:          userID,
		DisplayName: displayName,
	}
	return object, nil
}

func (c *Converter) SCIMGroupToObject(group *model.Group) (*dsc.Object, error) {
	if c.cfg.Group == nil {
		return nil, ErrGroupsNotEnabled
	}

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

	sourceGroupType := c.cfg.Group.SourceObjectType
	if sourceGroupType == "" {
		return nil, ErrSourceGroupTypeNotSet
	}

	object := &dsc.Object{
		Type:        sourceGroupType,
		Properties:  props,
		Id:          objID,
		DisplayName: displayName,
	}
	return object, nil
}

func (c *Converter) TransformResource(resource map[string]interface{}, objType string) (*msg.Transform, error) {
	template, err := c.cfg.GetTemplate()
	if err != nil {
		return nil, err
	}

	vars, err := c.cfg.ToTemplateVars()
	if err != nil {
		return nil, err
	}

	transformInput := make(map[string]interface{})
	transformInput["input"] = resource
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
