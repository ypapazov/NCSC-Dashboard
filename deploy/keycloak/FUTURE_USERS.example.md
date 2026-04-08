# Future Keycloak users (examples — not applied)

Use this file as a **design reference** when you create additional identities in Keycloak (Admin Console, Admin API, or realm export).  
These blocks are **not** imported automatically; copy fields into your realm export, `kcadm`, or Terraform when you are ready.

**Today:** the only baked-in realm user is `platform-admin` (see `fresnel-realm.json`).  
**Self-service sign-up:** disabled in the realm (`registrationAllowed: false`). To allow open registration again (e.g. a public pilot), set `registrationAllowed` to `true` in realm settings and tighten flows, CAPTCHA, and email verification before production.

---

## Role mapping reminder

Fresnel **application** roles (Cedar, `fresnel_iam.role_assignments`, roots) are stored in **PostgreSQL**, not only in Keycloak.  
Keycloak users are the identity; after first OIDC login, link `keycloak_sub` to `fresnel.users` (email match or admin insert).  
Future automation might sync Keycloak groups → app roles — that is **not** implemented in the PoC.

---

## Example: sector root (Government)

```json
{
  "username": "gov-sector-root",
  "enabled": true,
  "emailVerified": true,
  "email": "sector-root-gov@example.invalid",
  "firstName": "Sector",
  "lastName": "Root Gov",
  "credentials": [
    {
      "type": "password",
      "value": "REPLACE_WITH_SECRET",
      "temporary": true
    }
  ]
}
```

**Later:** insert matching `fresnel.users` row and `fresnel_iam.root_designations` / `role_assignments` for the Government sector scope.

---

## Example: organization root (Org A)

```json
{
  "username": "orgA-root",
  "enabled": true,
  "emailVerified": true,
  "email": "orga-root@example.invalid",
  "firstName": "Org A",
  "lastName": "Root",
  "credentials": [
    {
      "type": "password",
      "value": "REPLACE_WITH_SECRET",
      "temporary": true
    }
  ]
}
```

---

## Example: org admin (contributor management)

```json
{
  "username": "orgA-admin",
  "enabled": true,
  "emailVerified": true,
  "email": "orga-admin@example.invalid",
  "firstName": "Org A",
  "lastName": "Admin",
  "credentials": [
    {
      "type": "password",
      "value": "REPLACE_WITH_SECRET",
      "temporary": true
    }
  ]
}
```

---

## Example: contributor

```json
{
  "username": "orgA-contributor",
  "enabled": true,
  "emailVerified": true,
  "email": "contributor@example.invalid",
  "firstName": "Casey",
  "lastName": "Contributor",
  "credentials": [
    {
      "type": "password",
      "value": "REPLACE_WITH_SECRET",
      "temporary": true
    }
  ]
}
```

---

## Example: viewer (read-only)

```json
{
  "username": "orgA-viewer",
  "enabled": true,
  "emailVerified": true,
  "email": "viewer@example.invalid",
  "firstName": "Taylor",
  "lastName": "Viewer",
  "credentials": [
    {
      "type": "password",
      "value": "REPLACE_WITH_SECRET",
      "temporary": true
    }
  ]
}
```

---

## Example: liaison (cross-org read per policy)

```json
{
  "username": "liaison-acme",
  "enabled": true,
  "emailVerified": true,
  "email": "liaison@example.invalid",
  "firstName": "Jamie",
  "lastName": "Liaison",
  "credentials": [
    {
      "type": "password",
      "value": "REPLACE_WITH_SECRET",
      "temporary": true
    }
  ]
}
```

---

## TOTP / WebAuthn (future)

The implementation plan targets **required OTP** for production-like accounts. When you enable it:

- Realm: **Authentication** → flows → add OTP to browser flow.
- Per-user **Required actions** → **Configure OTP** (or use a conditional authenticator).

Do not commit real secrets or production passwords into JSON; use Keycloak’s credential reset or Admin API in CI.
