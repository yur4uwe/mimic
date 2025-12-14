#!/bin/bash
set -eu

usage() {
  echo "Usage: $0 linux|win"
  exit 2
}

if [ $# -ne 1 ]; then
  usage
fi

case "$1" in
  linux)
    installer="scripts/install.sh"
    bin="mimic"
    osname="linux"
    ;;
  win)
    installer="scripts/install.ps1"
    bin="mimic.exe"
    osname="win"
    ;;
  *)
    usage
    ;;
esac

mkdir -p build

copy_or_fail() {
  if [ -f "$1" ]; then
    cp -p "$1" build/
  else
    echo "Missing required file: $1" >&2
    exit 1
  fi
}

copy_or_fail README.md
copy_or_fail LICENCE
copy_or_fail INSTALL.md
copy_or_fail example-config.toml
copy_or_fail "$bin"

if [ -f "$installer" ]; then
  cp -p "$installer" build/
else
  echo "Missing installer: $installer" >&2
  exit 1
fi

echo "Collected files into build/:"
ls -lah build

TS=$(date -u +%Y%m%d%H%M%S)
OUTDIR="$(pwd)/.."

if [ "$osname" = "linux" ]; then
  ARCHIVE_NAME="mimic-linux-${TS}.tar.gz"
  tar -C build -czf "${OUTDIR}/${ARCHIVE_NAME}" .
  echo "Created ${OUTDIR}/${ARCHIVE_NAME}"
else
  ARCHIVE_NAME="mimic-win-${TS}.zip"
  if ! command -v zip >/dev/null 2>&1; then
    echo "zip not found; please install zip to create Windows archive" >&2
    exit 1
  fi
  zip -j "${OUTDIR}/${ARCHIVE_NAME}" build/* >/dev/null
  echo "Created ${OUTDIR}/${ARCHIVE_NAME}"
fi