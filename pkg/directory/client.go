package directory

import (
	client "github.com/aserto-dev/go-aserto"
	"github.com/aserto-dev/go-aserto/ds/v3"
)

func GetDirectoryClient(cfg *client.Config) (*ds.Client, error) {
	conn, err := cfg.Connect()
	if err != nil {
		return nil, err
	}

	return ds.FromConnection(conn)
}
