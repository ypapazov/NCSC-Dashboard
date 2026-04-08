# Cloud deployment guide (MVP framing)

This document describes how to run Fresnel **outside your laptop** in a **minimal, replaceable** way. **AWS is used only as a convenient first landing zone** for an MVP or pilot. The architecture is deliberately **not** “an AWS product”: avoid designs that only make sense on a single vendor’s managed PaaS if you expect to move to another region, another cloud, or on‑premises later.

**Goals**

- Get TLS, networking, secrets, backups, and observability to a **good-enough** bar for a controlled deployment.
- Keep application components **portable**: PostgreSQL, Keycloak, the Go API, nginx (or equivalent), and ClamAV are the same containers or binaries everywhere.
- Treat **infrastructure-as-code** (e.g. Terraform) as **bootstrap and documentation**, not as the long-term identity of the product.

**Non-goals**

- Building a “proper” AWS-native platform (EKS-heavy service mesh, vendor-specific auth beyond standard OIDC, etc.) unless you explicitly choose that later.
- Optimizing cost and HA to production final form; this is a **starting point**.

---

## 1. Local Docker Compose (what you have today)

### 1.1 What `make compose-up` does

From the repository root, `make compose-up` typically:

1. Generates **dev TLS** certificates under `deploy/nginx/certs/` (self-signed) if missing.
2. Starts **Docker Compose** (`deploy/docker-compose.yml`): PostgreSQL (with pgvector), Keycloak, ClamAV (optional / platform-specific), the **Fresnel API** image, and **nginx** on `https://localhost`.

The **Fresnel** container runs database **migrations on startup** (see application `main`): you do **not** need a separate `make migrate` step for the default Compose path, as long as the API container starts successfully and can reach Postgres.

### 1.2 Is “just register on Keycloak” correct?

**No open self-registration** in the shipped realm: `fresnel-realm.json` sets **`registrationAllowed: false`**. Users are **not** meant to sign up themselves on the login page.

You log in with **accounts that already exist**:

- A **pre-imported** realm user `platform-admin` (email `admin@fresnel.local`, password set in `fresnel-realm.json` — **change it** after first login).
- Optionally, identities you create in the **Keycloak Admin Console** (master admin credentials from `KC_BOOTSTRAP_ADMIN_*` in Compose let you open `http://localhost:8081` and manage the `fresnel` realm).

The **Fresnel** app then links the OIDC `sub` to rows in PostgreSQL (and can link by **email** on first login if a matching `fresnel.users` row exists). Migrations `007` / `008` seed dev and platform-admin rows for that flow.

### 1.3 Two different “admins”

| Credential | Purpose |
|------------|---------|
| Keycloak **bootstrap** admin (`KC_BOOTSTRAP_ADMIN_*`) | Administer Keycloak itself (all realms). |
| Realm user **`platform-admin`** | End-user login to **Fresnel** via OIDC (plus DB row from migration `008`). |

---

## 2. Why AWS might be the first cloud

Common reasons: existing org accounts, billing, IAM familiarity, and quick access to VMs, load balancers, and managed PostgreSQL. **None of these require** adopting AWS-specific application runtimes. You can run the **same** containers or packages you use locally.

---

## 3. AWS MVP building blocks (portable pattern)

Think in layers. Swap **AWS** for “another VPC + VM + managed DB” with minimal application change.

### 3.1 Network

- **VPC** with public subnets (for an internet-facing load balancer) and **private** subnets (for databases and internal services) if you want a minimal private tier.
- **Security groups** as coarse firewalls: only ALB → nginx/API, API → Postgres/Keycloak, operator access via SSM Session Manager or bastion (avoid SSH on `0.0.0.0/0`).

### 3.2 Compute (pick one MVP style)

| Option | Pros | Cons |
|--------|------|------|
| **EC2** (one or few instances) + Docker Compose / systemd | Fastest path, mirrors dev, easy escape hatch | You operate patching and scaling |
| **ECS Fargate** + task definitions | No instance SSH, AWS schedules containers | More moving parts for a small team |
| **EKS** | Powerful if you already run Kubernetes | Usually overkill for first Fresnel pilot |

For an **MVP**, **EC2 + Compose** or a **single small ASG** is often enough; you can move to Kubernetes later **without** changing the Fresnel codebase if the app stays stateless and talks to Postgres/Keycloak over the network.

### 3.3 PostgreSQL

- **Amazon RDS for PostgreSQL** (or **Aurora PostgreSQL**) in **private subnets**, automated backups, encryption at rest, restricted SGs.
- Alternatively, **PostgreSQL on the same host as Compose** for the smallest pilot (not ideal for production; acceptable for a short internal demo if backups exist).

