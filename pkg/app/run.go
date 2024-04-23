package app

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/aserto-dev/certs"
	"github.com/aserto-dev/logger"
	"github.com/aserto-dev/scim/pkg/app/handlers/groups"
	"github.com/aserto-dev/scim/pkg/app/handlers/users"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
)

func Run(cfgPath string, logWriter logger.Writer, errWriter logger.ErrWriter) error {
	loggerConfig, err := config.NewLoggerConfig(cfgPath)
	if err != nil {
		return err
	}
	scimLogger, err := logger.NewLogger(logWriter, errWriter, loggerConfig)
	if err != nil {
		return err
	}
	certGenerator := certs.NewGenerator(scimLogger)

	cfg, err := config.NewConfig(cfgPath, scimLogger, certGenerator)
	if err != nil {
		return err
	}

	userHandler, err := users.NewUsersResourceHandler(cfg, scimLogger)
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

	groupHandler, err := groups.NewGroupResourceHandler(cfg, scimLogger)
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

	serverArgs := &scim.ServerArgs{
		ServiceProviderConfig: &scim.ServiceProviderConfig{
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

	server, err := scim.NewServer(serverArgs)
	if err != nil {
		return err
	}

	app := new(application)
	app.cfg = &cfg.Server.Auth

	srv := &http.Server{
		Addr:         cfg.Server.ListenAddress,
		Handler:      app.auth(server.ServeHTTP),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return srv.ListenAndServeTLS(cfg.Server.Certs.TLSCertPath, cfg.Server.Certs.TLSKeyPath)
}

type application struct {
	cfg *config.AuthConfig
}

func (app *application) auth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.cfg.Basic.Enabled && !app.cfg.Bearer.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if ok && app.cfg.Basic.Enabled {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(app.cfg.Basic.Username))
			expectedPasswordHash := sha256.Sum256([]byte(app.cfg.Basic.Password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		} else if app.cfg.Bearer.Enabled {
			reqToken := r.Header.Get("Authorization")
			splitToken := strings.Split(reqToken, "Bearer ")
			if len(splitToken) == 2 {
				if subtle.ConstantTimeCompare([]byte(app.cfg.Bearer.Token), []byte(splitToken[1])) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
