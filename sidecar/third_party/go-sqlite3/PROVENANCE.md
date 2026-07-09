# Vendored SQLCipher-capable go-sqlite3 fork

This directory is an **in-repo vendored fork** of `mattn/go-sqlite3`, extended with
[SQLite3MultipleCiphers](https://utelle.github.io/SQLite3MultipleCiphers/) so the sidecar
can encrypt its SQLite database at rest. The sidecar consumes it through a path `replace`
in [`../../go.mod`](../../go.mod):

```
replace github.com/mattn/go-sqlite3 => ./third_party/go-sqlite3
```

A `replace` (not a direct import) is required because `github.com/goptics/sqliteq` opens the
same database file via a hardcoded `sql.Open("sqlite3", …)`. The `replace` redirects **every**
importer — our `db` package, `golang-migrate`'s sqlite3 driver, and `sqliteq` — to this one
encryption-capable driver, all registered under the driver name `"sqlite3"`.

## Origin

| Field | Value |
|-------|-------|
| Upstream repo | https://github.com/jgiannuzzi/go-sqlite3 |
| Branch | `sqlite3mc-2.2.7` |
| Commit | `2c447b9a2806` (2026-02-27) |
| Go pseudo-version | `v1.14.35-0.20260227142656-2c447b9a2806` |
| Base | mattn/go-sqlite3 v1.14.35 |
| Amalgamation | SQLite **3.51.2** + SQLite3MultipleCiphers **2.2.7** |

## Integrity (verify after any update)

```
sqlite3-binding.c  sha256 = 841e5837334041b5806237711d339527d738bdbe9ce64df035dbc145881cd24a
sqlite3-binding.h  sha256 = 50819ff087b56d905606b179639c67a5ca0b4f661b41715966dfcf8bcd063fe0
```

## Upstream delta (`UPSTREAM-CHANGES.patch`)

[`UPSTREAM-CHANGES.patch`](UPSTREAM-CHANGES.patch) is the fork's **code-only** diff versus canonical
upstream mattn/go-sqlite3 — an audit aid so a reviewer can see exactly what SQLite3MultipleCiphers
adds (the `_cipher` / `_key` / `PRAGMA rekey` glue in `sqlite3.go`, the cipher tests, and the
amalgamation build tooling under `upgrade/`) without reading 14 MB of vendored source.

- **Base:** mattn/go-sqlite3 commit `208733130eafb38bd1f570eb7172267a3e64843d` (the merge-base of the
  `sqlite3mc-2.2.7` branch with mattn `master`) → fork tip `2c447b9a2806`.
- **Excluded:** `sqlite3-binding.c`, `sqlite3-binding.h`, `sqlite3ext.h` — the wholesale
  SQLite3MultipleCiphers amalgamation swap, which is not line-reviewable and whose integrity is
  anchored by the SHA-256s above instead.
- **Not a reproduction patch.** Because the amalgamation is excluded it will not `git apply` back
  into the full tree; it is for review only. It also reflects the full upstream branch delta, so it
  includes files our vendored copy omits (`*_test.go` and the nested `upgrade/` module that Go's
  module packaging drops).
- **Regenerate:** re-run `./update.sh` (it rebuilds this patch automatically), or see the
  `git merge-base` + `git diff` recipe there.

## Properties relied upon

- **Encryption is compiled in by default** — it is baked into `sqlite3-binding.c`, gated only
  by `//go:build cgo`. There is **no cipher build tag**; the existing `-tags assert` builds are
  unchanged. If CGO is ever disabled the sidecar will not compile (intended — a plaintext build
  must never ship).
- **Self-contained crypto** — SQLite3MultipleCiphers bundles its own AES/ChaCha
  implementations; no OpenSSL or system `libsqlcipher` is linked (important for the Windows
  MSYS2/MinGW build).
- **Load-extension enabled by default** — required to load the `sqlite-vector` extension
  against the cipher core (validated).
- **Cipher scheme used by Timeboxxing:** `sqlcipher` (authenticated AES-256-CBC + per-page
  HMAC), selected via the DSN param `_cipher=sqlcipher`; the raw 32-byte key is supplied via
  `_key=x'<64 hex>'`.

## Updating

Run `./update.sh <branch-or-pseudo-version>` (see `update.sh`), then re-record the SHA-256s and the
**Upstream delta** base commit above, review the diff (and the regenerated `UPSTREAM-CHANGES.patch`)
in a normal PR, and re-run `go test ./...` in `sidecar/`. The amalgamation is committed as C
**source**, so every change is code-reviewable.

## License

MIT — see `LICENSE` (same terms as SQLite3 and SQLite3MultipleCiphers, both public-domain /
MIT). The `.c/.h` amalgamation carries the upstream SQLite / SQLite3MultipleCiphers licensing.
