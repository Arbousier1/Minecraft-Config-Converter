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
- `POST /api/convert`
- `ItemsAdder -> CraftEngine` conversion path, including resource-pack migration, furniture handling, armor handling, and complex item templates
- `Nexo -> CraftEngine` conversion path with resource-pack migration
- `GET /api/download/:filename`
- `POST /api/heartbeat`
- `POST /api/shutdown`
- YAML loading with UTF-8, GBK, Latin-1, and tab-indentation fallback
- Package structure analysis ported from `src/analyzer.py`

Not ported yet:
- Cross-source parity testing against the Python implementation on a wider sample of real packs
- Additional converter targets beyond the current CraftEngine output

## Run

```bash
go run ./cmd/mcc
```

The server listens on `http://127.0.0.1:5000`.

## Notes

- The branch now compiles locally with `go build ./...`.
- The existing Python implementation is intentionally kept in place as a reference during migration validation.
