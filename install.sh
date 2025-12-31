#!/bin/sh
set -e

REPO_URL="${SCDL_REPO_URL:-https://github.com/nathan-tomodachi/scdl}"
REPO_NAME="${SCDL_REPO_NAME:-scdl}"
REF="${SCDL_REF:-master}"
VERSION="${SCDL_VERSION:-}"
INSTALL_DIR="${SCDL_INSTALL_DIR:-$HOME/.local/bin}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf "Missing required command: %s\n" "$1" >&2
    exit 1
  fi
}

prompt_yes_no() {
  printf "%s [y/N]: " "$1"
  read ans
  case "$ans" in
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

case "$(uname -s)" in
  Darwin|Linux) ;;
  *)
    printf "Unsupported OS. This installer supports macOS and Linux only.\n" >&2
    exit 1
    ;;
esac

require_cmd curl
require_cmd tar
require_cmd go
require_cmd mktemp

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

if [ -n "$VERSION" ] && [ -z "${SCDL_REF:-}" ]; then
  REF="v$VERSION"
fi

tarball_url="$REPO_URL/archive/refs/heads/$REF.tar.gz"
if [ -n "$VERSION" ]; then
  tarball_url="$REPO_URL/archive/refs/tags/$REF.tar.gz"
fi
printf "Downloading source from %s\n" "$tarball_url"
curl -fsSL "$tarball_url" -o "$tmp_dir/scdl.tar.gz"

tar -xzf "$tmp_dir/scdl.tar.gz" -C "$tmp_dir"
src_dir="$tmp_dir/${REPO_NAME}-${REF}"

if [ ! -d "$src_dir" ]; then
  printf "Unexpected archive layout. Expected %s\n" "$src_dir" >&2
  exit 1
fi

printf "Building scdl...\n"
cd "$src_dir"
if [ -n "$VERSION" ]; then
  go build -ldflags "-X main.Version=$VERSION" -o "$tmp_dir/scdl" ./
else
  go build -o "$tmp_dir/scdl" ./
fi

mkdir -p "$INSTALL_DIR"
install -m 755 "$tmp_dir/scdl" "$INSTALL_DIR/scdl"
printf "Installed scdl to %s\n" "$INSTALL_DIR/scdl"

case ":$PATH:" in
  *:"$INSTALL_DIR":*)
    printf "PATH already includes %s\n" "$INSTALL_DIR"
    ;;
  *)
    if prompt_yes_no "Add $INSTALL_DIR to your PATH?"; then
      shell_name="$(basename "${SHELL:-sh}")"
      case "$shell_name" in
        zsh) profile="$HOME/.zshrc" ;;
        bash)
          if [ -f "$HOME/.bash_profile" ]; then
            profile="$HOME/.bash_profile"
          else
            profile="$HOME/.bashrc"
          fi
          ;;
        fish) profile="$HOME/.config/fish/config.fish" ;;
        *) profile="$HOME/.profile" ;;
      esac

      mkdir -p "$(dirname "$profile")"
      if [ "$shell_name" = "fish" ]; then
        printf "\nset -gx PATH %s $PATH\n" "$INSTALL_DIR" >> "$profile"
      else
        printf "\nexport PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR" >> "$profile"
      fi
      printf "Updated %s. Restart your shell to use scdl.\n" "$profile"
    else
      printf "Skipping PATH update. Ensure %s is in your PATH to run scdl.\n" "$INSTALL_DIR"
    fi
    ;;
esac
