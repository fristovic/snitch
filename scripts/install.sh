#!/usr/bin/env bash
# Snitch terminal installer — macOS only
# Usage: curl -fsSL https://raw.githubusercontent.com/fristovic/snitch/main/scripts/install.sh | bash
set -euo pipefail

REPO="${SNITCH_REPO:-fristovic/snitch}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
SHARE_DIR="${SHARE_DIR:-$HOME/.local/share/snitch}"
DATA_DIR="${DATA_DIR:-$HOME/.snitch}"
VERSION="${SNITCH_VERSION:-}"
INSTALL_FROM_SOURCE="${SNITCH_INSTALL_FROM_SOURCE:-0}"
MENUBAR="${SNITCH_MENUBAR:-1}"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
error() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    *)      error "Snitch requires macOS. Unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)             error "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_version() {
  if [[ -n "$VERSION" && "$VERSION" != "latest" ]]; then
    echo "${VERSION#v}"
    return
  fi
  local json tag
  json="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)" || true
  if [[ -n "$json" ]]; then
    if command -v python3 >/dev/null 2>&1; then
      tag="$(printf '%s' "$json" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tag_name',''))" 2>/dev/null)" || true
    fi
    if [[ -z "$tag" ]]; then
      tag="$(printf '%s' "$json" | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\(v[^"]*\)".*/\1/p' | head -1)"
    fi
  fi
  if [[ -z "$tag" ]]; then
    warn "no GitHub release found; will try building from source"
    echo ""
    return
  fi
  echo "${tag#v}"
}

download_release() {
  local os="$1" arch="$2" ver="$3"
  local archive="snitch_${ver}_${os}_${arch}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/v${ver}/${archive}"
  local tmp
  tmp="$(mktemp -d)"

  info "Downloading ${url}"
  if ! curl -fsSL "$url" -o "${tmp}/${archive}"; then
    rm -rf "$tmp"
    return 1
  fi
  tar -xzf "${tmp}/${archive}" -C "$tmp"
  if [[ -f "${tmp}/snitchbar" && ! -d "${tmp}/Snitch Bar.app" && -x "${tmp}/scripts/bundle-snitchbar.sh" ]]; then
    "${tmp}/scripts/bundle-snitchbar.sh" "${tmp}/snitchbar" "${ver}" "${tmp}/snitchd"
  fi
  mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$SHARE_DIR"
  install -m 755 "${tmp}/snitch" "${INSTALL_DIR}/snitch"
  if [[ -d "${tmp}/Snitch Bar.app" ]]; then
    rm -rf "${SHARE_DIR}/Snitch Bar.app"
    cp -R "${tmp}/Snitch Bar.app" "${SHARE_DIR}/"
    if [[ -x "${SHARE_DIR}/Snitch Bar.app/Contents/MacOS/snitchd" ]]; then
      ln -sf "${SHARE_DIR}/Snitch Bar.app/Contents/MacOS/snitchd" "${INSTALL_DIR}/snitchd"
    elif [[ -f "${tmp}/snitchd" ]]; then
      install -m 755 "${tmp}/snitchd" "${INSTALL_DIR}/snitchd"
    fi
  elif [[ -f "${tmp}/snitchd" ]]; then
    install -m 755 "${tmp}/snitchd" "${INSTALL_DIR}/snitchd"
  fi
  rm -rf "$tmp"
  info "Installed binaries to ${INSTALL_DIR}"
}

build_from_source() {
  if ! command -v go >/dev/null 2>&1; then
    error "Go is not installed and no release binary is available. Install Go 1.24+ or set SNITCH_VERSION to a published release."
  fi
  local root="${SNITCH_SOURCE_DIR:-}"
  local ver="${1:-dev}"
  if [[ -z "$root" ]]; then
    root="$(mktemp -d)"
    info "Cloning source..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$root"
    mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$SHARE_DIR"
    info "Building from source..."
    (cd "$root" && go build -ldflags "-s -w -X github.com/fristovic/snitch/internal/version.Version=${ver}" -o "${INSTALL_DIR}/snitchd" ./cmd/snitchd)
    (cd "$root" && go build -ldflags "-s -w -X github.com/fristovic/snitch/internal/version.Version=${ver}" -o "${INSTALL_DIR}/snitch" ./cmd/snitch)
    (cd "$root" && CGO_ENABLED=1 go build -ldflags "-s -w -X github.com/fristovic/snitch/internal/version.Version=${ver}" -o "${INSTALL_DIR}/snitchbar" ./cmd/snitchbar)
    (cd "$root" && ./scripts/bundle-snitchbar.sh "${INSTALL_DIR}/snitchbar" "${ver}" "${INSTALL_DIR}/snitchd")
    rm -rf "${SHARE_DIR}/Snitch Bar.app"
    cp -R "${INSTALL_DIR}/Snitch Bar.app" "${SHARE_DIR}/"
    rm -rf "${INSTALL_DIR}/Snitch Bar.app"
    rm -f "${INSTALL_DIR}/snitchbar"
    rm -rf "$root"
  else
    mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$SHARE_DIR"
    info "Building from source..."
    (cd "$root" && go build -ldflags "-s -w -X github.com/fristovic/snitch/internal/version.Version=${ver}" -o "${INSTALL_DIR}/snitchd" ./cmd/snitchd)
    (cd "$root" && go build -ldflags "-s -w -X github.com/fristovic/snitch/internal/version.Version=${ver}" -o "${INSTALL_DIR}/snitch" ./cmd/snitch)
    (cd "$root" && CGO_ENABLED=1 go build -ldflags "-s -w -X github.com/fristovic/snitch/internal/version.Version=${ver}" -o "${INSTALL_DIR}/snitchbar" ./cmd/snitchbar)
    (cd "$root" && ./scripts/bundle-snitchbar.sh "${INSTALL_DIR}/snitchbar" "${ver}" "${INSTALL_DIR}/snitchd")
    rm -rf "${SHARE_DIR}/Snitch Bar.app"
    cp -R "${INSTALL_DIR}/Snitch Bar.app" "${SHARE_DIR}/"
    rm -rf "${INSTALL_DIR}/Snitch Bar.app"
    rm -f "${INSTALL_DIR}/snitchbar"
  fi
  info "Built and installed to ${INSTALL_DIR}"
}

