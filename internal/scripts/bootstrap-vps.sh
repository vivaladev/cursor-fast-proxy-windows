#!/bin/bash
# Install sing-box VLESS+Reality for Cursor on TCP 8443.
# Does not touch AmneziaWG, port 443, or SSH.
set -euo pipefail

SINGBOX_VERSION="1.14.0"
SINGBOX="/usr/local/bin/sing-box"
CONF_DIR="/etc/sing-box-cursor"
KEYS="$CONF_DIR/keys.env"
CONFIG="$CONF_DIR/config.json"
UNIT="/etc/systemd/system/sing-box-cursor.service"

log() { printf '%s\n' "$*" >&2; }

sq() {
  local value=$1
  value=${value//\'/\'\\\'\'}
  printf "'%s'" "$value"
}

json_escape() {
  local value=$1
  value=${value//\\/\\\\}
  value=${value//\"/\\\"}
  printf '%s' "$value"
}

install_singbox() {
  if [[ -x "$SINGBOX" ]]; then
    return
  fi
  local arch asset url tmp bin
  case "$(uname -m)" in
    x86_64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) log "неизвестная архитектура $(uname -m)"; exit 1 ;;
  esac
  asset="sing-box-${SINGBOX_VERSION}-linux-${arch}.tar.gz"
  url="https://github.com/SagerNet/sing-box/releases/download/v${SINGBOX_VERSION}/${asset}"
  if ! command -v curl >/dev/null 2>&1; then
    apt-get update
    apt-get install -y curl ca-certificates tar
  fi
  tmp="$(mktemp -d)"
  curl -fsSL -o "$tmp/sing-box.tgz" "$url"
  tar -xzf "$tmp/sing-box.tgz" -C "$tmp"
  bin="$(find "$tmp" -type f -name sing-box | head -1)"
  if [[ -z "$bin" ]]; then
    log "в архиве sing-box нет бинаря"
    exit 1
  fi
  install -m 0755 "$bin" "$SINGBOX"
  rm -rf "$tmp"
}

write_keys() {
  local uuid priv pub short keypair
  uuid="$("$SINGBOX" generate uuid | tr -d '[:space:]')"
  keypair="$("$SINGBOX" generate reality-keypair)"
  priv="$(printf '%s\n' "$keypair" | sed -n 's/^PrivateKey: //p' | head -1 | tr -d '[:space:]')"
  pub="$(printf '%s\n' "$keypair" | sed -n 's/^PublicKey: //p' | head -1 | tr -d '[:space:]')"
  short="$("$SINGBOX" generate rand --hex 8 | tr -d '[:space:]')"
  if [[ -z "$uuid" || -z "$priv" || -z "$pub" || -z "$short" ]]; then
    log "sing-box не сгенерировал ключи"
    exit 1
  fi
  umask 077
  {
    printf 'UUID=%s\n' "$(sq "$uuid")"
    printf 'PRIVATE_KEY=%s\n' "$(sq "$priv")"
    printf 'PUBLIC_KEY=%s\n' "$(sq "$pub")"
    printf 'SHORT_ID=%s\n' "$(sq "$short")"
  } > "$KEYS"
  chmod 600 "$KEYS"
}

write_config() {
  local uuid priv pub short
  # shellcheck disable=SC1090
  source "$KEYS"
  uuid=$UUID
  priv=$PRIVATE_KEY
  pub=$PUBLIC_KEY
  short=$SHORT_ID
  umask 077
  cat > "$CONFIG" <<EOF
{
  "log": { "level": "warn", "timestamp": true },
  "inbounds": [{
    "type": "vless",
    "listen": "0.0.0.0",
    "listen_port": 8443,
    "users": [{ "uuid": "$(json_escape "$uuid")", "flow": "xtls-rprx-vision" }],
    "tls": {
      "enabled": true,
      "server_name": "www.cloudflare.com",
      "reality": {
        "enabled": true,
        "handshake": { "server": "www.cloudflare.com", "server_port": 443 },
        "private_key": "$(json_escape "$priv")",
        "short_id": ["$(json_escape "$short")"]
      }
    }
  }],
  "outbounds": [{ "type": "direct", "tag": "direct" }]
}
EOF
  chmod 600 "$CONFIG"
  "$SINGBOX" check -c "$CONFIG"
}

write_unit() {
  cat > "$UNIT" <<EOF
[Unit]
Description=sing-box cursor egress
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=$SINGBOX run -c $CONFIG
Restart=on-failure
RestartSec=2
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable --now sing-box-cursor
}

open_firewall() {
  if command -v ufw >/dev/null 2>&1; then
    if ufw status 2>/dev/null | grep -q 'Status: active'; then
      ufw allow 8443/tcp
    fi
  fi
}

install_singbox
mkdir -p "$CONF_DIR"
chmod 700 "$CONF_DIR"
if [[ ! -f "$KEYS" ]]; then
  write_keys
fi
write_config
write_unit
open_firewall

# shellcheck disable=SC1090
source "$KEYS"
printf 'UUID=%s\n' "$UUID"
printf 'PUBLIC_KEY=%s\n' "$PUBLIC_KEY"
printf 'SHORT_ID=%s\n' "$SHORT_ID"
