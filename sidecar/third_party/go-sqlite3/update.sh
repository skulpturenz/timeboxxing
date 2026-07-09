#!/usr/bin/env bash
# Re-vendor the SQLCipher-capable go-sqlite3 fork from upstream.
#
# Usage:
#   ./update.sh                       # re-pull the pinned sqlite3mc branch
#   ./update.sh sqlite3mc-2.2.8       # pull a different branch/tag
#
# After running: update the SHA-256s + Origin table in PROVENANCE.md, review the diff in a PR,
# and run `go test ./...` in ../../ (sidecar). Requires network + Go module cache.
set -euo pipefail

UPSTREAM="github.com/jgiannuzzi/go-sqlite3"
REF="${1:-sqlite3mc-2.2.7}"
HERE="$(cd "$(dirname "$0")" && pwd)"

echo ">> resolving ${UPSTREAM}@${REF} ..."
VERSION="$(cd "$(mktemp -d)" && go mod init tmp.update >/dev/null 2>&1 \
  && go mod download -json "${UPSTREAM}@${REF}" | sed -n 's/.*"Version": "\(.*\)",/\1/p' | head -1)"
if [[ -z "${VERSION}" ]]; then echo "!! could not resolve ${REF}" >&2; exit 1; fi
echo ">> resolved ${VERSION}"

SRC="$(go env GOMODCACHE)/${UPSTREAM}@${VERSION}"
if [[ ! -d "${SRC}" ]]; then echo "!! module cache missing ${SRC}" >&2; exit 1; fi

echo ">> syncing source into ${HERE} (excluding tests/examples) ..."
# Preserve our own metadata files; replace everything else with upstream.
rsync -a --delete \
  --exclude='*_test.go' --exclude='testdata/' --exclude='.github/' --exclude='_example/' \
  --exclude='PROVENANCE.md' --exclude='update.sh' --exclude='UPSTREAM-CHANGES.patch' \
  "${SRC}/" "${HERE}/"
chmod -R u+w "${HERE}"
rm -rf "${HERE}/_example"

echo ">> new SHA-256s (paste into PROVENANCE.md):"
shasum -a 256 "${HERE}/sqlite3-binding.c" "${HERE}/sqlite3-binding.h" | sed "s#${HERE}/##"

# Regenerate UPSTREAM-CHANGES.patch: the fork's code-only delta vs its mattn merge-base, excluding
# the amalgamation (whose integrity is anchored by the SHA-256s above). Requires network + git.
# Only works when REF is a branch name (e.g. sqlite3mc-2.2.7), not a tag/pseudo-version.
echo ">> regenerating UPSTREAM-CHANGES.patch (fork code delta vs mattn) ..."
PATCHTMP="$(mktemp -d)"
git clone --no-checkout --filter=blob:none "https://${UPSTREAM}" "${PATCHTMP}/repo" >/dev/null 2>&1
git -C "${PATCHTMP}/repo" remote add upstream https://github.com/mattn/go-sqlite3
git -C "${PATCHTMP}/repo" fetch --no-tags upstream >/dev/null 2>&1
BASE="$(git -C "${PATCHTMP}/repo" merge-base "origin/${REF}" upstream/master)"
git -C "${PATCHTMP}/repo" diff "${BASE}" "origin/${REF}" -- . \
  ':(exclude)sqlite3-binding.c' ':(exclude)sqlite3-binding.h' ':(exclude)sqlite3ext.h' \
  > "${HERE}/UPSTREAM-CHANGES.patch"
rm -rf "${PATCHTMP}"
echo ">> patch regenerated; base (mattn) commit = ${BASE}"

echo ">> done. Update PROVENANCE.md Origin+integrity+delta-base, then: (cd ../../ && go build -tags assert ./... && go test ./db/...)"
