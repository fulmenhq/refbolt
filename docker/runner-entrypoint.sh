#!/bin/sh
set -eu

if [ "$#" -gt 0 ]; then
    exec "$@"
fi

crontab_path="${SUPERCRONIC_CRONTAB:-/etc/refbolt/crontab}"
safe_directory="${REFBOLT_GIT_SAFE_DIRECTORY:-}"

if [ ! -f "$crontab_path" ]; then
    echo "missing crontab file: $crontab_path" >&2
    exit 1
fi

if [ ! -s "$crontab_path" ]; then
    echo "crontab file is empty: $crontab_path" >&2
    exit 1
fi

if command -v git >/dev/null 2>&1 && [ -n "$safe_directory" ]; then
    existing_safe_directories="$(git config --global --get-all safe.directory 2>/dev/null || true)"
    if ! printf '%s\n' "$existing_safe_directories" | grep -Fx -- "$safe_directory" >/dev/null 2>&1; then
        git config --global --add safe.directory "$safe_directory"
        echo "configured git safe.directory: $safe_directory"
    fi
fi

echo "starting supercronic with $crontab_path"
exec /usr/local/bin/supercronic "$crontab_path"
