# scim
Aserto SCIM service

### start service
```
go run ./cmd/aserto-scim/main.go run --address {directory addr} --tenant-id {tenant-id} --api-key {api key}
```

### list users

`curl 127.0.0.1:8080/Users`

### create user
```
curl -X 'POST' -H 'Content-Type: application/json' 'http://127.0.0.1:8080/Users' --data-binary @- <<EOF
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "test1234",
  "emails": [
    {
	  "value": "test@example.com"
    }
  ]
}
EOF
```

### delete user
`curl -X 'DELETE' -H 'Content-Type: application/json' 'http://127.0.0.1:8080/Users/{user id}' `