install_legacy_daemon_cleanup() {
  local plist_dest="$HOME/Library/LaunchAgents/com.snitch.daemon.plist"
  launchctl bootout "gui/$(id -u)/com.snitch.daemon" 2>/dev/null || \
    launchctl unload "$plist_dest" 2>/dev/null || true
  if [[ -f "$plist_dest" ]]; then
    rm -f "$plist_dest"
    info "Removed legacy snitchd LaunchAgent (daemon is managed by Snitch Bar)"
  fi
}

install_menubar_macos() {
  if [[ "$MENUBAR" != "1" ]]; then
    info "Skipping menu bar (SNITCH_MENUBAR=0)"
    return
  fi
  local app="${SHARE_DIR}/Snitch Bar.app/Contents/MacOS/snitchbar"
  if [[ ! -x "$app" ]]; then
    warn "Snitch Bar.app not found at ${SHARE_DIR}; skipping menubar LaunchAgent"
    return
  fi
  local plist_dest="$HOME/Library/LaunchAgents/com.snitch.menubar.plist"
  local plist_src="${DATA_DIR}/com.snitch.menubar.plist"
  if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local candidate="${script_dir}/../install/macos/com.snitch.menubar.plist"
    if [[ -f "$candidate" ]]; then
      plist_src="$candidate"
    fi
  fi
  if [[ ! -f "$plist_src" ]]; then
    curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/install/macos/com.snitch.menubar.plist" -o "$plist_src"
  fi
  mkdir -p "$HOME/Library/LaunchAgents"
  sed "s|__SNITCHBAR_APP__|${SHARE_DIR}/Snitch Bar.app|g; s|__HOME__|${HOME}|g" "$plist_src" > "$plist_dest"
  launchctl bootout "gui/$(id -u)/com.snitch.menubar" 2>/dev/null || \
    launchctl unload "$plist_dest" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist_dest" 2>/dev/null || \
    launchctl load "$plist_dest"
  info "Menu bar registered (LaunchAgent)"
}

ensure_path() {
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*) return ;;
  esac
  warn "${INSTALL_DIR} is not on your PATH (will be added to shell profile)"
}

configure_path() {
  local marker_begin="# >>> snitch path >>>"
  local marker_end="# <<< snitch path <<<"
  local block="${marker_begin}
export PATH=\"${INSTALL_DIR}:\$PATH\"
${marker_end}"
  local shell="${SHELL:-}"
  local profile=""
  if [[ "$shell" == *zsh* ]]; then
    profile="$HOME/.zshrc"
  elif [[ "$shell" == *bash* ]]; then
    profile="$HOME/.bashrc"
    [[ -f "$HOME/.bash_profile" ]] && profile="$HOME/.bash_profile"
  elif [[ -f "$HOME/.zshrc" ]]; then
    profile="$HOME/.zshrc"
  else
    profile="$HOME/.bashrc"
  fi
  if [[ -f "$profile" ]] && grep -qF "$marker_begin" "$profile" 2>/dev/null; then
    return
  fi
  if [[ -f "$profile" ]] && grep -qF "${INSTALL_DIR}" "$profile" 2>/dev/null; then
    return
  fi
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      printf '\n%s\n' "$block" >> "$profile"
      info "Added ${INSTALL_DIR} to PATH in ${profile}"
      ;;
  esac
}

print_next_steps() {
  cat <<EOF

Snitch is installed.

Snitch Bar manages the lie detector — it opens from your menu bar at login (or open it manually).
Use **Start Snitching** / **Stop Snitching** in the menu to pause or resume.

  snitch status           # daemon health (when Snitching...)
  snitch lies             # full lie history

Advanced:
  snitch log              # failed runs (power users; --watch overlaps menu bar)
  snitch dashboard        # interactive TUI

Open a new terminal (or: exec \$SHELL) if PATH was updated.

Homebrew:
  brew tap fristovic/snitch
  brew install snitch
  open "\$(brew --prefix)/opt/snitch/Snitch Bar.app"

curl / manual:
  open "\$HOME/.local/share/snitch/Snitch Bar.app"

EOF
}

detect_cursor() {
  if [[ -d "/Applications/Cursor.app" || -d "$HOME/.cursor" ]]; then
    return 0
  fi
  warn "Cursor not detected (/Applications/Cursor.app or ~/.cursor). Install Cursor first, then re-run."
  return 1
}

main() {
  local os arch ver
  os="$(detect_os)"
  arch="$(detect_arch)"
  ver="$(resolve_version)"
  detect_cursor || true

  if [[ "$INSTALL_FROM_SOURCE" == "1" || -z "$ver" ]]; then
    build_from_source "${ver:-dev}"
  else
    if ! download_release "$os" "$arch" "$ver"; then
      warn "release download failed; falling back to source build"
      build_from_source "$ver"
    fi
  fi

  install_legacy_daemon_cleanup

  install_menubar_macos
  launchctl kickstart -k "gui/$(id -u)/com.snitch.menubar" 2>/dev/null || true

  ensure_path
  configure_path
  print_next_steps
}

main "$@"
