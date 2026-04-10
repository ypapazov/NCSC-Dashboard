# Fetch dependencies
FROM golang:1.23-alpine AS fetch
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# Generate templ
FROM ghcr.io/a-h/templ:latest AS generate
COPY --chown=65532:65532 . /src
WORKDIR /src
RUN ["templ", "generate"]

# Build
FROM golang:1.23-alpine AS build
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY --from=fetch /go /go
COPY --from=generate /src /src
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/fresnel ./cmd/fresnel

# Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -H -u 65532 nonroot
WORKDIR /app
COPY --from=build /out/fresnel /app/fresnel
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/fresnel"]
