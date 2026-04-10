# Build
FROM golang:1.23-alpine AS build
RUN apk add --no-cache ca-certificates git
COPY --from=ghcr.io/a-h/templ:latest /templ /usr/local/bin/templ
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN templ generate
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/fresnel ./cmd/fresnel

# Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -H -u 65532 nonroot
WORKDIR /app
COPY --from=build /out/fresnel /app/fresnel
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/fresnel"]
