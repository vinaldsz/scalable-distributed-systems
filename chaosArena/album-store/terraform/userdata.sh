#!/bin/bash
set -e

# Install PostgreSQL 15
apt-get update -y
apt-get install -y postgresql-15 postgresql-client-15

# Start and enable PostgreSQL
systemctl enable postgresql
systemctl start postgresql

# Create DB and user
sudo -u postgres psql <<SQL
CREATE USER albumuser WITH PASSWORD 'secret';
CREATE DATABASE albumstore OWNER albumuser;
GRANT ALL PRIVILEGES ON DATABASE albumstore TO albumuser;
SQL

# Create app directory
mkdir -p /opt/album-store

# Write systemd service
cat > /etc/systemd/system/album-store.service <<EOF
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
EOF

systemctl daemon-reload
systemctl enable album-store
