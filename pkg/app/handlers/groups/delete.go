package groups

import (
	"net/http"

	"github.com/aserto-dev/scim/pkg/convert"
	"github.com/aserto-dev/scim/pkg/directory"
	serrors "github.com/elimity-com/scim/errors"
)

func (u GroupResourceHandler) Delete(r *http.Request, id string) error {
	u.logger.Trace().Str("id", id).Msg("deleting group")

	dirClient, err := u.getDirectoryClient(r)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get directory client")
		return serrors.ScimErrorInternal
	}

	scimConfigMap, err := dirClient.GetTransformConfigMap(r.Context())
	if err != nil {
		return err
	}
	scimConfig, err := convert.TransformConfigFromMap(u.cfg.SCIM.TransformDefaults, scimConfigMap)
	if err != nil {
		return err
	}

	sync := directory.NewSync(scimConfig, dirClient)
	err = sync.DeleteGroup(r.Context(), id)

	return err
}
