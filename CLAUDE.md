# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o webs.exe .    # build
go run .                  # run (or: go run main.go config.go)
go mod tidy               # sync dependencies
```

No tests, linters, or CI exist in this project.

## Architecture

A single-binary live-reload static file server. Two Go source files, package `main`:

- **`main.go`** — all server logic. `WebsServer` (hub pattern) holds connected WebSocket clients, runs an event loop (`run()`), and implements `http.Handler`. On file changes (`fsnotify`), broadcasts `{"command":"reload","path":"/"}` to every connected browser via WebSocket at `/ws`. Falls back through ephemeral ports (49152–49251) if the preferred port is busy.
- **`config.go`** — reads `webs.toml` from the executable's directory via `BurntSushi/toml`. Defines `Config` struct with `Port`, `Dir`, `WatchExts`.

## Config priority

CLI flags `--port` / `--dir` override `webs.toml` values. The `watch_exts` option has no CLI equivalent — it only comes from the toml file. An empty `watch_exts` means "watch all file types."

## Key dependencies

- `github.com/fsnotify/fsnotify` — filesystem watcher
- `github.com/gorilla/websocket` — WebSocket server
- `github.com/BurntSushi/toml` — TOML config parsing
