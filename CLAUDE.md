# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Yeelight smart-bulb controller in Go. Two binaries share one protocol library (`pkg/yeelight`):

- **`cmd/ui`** — Qt desktop GUI (primary app). Uses `github.com/therecipe/qt` bindings.
- **`main.go`** (root, `package main`) — standalone CLI demo: discover devices, run a hardcoded RGB color flow on the ambient light. Not the GUI.

## Commands

```bash
go build ./...                 # build everything
go run .                       # run root CLI demo (main.go)
go run ./cmd/ui                # run the Qt GUI
go vet ./...                   # vet
go test ./...                  # tests (pkg/yeelight, pkg/audio; cmd/ui has none — Qt)
```

`cmd/ui` links against Qt via `therecipe/qt` cgo bindings — it needs the Qt dev libraries + a working `therecipe/qt` environment to compile. The `pkg/yeelight` library and root CLI have no Qt dependency and build with plain Go.

## Architecture

### `pkg/yeelight` — protocol library (no UI deps)

The Yeelight LAN protocol: SSDP discovery over UDP, then a persistent TCP control connection per device.

- **Discovery** (`discover.go`): `Discover()` sends an `M-SEARCH` SSDP datagram to the multicast group (default `239.255.255.250:1982`), collects replies until `Timeout`, dedupes by IP, parses each into an `Info` (id/model/fw/name + `Methods` capability set), then dials a `Device` per result.
- **Device** (`device.go`): holds a TCP conn to `<ip>:55443`. A `listen()` goroutine reads newline-delimited JSON. Request/response are correlated by integer `id`:
  - `SendCommand` assigns an id via `genID()` (wraps at `maxId`), writes JSON+`\r\n`, then blocks in `waitResponse` on a per-id channel in the `pending` map (5s timeout).
  - Messages with `id == 0` are **async notifications**; `method == "props"` ones feed `updateData` → `mergeData` (only non-nil fields overwrite) and pulse `updatedChan` (non-blocking).
  - **Device state is private and mutex-guarded.** The props live in `d.data` (was a public `Data` field) behind `dataMu`, written by the `listen()` goroutine and `FetchProps`. Read it from anywhere via **`Snapshot()`**, which returns a shallow copy (`mergeValue` only ever *replaces* pointer fields, never mutates in place, so a struct copy is a consistent snapshot). `ApplyLocal(Data)` optimistically merges caller-known state (e.g. a power we just set) for firmware that won't echo it back — without it a later props message re-applies the stale value and fights the user. `updateData` also mirrors `power`↔`main_power` (same light; firmware emits one or the other) so readers can rely on `Power` alone.
  - **`SendCommand` does NOT pre-reject on `d.Methods`** — the advertised support list is a hint, not a gate (some firmware supports methods, notably `set_music`, that it omits from the list). The command is always sent and the device is the authority; a genuinely unsupported method comes back as an `*Error` for which `IsUnsupported(err)` (`command.go`) reports true.
- **Data model** (`data.go`): `Data` is an all-pointer struct (nil = unknown). `mergeData`/`mergeValue` merge partial updates. `Info` carries identity + `Methods map[Method]bool`.
- **`FetchProps`** (`device.go`): one `get_prop` with `AllProperties`, then maps the result slice back **positionally** — result index *i* corresponds to `AllProperties[i]`. Keep `AllProperties` order and the parse switch in sync when adding properties.
- **Commands**: build with `C(method, params...)` (`command.go`). Method/Property/Effect names live in `const.go`. Color flows: build `ColorFlow{...}.Build()` and splat into `C()` (`color_flow.go`).
- **Music mode** (`device_music.go`): `StartMusic()` opens a local TCP listener, sends `set_music` (action 1) telling the bulb to dial back, and blocks until it connects. The returned `*Music` bypasses the bulb's ~60-cmd/min rate limit: `Send()` writes **fire-and-forget** (no reply on the music channel), so it's safe to push dozens of updates/sec. `Stop()` closes the conn and sends `set_music` (action 0). Reaches bulbs that support music mode but omit `set_music` from their advertised list.

### `pkg/audio` — system audio capture (Linux only, no UI deps)

