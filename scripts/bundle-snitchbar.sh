#!/usr/bin/env bash
# Bundle snitchbar + snitchd into Snitch Bar.app (no Dock icon).
set -euo pipefail

BINARY="${1:?usage: bundle-snitchbar.sh <snitchbar-path> [version] [snitchd-path]}"
VERSION="${2:-dev}"
SNITCHD="${3:-${SNITCHD_PATH:-}}"
OUT_DIR="$(dirname "$BINARY")"
APP_NAME="Snitch Bar.app"
APP_DIR="${OUT_DIR}/${APP_NAME}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if [[ -z "$SNITCHD" ]]; then
  if [[ -x "${OUT_DIR}/snitchd" ]]; then
    SNITCHD="${OUT_DIR}/snitchd"
  else
    bar_folder="$(basename "$OUT_DIR")"
    if [[ "$bar_folder" == snitchbar_* ]]; then
      target="${bar_folder#snitchbar_}"
      candidate="${OUT_DIR}/../snitchd_${target}/snitchd"
      if [[ -x "$candidate" ]]; then
        SNITCHD="$candidate"
      fi
    fi
  fi
fi

rm -rf "$APP_DIR"
mkdir -p "${APP_DIR}/Contents/MacOS" "${APP_DIR}/Contents/Resources"

cp "$BINARY" "${APP_DIR}/Contents/MacOS/snitchbar"
chmod 755 "${APP_DIR}/Contents/MacOS/snitchbar"

if [[ -n "$SNITCHD" && -x "$SNITCHD" ]]; then
  cp "$SNITCHD" "${APP_DIR}/Contents/MacOS/snitchd"
  chmod 755 "${APP_DIR}/Contents/MacOS/snitchd"
else
  echo "warning: snitchd not found — app bundle will not include daemon" >&2
fi

sed "s/SNITCH_VERSION/${VERSION}/g" \
  "${ROOT}/install/macos/Snitch-Bar-Info.plist" \
  > "${APP_DIR}/Contents/Info.plist"

if [[ -d "${ROOT}/assets/menubar" ]]; then
  cp "${ROOT}/assets/menubar/"icon*.png "${APP_DIR}/Contents/Resources/" 2>/dev/null || true
fi

echo "bundled ${APP_DIR}"
