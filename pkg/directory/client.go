package directory

import (
	"context"
	"encoding/json"

	"github.com/aserto-dev/go-aserto/client"
	dsr3 "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw3 "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/scim/pkg/config"
	"github.com/pkg/errors"
)

type DirectoryClient struct {
	Reader dsr3.ReaderClient
	Writer dsw3.WriterClient
}

func connect(ctx context.Context, cfg *client.Config) (*client.Connection, error) {
	opts := []client.ConnectionOption{
		client.WithAddr(cfg.Address),
		client.WithInsecure(cfg.Insecure),
	}

	if cfg.APIKey != "" {
		opts = append(opts, client.WithAPIKeyAuth(cfg.APIKey))
	}
	if cfg.TenantID != "" {
		opts = append(opts, client.WithTenantID(cfg.TenantID))
	}

	conn, err := client.NewConnection(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func GetDirectoryClient(ctx context.Context, cfg *client.Config) (*DirectoryClient, error) {
	dirConn, err := connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &DirectoryClient{
		Reader: dsr3.NewReaderClient(dirConn.Conn),
		Writer: dsw3.NewWriterClient(dirConn.Conn),
	}, nil
}

func (d *DirectoryClient) GetTransformConfigMap(ctx context.Context) (map[string]interface{}, error) {
	varsResp, err := d.Reader.GetObject(ctx, &dsr3.GetObjectRequest{
		ObjectType: "scim_config",
		ObjectId:   "scim_config",
	})
	if err != nil {
		return nil, err
	}

	return varsResp.Result.Properties.AsMap(), nil
}

func (d *DirectoryClient) GetTransformConfig(ctx context.Context) (*config.TransformConfig, error) {
	t, err := d.GetTransformConfigMap(ctx)
	if err != nil {
		return &config.TransformConfig{}, err
	}

	cfg := &config.TransformConfig{}
	jsonData, err := json.Marshal(t)
	if err != nil {
		return &config.TransformConfig{}, errors.Wrap(err, "failed to marshal transform config")
	}

	if err := json.Unmarshal(jsonData, cfg); err != nil {
		return &config.TransformConfig{}, errors.Wrap(err, "failed to unmarshal transform config")
	}

	return cfg, nil
}