The Fresnel app only needs a **`DATABASE_URL`**; it does not care whether RDS, Aurora, or a VM runs Postgres.

### 3.4 Identity (Keycloak)

- Run Keycloak as a **container** on EC2 or as another service task; use a **hostname** and TLS.
- Point Fresnel’s **`KEYCLOAK_ISSUER`** at `https://<your-host>/realms/fresnel` (or your realm name).
- Realm JSON and secrets should be **managed** (see §5), not hand-edited on servers forever.

### 3.5 Edge and TLS

- **Application Load Balancer** (or NLB if you prefer L4) terminating TLS with **ACM** certificates.
- nginx (or ALB alone) forwards to the Fresnel API. Keep **security headers** and rate limits at the edge as in `deploy/nginx/`.

### 3.6 Attachments and ClamAV

- **EBS** or **EFS** (or S3 with app changes later) for attachment storage; ClamAV as a sidecar or separate task with a **private** socket/network path to the API.

### 3.7 Observability (minimal)

- **CloudWatch Logs** agent or Docker logging driver → central logs; **metrics** on ALB + RDS; **alarms** on 5xx and DB CPU. Avoid building a bespoke observability platform for the MVP.

---

## 4. Terraform: use it as disposable bootstrap

Terraform (or OpenTofu) is useful to **encode** VPC, RDS, ALB, SGs, and EC2 user-data that installs Docker and checks out a known **version** of Compose files.

**Principles**

- **State** in a remote backend (e.g. S3 + DynamoDB lock); **never** commit raw `terraform.tfstate` with secrets.
- **Modules** can wrap VPC + RDS + EC2, but the **application** remains defined in this repo (Compose/K8s manifests), not hidden inside provider-only constructs.
- It is OK to **throw away** the first Terraform tree and rewrite it when you move to another region or cloud: the **source of truth** for Fresnel remains this repository, not the TF module names.

This is **not** “Terraform is the product”; it is “Terraform bootstraps disposable infrastructure.”

---

## 5. Secrets and configuration

- **Never** bake production secrets into images. Use **SSM Parameter Store**, **Secrets Manager**, or another secret store; inject at runtime as env vars (`HMAC_SECRET`, `KEYCLOAK_CLIENT_SECRET`, DB password, etc.).
- **Keycloak** client secrets and realm imports should be **templated** or loaded from secure storage, not copied from dev defaults.
- **Rotate** the `platform-admin` password and all dev defaults before any shared environment goes live.

---

## 6. What to avoid locking in early

- **Vendor-specific application glue** that only deploys on one PaaS (unless you accept that lock-in).
- **Hard-coded** AWS endpoints inside the Go app (use env vars and standard URLs).
- **Treating** one region’s Terraform modules as the permanent “official” way to run Fresnel; document instead the **interfaces** (Postgres URL, OIDC issuer, attachment path, ClamAV address).

---

## 7. Path off AWS (on‑prem or another cloud)

The portable unit is:

1. **Linux hosts** (or K8s) + **containers** or binaries.
2. **PostgreSQL** reachable on the network.
3. **Keycloak** (or any OIDC provider) with the same client settings.
4. **TLS** at the edge and **secrets** in a vault-like store.

Replace RDS with a **managed Postgres** elsewhere or a **patroni** cluster; replace ALB with **another** L7 load balancer; keep Fresnel’s **environment contract** stable.

---

## 8. Checklist before calling an AWS MVP “live”

- [ ] TLS everywhere user-facing; no dev self-signed certs.
- [ ] Postgres backups tested restore (RDS snapshot or logical dump).
- [ ] Keycloak admin and realm credentials **not** default dev values.
- [ ] `HMAC_SECRET` and OIDC client secret rotated from Compose defaults.
- [ ] nginx or ALB: rate limits and security headers verified (see requirements doc).
- [ ] Incident path: who can SSH/SSM, who has Keycloak admin, where logs go.

---

## 9. Summary

- **Local:** `make compose-up` brings the stack up; migrations run with the API; **registration is closed** in Keycloak; use **pre-created** users (e.g. `platform-admin`) or admin-created users.
- **AWS MVP:** use **standard** building blocks (VPC, ALB, RDS, EC2/ECS) **without** tying the application to AWS-only services in code.
- **Terraform:** optional **bootstrap**; keep the app portable and treat cloud choice as **replaceable**.

When you outgrow this document, split “runbook” (ops) from “architecture” (`ARCHITECTURE.md`) and keep this file as the **early cloud positioning** narrative, not the final operations manual.
