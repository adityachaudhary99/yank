#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=/dev/null
source ./install.sh --source-only
case "$(detect_arch)" in
  amd64|arm64) echo "arch-ok" ;;
  *) echo "unexpected arch"; exit 1 ;;
esac
echo "all-ok"
