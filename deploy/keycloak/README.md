# Keycloak (Fresnel realm)

## Issuer / Docker networking

Browsers usually open Keycloak at **`http://localhost:8081`** (published port). Tokens then carry `iss: http://localhost:8081/realms/fresnel`. The Fresnel container validates that issuer **and** fetches JWKS from the internal URL **`http://keycloak:8080/...`** (same realm, same keys).

For **token exchange and refresh**, the API POSTs to **`KEYCLOAK_TOKEN_URL`** (see `docker-compose.yml`), targeting **`host.docker.internal:8081`** so Keycloak accepts refresh tokens whose issuer matches the browser flow. If refresh still fails on your OS, set `KEYCLOAK_TOKEN_URL` to a URL the Fresnel container can reach that matches your Keycloak `iss`.

- **Realm file:** `fresnel-realm.json` is mounted into the Keycloak container and imported on startup (`--import-realm` in `docker-compose.yml`).
- **Self-registration:** disabled (`registrationAllowed: false`). Users are created by admins or by editing the realm / using the Admin API.
- **Pre-created app login:** `platform-admin` / password in `fresnel-realm.json` (change immediately after first boot). Email `admin@fresnel.local` matches the PostgreSQL row from migration `008_m1_platform_admin.sql` for Fresnel linking.
- **Keycloak super-admin (container):** `KC_BOOTSTRAP_ADMIN_*` in Compose is only for the **master** Keycloak admin UI (`/admin`), not the Fresnel realm users.
- **Future user JSON sketches:** see `FUTURE_USERS.example.md` (not imported).

If realm import fails to create users (some Keycloak versions expect users in a separate `*-users-*.json` file), create `platform-admin` manually in the Admin Console with the same email and assign a password; Fresnel will link by email on first OIDC login.
