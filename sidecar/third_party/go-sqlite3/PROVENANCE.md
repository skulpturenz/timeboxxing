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

## How this fork is produced

We **regenerate** the fork ourselves onto the latest upstream rather than vendoring a third-party
branch verbatim (jgiannuzzi's fork has no cipher branch newer than `sqlite3mc-2.2.7`). It is
composed of three upstream pieces, combined by [`update.sh`](update.sh):

| Piece | Source | Version |
|-------|--------|---------|
| Go driver | [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | **v1.14.47** — commit `693de126` (2026-06-21) |
| Cipher amalgamation (`sqlite3-binding.c/.h`) | [utelle/SQLite3MultipleCiphers](https://github.com/utelle/SQLite3MultipleCiphers) release | **2.3.5** — SQLite core **3.53.2** (2026-06) |
| Cipher Go glue (`_cipher`/`_key`/`PRAGMA rekey` in `sqlite3.go`) | jgiannuzzi's SQLite3MC work, carried as [`UPSTREAM-CHANGES.patch`](UPSTREAM-CHANGES.patch) | applied onto the mattn base |

The mattn stock SQLite core (3.53.2) and the SQLite3MC core (3.53.2) match, so there is no
version skew between the Go driver and the cipher amalgamation.

## Integrity (verify after any update)

```
sqlite3-binding.c  sha256 = 7661b7229620c6d3553f6eaf7b5b1a6782082417de3dae27006301215b245103
sqlite3-binding.h  sha256 = de0523d757e7c810236e69ab1a41fe805092e76a928db4a995bb756e408b9ea2
```

The SQLite3MC amalgamation is the `sqlite3mc-2.3.5-sqlite-3.53.2-amalgamation.zip` asset from the
utelle 2.3.5 release, wrapped in mattn's `#ifndef USE_LIBSQLITE3` guard by the `upgrade/` tool.

## Upstream delta (`UPSTREAM-CHANGES.patch`)

[`UPSTREAM-CHANGES.patch`](UPSTREAM-CHANGES.patch) is the fork's **code-only** diff versus canonical
mattn/go-sqlite3 **v1.14.47** — an audit aid showing exactly what the cipher integration adds (the
`_cipher` / `_key` / `PRAGMA rekey` glue in `sqlite3.go`, the cipher tests, and the amalgamation
build tooling under `upgrade/`) without reading the vendored amalgamation.

- **Base:** mattn/go-sqlite3 `v1.14.47` (`693de126`). **Excluded:** `sqlite3-binding.c`,
  `sqlite3-binding.h`, `sqlite3ext.h` — the amalgamation, whose integrity is anchored by the
  SHA-256s above instead.
- It is a **review** artifact, not a reproduction patch (the amalgamation is excluded, so it will
  not `git apply` into the full tree), and it reflects the full upstream branch delta, so it
  includes files our vendored copy omits (`*_test.go` and the nested `upgrade/` module).
- This same patch is the vehicle `update.sh` re-applies onto a newer mattn release when updating.

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
  `_key=x'<64 hex>'`. Stable across SQLite3MC releases, so an on-disk database is portable across
  fork updates.

## Updating

Run [`./update.sh`](update.sh) to regenerate onto the latest mattn + latest SQLite3MC (it clones
mattn, re-applies `UPSTREAM-CHANGES.patch`, runs the `upgrade/` tool, and re-vendors here). Then
re-record the SHA-256s and the version table above, review the diff (and the regenerated
`UPSTREAM-CHANGES.patch`) in a normal PR, and run `go test -tags assert ./db/...` in `sidecar/`. The
amalgamation is committed as C **source**, so every change is code-reviewable.

## License

MIT — see `LICENSE` (same terms as SQLite3 and SQLite3MultipleCiphers, both public-domain /
MIT). The `.c/.h` amalgamation carries the upstream SQLite / SQLite3MultipleCiphers licensing.
