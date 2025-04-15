package config

import (
	"strings"

	"github.com/pkg/errors"
)

var ErrInvalidConfig = errors.New("invalid config")

type Config struct {
	User      *User       `json:"user"`
	Group     *Group      `json:"group"`
	Role      *Role       `json:"role"`
	Relations []*Relation `json:"relations"`
}

type User struct {
	ObjectType         string            `json:"object_type"`
	IdentityObjectType string            `json:"identity_object_type"`
	IdentityRelation   string            `json:"identity_relation"`
	PropertyMapping    map[string]string `json:"property_mapping"`
	SourceObjectType   string            `json:"source_object_type"`
	ManagerRelation    string            `json:"manager_relation"`
}

type Group struct {
	ObjectType          string `json:"object_type"`
	GroupMemberRelation string `json:"group_member_relation"`
	SourceObjectType    string `json:"source_object_type"`
}
type Role struct {
	ObjectType   string `json:"object_type"`
	RoleRelation string `json:"role_relation"`
}

type Relation struct {
	SubjectType     string `json:"subject_type"`
	SubjectID       string `json:"subject_id"`
	ObjectType      string `json:"object_type"`
	ObjectID        string `json:"object_id"`
	Relation        string `json:"relation"`
	SubjectRelation string `json:"subject_relation"`
}

func (cfg *Config) Validate() error {
	if cfg.User.ObjectType == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.user_object_type is required")
	}

	if cfg.User.IdentityObjectType == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.identity_object_type is required")
	}

	if cfg.User.IdentityRelation == "" {
		return errors.Wrap(ErrInvalidConfig, "scim.identity_relation is required")
	}

	object, relation, found := strings.Cut(cfg.User.IdentityRelation, "#")

	if !found {
		return errors.Wrap(ErrInvalidConfig, "identity relation must be in the format object#relation")
	}

	if object != cfg.User.IdentityObjectType && object != cfg.User.ObjectType {
		return errors.Wrapf(ErrInvalidConfig, "identity relation object type [%s] doesn't match user or identity type", object)
	}

	if relation == "" {
		return errors.Wrap(ErrInvalidConfig, "identity relation is required")
	}

	if cfg.Group != nil {
		if cfg.Group.ObjectType == "" {
			return errors.Wrap(ErrInvalidConfig, "scim.group_object_type is required")
		}

		if cfg.Group.GroupMemberRelation == "" {
			return errors.Wrap(ErrInvalidConfig, "scim.group_member_relation is required")
		}
	}

	return nil
}
