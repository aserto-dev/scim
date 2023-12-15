package groups

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u GroupResourceHandler) Delete(r *http.Request, id string) error {
	_, err := u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
		ObjectType:    "group",
		ObjectId:      id,
		WithRelations: true,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
	}

	return err
}
