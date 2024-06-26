package app

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/aserto-dev/certs"
	"github.com/aserto-dev/logger"
	"github.com/aserto-dev/scim/pkg/app/handlers/groups"
	"github.com/aserto-dev/scim/pkg/app/handlers/users"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/rs/zerolog"
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

	userHandler := users.NewUsersResourceHandler(cfg, scimLogger)

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

	groupHandler := groups.NewGroupResourceHandler(cfg, scimLogger)

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

	server, err := scim.NewServer(serverArgs, scim.WithLogger(NewLogger(scimLogger)))
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
		if app.cfg.Anonymous {
			next.ServeHTTP(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if ok && app.cfg.Basic.Enabled {
			if app.cfg.Basic.Passthrough {
				// let the directory authenticate the user
				authContext := context.WithValue(r.Context(), common.ContextKeyTenantID, username)
				authContext = context.WithValue(authContext, common.ContextKeyAPIKey, password)
				next.ServeHTTP(w, r.WithContext(authContext))
				return
			} else {
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
			}
		} else if app.cfg.Bearer.Enabled {
			reqToken := r.Header.Get("Authorization")
			splitToken := strings.Split(reqToken, "Bearer ")
			if len(splitToken) == 2 {
				if app.cfg.Bearer.Passthrough {
					// let the directory authenticate the user
					username, password, ok := app.parseToken(splitToken[1])
					if ok {
						authContext := context.WithValue(r.Context(), common.ContextKeyTenantID, username)
						authContext = context.WithValue(authContext, common.ContextKeyAPIKey, password)
						next.ServeHTTP(w, r.WithContext(authContext))
						return
					}
				}
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

func (app *application) parseToken(auth string) (username, password string, ok bool) {
	c, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", false
	}
	cs := string(c)
	username, password, ok = strings.Cut(cs, ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

type scimLogger struct {
	log *zerolog.Logger
}

func NewLogger(l *zerolog.Logger) scimLogger {
	return scimLogger{
		log: l,
	}
}

func (l scimLogger) Error(args ...interface{}) {
	l.log.Error().Any("args", args).Msg("error occured")
}
