# AWS to vSphere Migration Guide

**Scope**: Step-by-step procedure for migrating a running Fresnel instance from AWS (EC2 + RDS or EC2 + Docker Compose) to an on-premises vSphere VM.
**Prerequisite**: The vSphere VM is provisioned per `HOSTING_REQUIREMENTS.md`.

---

## Overview

The migration is a data transfer exercise, not an application rewrite. Fresnel's runtime contract is:

- A `DATABASE_URL` pointing at PostgreSQL
- A `KEYCLOAK_ISSUER` pointing at an OIDC provider
- A filesystem path for attachments
- TLS at the edge
- Environment variables for secrets

None of these are AWS-specific. The procedure is: export data, transfer securely, import on the new host, switch DNS.

---

## 1. Pre-migration preparation (days before cutover)

### 1.1 Provision the vSphere VM

Follow the checklist in `HOSTING_REQUIREMENTS.md`:

- [ ] VM created (4 vCPU, 16 GB RAM, 50 GB OS disk, 100 GB data disk)
- [ ] OS installed (Ubuntu 24.04 LTS or AlmaLinux 9)
- [ ] LUKS encryption enabled on both disks
- [ ] Docker Engine 27.x + Compose v2 installed
- [ ] NTP configured
- [ ] DNS A record prepared (can point to old IP until cutover)
- [ ] TLS certificate issued for the platform hostname
- [ ] SMTP relay reachable
- [ ] Firewall rules applied (nftables)
- [ ] Backup storage provisioned (separate from VM datastores)

### 1.2 Prepare Docker Compose on the target

Clone the Fresnel repository onto the vSphere VM. Copy the `deploy/` directory. Update `docker-compose.yml` environment variables:

| Variable | AWS value (example) | vSphere value |
|---|---|---|
| `DATABASE_URL` | `postgres://...@rds-host:5432/fresnel?sslmode=require` | `postgres://fresnel:$PASS@postgres:5432/fresnel?sslmode=disable` |
| `KEYCLOAK_ISSUER` | `https://kc.example.org/realms/fresnel` | `http://keycloak:8080/realms/fresnel` |
| `KEYCLOAK_EXTERNAL_URL` | `https://kc.example.org/realms/fresnel` | `https://fresnel.example.org/realms/fresnel` (or separate hostname) |
| `APP_PUBLIC_URL` | `https://fresnel.example.org` | `https://fresnel.example.org` (same if DNS is repointed) |

### 1.3 Lower DNS TTL

Set the DNS TTL for the platform hostname to **60 seconds**, at least 24 hours before the cutover. This ensures fast propagation when you switch the A record.

### 1.4 Generate a GPG key pair for encrypted transfers

All database exports and attachment archives are encrypted before leaving AWS. Generate a key pair on the vSphere VM (the destination):

```bash
gpg --batch --gen-key <<'EOF'
%no-protection
Key-Type: RSA
Key-Length: 4096
Name-Real: Fresnel Migration
Name-Email: migration@fresnel.local
Expire-Date: 30d
%commit
EOF

# Export the public key to transfer to the AWS host
gpg --export --armor migration@fresnel.local > /tmp/fresnel-migration.pub
```

Copy the public key to the AWS host (via SCP, SSM, or paste). Import it there:

```bash
gpg --import /tmp/fresnel-migration.pub
```

---

## 2. Initial data sync (optional, reduces cutover time)

If the database is large (> 1 GB) or attachments are numerous, do an initial sync days before cutover to reduce the final delta.

### 2.1 PostgreSQL full export (encrypted)

On the AWS host (or against RDS):

```bash
pg_dump -Fc -h $RDS_HOST -U fresnel -d fresnel \
  | gpg --encrypt --recipient migration@fresnel.local \
  > fresnel-initial.dump.gpg
```

Transfer to the vSphere VM:

```bash
scp fresnel-initial.dump.gpg user@vsphere-host:/data/migration/
```

On the vSphere VM, decrypt and restore into the local Postgres container:

```bash
gpg --decrypt fresnel-initial.dump.gpg \
  | docker exec -i $(docker compose ps -q postgres) \
    pg_restore -U fresnel -d fresnel --clean --if-exists
```

### 2.2 Attachments initial sync (encrypted archive)

On the AWS host:

```bash
tar czf - -C /path/to/attachments . \
  | gpg --encrypt --recipient migration@fresnel.local \
  > attachments-initial.tar.gz.gpg
```

Transfer and extract on the vSphere VM:

```bash
gpg --decrypt attachments-initial.tar.gz.gpg \
  | tar xzf - -C /data/fresnel-attachments/
```

---

## 3. Cutover procedure

### 3.1 Enable maintenance mode on AWS

Stop the Fresnel API container to prevent new writes. See the maintenance mode section below.

```bash
# On the AWS host:
docker compose stop fresnel
```

At this point, users see a maintenance page (served by nginx) or a connection error depending on your setup. Keycloak and Postgres remain running for the final export.

### 3.2 Final database export (encrypted)

This captures all data since the initial sync:

