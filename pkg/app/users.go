package app

import (
	"net/http"

	"github.com/aserto-dev/scim/common/handlers/users"
	"github.com/elimity-com/scim"
)

type UsersResourceHandler struct {
	handler *users.UsersResourceHandler
}

func NewUsersResourceHandler(handler *users.UsersResourceHandler) (*UsersResourceHandler, error) {
	return &UsersResourceHandler{
		handler: handler,
	}, nil
}

func (u UsersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	return u.handler.Create(r.Context(), attributes)
}

func (u UsersResourceHandler) Delete(r *http.Request, id string) error {
	return u.handler.Delete(r.Context(), id)
}

func (u UsersResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	return u.handler.Get(r.Context(), id)
}

func (u UsersResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	return u.handler.GetAll(r.Context(), params)
}

func (u UsersResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	return u.handler.Patch(r.Context(), id, operations)
}

func (u UsersResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	return u.handler.Replace(r.Context(), id, attributes)
}
