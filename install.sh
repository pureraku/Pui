#!/bin/sh
set -eu

repo_url=${1:-${PUI_REPO_URL:-}}
if [ -z "$repo_url" ] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  repo_url=$(git remote get-url origin 2>/dev/null || true)
fi
if [ -z "$repo_url" ]; then
  echo "Usage: PUI_REPO_URL=https://github.com/OWNER/REPO.git sh install.sh" >&2
  echo "   or: sh install.sh https://github.com/OWNER/REPO.git" >&2
  exit 2
fi

install_root=${PUI_INSTALL_ROOT:-"$HOME/.pui"}
target_dir=${PUI_TARGET_DIR:-"$install_root/target"}
mkdir -p "$install_root" "$target_dir"
if [ -d "$install_root/.git" ]; then
  if git -C "$install_root" rev-parse --verify HEAD >/dev/null 2>&1; then
    git -C "$install_root" pull --ff-only
  else
    echo "The checkout has no commits yet; keeping its local source files."
  fi
elif [ -z "$(ls -A "$install_root")" ]; then
  rmdir "$install_root"
  git clone "$repo_url" "$install_root"
else
  # Keep an existing data directory while placing the repository checkout at ~/.pui.
  stage=$(mktemp -d "${TMPDIR:-/tmp}/pui-install.XXXXXX")
  trap 'rm -rf "$stage"' EXIT HUP INT TERM
  git clone "$repo_url" "$stage/repo"
  cp -R "$stage/repo/." "$install_root/"
fi

if [ ! -f "$install_root/src/main.go" ]; then
  echo "No Pui source was found in the cloned repository ($install_root/src/main.go)." >&2
  echo "Push the app source to the repository, then rerun the installer." >&2
  exit 1
fi
command -v go >/dev/null 2>&1 || { echo "Go is required to build Pui." >&2; exit 1; }
mkdir -p "$target_dir"
(
  cd "$install_root"
  go build -o "$target_dir/pui" ./src
)
chmod 755 "$target_dir/pui"

case ":${PATH}:" in
  *":$target_dir:"*) path_added=0 ;;
  *) path_added=1 ;;
esac
shell_name=$(basename "${SHELL:-sh}")
case "$shell_name" in
  zsh) profile=${ZDOTDIR:-"$HOME"}/.zprofile ;;
  bash) profile=$HOME/.bash_profile ;;
  fish) profile=${XDG_CONFIG_HOME:-"$HOME/.config"}/fish/config.fish ;;
  *) profile=$HOME/.profile ;;
esac
if [ "$path_added" -eq 1 ]; then
  mkdir -p "$(dirname "$profile")"
  touch "$profile"
  if [ "$shell_name" = fish ]; then
    line="fish_add_path $target_dir"
  else
    line="export PATH=\"$target_dir:\$PATH\""
  fi
  if ! grep -Fqx "$line" "$profile"; then
    printf '\n# Pui executable\n%s\n' "$line" >> "$profile"
  fi
  echo "Added $target_dir to PATH in $profile. Open a new terminal to use it."
else
  echo "$target_dir is already on PATH."
fi

echo "Pui source: $install_root/src/"
echo "Pui executable: $target_dir/pui"
echo "Pui data: $install_root/data/"
if ! command -v nvim >/dev/null 2>&1; then
  echo "Neovim is not currently on PATH. Install it before editing notes."
fi
