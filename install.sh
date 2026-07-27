#!/bin/sh
# Rumpty CLI installer — https://get.rumptycloud.com
#
#   curl -fsSL https://get.rumptycloud.com | sh
#   curl -fsSL https://get.rumptycloud.com | RUMPTY_VERSION=v0.0.3 sh
#   curl -fsSL https://get.rumptycloud.com | RUMPTY_INSTALL_DIR="$HOME/.local/bin" sh
#
# Env overrides:
#   RUMPTY_VERSION      Install a specific tag (e.g. v0.0.3). Default: latest release.
#   RUMPTY_INSTALL_DIR  Where to place the binary. Default: /usr/local/bin
#                       (falls back to $HOME/.local/bin when not writable).
#   RUMPTY_BASE_URL     Base URL for release archives. Default: GitHub releases.
#                       Point at your own bucket/CDN mirror to avoid GitHub.
set -eu

REPO="Sanmo-Labs/rumpty-cli"
BIN="rumpty"
BASE_URL="${RUMPTY_BASE_URL:-https://github.com/${REPO}/releases/download}"
INSTALL_DIR="${RUMPTY_INSTALL_DIR:-/usr/local/bin}"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33mwarn:\033[0m %s\n' "$1" >&2; }
error() { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

# --- Branding ---------------------------------------------------------------
ORANGE='\033[38;5;208m'; BOLD='\033[1m'; DIM='\033[2m'; RESET='\033[0m'
if [ -n "${NO_COLOR:-}" ] || [ ! -t 1 ]; then ORANGE=''; BOLD=''; DIM=''; RESET=''; fi

banner() {
  printf '%b' "$ORANGE"
  cat <<'ART'
                            _
 _ __ _   _ _ __ ___  _ __ | |_ _   _
| '__| | | | '_ ` _ \| '_ \| __| | | |
| |  | |_| | | | | | | |_) | |_| |_| |
|_|   \__,_|_| |_| |_| .__/ \__|\__, |
                     |_|        |___/  cloud
ART
  printf '%b' "$RESET"
}

# --- Downloader (curl or wget) ---------------------------------------------
if command -v curl >/dev/null 2>&1; then
  dl()  { curl -fsSL "$1"; }          # to stdout
  dlo() { curl -fsSL -o "$1" "$2"; }  # to file
elif command -v wget >/dev/null 2>&1; then
  dl()  { wget -qO- "$1"; }
  dlo() { wget -qO "$1" "$2"; }
else
  error "need curl or wget installed"
fi
command -v tar >/dev/null 2>&1 || error "need tar installed"

# --- Detect platform --------------------------------------------------------
OS="$(uname -s)"; ARCH="$(uname -m)"
case "$OS" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) error "unsupported OS: $OS — download from https://github.com/${REPO}/releases" ;;
esac
case "$ARCH" in
  x86_64|amd64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) error "unsupported architecture: $ARCH" ;;
esac

# --- Resolve version --------------------------------------------------------
TAG="${RUMPTY_VERSION:-}"
if [ -z "$TAG" ]; then
  info "Resolving latest release..."
  TAG="$(dl "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -n1 \
    | sed -E 's/.*"tag_name" *: *"([^"]+)".*/\1/')"
  [ -n "$TAG" ] || error "could not resolve latest version; pass RUMPTY_VERSION=vX.Y.Z"
fi
VERSION="${TAG#v}"   # goreleaser archives drop the leading v

ARCHIVE="rumpty-cli_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="${BASE_URL}/${TAG}/${ARCHIVE}"
SUMS_URL="${BASE_URL}/${TAG}/rumpty-cli_${VERSION}_checksums.txt"

# --- Download + verify ------------------------------------------------------
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT INT TERM

info "Downloading ${BIN} ${TAG} (${OS}/${ARCH})..."
dlo "$TMP/$ARCHIVE" "$URL" || error "download failed: $URL"

if dlo "$TMP/checksums.txt" "$SUMS_URL" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then SHA="sha256sum";
  elif command -v shasum >/dev/null 2>&1; then SHA="shasum -a 256"; else SHA=""; fi
  if [ -n "$SHA" ]; then
    ( cd "$TMP" && grep " ${ARCHIVE}\$" checksums.txt | $SHA -c - ) >/dev/null 2>&1 \
      && info "Checksum OK" || error "checksum verification failed"
  else
    warn "no sha256 tool; skipping checksum verification"
  fi
else
  warn "checksums not found; skipping verification"
fi

# --- Extract + install ------------------------------------------------------
tar -xzf "$TMP/$ARCHIVE" -C "$TMP" || error "failed to extract archive"
[ -f "$TMP/$BIN" ] || error "binary '$BIN' not found in archive"

install_to() {
  mkdir -p "$1" 2>/dev/null || return 1
  if [ -w "$1" ]; then
    install -m 0755 "$TMP/$BIN" "$1/$BIN"
  elif command -v sudo >/dev/null 2>&1; then
    info "Installing to $1 (requires sudo)"
    sudo install -m 0755 "$TMP/$BIN" "$1/$BIN"
  else
    return 1
  fi
}

if ! install_to "$INSTALL_DIR"; then
  warn "cannot write to $INSTALL_DIR; falling back to \$HOME/.local/bin"
  INSTALL_DIR="$HOME/.local/bin"
  install_to "$INSTALL_DIR" || error "install failed"
fi
info "Installed $BIN -> $INSTALL_DIR/$BIN"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) : ;;
  *) warn "$INSTALL_DIR is not on your PATH — add: export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac

VERSION_OUT="$("$INSTALL_DIR/$BIN" --version 2>/dev/null | head -n1 || true)"

printf '\n'
banner
printf '\n'
printf '  %bRumpty CLI installed%b   %s\n' "$BOLD" "$RESET" "${VERSION_OUT:-$TAG}"
printf '  %b%s/%s%b\n\n' "$DIM" "$INSTALL_DIR" "$BIN" "$RESET"
printf '  %bGet started%b\n\n' "$BOLD" "$RESET"
printf '  %b1.%b Create an API key   %bhttps://console.rumptycloud.com/profile%b  (API keys tab)\n' "$ORANGE" "$RESET" "$DIM" "$RESET"
printf '  %b2.%b Add it to your env  export RUMPTY_API_KEY="rumpty_..."\n' "$ORANGE" "$RESET"
printf '  %b3.%b Sign in             rumpty login --token "$RUMPTY_API_KEY"\n' "$ORANGE" "$RESET"
printf '  %b4.%b Open your first VM  rumpty ssh <vm-name> --ws <workspace>\n\n' "$ORANGE" "$RESET"
printf '  %bManage everything at%b %bhttps://console.rumptycloud.com%b\n\n' "$DIM" "$RESET" "$ORANGE" "$RESET"
