package app

import (
	"net/http"

	"github.com/aserto-dev/scim/common/handlers"
	"github.com/elimity-com/scim"
)

type ResourceHandler struct {
	handler handlers.ResourceHandler
}

func NewResourceHandler(handler handlers.ResourceHandler) (scim.ResourceHandler, error) {
	return &ResourceHandler{
		handler: handler,
	}, nil
}

func (g ResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	return g.handler.Create(r.Context(), attributes)
}

func (g ResourceHandler) Delete(r *http.Request, id string) error {
	return g.handler.Delete(r.Context(), id)
}

func (g ResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	return g.handler.Get(r.Context(), id)
}

func (g ResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	return g.handler.GetAll(r.Context(), params)
}

func (g ResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	return g.handler.Patch(r.Context(), id, operations)
}

func (g ResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	return g.handler.Replace(r.Context(), id, attributes)
}
