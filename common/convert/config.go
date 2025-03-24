package convert

import (
	"encoding/json"
	"strings"

	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	"github.com/aserto-dev/scim/common"
	"github.com/aserto-dev/scim/common/config"
	"github.com/pkg/errors"
)

type TemplateName int

const (
	Users TemplateName = iota
	UsersGroups
	UsersGroupsRoles
)

var ErrInvalidConfig = errors.New("invalid config")

func (t TemplateName) String() string {
	switch t {
	case Users:
		return "users"
	case UsersGroups:
		return "users-groups"
	case UsersGroupsRoles:
		return "users-groups-roles"
	}
	return "unknown"
}

type TransformConfig struct {
	template TemplateName
	*config.SCIMConfig
	IdentityObjectType string `json:"identity_object_type,omitempty"`
	IdentityRelation   string `json:"identity_relation,omitempty"`
}

func NewTransformConfig(cfg *config.SCIMConfig) (*TransformConfig, error) {
	template := Users

	if cfg.Group != nil {
		template = UsersGroups
		if cfg.Role != nil {
			template = UsersGroupsRoles
		}
	}

	object, relation, found := strings.Cut(cfg.User.IdentityRelation, "#")
	if !found {
		return nil, errors.Wrap(ErrInvalidConfig, "identity relation must be in the format object#relation")
	}
	if object != cfg.User.IdentityObjectType && object != cfg.User.ObjectType {
		return nil, errors.Wrapf(ErrInvalidConfig, "identity relation object type [%s] doesn't match user or identity type", object)
	}
	if relation == "" {
		return nil, errors.Wrap(ErrInvalidConfig, "identity relation is required")
	}

	return &TransformConfig{
		SCIMConfig:         cfg,
		template:           template,
		IdentityObjectType: object,
		IdentityRelation:   relation,
	}, nil
}

func (c *TransformConfig) Groups() bool {
	return c.SCIMConfig.Group != nil
}

func (c *TransformConfig) ToTemplateVars() (map[string]interface{}, error) {
	var result map[string]interface{}

	cfg, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ScimConfig to json")
	}
	err = json.Unmarshal(cfg, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal ScimConfig to map")
	}

	return result, nil
}

func (c *TransformConfig) GetTemplate() ([]byte, error) {
	template, err := common.GetTemplateContent(c.template.String())
	if err != nil {
		return nil, err
	}

	return template, nil
}

func (c *TransformConfig) GetIdentityRelation(userID, identity string) (*dsc.Relation, error) {
	switch c.IdentityObjectType {
	case c.User.IdentityObjectType:
		return &dsc.Relation{
			SubjectId:   userID,
			SubjectType: c.User.ObjectType,
			Relation:    c.IdentityRelation,
			ObjectType:  c.User.IdentityObjectType,
			ObjectId:    identity,
		}, nil
	case c.User.ObjectType:
		return &dsc.Relation{
			SubjectId:   identity,
			SubjectType: c.User.IdentityObjectType,
			Relation:    c.IdentityRelation,
			ObjectType:  c.User.ObjectType,
			ObjectId:    userID,
		}, nil
	default:
		return nil, errors.New("invalid identity relation")
	}
}
