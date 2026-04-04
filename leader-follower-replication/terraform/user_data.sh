#!/usr/bin/env bash
# Runs on EC2 first boot.
# 1. Installs Docker + Compose plugin
# 2. Logs in to ECR
# 3. Writes docker-compose.prod.yml
# 4. Pulls image and starts all 20 containers

set -euo pipefail
exec > >(tee /var/log/user-data.log | logger -t user-data) 2>&1

ECR_REPO="${ecr_repo}"
REGION="${region}"
IMAGE="$ECR_REPO:latest"

# ─── Install Docker ────────────────────────────────────────────────────────────
dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

# Docker Compose v2 plugin
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64" \
     -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# ─── ECR login ────────────────────────────────────────────────────────────────
aws ecr get-login-password --region "$REGION" | \
  docker login --username AWS --password-stdin "$ECR_REPO"

# ─── Write docker-compose.prod.yml ────────────────────────────────────────────
mkdir -p /opt/kv-cluster
cat > /opt/kv-cluster/docker-compose.yml <<COMPOSE
version: "3.9"

x-node: &node
  image: $IMAGE
  healthcheck:
    test: ["CMD-SHELL", "wget -qO- http://localhost:5000/health || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 6
    start_period: 15s
  restart: unless-stopped
  networks:
    - kv-net

services:
  # ── LF W=5,R=1 (ports 8010-8014) ──────────────────────────────────────────
  lf1-leader:
    <<: *node
    ports: ["8010:5000"]
    environment:
      NODE_ID: "0"
      NODE_ROLE: leader
      WRITE_QUORUM: "5"
      READ_QUORUM: "1"
      PEER_URLS: "http://lf1-n1:5000,http://lf1-n2:5000,http://lf1-n3:5000,http://lf1-n4:5000"

  lf1-n1:
    <<: *node
    ports: ["8011:5000"]
    environment: {NODE_ID: "1", NODE_ROLE: follower, WRITE_QUORUM: "5", READ_QUORUM: "1", LEADER_URL: "http://lf1-leader:5000"}

  lf1-n2:
    <<: *node
    ports: ["8012:5000"]
    environment: {NODE_ID: "2", NODE_ROLE: follower, WRITE_QUORUM: "5", READ_QUORUM: "1", LEADER_URL: "http://lf1-leader:5000"}

  lf1-n3:
    <<: *node
    ports: ["8013:5000"]
    environment: {NODE_ID: "3", NODE_ROLE: follower, WRITE_QUORUM: "5", READ_QUORUM: "1", LEADER_URL: "http://lf1-leader:5000"}

  lf1-n4:
    <<: *node
    ports: ["8014:5000"]
    environment: {NODE_ID: "4", NODE_ROLE: follower, WRITE_QUORUM: "5", READ_QUORUM: "1", LEADER_URL: "http://lf1-leader:5000"}

  # ── LF W=1,R=5 (ports 8020-8024) ──────────────────────────────────────────
  lf2-leader:
    <<: *node
    ports: ["8020:5000"]
    environment:
      NODE_ID: "0"
      NODE_ROLE: leader
      WRITE_QUORUM: "1"
      READ_QUORUM: "5"
      PEER_URLS: "http://lf2-n1:5000,http://lf2-n2:5000,http://lf2-n3:5000,http://lf2-n4:5000"

  lf2-n1:
    <<: *node
    ports: ["8021:5000"]
    environment: {NODE_ID: "1", NODE_ROLE: follower, WRITE_QUORUM: "1", READ_QUORUM: "5", LEADER_URL: "http://lf2-leader:5000"}

  lf2-n2:
    <<: *node
    ports: ["8022:5000"]
    environment: {NODE_ID: "2", NODE_ROLE: follower, WRITE_QUORUM: "1", READ_QUORUM: "5", LEADER_URL: "http://lf2-leader:5000"}

  lf2-n3:
    <<: *node
    ports: ["8023:5000"]
    environment: {NODE_ID: "3", NODE_ROLE: follower, WRITE_QUORUM: "1", READ_QUORUM: "5", LEADER_URL: "http://lf2-leader:5000"}

  lf2-n4:
    <<: *node
    ports: ["8024:5000"]
    environment: {NODE_ID: "4", NODE_ROLE: follower, WRITE_QUORUM: "1", READ_QUORUM: "5", LEADER_URL: "http://lf2-leader:5000"}

  # ── LF W=3,R=3 (ports 8030-8034) ──────────────────────────────────────────
  lf3-leader:
    <<: *node
    ports: ["8030:5000"]
    environment:
      NODE_ID: "0"
      NODE_ROLE: leader
      WRITE_QUORUM: "3"
      READ_QUORUM: "3"
      PEER_URLS: "http://lf3-n1:5000,http://lf3-n2:5000,http://lf3-n3:5000,http://lf3-n4:5000"

  lf3-n1:
    <<: *node
    ports: ["8031:5000"]
    environment: {NODE_ID: "1", NODE_ROLE: follower, WRITE_QUORUM: "3", READ_QUORUM: "3", LEADER_URL: "http://lf3-leader:5000"}

  lf3-n2:
    <<: *node
    ports: ["8032:5000"]
    environment: {NODE_ID: "2", NODE_ROLE: follower, WRITE_QUORUM: "3", READ_QUORUM: "3", LEADER_URL: "http://lf3-leader:5000"}

  lf3-n3:
    <<: *node
    ports: ["8033:5000"]
    environment: {NODE_ID: "3", NODE_ROLE: follower, WRITE_QUORUM: "3", READ_QUORUM: "3", LEADER_URL: "http://lf3-leader:5000"}

  lf3-n4:
    <<: *node
    ports: ["8034:5000"]
    environment: {NODE_ID: "4", NODE_ROLE: follower, WRITE_QUORUM: "3", READ_QUORUM: "3", LEADER_URL: "http://lf3-leader:5000"}

  # ── Leaderless W=5,R=1 (ports 8040-8044) ──────────────────────────────────
  ll-n0:
    <<: *node
    ports: ["8040:5000"]
    environment:
      NODE_ID: "0"
      NODE_ROLE: leaderless
      WRITE_QUORUM: "5"
      READ_QUORUM: "1"
      PEER_URLS: "http://ll-n1:5000,http://ll-n2:5000,http://ll-n3:5000,http://ll-n4:5000"

  ll-n1:
    <<: *node
    ports: ["8041:5000"]
    environment: {NODE_ID: "1", NODE_ROLE: leaderless, WRITE_QUORUM: "5", READ_QUORUM: "1", PEER_URLS: "http://ll-n0:5000,http://ll-n2:5000,http://ll-n3:5000,http://ll-n4:5000"}

  ll-n2:
    <<: *node
    ports: ["8042:5000"]
    environment: {NODE_ID: "2", NODE_ROLE: leaderless, WRITE_QUORUM: "5", READ_QUORUM: "1", PEER_URLS: "http://ll-n0:5000,http://ll-n1:5000,http://ll-n3:5000,http://ll-n4:5000"}

  ll-n3:
    <<: *node
    ports: ["8043:5000"]
    environment: {NODE_ID: "3", NODE_ROLE: leaderless, WRITE_QUORUM: "5", READ_QUORUM: "1", PEER_URLS: "http://ll-n0:5000,http://ll-n1:5000,http://ll-n2:5000,http://ll-n4:5000"}

  ll-n4:
    <<: *node
    ports: ["8044:5000"]
    environment: {NODE_ID: "4", NODE_ROLE: leaderless, WRITE_QUORUM: "5", READ_QUORUM: "1", PEER_URLS: "http://ll-n0:5000,http://ll-n1:5000,http://ll-n2:5000,http://ll-n3:5000"}

networks:
  kv-net:
    driver: bridge
COMPOSE

# ─── Pull image and start ──────────────────────────────────────────────────────
cd /opt/kv-cluster
docker pull "$IMAGE"
docker compose up -d

echo "KV cluster started. Check: docker compose ps"
