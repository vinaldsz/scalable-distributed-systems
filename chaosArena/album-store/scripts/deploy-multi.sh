#!/bin/bash
set -e

# Usage: ./scripts/deploy-multi.sh <key-path> <ec2-1-ip> <ec2-2-ip> <ec2-1-private-ip>
# EC2-1 runs PostgreSQL + App, EC2-2 runs App only (connects to EC2-1's PostgreSQL)

KEY=$1
EC2_1_IP=$2
EC2_2_IP=$3
DB_PRIVATE_IP=$4

if [ -z "$KEY" ] || [ -z "$EC2_1_IP" ] || [ -z "$EC2_2_IP" ] || [ -z "$DB_PRIVATE_IP" ]; then
  echo "Usage: ./scripts/deploy-multi.sh <key-path> <ec2-1-ip> <ec2-2-ip> <ec2-1-private-ip>"
  exit 1
fi

SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"

# ─── Build binary once ───
echo "→ Building binary for Linux..."
GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -o bin/server-linux ./cmd/server

# ─── Generate per-instance .env files ───
generate_env() {
  local DB_HOST=$1
  cat <<ENVEOF
APP_PORT=8080
DATABASE_URL=postgres://albumuser:secret@${DB_HOST}:5432/albumstore?sslmode=disable
DB_MAX_CONNS=25
DB_MIN_CONNS=3
AWS_REGION=us-west-2
S3_BUCKET=album-store-photos-chaos
WORKER_COUNT=100
WORKER_QUEUE_CAP=1000
MAX_CONCURRENT_UPLOADS=25
MAX_UPLOAD_MB=200
ENVEOF
}

# EC2-1 uses localhost for DB
generate_env "localhost" > /tmp/env-ec2-1
# EC2-2 uses EC2-1's private IP for DB
generate_env "$DB_PRIVATE_IP" > /tmp/env-ec2-2

# ─── Deploy function ───
deploy_instance() {
  local IP=$1
  local ENV_FILE=$2
  local LABEL=$3

  echo ""
  echo "═══ Deploying to $LABEL ($IP) ═══"

  echo "  → Stopping service..."
  ssh -i "$KEY" $SSH_OPTS ubuntu@$IP "sudo systemctl stop album-store || true" 2>/dev/null

  echo "  → Copying binary..."
  scp -i "$KEY" $SSH_OPTS bin/server-linux ubuntu@$IP:/opt/album-store/server
  ssh -i "$KEY" $SSH_OPTS ubuntu@$IP "chmod +x /opt/album-store/server"

  echo "  → Copying .env..."
  scp -i "$KEY" $SSH_OPTS "$ENV_FILE" ubuntu@$IP:/opt/album-store/.env

  echo "  → Installing systemd service..."
  ssh -i "$KEY" $SSH_OPTS ubuntu@$IP "sudo tee /etc/systemd/system/album-store.service > /dev/null <<EOF
[Unit]
Description=Album Store API
After=network.target

[Service]
User=ubuntu
WorkingDirectory=/opt/album-store
EnvironmentFile=/opt/album-store/.env
ExecStart=/opt/album-store/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF"
  ssh -i "$KEY" $SSH_OPTS ubuntu@$IP "sudo systemctl daemon-reload && sudo systemctl enable album-store"
}

# ─── Run migration on EC2-1 (DB host) ───
echo ""
echo "═══ Running DB migration on EC2-1 ($EC2_1_IP) ═══"
scp -i "$KEY" $SSH_OPTS internal/db/migrations/001_init.sql ubuntu@$EC2_1_IP:/tmp/001_init.sql
ssh -i "$KEY" $SSH_OPTS ubuntu@$EC2_1_IP "sudo -u postgres psql -d albumstore -f /tmp/001_init.sql" 2>/dev/null

echo "  → Granting permissions..."
ssh -i "$KEY" $SSH_OPTS ubuntu@$EC2_1_IP "sudo -u postgres psql -d albumstore -c \"GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO albumuser; GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO albumuser; ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO albumuser; ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO albumuser;\"" 2>/dev/null

# ─── Deploy to both instances ───
deploy_instance "$EC2_1_IP" "/tmp/env-ec2-1" "EC2-1 (DB+App)"
deploy_instance "$EC2_2_IP" "/tmp/env-ec2-2" "EC2-2 (App)"

# ─── Start services ───
echo ""
echo "═══ Starting services ═══"
ssh -i "$KEY" $SSH_OPTS ubuntu@$EC2_1_IP "sudo systemctl restart album-store"
ssh -i "$KEY" $SSH_OPTS ubuntu@$EC2_2_IP "sudo systemctl restart album-store"

echo "  → Waiting for services to come up..."
sleep 4

# ─── Health checks ───
echo ""
echo "═══ Health Checks ═══"
echo -n "  EC2-1 ($EC2_1_IP): "
curl -s --max-time 5 http://$EC2_1_IP:8080/health || echo "FAILED"
echo ""
echo -n "  EC2-2 ($EC2_2_IP): "
curl -s --max-time 5 http://$EC2_2_IP:8080/health || echo "FAILED"
echo ""

# ─── Cleanup ───
rm -f /tmp/env-ec2-1 /tmp/env-ec2-2

echo ""
echo "✓ Multi-instance deployment complete!"
echo "  Submit to ChaosArena using the ALB DNS name (port 80)"
