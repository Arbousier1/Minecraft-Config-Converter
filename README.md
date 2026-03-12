# MCC

Minecraft Config Converter 的 Go 版本。

## Current Scope

- `ItemsAdder -> CraftEngine`
- `Nexo -> CraftEngine`
- 包结构分析
- 嵌入式 Web UI
- 结果下载、心跳、关闭接口

## Run

```bash
go run ./cmd/mcc
```

默认监听 `http://127.0.0.1:5000`。

## Build

```bash
go build ./...
go build -o dist/mcc-webview2.exe ./cmd/mcc
```

## GitHub Actions

Push 到 `rewrite/go`，或在 GitHub 上手动触发 `Build Go` workflow。
工作流会在 Windows runner 上先构建前端，再生成并上传 `dist/mcc-webview2.exe` artifact。

## Structure

- `cmd/mcc`: 程序入口
- `internal/server`: HTTP 服务
- `internal/analyzer`: 包分析
- `internal/converter/iace`: ItemsAdder 转换
- `internal/converter/nexoce`: Nexo 转换
- `internal/fileutil`: 共享文件 I/O
- `web`: 前端静态资源
