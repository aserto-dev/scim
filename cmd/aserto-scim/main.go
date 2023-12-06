package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aserto-dev/scim/handler"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/aserto-dev/scim/pkg/version"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/spf13/cobra"
)

var (
	// flagAddress    string
	// flagTenantID   string
	// flagAPIKey     string
	flagConfigPath string
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
		return start(flagConfigPath)
	},
}

// nolint: gochecknoinits
func init() {
	// cmdRun.Flags().StringVarP(&flagAddress, "address", "a", "directory.eng.aserto.com:8443", "directory address")
	// cmdRun.Flags().StringVarP(&flagTenantID, "tenant-id", "t", "", "tenant ID")
	// cmdRun.Flags().StringVarP(&flagAPIKey, "api-key", "k", "", "API key")
	cmdRun.Flags().StringVarP(&flagConfigPath, "config", "c", "", "config path")

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

func start(cfgPath string) error {
	cfg, err := config.NewConfig(flagConfigPath)
	if err != nil {
		return err
	}

	userHandler, err := handler.NewUsersResourceHandler(cfg)
	if err != nil {
		return err
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
			SupportFiltering: true,
			SupportPatch:     true,
			AuthenticationSchemes: []scim.AuthenticationScheme{
				{
					Type:        scim.AuthenticationTypeHTTPBasic,
					Name:        "HTTP Basic",
					Description: "Authentication scheme using the HTTP Basic Standard",
					SpecURI:     optional.NewString("https://tools.ietf.org/html/rfc7617"),
				}},
		},
		ResourceTypes: []scim.ResourceType{
			userType,
			groupType,
		},
	}

	app := new(application)
	app.auth.username = cfg.Server.Auth.Username
	app.auth.password = cfg.Server.Auth.Password

	srv := &http.Server{
		Addr:         cfg.Server.ListenAddress,
		Handler:      app.basicAuth(server.ServeHTTP),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return srv.ListenAndServe()
}

type application struct {
	auth struct {
		username string
		password string
	}
}

func (app *application) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(app.auth.username))
			expectedPasswordHash := sha256.Sum256([]byte(app.auth.password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
