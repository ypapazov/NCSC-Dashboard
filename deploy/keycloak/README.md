# Keycloak (Fresnel realm)

## Realm configuration

The realm definition is **not committed to source control** — it contains deployment-specific URLs and credentials.

**First-time setup:**

```bash
cp fresnel-realm.json.example fresnel-realm.json
```

Edit `fresnel-realm.json`:

1. Replace `CHANGEME.example.org` with your actual domain (redirect URIs, web origins, post-logout URIs).
2. Replace the `CHANGEME` password for `platform-root` with a strong initial password.
3. Add any additional seed users you need (or create them later via the Admin Console).

The file is imported by Keycloak on first startup (`--import-realm` with `IGNORE_EXISTING` strategy). Once the realm exists, Keycloak skips the import on subsequent restarts — changes made via the Admin Console are preserved.

## Docker networking

Keycloak is proxied by nginx at `/realms/`, `/resources/`, and `/js/`. The browser accesses Keycloak through the same origin as the app (no CORS issues).

The Fresnel container validates tokens using the **internal** issuer URL (`http://keycloak:8080/realms/fresnel`) and also accepts the **external** issuer (`https://yourdomain/realms/fresnel`).

## Key points

- **Self-registration:** disabled (`registrationAllowed: false`). Users are created by admins.
- **Pre-created user:** `platform-root` with email `admin@fresnel.local` matches the PostgreSQL row from migration `008_m1_platform_admin.sql`. Fresnel links OIDC identities by email on first login.
- **Keycloak super-admin:** `KC_BOOTSTRAP_ADMIN_*` in Compose is for the **master** realm admin UI (`/admin`), not the Fresnel realm users.
- **Admin console access:** not exposed externally. Use SSM port forwarding:

```bash
aws ssm start-session \
  --target <instance-id> \
  --document-name AWS-StartPortForwardingRemoteHost \
  --parameters '{"host":["localhost"],"portNumber":["8080"],"localPortNumber":["8080"]}'
```

Then open `http://localhost:8080/admin/`.
