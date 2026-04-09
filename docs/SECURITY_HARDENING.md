# Security hardening guide (Fresnel)

This guide complements the default Docker/nginx configuration in `deploy/` with production-oriented controls: WAF, brute-force throttling, headers, identity hardening, database permissions, network segmentation, backups, and incident response patterns.

It is descriptive: some items are **not implemented in code** (notably break-glass flows, marked as Phase 2).

---

## nginx ModSecurity + OWASP Core Rule Set (CRS)

**Goal**: Block common web attacks (SQLi, XSS, RCE probes) at the reverse proxy before they reach Fresnel.

**Outline**:

1. Use an nginx build with ModSecurity (e.g. OWASP ModSecurity CRS Docker image, or compile nginx with the ModSecurity-nginx connector).
2. Mount CRS rule files under `/etc/nginx/modsec/crs/` and include `crs-setup.conf` plus rule exclusions tuned for your APIs (JSON APIs often need relaxed rules for large bodies on attachment uploads).
3. Enable the connector in `nginx.conf`:

```nginx
modsecurity on;
modsecurity_rules_file /etc/nginx/modsec/main.conf;
```

4. Start in **DetectionOnly** mode, review false positives in the audit log, then switch to **On** for blocking.
5. Exempt or narrow rules for `/api/v1/events/{id}/attachments` if file uploads trigger CRS body inspection limits.

Document your exclusions: each one should reference a ticket and expiry review date.

---

## fail2ban for nginx access logs

**Goal**: Ban IPs that generate sustained 401/403/404 noise or obvious exploit scans.

**Outline**:

1. Install `fail2ban` on the host (or a sidecar that reads mounted logs).
2. Create a filter, e.g. `/etc/fail2ban/filter.d/fresnel-nginx.conf`:

```ini
[Definition]
failregex = ^<HOST> .* "(?:GET|POST|PUT|DELETE).*" (401|403|404) 
ignoreregex =
```

3. Create a jail, e.g. `/etc/fail2ban/jail.d/fresnel.conf`:

```ini
[fresnel-nginx]
enabled = true
port = http,https
filter = fresnel-nginx
logpath = /var/log/nginx/access.log
maxretry = 20
findtime = 600
bantime = 3600
```

4. Reload fail2ban and monitor `fail2ban-client status fresnel-nginx`.

Tune thresholds so legitimate API clients behind NAT are not banned; consider separate jails for `/api/` vs static assets.

---

## Content-Security-Policy (CSP)

**Purpose**: Reduce XSS impact by restricting script, connect, and frame sources.

**Current values** (from `deploy/nginx/security-headers.conf`):

```http
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' http://localhost:8081; frame-ancestors 'none'; form-action 'self'; base-uri 'self'
```

**Customization**:

- Replace `http://localhost:8081` in `connect-src` with your **production Keycloak origin** (scheme + host + port), e.g. `https://auth.example.com`.
- If you serve assets from a CDN, add hosts to `script-src` / `style-src` / `font-src` explicitly.
- Avoid `unsafe-inline` for scripts if you can adopt nonces or hashes (may require template changes).
- After changes, verify login, token refresh, and HTMX/API calls in browser devtools (CSP violations appear in the console).

---

## Rate limiting

The sample `deploy/nginx/nginx.conf` defines:

- **global** zone: `100r/s` with burst 200 on `/`.
- **api** zone: `50r/s` with burst 100 on `/api/`.

**Recommendations**:

- Lower rates for unauthenticated endpoints if you add any; keep authenticated API limits aligned with expected concurrent users.
- Consider **per-JWT** or **per-API-key** limits inside the application for expensive operations (not shown here).
- Use `limit_conn` for concurrent connections from a single IP if you see abuse patterns.

---

## TOTP enforcement in Keycloak

**Goal**: Add a second factor for privileged accounts.

**Steps (Keycloak admin console)**:

