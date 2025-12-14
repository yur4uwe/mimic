#!/usr/bin/env bash
set -euo pipefail

TIMESTAMP() { date +"%Y%m%d%H%M%S"; }

if [ "$(id -u)" -ne 0 ]; then
  echo "This script needs root privileges. Re-running with sudo..."
  exec sudo bash "$0" "$@"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_BIN="$SCRIPT_DIR/mimic"
SRC_CONFIG="$SCRIPT_DIR/example-config.toml"

if [ ! -f "$SRC_BIN" ]; then
  echo "Error: mimic binary not found at $SRC_BIN"
  exit 1
fi
if [ ! -f "$SRC_CONFIG" ]; then
  echo "Warning: example config not found at $SRC_CONFIG"
fi

DEFAULT_BIN_DIR="/usr/local/bin"
DEFAULT_CONFIG_DIR="/etc/mimic"
DEFAULT_RUN_USER="root"

BIN_DIR="${BIN_DIR:-$DEFAULT_BIN_DIR}"
INSTALL_BIN_PATH="${INSTALL_BIN_PATH:-$BIN_DIR/mimic}"

if [ -n "${CONFIG_DIR:-}" ]; then
  CONFIG_DIR="$CONFIG_DIR"
else
  if [ "$(id -u)" -eq 0 ]; then
    CONFIG_DIR="$DEFAULT_CONFIG_DIR"
  else
    if [ -n "${XDG_CONFIG_HOME:-}" ]; then
      CONFIG_DIR="${XDG_CONFIG_HOME}/mimic"
    else
      CONFIG_DIR="${HOME}/.config/mimic"
    fi
  fi
fi
INSTALL_CONFIG_FILE="${INSTALL_CONFIG_FILE:-$CONFIG_DIR/config.toml}"

DEF_MPOINT="/mnt/mimic"
DEF_URL="https://webdav.exaple.com"
DEF_USERNAME="user"
DEF_PASSWORD="pass"
DEF_TTL="1s"
DEF_MAX_ENTRIES="100"
DEF_VERBOSE="true"
DEF_ERR="stderr"
DEF_STD="stdout"

mkdir -p "$BIN_DIR"

if [ -f "$INSTALL_BIN_PATH" ]; then
  bk="$INSTALL_BIN_PATH.bak.$(TIMESTAMP)"
  echo "Backing up existing binary to $bk"
  mv "$INSTALL_BIN_PATH" "$bk"
fi

echo "Installing binary to $INSTALL_BIN_PATH..."
install -m 0755 "$SRC_BIN" "$INSTALL_BIN_PATH"
chmod +x "$INSTALL_BIN_PATH"

mkdir -p "$CONFIG_DIR"
if [ -f "$INSTALL_CONFIG_FILE" ]; then
  bk="$INSTALL_CONFIG_FILE.bak.$(TIMESTAMP)"
  echo "Backing up existing config to $bk"
  mv "$INSTALL_CONFIG_FILE" "$bk"
fi

echo
echo "Configuration"
echo "-------------"
echo "The installer will write a TOML config file at: $INSTALL_CONFIG_FILE"
echo "Mimic expects the remote server to use HTTP Basic Authentication (username/password)."
read -r -p "Would you like to provide configuration values now? (Y/n) " cfgnow
cfgnow="${cfgnow:-Y}"

if [[ "$cfgnow" =~ ^([yY]|[yY][eE][sS])$ ]]; then
  read -r -p "Mount point [$DEF_MPOINT]: " mpoint
  mpoint="${mpoint:-$DEF_MPOINT}"

  read -r -p "Server URL [$DEF_URL]: " url
  url="${url:-$DEF_URL}"

  read -r -p "Username [$DEF_USERNAME]: " username
  username="${username:-$DEF_USERNAME}"

  read -r -s -p "Password [leave empty to use default example (pass) or press Enter]: " password
  echo
  if [ -z "$password" ]; then
    password="$DEF_PASSWORD"
  fi

  read -r -p "Cache TTL (e.g. 1s) [$DEF_TTL]: " ttl
  ttl="${ttl:-$DEF_TTL}"

  read -r -p "Cache max-entries [$DEF_MAX_ENTRIES]: " max_entries
  max_entries="${max_entries:-$DEF_MAX_ENTRIES}"

  read -r -p "Verbose logging (true/false) [$DEF_VERBOSE]: " verbose
  verbose="${verbose:-$DEF_VERBOSE}"

  read -r -p "stderr target (stderr|file path|discard) [$DEF_ERR]: " err
  err="${err:-$DEF_ERR}"

  read -r -p "stdout target (stdout|file path|discard) [$DEF_STD]: " std
  std="${std:-$DEF_STD}"

  mkdir -p "$CONFIG_DIR"

  cat >"$INSTALL_CONFIG_FILE" <<EOF
# Mimic configuration (generated)
mpoint = "$mpoint"
url = "$url"
username = "$username"
password = "$password"
ttl = "$ttl"
max-entries = $max_entries
verbose = $verbose
err = "$err"
std = "$std"
EOF

  chmod 0640 "$INSTALL_CONFIG_FILE"
  echo "Wrote configuration to $INSTALL_CONFIG_FILE"
else
  if [ -f "$SRC_CONFIG" ]; then
    echo "Installing example config..."
    install -m 0644 "$SRC_CONFIG" "$INSTALL_CONFIG_FILE"
  else
    echo "Creating default config file at $INSTALL_CONFIG_FILE"
    cat >"$INSTALL_CONFIG_FILE" <<EOF
mpoint = "$DEF_MPOINT"
url = "$DEF_URL"
username = "$DEF_USERNAME"
password = "$DEF_PASSWORD"
ttl = "$DEF_TTL"
max-entries = $DEF_MAX_ENTRIES
verbose = $DEF_VERBOSE
err = "$DEF_ERR"
std = "$DEF_STD"
EOF
    chmod 0640 "$INSTALL_CONFIG_FILE"
  fi
  echo "Note: server must use Basic Auth (username/password)."
fi

echo
echo "Installation completed"
echo "------------------------------"
echo "Binary: $INSTALL_BIN_PATH"
echo "Config: $INSTALL_CONFIG_FILE"
echo
echo "To run the application manually, you can execute:"
echo "  sudo $INSTALL_BIN_PATH --config $INSTALL_CONFIG_FILE"
echo
echo "To edit config later look for it here: $INSTALL_CONFIG_FILE"
echo "You can find where mimic gets its config file by running: mimic --where-config"