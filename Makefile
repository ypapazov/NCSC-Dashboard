.PHONY: build test lint compose-up compose-dev compose-down migrate seed run certs generate

generate:
	templ generate

build: generate
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/fresnel ./cmd/fresnel

test:
	go test -count=1 ./...

lint:
	golangci-lint run ./...

certs:
	@mkdir -p deploy/nginx/certs
	openssl req -x509 -nodes -days 825 -newkey rsa:2048 \
		-keyout deploy/nginx/certs/server.key \
		-out deploy/nginx/certs/server.crt \
		-subj "/CN=localhost/O=Fresnel Dev/C=US"

compose-up: certs
	docker compose -f deploy/docker-compose.yml up --build -d

compose-dev: certs
	docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up --build -d

compose-down:
	docker compose -f deploy/docker-compose.yml down

migrate:
	go run ./cmd/fresnel migrate

seed:
	./scripts/seed-dev-data.sh

run: generate
	LISTEN_ADDR=:8080 \
	DATABASE_URL=$${DATABASE_URL:-postgres://fresnel:fresnel@127.0.0.1:5432/fresnel?sslmode=disable} \
	KEYCLOAK_ISSUER=$${KEYCLOAK_ISSUER:-http://127.0.0.1:8081/realms/fresnel} \
	KEYCLOAK_CLIENT_ID=$${KEYCLOAK_CLIENT_ID:-fresnel-app} \
	KEYCLOAK_EXTERNAL_URL=$${KEYCLOAK_EXTERNAL_URL:-https://localhost/realms/fresnel} \
	APP_PUBLIC_URL=$${APP_PUBLIC_URL:-https://localhost} \
	go run ./cmd/fresnel
