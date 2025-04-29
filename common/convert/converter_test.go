package convert_test

import (
	"testing"

	"github.com/aserto-dev/scim/common/config"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/stretchr/testify/require"
)

var ScimUser map[string]any = map[string]any{
	"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
	"userName": "foobar",
	"name": map[string]any{
		"givenName":  "foo",
		"familyName": "bar",
	},
	"emails": []map[string]any{
		{
			"primary": true,
			"value":   "foo@bar.com",
			"type":    "work",
		},
	},
	"displayName": "foo bar",
	"externalId":  "fooooo",
	"locale":      "en-US",
	"groups":      []any{},
	"active":      true,
}

func TestTransform(t *testing.T) {
	assert := require.New(t)

	cfg := config.Config{
		User: &config.User{
			IdentityObjectType: "identity",
			IdentityRelation:   "identity#identitifier",
			ObjectType:         "user",
			SourceObjectType:   "scim:user",
			ManagerRelation:    "manager",
		},
	}

	sCfg, err := convert.NewTransformConfig(&cfg)
	assert.NoError(err)

	cvt := convert.NewConverter(sCfg)
	msg, err := cvt.TransformResource(ScimUser, "user")
	assert.NoError(err)

	assert.NotNil(msg)
	assert.NotEmpty(msg.GetObjects())
	assert.NotEmpty(msg.GetRelations())
	assert.Len(msg.GetRelations(), 3)

	assert.Equal("foo@bar.com", msg.GetRelations()[1].GetObjectId())
	assert.Equal("identity", msg.GetRelations()[1].GetObjectType())
	assert.Equal("identitifier", msg.GetRelations()[1].GetRelation())
	assert.Equal("foobar", msg.GetRelations()[1].GetSubjectId())
	assert.Equal("user", msg.GetRelations()[1].GetSubjectType())

	assert.Equal("foobar", msg.GetRelations()[0].GetSubjectId())
	assert.Equal("user", msg.GetRelations()[0].GetSubjectType())

	assert.Equal("fooooo", msg.GetRelations()[2].GetObjectId())
	assert.Equal("identity", msg.GetRelations()[2].GetObjectType())
}

func TestTransformUserIdentifier(t *testing.T) {
	assert := require.New(t)

	cfg := config.Config{
		User: &config.User{
			IdentityObjectType: "identity",
			IdentityRelation:   "user#identitifier",
			ObjectType:         "user",
			SourceObjectType:   "scim:user",
			ManagerRelation:    "manager",
		},
	}

	sCfg, err := convert.NewTransformConfig(&cfg)
	assert.NoError(err)

	cvt := convert.NewConverter(sCfg)
	msg, err := cvt.TransformResource(ScimUser, "user")
	assert.NoError(err)

	assert.NotNil(msg)
	assert.NotEmpty(msg.GetObjects())
	assert.NotEmpty(msg.GetRelations())
	assert.Equal("foo@bar.com", msg.GetRelations()[1].GetSubjectId())
	assert.Equal("identity", msg.GetRelations()[1].GetSubjectType())
	assert.Equal("identitifier", msg.GetRelations()[1].GetRelation())
	assert.Equal("foobar", msg.GetRelations()[1].GetObjectId())
	assert.Equal("user", msg.GetRelations()[1].GetObjectType())

	assert.Equal("foobar", msg.GetRelations()[0].GetObjectId())
	assert.Equal("user", msg.GetRelations()[0].GetObjectType())

	assert.Equal("fooooo", msg.GetRelations()[2].GetSubjectId())
	assert.Equal("identity", msg.GetRelations()[2].GetSubjectType())
}
