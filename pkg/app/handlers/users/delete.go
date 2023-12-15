package users

import (
	"log"
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsr "github.com/aserto-dev/go-directory/aserto/directory/reader/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
)

func (u UsersResourceHandler) Delete(r *http.Request, id string) error {
	log.Println("DELETE", id)
	relations, err := u.dirClient.Reader.GetRelations(r.Context(), &dsr.GetRelationsRequest{
		SubjectType: "user",
		SubjectId:   id,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrObjectNotFound) {
			return serrors.ScimErrorResourceNotFound(id)
		}
		return err
	}

	for _, v := range relations.Results {
		if v.Relation == "identifier" {
			_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
				ObjectId:      v.ObjectId,
				ObjectType:    v.ObjectType,
				WithRelations: true,
			})
			if err != nil {
				return err
			}
		}
	}

	_, err = u.dirClient.Writer.DeleteObject(r.Context(), &dsw.DeleteObjectRequest{
		ObjectType:    "user",
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
