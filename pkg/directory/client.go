package directory

import (
	"context"

	"github.com/aserto-dev/go-aserto/client"
	dsr3 "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw3 "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
)

type DirectoryClient struct {
	Reader dsr3.ReaderClient
	Writer dsw3.WriterClient
}

func connect(cfg *client.Config) (*client.Connection, error) {
	ctx := context.Background()

	conn, err := client.NewConnection(ctx,
		client.WithAddr(cfg.Address),
		client.WithAPIKeyAuth(cfg.APIKey),
		client.WithTenantID(cfg.TenantID),
		client.WithInsecure(cfg.Insecure),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func GetDirectoryClient(cfg *client.Config) (*DirectoryClient, error) {
	dirConn, err := connect(cfg)
	if err != nil {
		return nil, err
	}
	return &DirectoryClient{
		Reader: dsr3.NewReaderClient(dirConn.Conn),
		Writer: dsw3.NewWriterClient(dirConn.Conn),
	}, nil
}
