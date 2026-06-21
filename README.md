# Yeelight

Desktop controller for Yeelight LAN smart bulbs, written in Go. Discovers bulbs
over SSDP, controls power/brightness/color, and syncs the ambient light to your
**screen** or **system audio**.

Two binaries share one protocol library (`pkg/yeelight`):

- **`cmd/ui`** — Qt desktop GUI. The main app.
- **`main.go`** (repo root) — tiny CLI demo: discover bulbs, run a hardcoded
  RGB color flow. Not the GUI.

Bulbs must have **LAN Control** enabled (Yeelight phone app → device settings).

## Build & run

Requires Go 1.26+.

```bash
go run ./cmd/ui     # the GUI
go run .            # the CLI demo
go build ./...      # build everything
```

`cmd/ui` links Qt via `github.com/therecipe/qt` (cgo) — it needs the Qt dev
libraries and a working `therecipe/qt` environment to compile. `pkg/yeelight`
and the CLI demo are plain Go, no Qt.

## Using the GUI

1. Launch — it auto-scans the LAN. Each bulb gets a tab. Click **Scan Device**
   to rescan if a bulb is missing.
2. Per bulb: toggle power, set brightness/color for the **main** and **ambient**
   light. Controls only appear for features the bulb advertises.
3. **Screen Sync** — pick a monitor + interval; the ambient light tracks that
   screen's average color.
4. **Music Sync** — the ambient light pulses to system audio (brightness =
   loudness, hue = tone).
5. Closing the window hides to the **system tray** (if available); the app keeps
   running. Quit from the tray.

Settings, themes, and per-bulb sync prefs persist to
`<UserConfigDir>/yeelight/config.toml`. A full log is at
`<UserConfigDir>/yeelight/yeelight.log` (and the in-app **Log** tab).
`<UserConfigDir>` is `~/.config` (Linux), `~/Library/Application Support`
(macOS), or `%AppData%` (Windows).

## Platform requirements

The core controls (discovery, power, brightness, color) work everywhere. Screen
Sync and Music Sync each need an OS capture backend:

| | Screen Sync | Music Sync (system audio) |
|---|---|---|
| **Linux / Wayland** | `grim` | `parec` (PulseAudio/PipeWire) |
| **Linux / X11** | ImageMagick (`import`) | `parec` (PulseAudio/PipeWire) |
| **Windows** | built-in (GDI) — no setup | built-in (WASAPI loopback) — no setup |
| **macOS** | built-in (`screencapture`)¹ | **BlackHole** virtual device² |

¹ macOS prompts for **Screen Recording** permission on first capture; sync stays
black until you grant it (System Settings → Privacy & Security → Screen
Recording).

² macOS has no built-in audio loopback. Install the free
[BlackHole](https://github.com/ExistentialAudio/BlackHole) virtual device, then
create a **Multi-Output Device** (Audio MIDI Setup) containing both your speakers
and BlackHole and select it as system output — that keeps sound audible while
Music Sync captures it. Without this, Music Sync captures the microphone instead.

Install the Linux tools from your package manager, e.g.:

```bash
# Debian/Ubuntu
sudo apt install grim imagemagick pulseaudio-utils
# Arch
sudo pacman -S grim imagemagick libpulse
```

`parec` works under both Wayland and X11 (PulseAudio is independent of the
display server).

### Firewall (Linux)

Discovery needs inbound UDP for SSDP replies. The app best-effort opens the port
via `ufw` (escalating through `pkexec` with a graphical prompt) on launch. If you
use a different firewall, allow inbound UDP on the discovery port yourself.

## Notes

- Sync sends are throttled to the bulb's quota; when a bulb supports **music
  mode** the app opens a music-mode channel to push high-rate updates without
  hitting the ~60-commands/min limit.
- Enabling Screen Sync and Music Sync on the *same* bulb at once isn't
  recommended — they drive the same ambient light and open independent sessions.

See `CLAUDE.md` for architecture and `docs/yeelight-spec.md` for the protocol.
