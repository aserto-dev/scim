package scim_test

import (
	"encoding/json"
	"testing"

	assets_test "github.com/aserto-dev/scim/pkg/test/assets"
	common_test "github.com/aserto-dev/scim/pkg/test/common"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
)

func TestScim(t *testing.T) {
	// Setup test containers
	tst := common_test.TestSetup(t)

	e := httpexpect.Default(t, "http://localhost:8081")

	// Create user for Rick
	rick := map[string]any{}
	err := json.Unmarshal(assets_test.Rick(), &rick)
	require.NoError(t, err)
	e.GET("/Users").WithBasicAuth("scim", "scim").Expect().Status(200)
	e.POST("/Users").WithBasicAuth("scim", "scim").WithJSON(rick).Expect().Status(201).Body().Contains("Rick Sanchez")
	e.GET("/Users").WithBasicAuth("scim", "scim").Expect().Status(200).Body().Contains("Rick Sanchez")
	e.GET("/Users/rick@the-citadel.com").WithBasicAuth("scim", "scim").Expect().Status(200).Body().Contains("Rick Sanchez")

	// Create user for Morty
	morty := map[string]any{}
	err = json.Unmarshal(assets_test.Morty(), &morty)
	require.NoError(t, err)
	e.POST("/Users").WithBasicAuth("scim", "scim").WithJSON(morty).Expect().Status(201).Body().Contains("Morty Smith")
	e.GET("/Users/morty@the-citadel.com").WithBasicAuth("scim", "scim").Expect().Status(200).Body().Contains("Morty Smith")

	require.True(t, tst.UserHasIdentity(t.Context(), "morty@the-citadel.com", "CiRmZDE2MTRkMy1jMzlhLTQ3ODEtYjdiZC04Yjk2ZjVhNTEwMGQSBWxvY2Fs"))
	require.True(t, tst.UserHasManager(t.Context(), "morty@the-citadel.com", "rick@the-citadel.com"))
	require.Equal(t, true, tst.ReadUserProperty(t.Context(), "morty@the-citadel.com", "enabled"))

	// Update Morty
	patchMorty := map[string]any{}
	err = json.Unmarshal(assets_test.Patch(), &patchMorty)
	require.NoError(t, err)
	e.PATCH("/Users/morty@the-citadel.com").WithBasicAuth("scim", "scim").WithJSON(patchMorty).Expect().Status(200).Body().Contains("Morty Smith")
	require.Equal(t, false, tst.ReadUserProperty(t.Context(), "morty@the-citadel.com", "enabled"))

	// Delete Morty
	e.DELETE("/Users/morty@the-citadel.com").WithBasicAuth("scim", "scim").Expect().Status(204)
	e.GET("/Users/morty@the-citadel.com").WithBasicAuth("scim", "scim").Expect().Status(404)

	t.Logf("topaz log:\n%s", tst.ContainerLogs(t.Context(), t))
}
