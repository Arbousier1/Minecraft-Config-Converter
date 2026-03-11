# Go Rewrite Branch

Branch: `rewrite/go`

## Status

This branch starts the Go rewrite without deleting the original Python code.

Implemented:
- Go module setup in `go.mod`
- HTTP server entrypoint in `cmd/mcc/main.go`
- Embedded frontend assets from `web/`
- `GET /`
- `POST /api/analyze`
- `GET /api/download/:filename`
- `POST /api/heartbeat`
- `POST /api/shutdown`
- YAML loading with UTF-8, GBK, Latin-1, and tab-indentation fallback
- Package structure analysis ported from `src/analyzer.py`

Not ported yet:
- `POST /api/convert`
- ItemsAdder to CraftEngine conversion logic
- Nexo to CraftEngine conversion logic
- Packaging converted output

## Run

```bash
go run ./cmd/mcc
```

The server listens on `http://127.0.0.1:5000`.

## Notes

- The current environment used for this rewrite does not have a working Go toolchain installed, so this branch was not compiled locally.
- The existing Python implementation is intentionally kept in place as a reference for the remaining converter port.
