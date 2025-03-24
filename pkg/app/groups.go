package app

import (
	"net/http"

	"github.com/aserto-dev/scim/common/handlers/groups"
	"github.com/elimity-com/scim"
)

type GroupResourceHandler struct {
	handler *groups.GroupResourceHandler
}

func NewGroupResourceHandler(handler *groups.GroupResourceHandler) (*GroupResourceHandler, error) {
	return &GroupResourceHandler{
		handler: handler,
	}, nil
}

func (g GroupResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	return g.handler.Create(r.Context(), attributes)
}

func (g GroupResourceHandler) Delete(r *http.Request, id string) error {
	return g.handler.Delete(r.Context(), id)
}

func (g GroupResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	return g.handler.Get(r.Context(), id)
}

func (g GroupResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	return g.handler.GetAll(r.Context(), params)
}

func (g GroupResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	return g.handler.Patch(r.Context(), id, operations)
}

func (g GroupResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	return g.handler.Replace(r.Context(), id, attributes)
}
