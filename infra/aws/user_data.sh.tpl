#!/usr/bin/env bash
set -euo pipefail

# ── Fresnel EC2 bootstrap (runs once on first launch) ──
# Installs Docker, clones the repo, and prepares the data volume mount point.
# The data volume is NOT formatted here — the operator must LUKS-format and
# unlock it manually before starting the stack (see OPERATIONS_GUIDE.md).

export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get upgrade -y

# Docker Engine (official repo)
apt-get install -y ca-certificates curl gnupg cryptsetup ntp
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

systemctl enable --now docker

# Fresnel repository
mkdir -p /opt/${project}
git clone https://github.com/CHANGEME/${project}.git /opt/${project} || true

# Data volume mount point
mkdir -p /data

cat <<'MSG'
══════════════════════════════════════════════════════════════
 Fresnel EC2 bootstrap complete.
 Next steps (manual — see OPERATIONS_GUIDE.md):
   1. SSM into the instance
   2. LUKS-format /dev/xvdf:  cryptsetup luksFormat /dev/xvdf
   3. Open and mount:         cryptsetup luksOpen /dev/xvdf fresnel-data
                              mkfs.ext4 /dev/mapper/fresnel-data
                              mount /dev/mapper/fresnel-data /data
   4. Create data dirs:       mkdir -p /data/{pgdata,attachments,backups}
   5. Configure env:          cp /opt/fresnel/.env.example /opt/fresnel/.env
                              # edit .env with production values
   6. Start the stack:        cd /opt/fresnel && scripts/deploy.sh
══════════════════════════════════════════════════════════════
MSG
