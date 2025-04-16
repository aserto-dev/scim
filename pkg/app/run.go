package app

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/aserto-dev/go-aserto/ds/v3"
	"github.com/aserto-dev/logger"
	"github.com/aserto-dev/scim/common/convert"
	"github.com/aserto-dev/scim/common/handlers/groups"
	"github.com/aserto-dev/scim/common/handlers/users"
	"github.com/aserto-dev/scim/pkg/app/directory"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/rs/zerolog"
)

type SCIMServer struct {
	server   *http.Server
	log      *zerolog.Logger
	cfg      *config.Config
	dsClient *ds.Client
}

func NewSCIMServer(cfgPath string, logWriter logger.Writer, errWriter logger.ErrWriter) (*SCIMServer, error) {
	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	scimLogger, err := logger.NewLogger(logWriter, errWriter, &cfg.Logging)
	if err != nil {
		return nil, err
	}

	return &SCIMServer{
		log: scimLogger,
		cfg: cfg,
	}, nil
}

func (s *SCIMServer) Run() error {
	dsClient, err := directory.GetDirectoryClient(&s.cfg.Directory)
	if err != nil {
		return err
	}

	s.dsClient = dsClient

	resourceTypes, err := s.resourceTypes()
	if err != nil {
		return err
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
				},
			},
		},
		ResourceTypes: resourceTypes,
	}

	server, err := scim.NewServer(serverArgs)
	if err != nil {
		return err
	}

	app := new(application)
	app.cfg = &s.cfg.Server.Auth

	tlsServerConfig, err := s.cfg.Server.Certs.ServerConfig()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              s.cfg.Server.ListenAddress,
		Handler:           app.auth(server.ServeHTTP),
		TLSConfig:         tlsServerConfig,
		IdleTimeout:       s.cfg.Server.IdleTimeout,
		ReadTimeout:       s.cfg.Server.ReadTimeout,
		WriteTimeout:      s.cfg.Server.WriteTimeout,
		ReadHeaderTimeout: s.cfg.Server.ReadHeaderTimeout,
	}

	s.server = srv
	s.log.Info().Str("address", s.cfg.Server.ListenAddress).Msg("Starting SCIM server")

	if s.cfg.Server.Certs.HasCert() {
		return srv.ListenAndServeTLS("", "")
	}

	fmt.Println("Starting SCIM server without TLS")

	return srv.ListenAndServe()
}

func (s *SCIMServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		s.log.Info().Msg("Shutting down SCIM server")
		return s.server.Shutdown(ctx)
	}

	s.server = nil

	if s.dsClient != nil {
		s.log.Info().Msg("Closing directory client connection")

		if err := s.dsClient.Close(); err != nil {
			s.log.Error().Err(err).Msg("Failed to close directory client")
		}
	}

	s.dsClient = nil
	s.log.Info().Msg("SCIM server shutdown complete")

	return nil
}

func (s *SCIMServer) userHandler(cfg *convert.TransformConfig) (scim.ResourceHandler, error) {
	usersLogger := s.log.With().Str("component", "users").Logger()

	usersResourceHandler, err := users.NewUsersResourceHandler(&usersLogger, cfg, s.dsClient)
	if err != nil {
		return nil, err
	}

	return NewResourceHandler(usersResourceHandler)
}

func (s *SCIMServer) groupHandler(cfg *convert.TransformConfig) (scim.ResourceHandler, error) {
	groupsLogger := s.log.With().Str("component", "groups").Logger()

	groupsResourceHandler, err := groups.NewGroupResourceHandler(&groupsLogger, cfg, s.dsClient)
	if err != nil {
		return nil, err
	}

	return NewResourceHandler(groupsResourceHandler)
}

func (s *SCIMServer) resourceTypes() ([]scim.ResourceType, error) {
	transformCfg, err := convert.NewTransformConfig(&s.cfg.SCIM)
	if err != nil {
		return nil, err
	}

	userHandler, err := s.userHandler(transformCfg)
	if err != nil {
		return nil, err
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

	groupHandler, err := s.groupHandler(transformCfg)
	if err != nil {
		return nil, err
	}

	groupType := scim.ResourceType{
		ID:          optional.NewString("Group"),
		Name:        "Group",
		Endpoint:    "/Groups",
		Description: optional.NewString("Group"),
		Schema:      schema.CoreGroupSchema(),
		Handler:     groupHandler,
	}

	return []scim.ResourceType{userType, groupType}, nil
}

const authzHeaderParts = 2

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
		if ok && app.cfg.Basic.Enabled && app.checkBasicAuth(username, password) {
			next.ServeHTTP(w, r)
			return
		} else if app.cfg.Bearer.Enabled {
			reqToken := r.Header.Get("Authorization")
			splitToken := strings.Split(reqToken, "Bearer ")

			if len(splitToken) == authzHeaderParts {
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

func (app *application) checkBasicAuth(username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))

	expectedUsernameHash := sha256.Sum256([]byte(app.cfg.Basic.Username))
	expectedPasswordHash := sha256.Sum256([]byte(app.cfg.Basic.Password))

	usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
	passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

	return usernameMatch && passwordMatch
}
