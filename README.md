# MCC (Minecraft Configuration Converter) 工具

MCC 是一个用于分析和转换 Minecraft 服务器插件配置的工具，主要专注于将 **ItemsAdder** 配置迁移到 **CraftEngine** 格式。它提供了一个基于 Web 的界面，便于文件管理和可视化转换过程。

## 主要功能

- **包分析**: 分析上传的 `.zip` 文件，检测配置格式（ItemsAdder, CraftEngine, Nexo）、内容类型（贴图、模型、物品）和完整性。
- **转换引擎**: 将 ItemsAdder 配置转换为 CraftEngine 格式。
- **资源迁移**: 自动迁移贴图和模型文件到目标目录结构。
- **Web 界面**: 基于 Flask 构建的简单拖拽式界面。
- **可执行文件构建**: 包含构建脚本，可将应用程序打包为独立的 `.exe` 文件。

## 