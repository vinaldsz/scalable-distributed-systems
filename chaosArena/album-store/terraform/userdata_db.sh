#!/bin/bash
set -e

# ─── Install PostgreSQL 15 ───
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

# ─── Configure PostgreSQL for remote access ───
PG_CONF="/etc/postgresql/15/main/postgresql.conf"
PG_HBA="/etc/postgresql/15/main/pg_hba.conf"

# Listen on all interfaces (so EC2-2 can connect)
sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" "$PG_CONF"

# Tune PostgreSQL for performance
cat >> "$PG_CONF" <<PGCONF
# Performance tuning
shared_buffers = 2GB
work_mem = 64MB
effective_cache_size = 6GB
random_page_cost = 1.1
max_connections = 200
PGCONF

# Allow connections from VPC CIDR (172.31.0.0/16)
echo "host albumstore albumuser 172.31.0.0/16 md5" >> "$PG_HBA"

# Restart PostgreSQL with new config
systemctl restart postgresql

# ─── TCP / Kernel Tuning ───
cat >> /etc/sysctl.conf <<SYSCTL
net.core.rmem_max=16777216
net.core.wmem_max=16777216
net.ipv4.tcp_rmem=4096 87380 16777216
net.ipv4.tcp_wmem=4096 65536 16777216
net.core.somaxconn=65535
net.core.netdev_max_backlog=65535
net.ipv4.tcp_max_syn_backlog=65535
net.ipv4.tcp_tw_reuse=1
SYSCTL
sysctl -p

# ─── App directory + systemd service ───
mkdir -p /opt/album-store
chown ubuntu:ubuntu /opt/album-store

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
