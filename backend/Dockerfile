FROM golang:1.26-alpine AS build
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/touchgal-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/touchgal-sync ./cmd/sync

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build /out/touchgal-api /usr/local/bin/touchgal-api
COPY --from=build /out/touchgal-sync /usr/local/bin/touchgal-sync
COPY internal/openapi ./internal/openapi
EXPOSE 8080
CMD ["touchgal-api"]
