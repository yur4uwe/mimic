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
    osname="linux"
    ;;
  win)
    installer="scripts/install.ps1"
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

arches=(amd64 arm64)
if [ "$osname" = "linux" ]; then
  goos=linux
else
  goos=windows
fi

OUTDIR="$(pwd)/.."
installer_name="$(basename "$installer")"

for arch in "${arches[@]}"; do
  echo "Packaging ${osname}/${arch}"
  stagedir="build/"
  rm -rf "${stagedir}"
  mkdir -p "${stagedir}"

  cp -p README.md LICENCE INSTALL.md example-config.toml "${stagedir}/"
  if [ -f "$installer" ]; then
    cp -p "$installer" "${stagedir}/"
  else
    echo "Missing installer: $installer" >&2
    rm -rf "${stagedir}"
    exit 1
  fi

  if [ "$goos" = "windows" ]; then
    outbin="mimic.exe"
  else
    outbin="mimic"
  fi
  echo "Building ${outbin} (GOOS=${goos} GOARCH=${arch})"
  GOOS="${goos}" GOARCH="${arch}" go build -o "${stagedir}/${outbin}" ./cmd/main/main.go

  if [ "$goos" = "linux" ]; then
    ARCHIVE_NAME="mimic-${osname}-${arch}.tar.gz"
    tar -C "${stagedir}" -czf "${OUTDIR}/${ARCHIVE_NAME}" .
    echo "Created ${OUTDIR}/${ARCHIVE_NAME}"
  else
    ARCHIVE_NAME="mimic-${osname}-${arch}.zip"
    if ! command -v zip >/dev/null 2>&1; then
      echo "zip not found; please install zip to create Windows archive" >&2
      rm -rf "${stagedir}"
      exit 1
    fi
    zip -j "${OUTDIR}/${ARCHIVE_NAME}" "${stagedir}"/* >/dev/null
    echo "Created ${OUTDIR}/${ARCHIVE_NAME}"
  fi

  rm -rf "${stagedir}"
done