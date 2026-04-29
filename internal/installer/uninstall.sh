#!/usr/bin/env sh
set -u

install_dir="${KIT_INSTALL_DIR:-$HOME/.local/bin}"

remove_path() {
  path="$1"
  kind="$2"
  if [ -z "$path" ]; then
    return
  fi
  if [ ! -e "$path" ] && [ ! -L "$path" ]; then
    echo "Skipped $kind: $path"
    return
  fi
  if rm -rf "$path" 2>/dev/null; then
    echo "Removed $kind: $path"
  else
    echo "Could not remove $kind: $path" >&2
  fi
}

remove_path "${KIT_BIN:-$install_dir/kit}" "binary"
remove_path "$HOME/.local/bin/kit" "binary"
remove_path "$HOME/bin/kit" "binary"

if command -v kit >/dev/null 2>&1; then
  found_kit="$(command -v kit)"
  case "$found_kit" in
    */*) remove_path "$found_kit" "binary" ;;
  esac
fi

remove_path "$HOME/.kit" "state"
remove_path "$HOME/.kit-server" "state"

echo "Uninstall complete."
echo "Remove ~/.kit/shims from PATH in your shell profile if you added it manually."
