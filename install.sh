#!/bin/sh
set -eu

repo="WolfCanCode/personal-ai-brain"
version="${BRAIN_VERSION:-}"
install_dir="${BRAIN_INSTALL_DIR:-$HOME/.local/bin}"

usage() {
  cat <<'EOF'
Install a tested Noevyra release binary.

Usage: install.sh [--version vX.Y.Z] [--bin-dir PATH]

Environment:
  BRAIN_VERSION       Release tag to install; defaults to the latest release.
  BRAIN_INSTALL_DIR   Destination directory; defaults to ~/.local/bin.
EOF
}

fail() {
  printf 'brain installer: %s\n' "$*" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || fail "--version requires a release tag"
      version=$2
      shift 2
      ;;
    --bin-dir)
      [ "$#" -ge 2 ] || fail "--bin-dir requires a path"
      install_dir=$2
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *) fail "unknown option: $1" ;;
  esac
done

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fail "supported systems are macOS and Linux; use install.ps1 on Windows" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [ -z "$version" ]; then
  latest_url=$(curl --proto '=https' --tlsv1.2 -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
  version=${latest_url##*/}
fi
printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' || fail "invalid release tag: $version"

asset="brain_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$version"
tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t brain-install)
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM
archive="$tmp_dir/$asset"
sums="$tmp_dir/SHA256SUMS"

printf 'Downloading Noevyra %s for %s/%s...\n' "$version" "$os" "$arch"
curl --proto '=https' --tlsv1.2 -fsSL "$base_url/$asset" -o "$archive"
curl --proto '=https' --tlsv1.2 -fsSL "$base_url/SHA256SUMS" -o "$sums"

expected=$(awk -v asset="$asset" '$2 == asset { print tolower($1); exit }' "$sums")
[ -n "$expected" ] || fail "checksum entry missing for $asset"
case "$expected" in
  *[!0-9a-f]*|'') fail "invalid checksum for $asset" ;;
esac
[ "${#expected}" -eq 64 ] || fail "invalid checksum length for $asset"

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$archive" | awk '{ print tolower($1) }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$archive" | awk '{ print tolower($1) }')
else
  fail "sha256sum or shasum is required"
fi
[ "$actual" = "$expected" ] || fail "SHA-256 verification failed for $asset"

mkdir -p "$tmp_dir/extract" "$install_dir"
tar -xOzf "$archive" brain > "$tmp_dir/extract/brain"
[ -f "$tmp_dir/extract/brain" ] || fail "archive does not contain the brain binary"
if command -v install >/dev/null 2>&1; then
  install -m 0755 "$tmp_dir/extract/brain" "$install_dir/brain"
else
  cp "$tmp_dir/extract/brain" "$install_dir/brain"
  chmod 0755 "$install_dir/brain"
fi

"$install_dir/brain" version >/dev/null
printf 'Installed Noevyra %s to %s/brain\n' "$version" "$install_dir"
case ":${PATH:-}:" in
  *:"$install_dir":*) ;;
  *) printf 'Add %s to PATH, then run: brain init\n' "$install_dir" ;;
esac

