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
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates wget && \
    addgroup -S -g 10001 touchgal && \
    adduser -S -D -H -u 10001 -G touchgal touchgal
COPY --from=build /out/touchgal-api /usr/local/bin/touchgal-api
COPY --from=build /out/touchgal-sync /usr/local/bin/touchgal-sync
EXPOSE 8080
USER 10001:10001
CMD ["touchgal-api"]
