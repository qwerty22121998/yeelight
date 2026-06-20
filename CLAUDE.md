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
go test ./...                  # tests (none exist yet)
```

`cmd/ui` links against Qt via `therecipe/qt` cgo bindings — it needs the Qt dev libraries + a working `therecipe/qt` environment to compile. The `pkg/yeelight` library and root CLI have no Qt dependency and build with plain Go.

## Architecture

### `pkg/yeelight` — protocol library (no UI deps)

The Yeelight LAN protocol: SSDP discovery over UDP, then a persistent TCP control connection per device.

- **Discovery** (`discover.go`): `Discover()` sends an `M-SEARCH` SSDP datagram to the multicast group (default `239.255.255.250:1982`), collects replies until `Timeout`, dedupes by IP, parses each into an `Info` (id/model/fw/name + `Methods` capability set), then dials a `Device` per result.
- **Device** (`device.go`): holds a TCP conn to `<ip>:55443`. A `listen()` goroutine reads newline-delimited JSON. Request/response are correlated by integer `id`:
  - `SendCommand` assigns an id via `genID()` (wraps at `maxId`), writes JSON+`\r\n`, then blocks in `waitResponse` on a per-id channel in the `pending` map (5s timeout).
  - Messages with `id == 0` are **async notifications**; `method == "props"` ones feed `updateData` → `mergeData` (only non-nil fields overwrite) and pulse `updatedChan` (non-blocking).
  - **`SendCommand` rejects any method not in `d.Methods`** — the device's advertised capability set gates everything.
- **Data model** (`data.go`): `Data` is an all-pointer struct (nil = unknown). `mergeData`/`mergeValue` merge partial updates. `Info` carries identity + `Methods map[Method]bool`.
- **`FetchProps`** (`device.go`): one `get_prop` with `AllProperties`, then maps the result slice back **positionally** — result index *i* corresponds to `AllProperties[i]`. Keep `AllProperties` order and the parse switch in sync when adding properties.
- **Commands**: build with `C(method, params...)` (`command.go`). Method/Property/Effect names live in `const.go`. Color flows: build `ColorFlow{...}.Build()` and splat into `C()` (`color_flow.go`).

### `cmd/ui` — Qt GUI

- `main.go`: boots `QApplication`, makes `UI`, calls `RenderMain`.
- `ui.go` (`UI`): top-level. "Devices" tab (one sub-tab per discovered device) + "Settings" tab. "Scan Device" runs `Discover` in a goroutine with a fake progress bar (`loadingTimeout`) and rebuilds device tabs.
- `device_controller.go` (`DeviceUI`): per-device controls split into Main Light / Ambient Light / Info groups. **Each control is rendered only if `device.Methods[...]` advertises support.** A goroutine watches `device.Updated()` and calls `update()` to sync slider/color state from `device.Data`.
- `setting.go` (`Setting`): shared config — discovery (SSDP addr, timeout) + effect (smooth/sudden, duration ms). Passed by pointer into every `DeviceUI`, so command effects read live settings.
- `util.go`: RGB int packing (`(r<<16)|(g<<8)|b`) and a reflection-based `allNotNil` guard used before dereferencing `Data`'s pointers.

## Conventions

- Capability-gate every device command — check `device.Methods[method]` (the UI does this per-widget; `SendCommand` enforces it again) rather than assuming a method exists.
- `Data` fields are pointers; guard with `allNotNil` (UI) before dereferencing.
- Adding a property: update `const.go` (`Property` const + `AllProperties`), `data.go` (`Data` field + `mergeData`), and the `FetchProps` parse switch — positional, so order matters.
- Logging is `log/slog` throughout the library.
