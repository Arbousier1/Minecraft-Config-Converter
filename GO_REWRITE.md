# Go Branch Status

Branch: `rewrite/go`

## Status

这个分支现在按纯 Go 结构维护，不再保留 Python 参考实现或 Python 打包链路。

已实现：

- Go HTTP 服务入口
- 嵌入式前端资源
- `POST /api/analyze`
- `POST /api/convert`
- `GET /api/download/:filename`
- `POST /api/heartbeat`
- `POST /api/shutdown`
- `ItemsAdder -> CraftEngine`
- `Nexo -> CraftEngine`
- YAML 兼容加载
- 共享文件 I/O 层，用于流式压缩和复制

## Run

```bash
go run ./cmd/mcc
```

## Build

```bash
go build ./...
go build -o dist/mcc-webview2.exe ./cmd/mcc
```
