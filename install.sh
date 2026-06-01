#!/usr/bin/env bash
set -euo pipefail

REPO="adityachaudhary99/yank"     # set to your GitHub owner/repo
BINARY="yank"

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo "unsupported" ;;
  esac
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *) echo "unsupported" ;;
  esac
}

install_yank() {
  local os arch tag url tmp dest
  os="$(detect_os)"; arch="$(detect_arch)"
  [ "$os" = unsupported ] || [ "$arch" = unsupported ] && { echo "unsupported platform"; exit 1; }
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -oP '"tag_name":\s*"\K[^"]+')"
  url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}_${os}_${arch}.tar.gz"
  tmp="$(mktemp -d)"
  echo "Downloading ${BINARY} ${tag} (${os}/${arch})..."
  curl -fsSL "$url" | tar -xz -C "$tmp"
  dest="${PREFIX:-$HOME/.local/bin}"
  mkdir -p "$dest"
  install -m 0755 "$tmp/${BINARY}" "$dest/${BINARY}"
  rm -rf "$tmp"
  echo "Installed to ${dest}/${BINARY}"
  echo "Ensure ${dest} is on your PATH."
}

# Allow sourcing for tests without executing.
if [ "${1:-}" = "--source-only" ]; then return 0 2>/dev/null || exit 0; fi
install_yank
