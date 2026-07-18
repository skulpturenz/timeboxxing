# `monitor` — foreground capture pipeline

The `monitor` module captures what the user is doing right now: it polls the OS
for the active window, detects idle, decorates each sample with derived context
(app metadata, browser tab/URL, location), and hands a stream of *change events*
to the timeline. It is the source of every observation that later becomes a
transition/usage event.

Everything downstream depends on one type — [`monitor.ForegroundProcess`](monitor.go)
— flowing through five cooperating packages.

## End-to-end data flow

```mermaid
flowchart LR
    OS[OS window APIs] -->|Poll| T[platform.Tracker]
    T -->|WindowInfo| C[monitor core<br/>poll/tick]
    I[idle.IdleDetector] -->|seconds idle| C
    C -->|ForegroundProcess| RB[(ring buffer<br/>Stream)]
    RB -->|GetChan| R[reporter<br/>dedup + fan-out]
    R -->|change events| E[enrichment pipeline<br/>stack.Stack]
    E -->|enriched| P[timeline projector]
```

The pipeline is assembled in [`app/app.go`](../app/app.go) →
`startForegroundProjection` ([app.go:113-136](../app/app.go#L113-L136)):

1. `stack.Stack()` builds the enrichment `Enricher` + the permission list.
2. `monitor.New(ctx, Options{Permissions})` starts the poll goroutine and
   requests those permissions.
3. `reporter.From(ctx, m.Stream)` starts the single consumer of the ring buffer.
4. `pubsub.Subscribe("timeline")` yields a change-event channel that a goroutine
   drains, `enrich`-es, and projects into the timeline event store.

Shutdown is a single `ctx` cancel that cascades: poll loop stops the ring buffer
→ reporter goroutine closes subscriber channels → the `range events` loop exits.

## Package map

| Package | Path | Role |
| --- | --- | --- |
| `monitor` | [`monitor.go`](monitor.go) | Core producer: poll loop, `ForegroundProcess`, ring buffer |
| `platform` | [`platform/`](platform/) | Per-OS active-window tracker (`Tracker`) |
| `idle` | [`idle/`](idle/) | Per-OS idle detection (`IdleDetector`) |
| `permission` | [`permission/`](permission/) | Shared permission vocabulary (stdlib-only leaf) |
| `reporter` | [`reporter/`](reporter/) | Pub/sub fan-out with per-subscriber change-dedup |
| `enrichment` | [`enrichment/`](enrichment/) | `Enricher` combinator kit (`Pipe`, `Or`) |
| `enrichment/app_metadata` | [`enrichment/app_metadata/`](enrichment/app_metadata/) | Friendly name, category, icon (local + Flathub/Winget feeds) |
| `enrichment/browser` | [`enrichment/browser/`](enrichment/browser/) | Browser detection + tab title/URL via CDP |
| `enrichment/location` | [`enrichment/location/`](enrichment/location/) | Lat/long + public IP |
| `enrichment/stack` | [`enrichment/stack/`](enrichment/stack/) | Assembles the default pipeline + permission set |
| `internal/wlproto` | [`internal/wlproto/`](internal/wlproto/) | Generated Wayland protocol bindings (`wlr`, `plasma`, `ext`) |

> **Go 1.26 note:** this module (`go 1.26.4`) uses the `new(expr)` builtin — a
> pointer to a *copy of a value*, not a zero of a type. That is why
> `new(windowInfo.AppName)` and `new(incoming)` appear throughout; there is no
> custom `new` helper.

## Core capture (`monitor.go`)

`ForegroundProcess` ([monitor.go:15-25](monitor.go#L15-L25)) is the unit of data.
Identity fields are **pointers** so "absent" is distinguishable from "empty" — an
idle sample leaves them all `nil`:

```go
type ForegroundProcess struct {
    AppName, AppIdentifier, AppPath *string
    PID                             *int32
    WindowTitle                     *string
    TitleSource                     *platform.TitleSource
    Timestamp                       time.Time
    Idle                            bool
    Enrichments                     map[string]any  // decorated downstream
}
```

`New` ([monitor.go:41-74](monitor.go#L41-L74)) constructs the tracker and idle
detector, requests permissions, applies defaults via `utils.Coalesce`, builds the
ring buffer, and launches the poll goroutine — the `Monitor` is **live on
construction**. Two failure policies differ deliberately:

- **Tracker creation is a hard error** — without a window source there is nothing
  to capture.
- **Idle detection degrades gracefully** to `idle.Nop()` ([monitor.go:47-50](monitor.go#L47-L50))
  — idle is best-effort.

Defaults: poll every **200 ms**, idle after **5 min**, ring buffer size **10**
(≈2 s of samples).

The poll loop ([monitor.go:76-88](monitor.go#L76-L88)) ticks a `time.Ticker` and
stops the ring buffer on `ctx.Done()`. Each `tick`
([monitor.go:90-121](monitor.go#L90-L121)):

- `tracker.Poll` fails → **skip the tick** (no sample).
- Idle (`idleSeconds >= idleAfter`) → emit a minimal `{Idle: true}` sample whose
  `Timestamp` is **back-dated** to when idleness began, then return early.
- Active → build a full `ForegroundProcess` from the `WindowInfo` with an empty
  `Enrichments` map ready for downstream enrichers.

### Ring buffer

`Stream *ringbuffer.RingBuffer[ForegroundProcess]` (from
`github.com/jonoton/go-ringbuffer`) is a **concurrency boundary**: an internal
goroutine serializes all access over channels, so the poll goroutine (producer)
and the reporter goroutine (consumer) never share memory directly. On overflow it
**drops the oldest** sample — acceptable because stale foreground state is worth
less than fresh state.

## Platform layer (`platform/`)

The OS-agnostic contract ([platform.go:114-122](platform/platform.go#L114-L122)):

```go
type Tracker interface {
    Poll(ctx) (WindowInfo, error)      // current foreground window
    Permissions() []permission.Status  // required perms + grant state
}
```

`WindowInfo` ([platform.go:24-32](platform/platform.go#L24-L32)) is the raw
per-tick observation (string fields, not pointers). `TitleSource`
([platform.go:16-21](platform/platform.go#L16-L21)) is a **provenance tag** —
`ax` / `osascript` / `window_api` / `none` — telling downstream consumers which
OS API produced the title and how much to trust it. Every backend runs its result
through `FinalizeWindowInfo` ([platform.go:39-57](platform/platform.go#L39-L57)),
which trims fields, derives a display name from path → identifier → title, and
falls back to `"Unknown app"` only when there is genuine foreground evidence.

`New(ctx, Config)` is defined **once per OS via build tags** — the Go build
selects exactly one. Linux is the only OS with additional *runtime* dispatch.

| OS | File(s) | Mechanism | cgo? | Setup / gotchas |
| --- | --- | --- | --- | --- |
| macOS | `darwin.go` | Accessibility API (`AXUIElementCreateSystemWide` + focused app), `osascript`/System Events fallback | yes (AppKit/ApplicationServices/CoreGraphics) | Needs **Accessibility** grant; without it only the osascript fallback works (which itself needs Automation permission) |
| Windows | `windows.go` | `GetForegroundWindow` + `QueryFullProcessImageName` via `x/sys/windows` lazy DLLs | no | No special permission needed |
| Linux (X11) | `linux_x11.go` | EWMH atoms over `BurntSushi/xgb` (`_NET_ACTIVE_WINDOW`, `_NET_WM_NAME`, `WM_CLASS`, `_NET_WM_PID`) | no | Synchronous poll; used for XWayland too |
| Linux (wlr) | `linux_wayland_wlr.go` | `zwlr_foreign_toplevel_manager_v1` (Sway, Hyprland, KWin, …) | no | Event-driven; **no PID** exposed by toplevel protocol |
| Linux (plasma) | `linux_wayland_plasma.go` | `org_kde_plasma_window_management` | no | KDE fallback when wlr absent; no PID |
| Linux (GNOME) | `linux_gnome.go` + `gnome_extension/` | Bundled Shell extension over D-Bus (Mutter implements no focus protocol) | no | Extension auto-installed but **needs logout/login** to activate; exposes PID via `get_pid()` |
| Linux (fallback) | `linux.go` `degradedBackend` | none | no | Keeps the app alive, explains itself via `Permissions()` |

**Two poll models:** X11 and GNOME poll *synchronously* each tick (XGB query /
D-Bus call); wlr and plasma are *event-driven* with a background reconnecting
goroutine and a mutex-guarded snapshot that `Poll` simply reads. Backend selection
lives in `selectLinuxBackend` ([linux.go](platform/linux.go)) and keys off
`WAYLAND_DISPLAY` / `XDG_SESSION_TYPE` / desktop-environment env vars plus the
compositor's advertised Wayland globals.

## Idle detection (`idle/`)

```go
type IdleDetector interface {
    SecondsSinceLastInput(ctx) (float64, error)
}
```

`Nop()` ([idle.go:20](idle/idle.go#L20)) always returns `(0, nil)` and is the
fallback when detection is unsupported. Each OS provides its own build-tagged
`New`. Two implementation shapes:

- **Pull** (macOS `CGEventSourceSecondsSinceLastEventType`, Windows
  `GetLastInputInfo`, Linux X11 MIT-SCREEN-SAVER): cheap, stateless, query on
  demand.
- **Push** (Linux Wayland): Wayland has no "query idle time" call, so
  `linux_wayland.go` uses the event-driven `ext-idle-notify-v1` protocol (bindings
  in `internal/wlproto/ext`). A long-lived goroutine tracks `idleSince` from
  compositor `Idled`/`Resumed` events, with a reconnect loop; `SecondsSinceLastInput`
  derives elapsed time from the cached state.

Linux `New` ([linux.go:15-32](idle/linux.go#L15-L32)) prefers Wayland, falls
through to X11 (works under XWayland too), then `Nop()` — it **never returns an
error**, always yielding a working detector.

## Reporter (`reporter/`)

`reporter.From` ([pub_sub_reporter.go:25-60](reporter/pub_sub_reporter.go#L25-L60))
spawns the **sole consumer** of the monitor's ring buffer and re-broadcasts to N
subscribers — a fan-in → fan-out bridge. `Subscribe`
([pub_sub_reporter.go:62-79](reporter/pub_sub_reporter.go#L62-L79)) hands out a
**buffered channel of size 1** (there is only ever one foreground process; items
carry timestamps so out-of-order processing is fine).

`publish` ([pub_sub_reporter.go:81-107](reporter/pub_sub_reporter.go#L81-L107))
does two things per subscriber:

1. **Change-dedup** via `isReported`
   ([is_reported.go](reporter/is_reported.go)): skip the sample unless it
   represents a new state. A sample is *new* when it's the first, when idle↔active
   flips, when the PID changes, or when the **window title changes for the same
   PID** — the browser-tab special case ([is_reported.go:22](reporter/is_reported.go#L22)),
   so a new tab in the same browser process counts as an event. Dedup state is
   per-subscriber, so a late subscriber still gets its own first event.
2. **Non-blocking send**: if the subscriber channel is full, drop the *incoming*
   sample and count it — one slow consumer can never stall the publisher or the
   others.

**Two independent drop policies** are deliberate and documented at
[pub_sub_reporter.go:99-101](reporter/pub_sub_reporter.go#L99-L101): the ring
buffer drops the **oldest**, the reporter drops the **newest**. Rationale — if the
foreground keeps changing faster than a consumer drains, that's noise, so dropping
the newest is fine and keeps the publisher non-blocking.

Net effect: subscribers receive **semantic change events**, not the raw 5/sec
sample stream.

## Enrichment (`enrichment/`)

`Enricher` ([enrichment.go:9](enrichment/enrichment.go#L9)) is a **type alias**:

```go
type Enricher = func(ctx, monitor.ForegroundProcess) (monitor.ForegroundProcess, bool)
```

The `bool` means "did I contribute anything." Two combinators:

- `Pipe(...)` — runs **all** enrichers, threading an accumulator (orthogonal
  dimensions: metadata + browser + location all apply).
- `Or(...)` — returns the **first** enricher that succeeds (fallback within one
  dimension).

Enrichers never touch process identity — they only write typed values into the
`Enrichments map[string]any` bag under well-known keys:

| Enricher | Key | Value type | External I/O & caching | Permission? |
| --- | --- | --- | --- | --- |
| `app_metadata` | `"appmetadata"` | `Metadata` (friendly name, description, `Category`, icon path) | Local OS metadata (plist/.desktop/PE) first; Flathub & Winget HTTP feeds as `Or` fallbacks. Icons cached under `<UserCacheDir>/timeboxxing/app-icons/`. **Memoized 15-min TTL** (metadata rarely changes) | none |
| `browser` | `"browser"` | `Tab` (browser, title, URL, domain) | Tab title parsed from window title; URL recovered via **Chrome DevTools Protocol** at `localhost:9222`. Self-rate-limited (500 ms), 1 s timeout, no HTTP retries. **Not memoized** (tab changes constantly) | none |
| `location` | `"location"` | `Environment` (nullable lat/long + public IP) | Location read non-blockingly from an OS provider on a background thread (macOS CoreLocation, Windows WinRT Geolocator; no-op elsewhere). Public IP via `api.ipify.org`, **memoized 1-hour TTL** | **yes** (location) |

The `location` enricher is the only one that guards against the **nil**
`Enrichments` map ([location.go:80-83](enrichment/location/location.go#L80-L83)),
because public IP is machine-wide and can reach here on an *idle* sample (whose map
is nil).

The default pipeline is assembled in one place —
`stack.Stack()` ([stack.go:14-32](enrichment/stack/stack.go#L14-L32)):

```go
enrichment.Pipe(
    enrichment.Or(Memoized(LocalMetadataEnricher), Memoized(FlathubEnricher), Memoized(WingetEnricher)),
    browser.Enrich(browser.NewCDPPoller(9222)),
    location.Enrich(locationProvider, location.NewPublicIPProvider()),
)
```

`Stack` also gathers the permission set — today only `location.Requestable`
contributes one. This factory is the single seam where sources, the CDP port, TTLs,
and permissions are chosen, keeping `app.go` ignorant of enrichment internals.

Memoization uses the [`memo`](../memo/) package (a `go-cache` TTL cache +
`singleflight` to dedup concurrent lookups); **only successes are cached**, and
processes with empty identity bypass the cache.

## Permissions (`permission/`)

A deliberately stdlib-only **leaf package** (imports only `context`) so both
`monitor` and the enrichment layer can depend on it without an import cycle — the
enrichment stack *produces* the permissions it needs and `monitor.New` *requests*
them.

```go
type Permission interface {  // active / requestable — Request triggers the OS prompt
    Name() string; HowToGrant() string; Granted() bool; Request(ctx)
}
type Status struct {  // passive report, returned by Tracker.Permissions() and enrichers
    Name string; Granted bool; HowToGrant string
}
```

There are no per-OS `Permission` implementations here — the package is purely the
shared vocabulary. Trackers report readiness via `[]Status`; the requestable
`Permission` is implemented in the location enricher.

## Wayland bindings (`internal/wlproto/`)

Three sibling subpackages, each `//go:build linux`, each with the same layout:
vendored protocol `*.xml`, generated `*.xml.go`, a hand-written `types.go`
(aliases to `neurlang/wayland/wl` so the generated file compiles standalone), and
`doc.go` with the `//go:generate` directive.

| Subpackage | Protocol | Consumed by |
| --- | --- | --- |
| `wlr` | `wlr-foreign-toplevel-management-unstable-v1` | `platform/linux_wayland_wlr.go` |
| `plasma` | `org_kde_plasma_window_management` | `platform/linux_wayland_plasma.go` |
| `ext` | `ext-idle-notify-v1` | **`idle/linux_wayland.go`** — *not* the tracker |

> **Regeneration gotcha:** `wayland-scanner` does not emit build tags, so after
> regenerating you must manually re-add `//go:build linux` to the generated file
> (noted in each `doc.go`).

## Cross-cutting design themes

- **Graceful degradation is a first-class state.** Idle falls back to `Nop`;
  Linux window tracking falls back through wlr → plasma → GNOME → X11 →
  `degradedBackend`; missing browser CDP or location providers just yield empty
  enrichments. The one hard failure is an un-constructable window tracker.
- **Absent vs empty.** Pointer fields on `ForegroundProcess` + Go 1.26 `new(expr)`
  cleanly express "this sample has no app identity" (idle) vs "the app has no name".
- **Change events, not a sample firehose.** Per-subscriber dedup turns a 5/sec raw
  stream into semantic transitions, with the browser-tab title special case.
- **Two independent drop policies** (buffer drops oldest, reporter drops newest)
  keep every stage non-blocking.
- **Goroutines per running pipeline:** poll loop, ring-buffer `run()`, reporter
  fan-out, timeline consumer — plus, on Wayland, the idle `loop()` and each
  event-driven window backend's reconnect goroutine.
