#!/bin/bash
set -e

# Usage: ./scripts/deploy.sh <ec2-public-ip> <key-path>
# Example: ./scripts/deploy.sh 1.2.3.4 ~/.ssh/my-key.pem

EC2_IP=$1
KEY=$2

if [ -z "$EC2_IP" ] || [ -z "$KEY" ]; then
  echo "Usage: ./scripts/deploy.sh <ec2-public-ip> <key-path>"
  exit 1
fi

echo "→ Building binary for Linux..."
GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -o bin/server-linux ./cmd/server

echo "→ Stopping service..."
ssh -i "$KEY" ubuntu@$EC2_IP "sudo systemctl stop album-store || true"

echo "→ Copying binary to EC2..."
scp -i "$KEY" -o StrictHostKeyChecking=no bin/server-linux ubuntu@$EC2_IP:/opt/album-store/server
ssh -i "$KEY" ubuntu@$EC2_IP "chmod +x /opt/album-store/server"

echo "→ Copying .env to EC2..."
scp -i "$KEY" -o StrictHostKeyChecking=no .env.ec2 ubuntu@$EC2_IP:/opt/album-store/.env

echo "→ Running DB migration..."
scp -i "$KEY" -o StrictHostKeyChecking=no internal/db/migrations/001_init.sql ubuntu@$EC2_IP:/tmp/001_init.sql
ssh -i "$KEY" ubuntu@$EC2_IP "sudo -u postgres psql -d albumstore -f /tmp/001_init.sql"

echo "→ Installing systemd service..."
ssh -i "$KEY" ubuntu@$EC2_IP "sudo tee /etc/systemd/system/album-store.service > /dev/null <<EOF
[Unit]
Description=Album Store API
After=network.target postgresql.service

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
ssh -i "$KEY" ubuntu@$EC2_IP "sudo systemctl daemon-reload && sudo systemctl enable album-store"

echo "→ Restarting service..."
ssh -i "$KEY" ubuntu@$EC2_IP "sudo systemctl restart album-store"

echo "→ Waiting for service to come up..."
sleep 3
ssh -i "$KEY" ubuntu@$EC2_IP "sudo systemctl status album-store --no-pager"

echo ""
echo "✓ Deployed! Health check:"
curl -s http://$EC2_IP:8080/health
echo ""
