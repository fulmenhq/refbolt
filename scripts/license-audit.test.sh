#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

cat > "$WORK/scanner" << 'SCANNER'
#!/usr/bin/env bash
set -euo pipefail
case "$LICENSE_TEST_MODE" in
    allowed) echo 'example.com/module,https://example.com/LICENSE,MIT' ;;
    forbidden) echo "example.com/module,https://example.com/LICENSE,$LICENSE_TEST_TYPE" ;;
    failed) exit 42 ;;
    partial) echo 'example.com/module,https://example.com/LICENSE,MIT'; exit 42 ;;
    empty) exit 0 ;;
esac
SCANNER
chmod +x "$WORK/scanner"

audit() {
    LICENSE_TEST_MODE="$1" LICENSE_TEST_TYPE="${2:-}" \
        make --no-print-directory -s -C "$WORK" -f "$ROOT/Makefile" \
        GO_LICENSES="$WORK/scanner" license-audit > "$WORK/output" 2>&1
}

audit allowed
cp "$WORK/dist/reports/license-inventory.csv" "$WORK/expected.csv"
for mode in failed partial empty; do
    if audit "$mode"; then
        echo "license-audit incorrectly passed for $mode scanner" >&2
        exit 1
    fi
    if rg -q 'No forbidden licenses detected' "$WORK/output"; then
        echo "license-audit printed success for $mode scanner" >&2
        exit 1
    fi
    cmp "$WORK/expected.csv" "$WORK/dist/reports/license-inventory.csv"
done

for license in GPL-3.0 LGPL-2.1 AGPL-3.0 MPL-2.0 CDDL-1.0; do
    if audit forbidden "$license"; then
        echo "license-audit incorrectly accepted $license" >&2
        exit 1
    fi
done

echo 'License audit regression checks passed'
