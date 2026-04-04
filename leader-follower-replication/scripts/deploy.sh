#!/usr/bin/env bash
# Build the Go Docker image, push to ECR, and optionally restart the cluster.
#
# Prerequisites:
#   - AWS CLI configured (or running in AWS Academy session)
#   - Docker running
#   - terraform apply already done (to create ECR repo + EC2)
#
# Usage:
#   ./scripts/deploy.sh                         # build + push only
#   ./scripts/deploy.sh --restart <ec2-ip>      # build + push + restart on EC2
#
# The ECR_REPO is read from terraform output automatically.

set -euo pipefail

REGION="${AWS_REGION:-us-west-2}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR/.."

# ─── Resolve ECR repo URL from Terraform output ───────────────────────────────
cd "$ROOT/terraform"
ECR_REPO=$(terraform output -raw ecr_repository_url 2>/dev/null)
if [[ -z "$ECR_REPO" ]]; then
  echo "ERROR: could not read ecr_repository_url from terraform output."
  echo "       Run: cd terraform && terraform apply"
  exit 1
fi
echo "ECR repo: $ECR_REPO"

# ─── Build ────────────────────────────────────────────────────────────────────
cd "$ROOT"
echo "Building Docker image..."
docker build \
  --platform linux/amd64 \
  -f node/Dockerfile \
  -t kv-node:latest \
  .

# ─── Tag + push ───────────────────────────────────────────────────────────────
echo "Logging in to ECR..."
aws ecr get-login-password --region "$REGION" | \
  docker login --username AWS --password-stdin "$ECR_REPO"

docker tag kv-node:latest "$ECR_REPO:latest"
echo "Pushing image to ECR..."
docker push "$ECR_REPO:latest"
echo "Push complete: $ECR_REPO:latest"

# ─── Optional: restart cluster on EC2 ────────────────────────────────────────
if [[ "${1:-}" == "--restart" && -n "${2:-}" ]]; then
  EC2_IP="$2"
  KEY="${SSH_KEY:-~/.ssh/id_rsa}"
  echo "Restarting cluster on $EC2_IP ..."
  ssh -i "$KEY" -o StrictHostKeyChecking=no "ec2-user@$EC2_IP" \
    "aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ECR_REPO && \
     cd /opt/kv-cluster && \
     docker compose pull && \
     docker compose up -d --remove-orphans"
  echo "Cluster restarted. Check: ssh ec2-user@$EC2_IP 'docker compose -f /opt/kv-cluster/docker-compose.yml ps'"
fi
