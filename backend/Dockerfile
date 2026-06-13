# syntax=docker/dockerfile:1.7
FROM golang:1.26.4-alpine AS build
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /out/touchgal-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/touchgal-sync ./cmd/sync

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/touchgal-api /usr/local/bin/touchgal-api
COPY --from=build /out/touchgal-sync /usr/local/bin/touchgal-sync
EXPOSE 8080
CMD ["touchgal-api"]
