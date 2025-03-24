package convert_test

import (
	"testing"

	"github.com/aserto-dev/scim/common/config"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/stretchr/testify/require"
)

var ScimUser map[string]interface{} = map[string]interface{}{
	"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
	"userName": "foobar",
	"name": map[string]interface{}{
		"givenName":  "foo",
		"familyName": "bar",
	},
	"emails": []map[string]interface{}{
		{
			"primary": true,
			"value":   "foo@bar.com",
			"type":    "work",
		},
	},
	"displayName": "foo bar",
	"externalId":  "fooooo",
	"locale":      "en-US",
	"groups":      []interface{}{},
	"active":      true,
}

func TestTransform(t *testing.T) {
	assert := require.New(t)

	cfg := config.SCIMConfig{
		User: &config.UserOptions{
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
	assert.NotEmpty(msg.Objects)
	assert.NotEmpty(msg.Relations)
	assert.Len(msg.Relations, 3)

	assert.Equal("foo@bar.com", msg.Relations[1].ObjectId)
	assert.Equal("identity", msg.Relations[1].ObjectType)
	assert.Equal("identitifier", msg.Relations[1].Relation)
	assert.Equal("foobar", msg.Relations[1].SubjectId)
	assert.Equal("user", msg.Relations[1].SubjectType)

	assert.Equal("foobar", msg.Relations[0].SubjectId)
	assert.Equal("user", msg.Relations[0].SubjectType)

	assert.Equal("fooooo", msg.Relations[2].ObjectId)
	assert.Equal("identity", msg.Relations[2].ObjectType)
}

func TestTransformUserIdentifier(t *testing.T) {
	assert := require.New(t)

	cfg := config.SCIMConfig{
		User: &config.UserOptions{
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
	assert.NotEmpty(msg.Objects)
	assert.NotEmpty(msg.Relations)
	assert.Equal("foo@bar.com", msg.Relations[1].SubjectId)
	assert.Equal("identity", msg.Relations[1].SubjectType)
	assert.Equal("identitifier", msg.Relations[1].Relation)
	assert.Equal("foobar", msg.Relations[1].ObjectId)
	assert.Equal("user", msg.Relations[1].ObjectType)

	assert.Equal("foobar", msg.Relations[0].ObjectId)
	assert.Equal("user", msg.Relations[0].ObjectType)

	assert.Equal("fooooo", msg.Relations[2].SubjectId)
	assert.Equal("identity", msg.Relations[2].SubjectType)
}
