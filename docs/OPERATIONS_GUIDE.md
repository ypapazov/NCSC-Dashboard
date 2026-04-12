# Fresnel Operations Guide

**Scope**: How to provision, deploy, upgrade, back up, restore, and maintain a Fresnel instance on AWS (EC2 + Docker Compose). The same procedures apply on vSphere with minor path adjustments — see `AWS_TO_VSPHERE_MIGRATION.md` for the migration path.

**Related documents**:
- `HOSTING_REQUIREMENTS.md` — VM specs, network, firewall, TLS, backups
- `CLOUD_DEPLOYMENT_GUIDE.md` — architectural rationale, encryption at rest
- `ZERO_DOWNTIME_DEPLOYS.md` — migration discipline, Keycloak change management
- `AWS_TO_VSPHERE_MIGRATION.md` — moving off AWS to on-prem

---

## 1. Infrastructure provisioning (AWS)

### 1.1 Prerequisites

- AWS CLI configured with credentials that can create VPC, EC2, EBS, ALB, ACM, IAM, and Route 53 resources.
- Terraform >= 1.5 installed.
- A registered domain name and (optionally) a Route 53 hosted zone for it.

### 1.2 Terraform apply

```bash
cd infra/aws
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars: set domain_name, route53_zone_id, etc.

terraform init
terraform plan
terraform apply
```

This creates:
- VPC with public/private subnets, NAT gateway
- EC2 instance (Ubuntu 24.04, t3.xlarge) in a private subnet with SSM access
- Separate 100 GB EBS data volume (AWS-encrypted, attached at `/dev/xvdf`)
- ALB with ACM TLS certificate, HTTP→HTTPS redirect
- Security groups: ALB accepts 443/80 from internet; app accepts 8080 from ALB only
- Route 53 A record (if zone ID provided)

### 1.3 First-time instance setup

After `terraform apply`, connect via SSM:

```bash
# Get the instance ID from Terraform output
aws ssm start-session --target $(terraform output -raw instance_id)
```

The user-data script has already installed Docker and cloned the repository to `/opt/fresnel`. Now complete the manual steps:

**a) LUKS-encrypt the data volume:**

```bash
sudo cryptsetup luksFormat /dev/xvdf
# Enter a strong passphrase. Store it OUTSIDE AWS (see CLOUD_DEPLOYMENT_GUIDE.md §5).

sudo cryptsetup luksOpen /dev/xvdf fresnel-data
sudo mkfs.ext4 /dev/mapper/fresnel-data
sudo mount /dev/mapper/fresnel-data /data
```

**b) Create data directories:**

```bash
sudo mkdir -p /data/{pgdata,attachments,backups}
sudo chown -R 1000:1000 /data  # Docker-accessible
```

**c) Configure environment:**

```bash
cd /opt/fresnel
cp .env.example .env
# Edit .env: set POSTGRES_PASSWORD, KC_ADMIN_PASSWORD, HMAC_SECRET,
# APP_PUBLIC_URL, KEYCLOAK_EXTERNAL_URL, SMTP settings, backup GPG recipient.
chmod 600 .env
```

Generate strong secrets:

```bash
# HMAC_SECRET (64 hex chars)
openssl rand -hex 32

# Database / Keycloak passwords
openssl rand -base64 24
```

**d) Install TLS certificates (if not using ALB termination):**

If running nginx behind ALB (ALB terminates TLS), you can use self-signed certs between ALB and nginx, or skip nginx TLS entirely and run HTTP internally. If running nginx as the edge (no ALB), place real certs:

```bash
cp /path/to/cert.pem deploy/nginx/certs/server.crt
cp /path/to/key.pem deploy/nginx/certs/server.key
```

**e) Start the stack:**

```bash
./scripts/deploy.sh --skip-backup  # No data to back up yet
```

Or manually:

```bash
cd deploy
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

**f) Verify:**

```bash
curl -s http://localhost:8080/api/v1/health | python3 -m json.tool
# Should return healthy status for db and keycloak
```

Log in via `https://fresnel.example.org` (or your domain) with the `platform-root` user from the Keycloak realm import, then immediately change the password.

---

## 2. Deploying updates

### 2.1 Standard deploy (application update)

From the production host:

```bash
cd /opt/fresnel
./scripts/deploy.sh
```

This script:
1. `git pull --ff-only` to get the latest code
2. Runs `scripts/backup.sh` (pre-deploy backup)
3. Builds the new Fresnel Docker image
4. Runs database migrations (as a one-shot container, before restart)
5. Restarts the `fresnel` service container
6. Waits for the health check to pass

**Expected downtime**: 2–10 seconds (container swap time). Migrations run before the restart, so the new container starts serving immediately.

To skip the pre-deploy backup (e.g., if you just ran one):

```bash
./scripts/deploy.sh --skip-backup
```

### 2.2 Full stack update (Postgres, Keycloak, nginx version bumps)

