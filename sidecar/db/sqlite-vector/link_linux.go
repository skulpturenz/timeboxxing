//go:build linux

package sqlitevector

// The bundled sqlite-vector extension references libm math symbols (e.g. exp)
// but does not link libm itself — it expects the host process to provide them,
// the way the system sqlite3 CLI (linked with -lm) does. The Go sidecar does
// not otherwise pull in libm, and the linker's default --as-needed would drop a
// plain -lm because no Go/cgo code references a libm symbol directly. Forcing
// libm into DT_NEEDED with --no-as-needed keeps it loaded in the process's
// global symbol scope, so dlopen of the extension at startup can resolve exp et
// al. Without this the sidecar fails on Linux with:
//   failed to load extension ... vector.so: undefined symbol: exp
//
// macOS and Windows resolve these symbols already, so this is Linux-only.

// #cgo LDFLAGS: -Wl,--no-as-needed -lm
import "C"
