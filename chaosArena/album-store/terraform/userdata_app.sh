#!/bin/bash
set -e

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

# ─── App directory + systemd service (no PostgreSQL) ───
mkdir -p /opt/album-store
chown ubuntu:ubuntu /opt/album-store

cat > /etc/systemd/system/album-store.service <<EOF
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
EOF

systemctl daemon-reload
systemctl enable album-store
