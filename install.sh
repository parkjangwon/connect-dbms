#!/usr/bin/env sh
set -eu

APP_NAME="connect-dbms"
REPO="parkjangwon/connect-dbms"
ACTION="${1:-install}"
CHANNEL="${CONNECT_DBMS_CHANNEL:-basic}"
BIN_DIR="${CONNECT_DBMS_BIN_DIR:-}"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/connect-dbms"

log() {
  printf '%s\n' "$*"
}

detect_os() {
  case "$(uname -s)" in
    Darwin) printf 'darwin' ;;
    Linux) printf 'linux' ;;
    *) log "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) log "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

resolve_bin_dir() {
  if [ -n "$BIN_DIR" ]; then
    printf '%s' "$BIN_DIR"
    return
  fi

  if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    printf '/usr/local/bin'
    return
  fi

  printf '%s' "$HOME/.local/bin"
}

latest_asset_url() {
  os="$1"
  arch="$2"
  api="https://api.github.com/repos/${REPO}/releases/latest"
  release_json="$(curl -fsSL "$api")"
  printf '%s' "$release_json" \
    | tr ',' '\n' \
    | grep '"browser_download_url"' \
    | cut -d '"' -f4 \
    | grep "/${APP_NAME}-${CHANNEL}-.*-${os}-${arch}\\.tar\\.gz$" \
    | head -n 1
}

install_or_update() {
  os="$(detect_os)"
  arch="$(detect_arch)"
  target_dir="$(resolve_bin_dir)"
  mkdir -p "$target_dir"

  asset_url="$(latest_asset_url "$os" "$arch" || true)"
  if [ -z "$asset_url" ]; then
    log "No release asset found for channel=${CHANNEL}, os=${os}, arch=${arch}" >&2
    exit 1
  fi

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT INT TERM

  archive_path="${tmp_dir}/${APP_NAME}.tar.gz"
  curl -fsSL "$asset_url" -o "$archive_path"
  tar -xzf "$archive_path" -C "$tmp_dir"

  install -m 0755 "${tmp_dir}/${APP_NAME}" "${target_dir}/${APP_NAME}"

  log "${APP_NAME} ${ACTION} complete"
  log "Installed to: ${target_dir}/${APP_NAME}"

  case ":$PATH:" in
    *":${target_dir}:"*) ;;
    *)
      log "Add ${target_dir} to your PATH if needed."
      ;;
  esac

  if [ "$CHANNEL" = "odbc" ]; then
    log "ODBC/full channel selected. Ensure unixODBC and the vendor ODBC driver are installed."
  fi
}

uninstall_app() {
  removed=0
  for dir in \
    "${CONNECT_DBMS_BIN_DIR:-}" \
    "/usr/local/bin" \
    "$HOME/.local/bin" \
    "$HOME/bin"
  do
    [ -n "$dir" ] || continue
    if [ -f "${dir}/${APP_NAME}" ]; then
      rm -f "${dir}/${APP_NAME}"
      log "Removed ${dir}/${APP_NAME}"
      removed=1
    fi
  done

  if [ -d "$CONFIG_DIR" ]; then
    rm -rf "$CONFIG_DIR"
    log "Removed ${CONFIG_DIR}"
    removed=1
  fi

  if [ "$removed" -eq 0 ]; then
    log "Nothing to remove"
  else
    log "${APP_NAME} uninstall complete"
  fi
}

case "$ACTION" in
  install|update)
    install_or_update
    ;;
  uninstall|remove|delete)
    uninstall_app
    ;;
  *)
    log "Usage: $0 [install|update|uninstall]" >&2
    exit 1
    ;;
esac
