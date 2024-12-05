# Configuring the scim service

## Single tenant setup
In a single tenant setup, scim syncs users to just one tenant, that has been configured. For this, the tenant id and directory read-write API Key need to be configured.

```
logging:
  prod: true
  log_level: info
server:
  listen_address: ":8080"
  auth:
    basic:
      enabled: true
      username: "scim"
      password: "scim"
    bearer:
      enabled: true
      token: "scim"
directory:
  address: "directory.prod.aserto.com:8443"
  tenant_id: "your_tenant_id"
  api_key: "your_directory_rw_api_key"
```

To let the directory handle auth, set the `passthrough` flag to `true`.

```
      auth:
        basic:
          enabled: true
          passthrough: true
        bearer:
          enabled: true
          passthrough: true
```
The bearer token used should be set as base64 encoded `<tenant-id>:<api-key>`

## Multi-tenant setup

For a multitenant setup, the directory config should not contain the tenant-id and api-key. These will be passed using the authorization header.

```
logging:
  prod: true
  log_level: info
server:
  listen_address: ":8080"
  auth:
    basic:
      enabled: true
      passthrough: true
    bearer:
      enabled: true
      passthrough: true
directory:
  address: "directory.prod.aserto.com:8443"
```

## Transform config

The transform config is being read from the tenant the users are being synced to. For this, a object type `scim_config` with id `scim_config` is being read. This config can be used to override the default values for the transformation templat, aswell as the transformation template used when syncing data.

Sample `scim_config` and default values:
```
{
  "group_mappings": [],
  "group_member_relation": "member",
  "group_object_type": "group",
  "identity_object_type": "identity",
  "identity_relation": "identifier",
  "manager_relation": "manager",
  "role_object_type": "group",
  "role_relation": "member",
  "source_group_type": "scim.2.0.group",
  "source_user_type": "scim.2.0.user",
  "template": "users-groups-roles-v1",
  "user_mappings": [],
  "user_object_type": "user"
}
```