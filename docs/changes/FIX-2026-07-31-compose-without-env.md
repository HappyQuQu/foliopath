# FIX-2026-07-31：Compose 无 `.env` 快速部署

- 类型：已批准发布与部署切片内的例行易用性优化
- Gate：沿用 Stage 5 单容器、只读媒体、非 root 与本地 SQLite 数据目录合同
- 行为：中英文 README 直接提供完整 Compose 配置和参数表，不添加额外的 `.env` 提示
- 数据：README 快速示例使用 Docker 自动创建的 `./data` 宿主目录，不要求手动创建目录
- 默认值：仓库 Compose 在无 `.env` 时使用 `evanqu/foliopath:latest`、`./library`、`./data` 和 `UTC`；镜像内部默认监听 `0.0.0.0:8080`
- 覆盖：保留 `.env.example`，供自定义镜像、媒体根、数据根、宿主监听地址、端口和时区；`FOLIOPATH_LISTEN` 不向普通部署暴露
- 安全：README 快速示例只保留必须的 `/library:ro`；仓库正式 Compose 继续保留非 root、只读容器根、capability 丢弃、`no-new-privileges` 和受限 tmpfs
- 回归：发布文档架构测试固定双语无 `.env` 说明、参数表及 Compose 默认值
