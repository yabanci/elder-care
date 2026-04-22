# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.20 AS server
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=builder /out/server /usr/local/bin/server
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/server"]

FROM alpine:3.20 AS seed
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=builder /out/seed /usr/local/bin/seed
ENTRYPOINT ["/usr/local/bin/seed"]
