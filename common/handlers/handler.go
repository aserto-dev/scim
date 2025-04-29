package handlers

import (
	"context"

	"github.com/elimity-com/scim"
)

type ResourceHandler interface {
	Create(ctx context.Context, attributes scim.ResourceAttributes) (scim.Resource, error)
	Get(ctx context.Context, id string) (scim.Resource, error)
	GetAll(ctx context.Context, params scim.ListRequestParams) (scim.Page, error)
	Patch(ctx context.Context, id string, operations []scim.PatchOperation) (scim.Resource, error)
	Replace(ctx context.Context, id string, attributes scim.ResourceAttributes) (scim.Resource, error)
	Delete(ctx context.Context, id string) error
}