1. Authentication → **Policies** / **Required actions**: enable **Configure OTP**.
2. For sensitive groups (e.g. platform administrators), set **Required user actions** to include OTP setup on next login, or use an authentication flow with **OTP Form** as a required step.
3. Disable weaker alternatives for those flows (e.g. avoid password-only for admin accounts where policy requires MFA).

Coordinate with user onboarding: backup codes and device loss procedures should be documented for your organization.

---

## Database security

**Roles**:

- **`fresnel_app`**: `SELECT`, `INSERT`, `UPDATE`, `DELETE` on application tables in schema `fresnel` as required; **no** superuser.
- **`fresnel_readonly`**: reporting / BI — `SELECT` only on safe views.
- **Audit schema** (`fresnel_audit`): grant the application role **`INSERT` only** on `audit_entries`; **revoke `UPDATE` and `DELETE`** so tampering requires superuser access. Implement via explicit `GRANT`/`REVOKE` in a dedicated migration or DBA script.

**Hardening checklist**:

- Enforce `sslmode=require` (or verify-full with CA) for application connections.
- Rotate credentials; store them in a secret manager, not Compose files in production.
- Enable `log_connections` / `log_disconnections` and appropriate statement logging for investigations (balance with volume and PII).
- Regular vacuum/analyze and monitoring for bloat and slow queries.

---

## Network segmentation

**Recommendations**:

- Place **PostgreSQL** and **Keycloak** on a private network segment not reachable from the public Internet; only Fresnel (and admin bastions) may connect.
- Expose **only 443** (and 80→443 redirect if used) to users; Keycloak admin console should be VPN- or IP-restricted in production.
- If using ClamAV over TCP/unix socket, restrict socket permissions and firewall rules so only Fresnel can reach the scanner.
- Egress: allow SMTP only to approved relays; default-deny outbound where possible.

---

## Backup encryption

- Encrypt backup artifacts **at rest** (disk encryption, GPG, or object-store SSE-KMS).
- Encrypt **in transit** to backup storage (TLS, SCP over VPN).
- Store keys separately from ciphertext; test **restore** quarterly including decryption steps.
- For Keycloak, include realm exports in the same encryption envelope as database dumps.

---

## Security headers checklist (current nginx include)

From `deploy/nginx/security-headers.conf`:

| Header | Current value |
|--------|----------------|
| `Content-Security-Policy` | `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' http://localhost:8081; frame-ancestors 'none'; form-action 'self'; base-uri 'self'` |
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains; preload` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=(), payment=()` |
| `X-XSS-Protection` | `0` (deprecated header set to disable misleading UA behavior) |

Review annually and after any change to Keycloak URLs or static asset hosting.

---

## Incident response: break-glass access (Phase 2 — not implemented)

**Intent**: When normal IdP or admin paths are unavailable, a **controlled, audited** emergency access path limits downtime without standing shared passwords.

**Recommended pattern** (documentation only for this repository):

1. **Dual control**: two approvers for activating break-glass (e.g. security + operations).
2. **Time-bound credentials**: short-lived certificates or one-time tokens issued from an offline or separate vault.
3. **Network gate**: access only from a bastion with MFA and source IP allowlists.
4. **Mandatory logging**: every break-glass session creates high-severity audit entries (who, when, why, ticket id) in `fresnel_audit` and infrastructure logs.
5. **Post-incident**: rotate any exposed secrets, revoke temporary access, and run a blameless review.

Implementing this requires product and operations work (runbooks, vault integration, optional API endpoints). It is **not** present in the current Fresnel codebase; treat this section as a Phase 2 target.

---

## Related files

- `deploy/nginx/nginx.conf` — TLS, upstream, rate limits.
- `deploy/nginx/security-headers.conf` — CSP and security headers.
- `deploy/docker-compose.yml` — service topology and default environment.
- `deploy/keycloak/fresnel-realm.json` — realm and client baseline (replace dev secrets in production).
