# Sidebar tab duplication on launch

## Symptom

On launch, parts of the sidebar (commonly the SSH / Netshares / VPN rows) and a
slice of the body content appear a second time, lower down in the terminal,
below the live frame's footer. Scrolling or any subsequent re-render makes the
ghost copy disappear.

The duplicated block is **not** the same as a "stale row left behind" bug — the
ghost is double-spaced (one blank line inserted between every original line),
which is the giveaway as to what's happening.

See `docs/sidebar-duplication-screenshot.png` if a fresh screenshot has been
captured for comparison; otherwise the original report's screenshot lives in
the chat thread that produced this fix.

## Root cause

Two things stack on launch:

1. `Init()` (`internal/tui/model.go`) batches a flurry of commands —
   `spinner.Tick`, `checkStatusCmd`, one `checkNetshareCmd` per netshare, one
   `checkWGCmd` per wireguard tunnel. Each completion triggers a re-render, so
   in the first ~100–300 ms there are many `View()` calls in quick succession,
   and the frame's body height fluctuates as Checking → Connected/Disconnected
   transitions remove spinners.

2. The very first `View()` call happens **before `WindowSizeMsg` arrives**, so
   `m.width` is still `0`. The previous code then clamped to 80 columns and
   emitted a frame at an unknown terminal geometry. When the real
   `WindowSizeMsg` lands and the next frame is laid out at the actual width,
   Bubble Tea's standard renderer has to reconcile the new frame against a
   committed frame that doesn't match the terminal's real cursor state.

The reconciliation isn't a clean overwrite — it ends up cursor-moving and
emitting line feeds that scroll the prior frame's lower portion *down* into
the rows below the live view, with extra LFs between lines. That's the
double-spaced ghost the user sees.

## Fix

Two small changes in `internal/tui/`:

- `view.go` — `View()` returns `""` while `m.width == 0`. No frame is committed
  until the renderer knows the real terminal geometry.
- `update.go` — on the first `WindowSizeMsg`, return `tea.ClearScreen` along
  with the model update. Belt-and-suspenders against any pre-resize paint that
  did sneak through.

The "first" detection in `update.go` keys off `m.width == 0` — the initial
zero-value of the field — so it costs no extra state.

## Why we ruled out alternatives

- **Body height stabilization (forcing a fixed body height).** Would mask but
  not address the underlying problem, and adds layout rigidity that's awkward
  given content panes have legitimately different sizes.
- **Switching renderers / Bubble Tea versions.** The standard renderer is fine
  once it's given a consistent geometry to reconcile against. Switching is a
  larger lift for a smaller problem.
- **Adding a sleep / debounce at startup.** Hides the race but doesn't fix it,
  and noticeably delays the first paint.

## How to reproduce if it returns

1. Launch `yabinui` in a wide terminal (so the gap between the clamped 80-col
   layout and the real width is large).
2. Have several netshares + wireguard tunnels configured so `Init()` dispatches
   many parallel checks.
3. Look for double-spaced ghost text below the live footer immediately on
   launch. Scrolling clears it.

If it does come back, the first thing to check is whether something in the
render path is committing output before `m.width` is set — a regression of the
`View()` guard would re-open exactly this bug. After that, look at whether
new commands have been added to `Init()` that could widen the startup
re-render storm.

## Terminal-emulator caveat

This is sensitive to terminal emulator + alt-screen behavior. The original
report came from an alacritty-style screenshot. If a future report shows the
bug *only* in one specific emulator, treat that emulator's alt-screen handling
as a contributing factor — the fixes above should still help, but a full
repro may need that specific emulator.
