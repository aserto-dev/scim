package scim_test

import (
	"encoding/json"
	"testing"

	assets_test "github.com/aserto-dev/scim/pkg/test/assets"
	common_test "github.com/aserto-dev/scim/pkg/test/common"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
)

const scimMediaType = "application/scim+json"

func TestScim(t *testing.T) {
	// Setup test containers
	tst := common_test.TestSetup(t)

	e := httpexpect.Default(t, "http://localhost:8081")

	// Create user for Rick
	rick := map[string]any{}
	err := json.Unmarshal(assets_test.Rick(), &rick)
	require.NoError(t, err)

	e.GET("/Users").WithBasicAuth("scim", "scim").Expect().Status(200)

	rickID := e.POST("/Users").WithBasicAuth("scim", "scim").WithJSON(rick).Expect().
		Status(201).JSON(httpexpect.ContentOpts{MediaType: scimMediaType}).Object().Value("id").String()

	rickID.NotEmpty()
	e.GET("/Users").WithBasicAuth("scim", "scim").Expect().Status(200).Body().Contains("Rick Sanchez")
	e.GET("/Users/"+rickID.Raw()).WithBasicAuth("scim", "scim").Expect().Status(200).Body().Contains("Rick Sanchez")

	// Create user for Morty
	morty := map[string]any{}
	err = json.Unmarshal(assets_test.Morty(), &morty)
	require.NoError(t, err)

	mortyID := e.POST("/Users").WithBasicAuth("scim", "scim").WithJSON(morty).Expect().
		Status(201).JSON(httpexpect.ContentOpts{MediaType: scimMediaType}).Object().Value("id").String()

	mortyID.NotEmpty()
	e.GET("/Users/"+mortyID.Raw()).WithBasicAuth("scim", "scim").Expect().Status(200).Body().Contains("Morty Smith")

	require.True(t, tst.UserHasIdentity(t.Context(), mortyID.Raw(), "CiRmZDE2MTRkMy1jMzlhLTQ3ODEtYjdiZC04Yjk2ZjVhNTEwMGQSBWxvY2Fs"))
	require.True(t, tst.UserHasManager(t.Context(), mortyID.Raw(), "rick@the-citadel.com"))
	require.Equal(t, true, tst.ReadUserProperty(t.Context(), mortyID.Raw(), "enabled"))

	// Update Morty
	patchMorty := map[string]any{}
	err = json.Unmarshal(assets_test.PatchOp(), &patchMorty)
	require.NoError(t, err)
	e.PATCH("/Users/"+mortyID.Raw()).WithBasicAuth("scim", "scim").WithJSON(patchMorty).Expect().Status(200).Body().Contains("Morty Smith")
	require.Equal(t, false, tst.ReadUserProperty(t.Context(), mortyID.Raw(), "enabled"))

	// Delete Morty
	e.DELETE("/Users/"+mortyID.Raw()).WithBasicAuth("scim", "scim").Expect().Status(204)
	e.GET("/Users/"+mortyID.Raw()).WithBasicAuth("scim", "scim").Expect().Status(404)

	group := map[string]any{}
	err = json.Unmarshal(assets_test.Group(), &group)
	require.NoError(t, err)

	groupID := e.POST("/Groups").WithBasicAuth("scim", "scim").WithJSON(group).Expect().
		Status(201).JSON(httpexpect.ContentOpts{MediaType: scimMediaType}).Object().Value("id").String()

	groupID.NotEmpty()
	e.GET("/Groups/"+groupID.Raw()).WithBasicAuth("scim", "scim").Expect().Status(200).Body().Contains("Evil Genius")

	t.Logf("topaz log:\n%s", tst.ContainerLogs(t.Context(), t))
}
