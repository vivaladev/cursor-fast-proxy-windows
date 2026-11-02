#!/bin/bash
# Remove the Cursor sing-box sidecar. Does not touch AmneziaWG, SSH, or the sing-box binary.
set -euo pipefail

UNIT="/etc/systemd/system/sing-box-cursor.service"
CONF_DIR="/etc/sing-box-cursor"

systemctl disable --now sing-box-cursor || true
rm -f "$UNIT"
systemctl daemon-reload || true
rm -rf "$CONF_DIR"

if command -v ufw >/dev/null 2>&1; then
  if ufw status 2>/dev/null | grep -q 'Status: active'; then
    ufw delete allow 8443/tcp || true
  fi
fi
