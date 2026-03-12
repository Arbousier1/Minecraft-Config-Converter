# Go Branch Status

Branch: `rewrite/go`

## Status

This branch is now desktop-first.

Implemented:

- Cross-platform desktop UI with `Fyne`
- Shared `PackageIndex` scan step
- `ItemsAdder -> CraftEngine`
- `Nexo -> CraftEngine`
- YAML compatibility loader
- Shared file I/O helpers for extraction, copy, and zip output
- Benchmark baseline for package indexing and conversion pipeline

## Run

```bash
go run ./cmd/mcc
```

## Build

```bash
go build ./...
go build -o dist/mcc-desktop.exe ./cmd/mcc
```

## Notes

- The real desktop UI is built when `cgo` is enabled.
- The repository keeps a `!cgo` desktop stub so `go test ./...` still passes in environments without a GCC toolchain.
