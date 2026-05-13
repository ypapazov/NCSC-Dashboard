# Fetch dependencies
FROM golang:1.24-alpine AS fetch
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# Generate help HTML (static/help is gitignored; must run before embed) and templ outputs
FROM golang:1.24-alpine AS generate
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TEMPL_VERSION=v0.3.1001
RUN go run ./cmd/helpgen help/en static/help/en
RUN go install github.com/a-h/templ/cmd/templ@${TEMPL_VERSION} \
	&& "$(go env GOPATH)/bin/templ" generate

# Build
FROM golang:1.24-alpine AS build
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
