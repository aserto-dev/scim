package common

type ContextKey string

var (
	ContextKeyTenantID = ContextKey("Aerto-Tenant-Id")
	ContextKeyAPIKey   = ContextKey("Aerto-API-Key")
)
