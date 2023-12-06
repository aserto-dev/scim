# scim
Aserto SCIM service

### start service
```
go run ./cmd/aserto-scim/main.go run -c ./config.yaml
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

### get user
`curl -X 'GET' 'http://127.0.0.1:8080/Users/{user id}' `

### delete user
`curl -X 'DELETE' 'http://127.0.0.1:8080/Users/{user id}' `