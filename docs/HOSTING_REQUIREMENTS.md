# Fresnel — Hosting Requirements

**Purpose**: Specifications for provisioning the Fresnel platform infrastructure.
**Deployment model**: Single virtual machine running Docker Compose.
**Target hypervisor**: vSphere (VMware). Adaptable to any hypervisor or bare metal.

---

## 1. Virtual Machine Specification

| Resource | PoC | Production (future) |
|---|---|---|
| vCPU | 4 | 8 |
| RAM | 16 GB | 32 GB |
| OS disk | 50 GB SSD | 50 GB SSD |
| Data disk | 100 GB SSD | 500 GB SSD |
| GPU | None | Optional — 1× NVIDIA GPU with ≥8 GB VRAM for AI module (not required for core platform) |

**Storage notes**:
- The OS disk holds the operating system, Docker images, and application binaries.
- The data disk holds PostgreSQL data, file attachments, audit logs, and backups. Must be SSD-backed — database performance degrades significantly on spinning disk.
- Data disk should be a separate vDisk from the OS disk so it can be snapshotted and backed up independently.
- All disks must be provisioned as thick-provisioned eager-zeroed for consistent I/O performance.

---

## 2. Operating System

| Requirement | Value |
|---|---|
| OS | Ubuntu 24.04 LTS **or** AlmaLinux 9 |
| Kernel | 5.15+ (ships with both) |
| Full-disk encryption | LUKS on both OS and data disks |
| Docker | Docker Engine 27.x + Docker Compose v2 |
| Time sync | NTP enabled (chrony or systemd-timesyncd) |

No other software is required on the host. All application components run in Docker containers.

---

## 3. Network Requirements

### 3.1 Inbound

| Port | Protocol | Source | Purpose |
|---|---|---|---|
| 443 | TCP | Users / Internet | HTTPS — application access |

**No other inbound ports are required.** SSH access (port 22) should be restricted to a management network or VPN, not exposed to the internet.

### 3.2 Outbound

| Port | Protocol | Destination | Purpose |
|---|---|---|---|
| 25 or 587 | TCP | On-prem SMTP relay | Nudge/escalation emails, control plane notifications |
| 53 | UDP/TCP | DNS servers | Name resolution |
| 443 | TCP | ClamAV signature mirrors (database.clamav.net) | Virus definition updates |

If the platform is deployed in an air-gapped environment, ClamAV signature updates must be loaded manually via offline mirror.

### 3.3 DNS

The platform requires a DNS A record pointing to the VM's IP address. Example: `fresnel.example.org → 10.0.1.50`.

A valid TLS certificate for this hostname is required (see Section 5).

### 3.4 SMTP

The platform sends email for:
- Event update nudges (daily/weekly based on severity)
- Escalation notifications
- Control plane alerts (root reassignments, policy changes)

An on-premises SMTP relay must be reachable from the VM. The platform authenticates to the relay if required (SMTP AUTH with STARTTLS). No cloud email service dependency.

| Setting | Value |
|---|---|
| Relay host | Provided by infrastructure team |
| Relay port | 25 (unauthenticated) or 587 (STARTTLS + auth) |
| From address | e.g., `fresnel-noreply@example.org` |

---

## 4. Firewall Rules (Host-Level)

The VM runs `nftables` with a default-deny policy. Required rules:

```
# Inbound
allow tcp dport 443 from any              # HTTPS
allow tcp dport 22 from <mgmt-network>    # SSH (restricted)

# Outbound
allow tcp dport {25, 587} to <smtp-relay> # Email
allow udp dport 53 to <dns-servers>       # DNS
allow tcp dport 53 to <dns-servers>       # DNS (TCP fallback)
allow tcp dport 443 to <clamav-mirrors>   # ClamAV updates

# Drop everything else
```

---

## 5. TLS Certificate