```bash
pg_dump -Fc -h $RDS_HOST -U fresnel -d fresnel \
  | gpg --encrypt --recipient migration@fresnel.local \
  > fresnel-final.dump.gpg
```

Transfer to vSphere and restore (same as 2.1, this is a full dump that replaces the initial one):

```bash
gpg --decrypt fresnel-final.dump.gpg \
  | docker exec -i $(docker compose ps -q postgres) \
    pg_restore -U fresnel -d fresnel --clean --if-exists
```

### 3.3 Final attachment delta (encrypted)

If few attachments were added since the initial sync, just re-sync the full directory. For a large volume, use rsync through an encrypted channel (SSH):

```bash
# From the AWS host, rsync to vSphere over SSH
rsync -az --progress /path/to/attachments/ user@vsphere-host:/data/fresnel-attachments/
```

The SSH channel provides encryption in transit. If the attachments must also be encrypted at rest on the intermediate transfer path, use the GPG archive method from section 2.2.

### 3.4 Export Keycloak state

**If Keycloak uses an external Postgres database** (recommended for anything beyond dev):

```bash
pg_dump -Fc -h $KC_DB_HOST -U keycloak -d keycloak \
  | gpg --encrypt --recipient migration@fresnel.local \
  > keycloak-final.dump.gpg
```

Restore on the vSphere Keycloak Postgres.

**If Keycloak uses the embedded H2 (dev mode)**:

Export the realm via the Admin REST API:

```bash
# Get an admin token
TOKEN=$(curl -s -X POST "https://kc.example.org/realms/master/protocol/openid-connect/token" \
  -d "client_id=admin-cli" \
  -d "username=admin" \
  -d "password=$KC_ADMIN_PASSWORD" \
  -d "grant_type=password" | jq -r .access_token)

# Export the fresnel realm (partial — does not include credentials)
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://kc.example.org/admin/realms/fresnel" > realm-export.json
```

Note: Keycloak's REST API realm export does **not** include user credentials. Users will need to reset their passwords, or you can use Keycloak's `kc.sh export` CLI against the running instance for a full export that includes credential hashes. For a PoC with few users, resetting passwords may be simpler.

On the vSphere VM, place the realm JSON in the Keycloak import directory and start Keycloak, or use `kc.sh import`.

### 3.5 Start the stack on vSphere

```bash
cd /opt/fresnel/deploy  # or wherever you placed the Compose files
docker compose up -d
```

Verify:
- Postgres is healthy: `docker compose exec postgres pg_isready`
- Keycloak is accessible: `curl -s http://localhost:8081/realms/fresnel`
- Fresnel API responds: `curl -s http://localhost:8080/api/v1/health`

### 3.6 Switch DNS

Update the DNS A record for the platform hostname to point at the vSphere VM's IP address.

```
fresnel.example.org → <vSphere-VM-IP>
```

With the 60-second TTL set earlier, propagation completes within 1–2 minutes.

### 3.7 Verify end-to-end

- [ ] Login works (Keycloak OIDC flow completes)
- [ ] Dashboard loads
- [ ] Events are visible with correct data
- [ ] Attachments download correctly
- [ ] New event creation works
- [ ] Audit log shows recent entries

---

## 4. Post-migration

### 4.1 Clean up AWS

Once verified and stable (give it a day or two):

- Terminate EC2 instances
- Delete RDS instances (take a final snapshot first, retain for 30 days)
- Remove EBS volumes
- Clean up security groups, ALB, and VPC resources
- Delete secrets from SSM/Secrets Manager
- Remove the GPG migration key pair from both hosts

### 4.2 Clean up migration artifacts

On the vSphere VM:

```bash
rm -f /data/migration/*.gpg
gpg --delete-secret-and-public-key migration@fresnel.local
```

### 4.3 Restore DNS TTL

Set the DNS TTL back to a normal value (e.g., 300–3600 seconds).

### 4.4 Verify backups

Ensure the on-prem backup schedule is running per `HOSTING_REQUIREMENTS.md`:

- [ ] `pg_dump` cron job active (daily at 02:00)
- [ ] Keycloak realm export cron job active (daily at 02:30)
- [ ] Attachment rsync/copy cron job active (daily at 03:00)
- [ ] Backup storage is on a separate datastore from the VM

---

## 5. Estimated timeline

| Phase | Duration | User impact |
|---|---|---|
| Pre-migration prep (1.x) | 1–2 days | None |
| Initial data sync (2.x) | Hours (background) | None |
| Cutover (3.1–3.6) | 15–30 minutes | Platform unavailable |
| Verification (3.7) | 15 minutes | Platform available, under observation |
| AWS decommission (4.1) | Next day | None |

Total user-facing downtime: **15–30 minutes**.

---

## 6. Rollback

If the vSphere deployment fails verification:

1. Switch DNS back to the AWS host's IP.
2. Restart the Fresnel API on AWS: `docker compose start fresnel`.
3. Service restores on AWS within 1–2 minutes (DNS TTL).
4. Investigate and retry migration later.

No data is lost in a rollback — the AWS instance was only stopped, not destroyed. Any writes that happened on the vSphere instance after DNS switch and before rollback would need to be manually reconciled (unlikely in a 15-minute window).
