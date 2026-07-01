#!/bin/sh
# macOS installer — delegates to the universal terminal installer.
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export SNITCH_SOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
exec "${SCRIPT_DIR}/../../scripts/install.sh" "$@"
