# MCC

Minecraft Config Converter in Go.

## Current Scope

- `ItemsAdder -> CraftEngine`
- `Nexo -> CraftEngine`
- Package analysis with a shared `PackageIndex`
- Cross-platform desktop UI built with `Fyne`

## Run

```bash
go run ./cmd/mcc
```

The desktop UI uses `cgo`. On Windows you need a GCC toolchain in `PATH`.

## Build

```bash
go build ./...
go build -o dist/mcc-desktop.exe ./cmd/mcc
```

## GitHub Actions

Push to `rewrite/go`, or trigger the `Build Go` workflow manually.
The workflow installs MinGW on a Windows runner, runs `go test ./...`, and uploads `dist/mcc-desktop.exe`.

## Structure

- `cmd/mcc`: desktop entrypoint
- `internal/desktopui`: Fyne desktop shell
- `internal/workflow`: zip analyze/convert workflow used by the desktop app
- `internal/analyzer`: package analysis
- `internal/converter/iace`: ItemsAdder conversion
- `internal/converter/nexoce`: Nexo conversion
- `internal/packageindex`: shared package scan index
- `internal/fileutil`: shared file I/O helpers
