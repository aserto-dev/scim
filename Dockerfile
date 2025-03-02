FROM alpine

RUN apk add --no-cache bash tzdata

WORKDIR /app

COPY aserto-scim /app/

ENTRYPOINT ["./aserto-scim"]