`Capture(ctx)` shells out to `parec` on `@DEFAULT_MONITOR@` (PulseAudio/PipeWire — mirrors `pkg/screen`'s subprocess approach) and returns a `<-chan Tick`. Each `Tick` reduces a ~93ms window to `Level` (0..1 RMS loudness, AGC-normalized via a decaying running peak → brightness) and `Tone` (0..1 zero-crossing rate, bass→treble → hue). No FFT — ZCR is a cheap tone proxy (`analyze`, marked `ponytail:`; swap an FFT band-split in if hue mapping feels wrong). Channel closes on ctx cancel or recorder exit.

### `cmd/ui` — Qt GUI

- `mainthread.go`: **Qt forbids touching widgets off the GUI thread.** `runOnUI(f)` pushes a closure onto `uiQueue`; `startUIDispatch()` (called once from `main`, on the GUI thread) runs a 16ms `QTimer` that drains the queue. Any background goroutine (discovery, `device.Updated()` watchers, async sends, music sync) that wants to mutate a widget MUST wrap it in `runOnUI`. (`ponytail:` poll-drain — swap for a moc signal only if latency matters.)
- `main.go`: `setupLogging()` tees `slog` into the in-app Log tab (`logs`) and, if it can open it, `<UserConfigDir>/yeelight/yeelight.log` (append-only). Then boots `QApplication`, calls `startUIDispatch()`, makes `UI`, calls `RenderMain`.
- `ui.go` (`UI`): top-level. "Devices" tab (one sub-tab per device, `emptyState` placeholder when none) + Settings + Log tabs. `scan()` runs off the GUI thread: closes old devices, `Discover`s (indeterminate busy progress bar via `runOnUI`), then `reRenderDevices` fetches props off-thread and rebuilds tabs on-thread. Auto-scans once on launch. `showStatus(msg)` shows a transient status-bar message (thread-safe). Window close hides to the system tray when one is available (app keeps running), else quits.
- `device_controller.go` (`DeviceUI`): per-device controls split into Main Light / Ambient Light / Info groups. **Each control is rendered only if `device.Methods[...]` advertises support.** A goroutine watches `device.Updated()` and `runOnUI(update)` to sync slider/color from `device.Snapshot()` (`setSlider` skips a slider the user is currently dragging). `send(ctx, cmd, onErr)` runs `SendCommand` off-thread (it blocks up to 5s), reports failures to the status bar, and runs `onErr` back on the GUI thread (e.g. to revert a slider). **Music Sync** button → `runMusicSync`: `audio.Capture` ticks drive the ambient light via an `rgbSender` (prefers `StartMusic` for high-rate sends, falls back to plain `bg_set_rgb`); a `waveViz` shows the live waveform.
- `waveviz.go` (`waveViz`): scrolling level-over-time audio visualizer (custom `paintEvent`, fixed-size ring of level+color samples). `push`/`reset` must be called on the GUI thread.
- `log.go`: the **Log tab**. `logSink` is the `io.Writer` `slog` tees into; it keeps the last 1000 records in a ring and pushes live records (via `runOnUI`) to the attached `logView`, which filters by level + search text and colors by level.
- `setting.go` (`Setting`): shared config — discovery (SSDP addr, timeout) + effect (smooth/sudden, duration ms). Passed by pointer into every `DeviceUI`, so command effects read live settings.
- `util.go`: RGB int packing (`(r<<16)|(g<<8)|b`), `hsvToColorInt` (music sync maps tone→hue), and a reflection-based `allNotNil` guard used before dereferencing `Data`'s pointers.

## Conventions

- The advertised `device.Methods` set is a UI rendering hint (the UI shows a control per-widget when a method is advertised), NOT a send-time gate. `SendCommand` always sends; let the device decide and use `IsUnsupported(err)` to detect a "method not supported" reply.
- **Qt thread safety**: touch widgets only on the GUI thread. From any other goroutine, wrap widget mutations in `runOnUI`. Conversely, never call a blocking network op (`SendCommand`, `StartMusic`, `FetchProps`, `Discover`) on the GUI thread — run it in a goroutine (use `DeviceUI.send` for commands).
- Read device props via `device.Snapshot()` (returns a copy); there is no public `Data` field. `Data` fields are pointers; guard with `allNotNil` (UI) before dereferencing.
- Adding a property: update `const.go` (`Property` const + `AllProperties`), `data.go` (`Data` field + `mergeData`), and the `FetchProps` parse switch — positional, so order matters.
- Logging is `log/slog` throughout the library.
