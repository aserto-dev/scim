package common_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aserto-dev/go-aserto"
	"github.com/aserto-dev/go-aserto/ds/v3"
	dsm "github.com/aserto-dev/go-directory/aserto/directory/model/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	"github.com/aserto-dev/logger"
	"github.com/aserto-dev/scim/pkg/app"
	assets_test "github.com/aserto-dev/scim/pkg/test/assets"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const containerStartTimeout = 300 * time.Second
const topazConfigFileMode = 0o700

type TestCase struct {
	Topaz           testcontainers.Container
	DirectoryClient *ds.Client
}

func (tst *TestCase) ContainerLogs(ctx context.Context, t *testing.T) string {
	require := require.New(t)

	logs, err := tst.Topaz.Logs(ctx)
	require.NoError(err)

	t.Cleanup(func() { _ = logs.Close() })

	logData, err := io.ReadAll(logs)
	require.NoError(err)

	return string(logData)
}

func TestSetup(t *testing.T) TestCase {
	t.Logf("\nTEST CONTAINER IMAGE: %q\n", TopazImage())

	req := testcontainers.ContainerRequest{
		AlwaysPullImage: true,
		Image:           TopazImage(),
		ExposedPorts:    []string{"9292/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            assets_test.TopazConfigReader(),
				ContainerFilePath: "/config/config.yaml",
				FileMode:          topazConfigFileMode,
			},
		},
		WaitingFor: wait.ForAll(
			wait.ForExposedPort(),
			wait.ForLog("Starting 0.0.0.0:9292 gRPC server"),
		).WithStartupTimeoutDefault(containerStartTimeout),
	}

	topaz, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	require.NoError(t, err)

	if err := topaz.Start(t.Context()); err != nil {
		require.NoError(t, err)
	}

	addr, err := MappedAddr(t.Context(), topaz, "9292")
	require.NoError(t, err)

	t.Setenv("ASERTO_SCIM_DIRECTORY_ADDRESS", addr)

	scimConfig, err := filepath.Abs("assets/config/scim.yaml")
	require.NoError(t, err)

	srv, err := app.NewSCIMServer(scimConfig, logger.TestLogger(os.Stdout), os.Stderr)
	require.NoError(t, err)

	go func() {
		srv.Run()
	}()

	time.Sleep(time.Second)

	dirCfg := aserto.Config{
		Address: addr,
		NoTLS:   true,
	}

	conn, err := dirCfg.Connect()
	require.NoError(t, err)

	dsClient := ds.FromConnection(conn)
	stream, err := dsClient.Model.SetManifest(t.Context())
	require.NoError(t, err)
	err = stream.Send(&dsm.SetManifestRequest{
		Msg: &dsm.SetManifestRequest_Body{
			Body: &dsm.Body{
				Data: assets_test.Manifest(),
			},
		},
	})
	require.NoError(t, err)
	_, err = stream.CloseAndRecv()
	require.NoError(t, err)

	t.Cleanup(func() {
		conn.Close()
		srv.Shutdown(t.Context())
		testcontainers.CleanupContainer(t, topaz)
	})

	return TestCase{
		Topaz:           topaz,
		DirectoryClient: dsClient,
	}
}

func (tst *TestCase) UserHasIdentity(ctx context.Context, user, identity string) bool {
	userResp, err := tst.DirectoryClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
		Relation:    "identifier",
		ObjectType:  "user",
		ObjectId:    user,
		SubjectType: "identity",
		SubjectId:   identity,
	})
	if err != nil {
		return false
	}

	return userResp.GetResult() != nil
}

func (tst *TestCase) UserHasManager(ctx context.Context, user, manager string) bool {
	userResp, err := tst.DirectoryClient.Reader.GetRelation(ctx, &dsr.GetRelationRequest{
		Relation:    "manager",
		ObjectType:  "user",
		ObjectId:    user,
		SubjectType: "user",
		SubjectId:   manager,
	})
	if err != nil {
		return false
	}

	return userResp.GetResult() != nil
}

func (tst *TestCase) ReadUserProperty(ctx context.Context, user, property string) any {
	userResp, err := tst.DirectoryClient.Reader.GetObject(ctx, &dsr.GetObjectRequest{
		ObjectType: "user",
		ObjectId:   user,
	})
	if err != nil || userResp.GetResult() == nil {
		return nil
	}

	return userResp.GetResult().GetProperties().GetFields()[property].AsInterface()
}

func TopazImage() string {
	image := os.Getenv("TOPAZ_TEST_IMAGE")
	if image != "" {
		return image
	}

	return "ghcr.io/aserto-dev/topaz:latest"
}

func MappedAddr(ctx context.Context, container testcontainers.Container, port string) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", host, mappedPort.Port()), nil
}