When you need to update base images (not just the Fresnel app):

```bash
cd /opt/fresnel

# 1. Backup first
./scripts/backup.sh --label pre-upgrade

# 2. Enable maintenance mode
./scripts/maintenance.sh on

# 3. Pull new images and restart everything
cd deploy
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 4. Wait and verify
sleep 10
curl -s http://localhost:8080/api/v1/health

# 5. Disable maintenance mode
cd /opt/fresnel
./scripts/maintenance.sh off
```

**Expected downtime**: 1–3 minutes (Keycloak startup is the bottleneck).

### 2.3 Rolling back

If the new version is broken:

```bash
# Check the git log for the previous commit
git log --oneline -5

# Reset to the previous version
git checkout <previous-commit>

# Rebuild and restart (skip backup — you just made one in deploy.sh)
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.prod.yml build fresnel
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.prod.yml up -d --no-deps fresnel
```

If the database migration is backwards-incompatible (which should not happen if you follow `ZERO_DOWNTIME_DEPLOYS.md`), restore from the pre-deploy backup:

```bash
./scripts/restore.sh /data/backups/<latest-pre-deploy-backup>
```

---

## 3. Backups

### 3.1 What gets backed up

| Component | Method | Location |
|-----------|--------|----------|
| PostgreSQL | `pg_dump -Fc` (custom format) | `/data/backups/<timestamp>/fresnel.dump[.gpg]` |
| Keycloak realm | REST API export (or JSON fallback) | `/data/backups/<timestamp>/keycloak-realm.json[.gpg]` |
| File attachments | `tar` archive | `/data/backups/<timestamp>/attachments.tar.gz[.gpg]` |

### 3.2 Running a backup manually

```bash
./scripts/backup.sh
```

With a label:

```bash
./scripts/backup.sh --label pre-migration
```

### 3.3 Encrypted backups

Set `FRESNEL_BACKUP_GPG_RECIPIENT` in `.env` (or export it) to a GPG key ID or email. The backup script encrypts every output file with that key.

To set up GPG encryption:

```bash
# Generate a key (or import an existing one)
gpg --batch --gen-key <<'EOF'
%no-protection
Key-Type: RSA
Key-Length: 4096
Name-Real: Fresnel Backup
Name-Email: backup@fresnel.local
Expire-Date: 1y
%commit
EOF

# Set the recipient in .env
echo 'FRESNEL_BACKUP_GPG_RECIPIENT=backup@fresnel.local' >> .env

# IMPORTANT: Export and store the private key somewhere safe (off this VM).
gpg --export-secret-keys --armor backup@fresnel.local > /safe/location/fresnel-backup.key
```

### 3.4 Automated daily backups (cron)

```bash
sudo crontab -e
```

Add:

```cron
# Fresnel daily backup at 02:00
0 2 * * * cd /opt/fresnel && ./scripts/backup.sh --label daily >> /var/log/fresnel-backup.log 2>&1
```

### 3.5 Backup retention

Old backups are automatically pruned after `FRESNEL_BACKUP_RETAIN_DAYS` (default: 30 days).

### 3.6 Off-host backup storage

The `/data/backups` directory is on the same LUKS volume as the database. For disaster recovery, copy backups off the host:

```bash
# Example: rsync to a NAS or backup server
rsync -az /data/backups/ backup-server:/backups/fresnel/

# Example: upload to S3 (if backup files are GPG-encrypted, the S3 copy is safe)
aws s3 sync /data/backups/ s3://fresnel-backups-CHANGEME/ --exclude "*.dump" --exclude "*.tar.gz"
```

---

## 4. Restoring from backup

### 4.1 Full restore

```bash
# List available backups
ls -lt /data/backups/

# Restore from a specific backup
./scripts/restore.sh /data/backups/20260410-020000-daily
```

The restore script:
1. Stops the Fresnel API (prevents writes)
2. Decrypts GPG-encrypted files (if applicable)
3. Restores the PostgreSQL dump (`pg_restore --clean --if-exists`)
4. Restores attachment files
5. Restarts the full stack
6. Waits for the health check

### 4.2 Restoring to a fresh host

On a new VM (provisioned per `HOSTING_REQUIREMENTS.md` or via Terraform):

```bash
# 1. Complete the first-time setup (section 1.3 a–d)
# 2. Copy the backup directory to the new host
scp -r old-host:/data/backups/latest /data/backups/latest
# 3. Restore
./scripts/restore.sh /data/backups/latest
```

### 4.3 Database-only restore (no attachment change)

```bash
# Decrypt if needed
gpg --decrypt /data/backups/20260410-020000/fresnel.dump.gpg > /tmp/fresnel.dump

# Stop the API
docker compose -f deploy/docker-compose.yml stop fresnel

# Restore
docker compose -f deploy/docker-compose.yml exec -T postgres \
  pg_restore -U fresnel -d fresnel --clean --if-exists < /tmp/fresnel.dump

# Restart
docker compose -f deploy/docker-compose.yml up -d
rm /tmp/fresnel.dump
```

