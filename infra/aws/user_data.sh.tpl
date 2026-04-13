#!/usr/bin/env bash
set -euo pipefail

# ── Fresnel EC2 bootstrap (runs once on first launch) ──
# Installs Docker + SSM agent, creates directory structure for the app and data.
# The data volume is NOT formatted here — the operator must LUKS-format and
# unlock it manually before starting the stack (see OPERATIONS_GUIDE.md).

export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get upgrade -y

# SSM agent (required for Session Manager access)
snap install amazon-ssm-agent --classic
systemctl enable --now snap.amazon-ssm-agent.amazon-ssm-agent.service

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

# Prepare directory for Fresnel repo (operator clones manually — see OPERATIONS_GUIDE.md)
mkdir -p /opt/${project}

# Data volume mount point
mkdir -p /data

# Identify the data volume device.
# Nitro instances (t3, m5, c5, etc.) expose EBS as NVMe: /dev/nvme1n1
# Older instance types use /dev/xvdf.
DATA_DEV=""
if [ -b /dev/nvme1n1 ]; then
  DATA_DEV=/dev/nvme1n1
elif [ -b /dev/xvdf ]; then
  DATA_DEV=/dev/xvdf
fi

cat <<MSG
══════════════════════════════════════════════════════════════
 Fresnel EC2 bootstrap complete.
 Data volume device: $${DATA_DEV:-NOT DETECTED — check lsblk}

 Next steps (manual — see OPERATIONS_GUIDE.md):
   1. SSM into the instance
   2. Clone repo:   sudo git clone <url> /opt/fresnel
   3. LUKS-format:  sudo cryptsetup luksFormat $${DATA_DEV}
   4. Open & mount: sudo cryptsetup luksOpen $${DATA_DEV} fresnel-data
                    sudo mkfs.ext4 /dev/mapper/fresnel-data
                    sudo mount /dev/mapper/fresnel-data /data
   5. Create dirs:  sudo mkdir -p /data/{pgdata,attachments,backups}
   6. Configure:    cp /opt/fresnel/.env.example /opt/fresnel/.env
   7. Start:        cd /opt/fresnel && ./scripts/deploy.sh
══════════════════════════════════════════════════════════════
MSG