| Requirement | Value |
|---|---|
| Type | X.509, RSA 2048+ or ECDSA P-256+ |
| Issued by | Organization CA or public CA (e.g., Let's Encrypt) |
| Subject | Must match the platform's DNS hostname |
| Format | PEM — separate cert file and private key file |
| Placement | Mounted into the nginx container at deployment time |

Self-signed certificates work for development but will cause browser warnings. For stakeholder-facing deployments, use a properly signed certificate.

---

## 6. Component Resource Breakdown

All components run as Docker containers on the single VM. Approximate resource consumption at PoC scale (~100 users, ~10k events):

| Container | CPU (steady) | CPU (peak) | RAM (steady) | RAM (peak) | Disk |
|---|---|---|---|---|---|
| PostgreSQL | 0.2 cores | 1.0 core | 512 MB | 2 GB | 5–50 GB (data volume, grows with usage) |
| Keycloak | 0.3 cores | 1.0 core | 768 MB | 1.5 GB | Negligible (config only) |
| Fresnel API | 0.1 cores | 0.5 core | 128 MB | 512 MB | Negligible (stateless binary) |
| nginx | 0.05 cores | 0.2 core | 64 MB | 128 MB | Negligible |
| ClamAV | 0.1 cores | 0.5 core | 1 GB | 1.5 GB | 500 MB (signature database) |
| **Total** | **~0.8 cores** | **~3.2 cores** | **~2.5 GB** | **~5.6 GB** | — |

The 4 vCPU / 16 GB specification provides comfortable headroom above peak usage. Keycloak and ClamAV are the largest memory consumers at idle.

**File attachments**: Stored on the data disk filesystem. Budget 25 MB × max attachments. At PoC scale with moderate attachment use, 5–10 GB.

**Backups**: Daily `pg_dump` output is typically 10–20% of live DB size. 30 days retention at PoC scale requires ~5–15 GB for backup storage.

---

## 7. Backup Requirements

| What | Method | Frequency | Retention | Storage |
|---|---|---|---|---|
| PostgreSQL database | `pg_dump` (full logical backup) | Daily, 02:00 local time | 30 days | Separate disk, NFS mount, or off-VM storage |
| Keycloak realm config | Keycloak realm export (JSON) | Daily, 02:30 local time | 30 days | Same as above |
| File attachments | Filesystem copy or rsync | Daily, 03:00 local time | 30 days | Same as above |
| TLS certificates | Manual — stored in config management | On change | Indefinite | Off-VM |

**RPO**: < 24 hours (worst case: lose one day of data).

**Restore procedure**: Documented in the deployment guide. Restore requires: fresh VM, Docker installed, backup files, TLS cert. Estimated restore time: 30–60 minutes.

Backup storage should **not** be on the same VM or the same datastore as the data disk. A VM failure should not destroy both live data and backups.

---

## 8. AI Module (Optional — Phase 2)

If the AI correlation module is activated in a future phase:

| Resource | Requirement |
|---|---|
| Additional RAM | +8 GB (for model loading) |
| GPU (optional) | NVIDIA with ≥8 GB VRAM (e.g., T4, A10, RTX 4060+) for real-time inference |
| GPU passthrough | vSphere GPU passthrough or vGPU configured |
| Disk | +10–20 GB for model weights |
| Network (outbound) | HTTPS to huggingface.co for initial model download (one-time, can be done offline) |

Without a GPU, the AI module operates in CPU-only batch mode (minutes per inference run instead of seconds). The core platform functions identically with or without the AI module.

---

## 9. Provisioning Checklist

- [ ] VM created with specified resources (4 vCPU, 16 GB RAM, 50 GB OS disk, 100 GB data disk)
- [ ] OS installed (Ubuntu 24.04 LTS or AlmaLinux 9)
- [ ] LUKS encryption enabled on both disks
- [ ] Docker Engine + Docker Compose installed
- [ ] NTP configured and synchronized
- [ ] DNS A record created for the platform hostname
- [ ] TLS certificate issued for the platform hostname
- [ ] SMTP relay reachable from the VM (test with `openssl s_client` or `swaks`)
- [ ] Firewall rules applied (nftables, default-deny, explicit allows per Section 4)
- [ ] SSH access restricted to management network
- [ ] Backup storage provisioned (separate from VM datastores)
- [ ] Data disk mounted at `/data` (or preferred mount point)
- [ ] Handoff to application team for Docker Compose deployment