---

## 5. Maintenance mode

### 5.1 Enabling / disabling

```bash
./scripts/maintenance.sh on      # Users see "under maintenance" page
./scripts/maintenance.sh off     # Normal operation resumes
./scripts/maintenance.sh status  # Check current state
```

Maintenance mode works via an nginx flag file. When enabled:
- All user-facing routes return a styled 503 page.
- The `/api/v1/health` endpoint remains accessible (for monitoring and deploy scripts).
- The Fresnel API container can be stopped, restarted, or upgraded without users seeing raw errors.

### 5.2 When to use maintenance mode

- Before full stack upgrades (section 2.2)
- Before database restores (section 4)
- During planned migration (see `AWS_TO_VSPHERE_MIGRATION.md`)
- **Not needed** for standard app-only deploys — the 2–10 second restart is short enough that users see at most a brief connection reset.

---

## 6. LUKS volume management

### 6.1 After a reboot

The data volume does not auto-mount. After a reboot:

```bash
sudo cryptsetup luksOpen /dev/xvdf fresnel-data
sudo mount /dev/mapper/fresnel-data /data
cd /opt/fresnel/deploy
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### 6.2 Automating unlock (optional)

For automated recovery without operator intervention, the LUKS passphrase must be retrievable from a non-AWS source at boot time. Options:

- **Clevis + Tang**: Network-bound disk encryption. A Tang server on your corporate network provides the unlock key. If the VM can reach the Tang server, it auto-unlocks. If network is unreachable, manual unlock is required. This is the recommended approach for higher availability.
- **HashiCorp Vault (transit unseal)**: A systemd unit fetches the passphrase from Vault over VPN at boot.
- **Manual SSM session**: Acceptable for a PoC with rare reboots.

### 6.3 Expanding the data volume

```bash
# 1. Resize the EBS volume in AWS console or Terraform (increase data_volume_size_gb)
# 2. On the instance:
sudo cryptsetup resize fresnel-data
sudo resize2fs /dev/mapper/fresnel-data
# No downtime needed — this works on a mounted filesystem.
```

---

## 7. Monitoring

### 7.1 Health endpoint

```bash
curl -s https://fresnel.example.org/api/v1/health
```

Returns JSON with database and Keycloak connectivity status.

### 7.2 Logs

```bash
# All containers
docker compose -f deploy/docker-compose.yml logs -f

# Just the API
docker compose -f deploy/docker-compose.yml logs -f fresnel

# Just Postgres
docker compose -f deploy/docker-compose.yml logs -f postgres
```

The Fresnel API emits structured JSON logs (slog). Pipe to `jq` for readability:

```bash
docker compose -f deploy/docker-compose.yml logs fresnel --no-log-prefix | jq .
```

### 7.3 Disk usage

```bash
df -h /data
du -sh /data/pgdata /data/attachments /data/backups
```

### 7.4 CloudWatch (if on AWS)

The Docker logging driver can forward to CloudWatch Logs. Add to `docker-compose.prod.yml`:

```yaml
services:
  fresnel:
    logging:
      driver: awslogs
      options:
        awslogs-region: eu-west-2
        awslogs-group: /fresnel/api
        awslogs-create-group: "true"
```

---

## 8. Compose file usage

The deployment uses Docker Compose file layering:

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Base service definitions (images, ports, health checks) |
| `docker-compose.dev.yml` | Dev overrides (exposed debug ports) |
| `docker-compose.prod.yml` | Production overrides (LUKS /data paths, secret injection from .env) |

**Development** (local laptop):

```bash
docker compose -f docker-compose.yml up --build -d
# Or: make compose-up (generates certs + runs the above)
```

**Production** (AWS or vSphere):

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

The `scripts/deploy.sh` script uses the production composition automatically when `/data` exists as a mount point. Otherwise it falls back to the base compose file.

---

## 9. Quick reference

| Task | Command |
|------|---------|
| Deploy app update | `./scripts/deploy.sh` |
| Full stack upgrade | `./scripts/maintenance.sh on && docker compose ... pull && up -d && ./scripts/maintenance.sh off` |
| Manual backup | `./scripts/backup.sh` |
| Restore from backup | `./scripts/restore.sh /data/backups/<dir>` |
| Enter maintenance | `./scripts/maintenance.sh on` |
| Exit maintenance | `./scripts/maintenance.sh off` |
| View logs | `docker compose -f deploy/docker-compose.yml logs -f fresnel` |
| Health check | `curl -s http://localhost:8080/api/v1/health` |
| Unlock LUKS after reboot | `sudo cryptsetup luksOpen /dev/xvdf fresnel-data && sudo mount /dev/mapper/fresnel-data /data` |
| Connect via SSM | `aws ssm start-session --target <instance-id>` |
