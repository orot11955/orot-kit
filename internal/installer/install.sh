#!/usr/bin/env sh
set -eu

base="${KIT_BASE_URL:-http://localhost:8080}"
install_dir="${KIT_INSTALL_DIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin|linux) ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

name="kit-$os-$arch"
url="$base/bin/$name"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

mkdir -p "$install_dir"
echo "Downloading $url"
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"
mv "$tmp" "$install_dir/kit"
echo "Installed kit to $install_dir/kit"
echo "Make sure $install_dir is in PATH."
