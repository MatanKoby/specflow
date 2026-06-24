#!/bin/sh
# specflow installer — fetches the prebuilt binary from GitHub Releases. No Go required.
#
#   curl -fsSL https://raw.githubusercontent.com/MatanKoby/specflow/main/install.sh | sh
#
# Environment overrides:
#   SPECFLOW_VERSION      tag to install (e.g. v0.1.0); default: latest published release
#   SPECFLOW_INSTALL_DIR  install directory; default: /usr/local/bin (falls back to ~/.local/bin)
set -eu

REPO="MatanKoby/specflow"
BINARY="specflow"

# ---- pretty output (suppressed when stdout is not a TTY) ----------------------
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  BOLD=$(printf '\033[1m'); DIM=$(printf '\033[2m'); RED=$(printf '\033[31m'); RESET=$(printf '\033[0m')
else
  BOLD=''; DIM=''; RED=''; RESET=''
fi
info() { printf '%s\n' "$*"; }
err()  { printf '%s%s%s\n' "$RED" "$*" "$RESET" >&2; }
die()  { err "error: $*"; exit 1; }

# ---- prerequisites ------------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  http_get() { curl -fsSL "$1"; }
  http_dl()  { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  http_get() { wget -qO- "$1"; }
  http_dl()  { wget -qO "$2" "$1"; }
else
  die "need curl or wget to download specflow"
fi
command -v tar >/dev/null 2>&1 || die "need tar to extract the archive"

# ---- detect OS / arch ---------------------------------------------------------
os=$(uname -s)
case "$os" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) die "unsupported OS '$os' — install with: go install github.com/$REPO/cmd/$BINARY@latest" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) die "unsupported architecture '$arch' — install with: go install github.com/$REPO/cmd/$BINARY@latest" ;;
esac

# ---- resolve version ----------------------------------------------------------
TAG="${SPECFLOW_VERSION:-}"
if [ -z "$TAG" ]; then
  info "${DIM}resolving latest release…${RESET}"
  TAG=$(http_get "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' | head -n1 | sed -E 's/.*"tag_name" *: *"([^"]+)".*/\1/')
  [ -n "$TAG" ] || die "could not resolve the latest release — set SPECFLOW_VERSION=vX.Y.Z, or no release is published yet"
fi
VERSION="${TAG#v}" # GoReleaser strips the leading v from archive names

ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$TAG"

info "${BOLD}specflow ${TAG}${RESET} → ${OS}/${ARCH}"

# ---- download + verify --------------------------------------------------------
TMP=$(mktemp -d 2>/dev/null || mktemp -d -t specflow)
trap 'rm -rf "$TMP"' EXIT INT TERM

info "${DIM}downloading ${ARCHIVE}…${RESET}"
http_dl "$BASE/$ARCHIVE" "$TMP/$ARCHIVE" \
  || die "download failed — '$ARCHIVE' may not exist for $TAG (check the release assets)"

if http_dl "$BASE/checksums.txt" "$TMP/checksums.txt" 2>/dev/null; then
  expected=$(grep " $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual=$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      actual=$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
    fi
    if [ -n "${actual:-}" ] && [ "$actual" != "$expected" ]; then
      die "checksum mismatch for $ARCHIVE (expected $expected, got $actual)"
    fi
    [ -n "${actual:-}" ] && info "${DIM}checksum ok${RESET}"
  fi
fi

tar -xzf "$TMP/$ARCHIVE" -C "$TMP" "$BINARY" || die "could not extract $BINARY from $ARCHIVE"

# ---- install ------------------------------------------------------------------
DIR="${SPECFLOW_INSTALL_DIR:-/usr/local/bin}"
if [ ! -d "$DIR" ] || [ ! -w "$DIR" ]; then
  if [ "$DIR" = "/usr/local/bin" ] && command -v sudo >/dev/null 2>&1; then
    info "${DIM}/usr/local/bin needs elevation — using sudo${RESET}"
    sudo install -m 0755 "$TMP/$BINARY" "$DIR/$BINARY" || die "install to $DIR failed"
  else
    DIR="$HOME/.local/bin"
    mkdir -p "$DIR"
    install -m 0755 "$TMP/$BINARY" "$DIR/$BINARY" || die "install to $DIR failed"
  fi
else
  install -m 0755 "$TMP/$BINARY" "$DIR/$BINARY" || die "install to $DIR failed"
fi

info ""
info "${BOLD}installed${RESET} $BINARY $TAG → $DIR/$BINARY"
case ":$PATH:" in
  *":$DIR:"*) info "run: ${BOLD}specflow init${RESET}" ;;
  *) info "add it to your PATH: ${BOLD}export PATH=\"$DIR:\$PATH\"${RESET}" ;;
esac
