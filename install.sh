#!/usr/bin/env bash
set -euo pipefail

REPO="r13v/plexus"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_VERSION="${INSTALL_VERSION:-latest}"

err() { printf 'plexus-install: %s\n' "$*" >&2; exit 1; }
log() { printf 'plexus-install: %s\n' "$*"; }

need() { command -v "$1" >/dev/null 2>&1 || err "missing required tool: $1"; }
need uname
need tar
need mkdir
need install
if command -v curl >/dev/null 2>&1; then
  DL="curl -fsSL"
  DL_OUT="curl -fsSL -o"
elif command -v wget >/dev/null 2>&1; then
  DL="wget -qO-"
  DL_OUT="wget -qO"
else
  err "need curl or wget"
fi

case "$(uname -s)" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) err "unsupported OS: $(uname -s) (linux/darwin only)" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) err "unsupported arch: $(uname -m) (amd64/arm64 only)" ;;
esac

if [ "$INSTALL_VERSION" = "latest" ]; then
  log "resolving latest release..."
  TAG="$($DL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -o '"tag_name":[[:space:]]*"[^"]*"' \
    | head -n1 \
    | sed 's/.*"\([^"]*\)"$/\1/')"
  [ -n "${TAG:-}" ] || err "could not determine latest tag"
else
  TAG="$INSTALL_VERSION"
fi

VERSION="${TAG#v}"
ARCHIVE="plexus_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE_URL="${BASE_URL}/${ARCHIVE}"
SUMS_URL="${BASE_URL}/checksums.txt"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

log "downloading ${ARCHIVE_URL}"
$DL_OUT "${TMP}/${ARCHIVE}" "$ARCHIVE_URL" || err "download failed: $ARCHIVE_URL"

log "downloading checksums.txt"
$DL_OUT "${TMP}/checksums.txt" "$SUMS_URL" || err "download failed: $SUMS_URL"

EXPECTED="$(awk -v name="$ARCHIVE" '$2 == name { print $1 }' "${TMP}/checksums.txt")"
[ -n "$EXPECTED" ] || err "checksum for ${ARCHIVE} not found in checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMP}/${ARCHIVE}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "${TMP}/${ARCHIVE}" | awk '{print $1}')"
else
  err "need sha256sum or shasum"
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  err "checksum mismatch: expected=$EXPECTED actual=$ACTUAL"
fi
log "checksum ok"

tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP"
[ -f "${TMP}/plexus" ] || err "binary 'plexus' not found in archive"

mkdir -p "$INSTALL_DIR"
install -m 0755 "${TMP}/plexus" "${INSTALL_DIR}/plexus"
log "installed plexus ${TAG} to ${INSTALL_DIR}/plexus"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) log "WARNING: ${INSTALL_DIR} is not in your PATH. Add it, e.g.: export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac
