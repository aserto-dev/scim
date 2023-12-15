package app

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/aserto-dev/scim/pkg/app/handlers/groups"
	"github.com/aserto-dev/scim/pkg/app/handlers/users"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
)

func Run(cfgPath string) error {
	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		return err
	}

	userHandler, err := users.NewUsersResourceHandler(cfg)
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

	groupHandler, err := groups.NewGroupResourceHandler(cfg)
	if err != nil {
		return err
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
	app.username = cfg.Server.Auth.Username
	app.password = cfg.Server.Auth.Password
	app.token = cfg.Server.Auth.Token

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
	username string
	password string
	token    string
}

func (app *application) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(app.username))
			expectedPasswordHash := sha256.Sum256([]byte(app.password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		if len(splitToken) == 2 {
			if subtle.ConstantTimeCompare([]byte(app.token), []byte(splitToken[1])) == 1 {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
