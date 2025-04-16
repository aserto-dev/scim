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
	"github.com/samber/lo"
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
		ID:         object.GetId(),
		ExternalID: eID,
		Attributes: attr,
		Meta:       meta,
	}
}

func (c *Converter) ObjectToResourceAttributes(object *dsc.Object) scim.ResourceAttributes {
	attr := object.GetProperties().AsMap()
	delete(attr, "password")

	return attr
}

func Unmarshal[S any, D any](source S, dest *D) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

func UserToResource(meta scim.Meta, user *model.User) (scim.Resource, error) {
	attributes := scim.ResourceAttributes{}

	err := Unmarshal(user, &attributes)
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
	attributes := scim.ResourceAttributes{}

	err := Unmarshal(user, &attributes)
	if err != nil {
		return nil, err
	}

	delete(attributes, "password")

	props, err := structpb.NewStruct(attributes)
	if err != nil {
		return nil, err
	}

	userID := lo.Ternary(user.ID != "", user.ID, user.UserName)

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

	attributes := scim.ResourceAttributes{}

	err := Unmarshal(group, &attributes)
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

func (c *Converter) TransformResource(resource map[string]any, objType string) (*msg.Transform, error) {
	template, err := c.cfg.GetTemplate()
	if err != nil {
		return nil, err
	}

	vars, err := c.cfg.ToTemplateVars()
	if err != nil {
		return nil, err
	}

	transformInput := map[string]any{
		"input":      resource,
		"vars":       vars,
		"objectType": objType,
	}
	transformer := transform.NewGoTemplateTransform(template)

	return transformer.TransformObject(transformInput)
}

func ProtobufStructToMap(s *structpb.Struct) (map[string]any, error) {
	b, err := protojson.Marshal(s)
	if err != nil {
		return nil, err
	}

	m := make(map[string]any)

	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
