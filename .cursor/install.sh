#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/.local/bin:${HOME}/go/bin:${PATH}"

install_sfetch() {
  if command -v sfetch >/dev/null 2>&1; then
    return 0
  fi
  if command -v apt-get >/dev/null 2>&1 && ! command -v minisign >/dev/null 2>&1; then
    sudo apt-get update
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y minisign
  fi
  curl -sSfL https://github.com/3leaps/sfetch/releases/latest/download/install-sfetch.sh | bash
  export PATH="${HOME}/.local/bin:${PATH}"
}

if command -v goneat >/dev/null 2>&1; then
  make dependencies
  goneat doctor tools --scope foundation --install --yes --no-cooling
else
  install_sfetch
  make bootstrap
fi

make embed-assets build
