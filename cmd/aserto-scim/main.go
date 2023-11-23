package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aserto-dev/go-aserto/client"
	"github.com/aserto-dev/scim/handler"
	"github.com/aserto-dev/scim/pkg/version"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/spf13/cobra"
)

var (
	flagAddress  string
	flagTenantID string
	flagAPIKey   string
)

var rootCmd = &cobra.Command{
	Use:           "aserto-scim [flags]",
	SilenceErrors: true,
	SilenceUsage:  true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("aserto-scim %s\n", version.GetInfo().Version)
	},
}

var cmdRun = &cobra.Command{
	Use:   "run [args]",
	Short: "Start SCIM service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &client.Config{
			Address:  flagAddress,
			TenantID: flagTenantID,
			APIKey:   flagAPIKey,
		}

		userHandler, err := handler.NewUsersResourceHandler(cfg)
		if err != nil {
			panic(err)
		}

		userType := scim.ResourceType{
			ID:          optional.NewString("User"),
			Name:        "User",
			Endpoint:    "/Users",
			Description: optional.NewString("User Account"),
			Schema:      schema.CoreUserSchema(),
			SchemaExtensions: []scim.SchemaExtension{
				{Schema: schema.ExtensionEnterpriseUser()},
			},
			Handler: userHandler,
		}

		groupHandler, err := handler.NewGroupResourceHandler(cfg)
		if err != nil {
			panic(err)
		}

		groupType := scim.ResourceType{
			ID:          optional.NewString("Group"),
			Name:        "Group",
			Endpoint:    "/Groups",
			Description: optional.NewString("Group"),
			Schema:      schema.CoreGroupSchema(),
			Handler:     groupHandler,
		}

		server := scim.Server{
			Config: scim.ServiceProviderConfig{
				DocumentationURI: optional.NewString("https://aserto.com/docs/scim"),
			},
			ResourceTypes: []scim.ResourceType{
				userType,
				groupType,
			},
		}

		return http.ListenAndServe(":8080", server)
	},
}

// nolint: gochecknoinits
func init() {
	cmdRun.Flags().StringVarP(&flagAddress, "address", "a", "directory.eng.aserto.com:8443", "directory address")
	cmdRun.Flags().StringVarP(&flagTenantID, "tenant-id", "t", "", "tenant ID")
	cmdRun.Flags().StringVarP(&flagAPIKey, "api-key", "k", "", "API key")

	rootCmd.AddCommand(cmdRun)
}

func main() {
	rootCmd.AddCommand(
		versionCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}
