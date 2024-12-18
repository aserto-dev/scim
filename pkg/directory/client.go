package directory

import (
	"context"

	client "github.com/aserto-dev/go-aserto"
	"github.com/aserto-dev/go-aserto/ds/v3"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
)

func GetTenantDirectoryClient(cfg *client.Config) (*ds.Client, error) {
	conn, err := cfg.Connect()
	if err != nil {
		return nil, err
	}

	return ds.FromConnection(conn)
}

func GetTransformConfigMap(ctx context.Context, rootClient *ds.Client, cfgKey string) (map[string]interface{}, error) {
	varsResp, err := rootClient.Reader.GetObject(ctx, &dsr.GetObjectRequest{
		ObjectType: cfgKey,
		ObjectId:   cfgKey,
	})
	if err != nil {
		return nil, err
	}

	return varsResp.Result.Properties.AsMap(), nil
}
