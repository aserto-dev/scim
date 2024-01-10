FROM alpine

RUN apk add --no-cache bash tzdata

WORKDIR /app

COPY dist/aserto-scim*_linux_amd64_v1/aserto-scim* /app/

ENTRYPOINT ["./aserto-scim"]
