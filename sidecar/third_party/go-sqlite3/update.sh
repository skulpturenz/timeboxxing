#!/usr/bin/env bash
# Regenerate the vendored SQLCipher-capable go-sqlite3 fork onto the latest upstream:
#   latest mattn/go-sqlite3 (Go driver) + the cipher Go glue (UPSTREAM-CHANGES.patch)
#   + the latest SQLite3MultipleCiphers amalgamation (via the fork's own upgrade/ tool).
#
# Usage:
#   ./update.sh                 # regenerate onto the latest mattn release + latest SQLite3MC
#   ./update.sh v1.14.47        # pin the mattn base tag
#
# After running: re-record the SHA-256s + version table in PROVENANCE.md, review the diff and the
# regenerated UPSTREAM-CHANGES.patch in a PR, and run `go test -tags assert ./db/...` in ../../.
# Requires network + git + Go.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
PATCH="$HERE/UPSTREAM-CHANGES.patch"
MATTN="https://github.com/mattn/go-sqlite3"

# Resolve the mattn base tag (arg, else latest from the Go proxy).
REF="${1:-}"
if [[ -z "$REF" ]]; then
  REF="$(curl -fsSL "https://proxy.golang.org/github.com/mattn/go-sqlite3/@latest" \
    | sed -n 's/.*"Version":"\([^"]*\)".*/\1/p')"
fi
[[ -n "$REF" ]] || { echo "!! could not resolve latest mattn tag" >&2; exit 1; }
echo ">> mattn base: $REF"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo ">> cloning mattn $REF ..."
git clone --depth 1 --branch "$REF" "$MATTN" "$TMP/fork" >/dev/null 2>&1

echo ">> applying cipher glue (UPSTREAM-CHANGES.patch) ..."
git -C "$TMP/fork" apply "$PATCH" || {
  echo "!! patch did not apply cleanly onto $REF — mattn likely moved code under the glue." >&2
  echo "!! Rebase UPSTREAM-CHANGES.patch onto $REF by hand, then re-run." >&2
  exit 1
}

# The upgrade tool has //go:build !cgo && upgrade && ignore; run it as an explicit file (which
# bypasses the build constraint) with -mod=mod so its transitive deps resolve.
echo ">> downloading latest SQLite3MultipleCiphers amalgamation ..."
( cd "$TMP/fork/upgrade" && CGO_ENABLED=0 go run -mod=mod upgrade.go )

echo ">> re-vendoring into $HERE ..."
rsync -a --delete \
  --exclude='*_test.go' --exclude='testdata/' --exclude='.github/' --exclude='_example/' \
  --exclude='upgrade/' --exclude='.git/' \
  --exclude='PROVENANCE.md' --exclude='update.sh' --exclude='UPSTREAM-CHANGES.patch' \
  "$TMP/fork/" "$HERE/"
chmod -R u+w "$HERE"

echo ">> regenerating UPSTREAM-CHANGES.patch (base $REF, amalgamation excluded) ..."
git -C "$TMP/fork" add -N sqlite3_cipher_test.go upgrade 2>/dev/null || true
git -C "$TMP/fork" diff -- . \
  ':(exclude)sqlite3-binding.c' ':(exclude)sqlite3-binding.h' ':(exclude)sqlite3ext.h' \
  > "$HERE/UPSTREAM-CHANGES.patch"

echo ">> new SHA-256s (paste into PROVENANCE.md):"
shasum -a 256 "$HERE/sqlite3-binding.c" "$HERE/sqlite3-binding.h" | sed "s#$HERE/#   #"
echo ">> versions:"
grep -m1 'define SQLITE_VERSION ' "$HERE/sqlite3-binding.c" | sed 's/^/   /'
grep -m1 'SQLITE3MC_VERSION_STRING' "$HERE/sqlite3-binding.c" | sed 's/^/   /'
echo ">> done. Update PROVENANCE.md (base $REF + SHAs + versions), then:"
echo "   (cd ../../ && go build -tags assert ./... && go test -tags assert ./db/...)"
