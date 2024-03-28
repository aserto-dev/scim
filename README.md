# scim
The Aserto SCIM service uses the SCIM 2.0 protocol to import data into the Aserto Directory. While not all features have been implemented yet, it supports the basic operations in order to manage users and groups using the SCIM core schemas.

### sample config.yaml
```yaml
---
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
scim:
  create_email_identities: true
  create_role_groups: true
```

### start service
```
go run ./cmd/aserto-scim/main.go run -c ./config.yaml
```

### run as docker container

```
docker run -p 8080:8080 -v {config directory}:/config:ro ghcr.io/aserto-dev/scim:latest run -c /config/config.yaml
```

### list users

```
curl  -X GET \
  'http://127.0.0.1:8080/Users' \
  --header 'Accept: */*' \
  --header 'Authorization: Bearer scim'
```

### create user
```
curl  -X POST \
  'http://127.0.0.1:8080/Users' \
  --header 'Accept: */*' \
  --header 'Authorization: Bearer scim' \
  --header 'Content-Type: application/json' \
  --data-raw '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "rsanchez",
    "name": {
        "givenName": "Rick",
        "familyName": "Sanchez"
    },
    "emails": [{
        "primary": true,
        "value": "rick@the-citadel.com",
        "type": "work"
    }],
    "displayName": "Rick Sanchez",
    "locale": "en-US",
    "groups": [],
    "active": true
}'
```

### get a user
`curl -X 'GET' 'http://127.0.0.1:8080/Users/{user id}' `

```
curl  -X GET \
  'http://127.0.0.1:8080/Users/rsanchez' \
  --header 'Accept: */*' \
  --header 'Authorization: Bearer scim'
```

### delete a user
`curl -X 'DELETE' 'http://127.0.0.1:8080/Users/{user id}'`

```
curl  -X DELETE \
  'http://127.0.0.1:8080/Users/rsanchez' \
  --header 'Accept: */*' \
  --header 'Authorization: Bearer scim'
```

### patch user
`curl -X 'PATCH' 'http://127.0.0.1:8080/Users/{user id}'`

```
curl  -X PATCH \
  'http://127.0.0.1:8080/Users/rsanchez' \
  --header 'Accept: */*' \
  --header 'Authorization: Bearer scim' \
  --header 'Content-Type: application/json' \
  --data-raw '{
"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
"Operations":[
{"op":"add","path": "nickName","value": "Madman"},
{"op":"add","path": "emails[type eq \"home\"].value","value": "rick@home"}
]}'
```

### create group
```
curl  -X POST \
  'http://127.0.0.1:8080/Groups' \
  --header 'Accept: */*' \
  --header 'Authorization: Bearer scim' \
  --header 'Content-Type: application/json' \
  --data-raw '{"displayName": "admin"}'
```

### add user to group
```
curl  -X PATCH \
  'http://127.0.0.1:8080/Users/rsanchez' \
  --header 'Accept: */*' \
  --header 'Authorization: Bearer scim' \
  --header 'Content-Type: application/json' \
  --data-raw '{
"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
"Operations":[
{"op":"add","path": "groups[type eq \"work\"].value","value": "admin"}
]}'
```
