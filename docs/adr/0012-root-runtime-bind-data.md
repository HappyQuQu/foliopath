# ADR-0012：默认 root 运行与免初始化 bind 数据目录

## 状态

已接受（2026-08-01）；替代 Stage 5 候选中固定 UID/GID `65532:65532` 的运行身份决定

## 决策角色

- 产品：确认默认部署必须直接使用 Docker 自动创建的 `./data` bind 目录
- 架构：保持单容器、单进程、SQLite 与固定 `/library`、`/app/data` 边界
- 安全／发布：接受 root runtime 的扩大风险，继续强制原媒体只读与路径失败关闭

## 背景与驱动因素

`FR-DEP-001～004` 要求普通 Docker Compose 部署无需额外初始化。固定非 root 身份与
`./data:/app/data` bind mount 冲突：短语法会创建 root-owned 宿主目录，但不会把它改成
UID/GID 65532，应用因无法创建 SQLite 数据库而重启。要求宿主机 `chown`、改用 named
volume 或在 Compose 覆盖用户均被产品操作者明确拒绝。

## 备选方案

1. 保留非 root 并要求首次 `chown`：安全边界较强，但不满足零初始化部署，拒绝。
2. 默认 named volume：可由 Docker 初始化，但不满足明确的宿主 bind 数据目录要求，拒绝。
3. root 启动器修复权限后降权：需要新增特权启动路径和身份切换实现，且会修改宿主目录
   所有权，复杂度与副作用更高，拒绝。
4. 应用进程默认以 root 运行：直接兼容 Docker 创建的 root-owned bind 目录，选择。

## 决策

- 正式镜像的 Dockerfile 不声明 `USER`，从固定的 root distroless 基础镜像继承 OCI 用户
  `0`；Compose 和快速启动示例不设置 `user`。
- `/app/data` 继续是唯一可写持久边界，允许直接使用 `./data:/app/data`。
- `/library` 必须继续以 `:ro` 挂载；API、`internal/files`、`openat2`、无 symlink/no-xdev
  和单一媒体根规则不变。root 身份不得成为跨越这些边界的理由。
- 权威 Compose 继续丢弃 capabilities、启用 `no-new-privileges` 和只读容器根；简化示例
  可以依赖 Docker 默认容器根，但必须保留 `/library:ro` 并警告只用于受信 LAN。
- 不向应用添加修改宿主权限、任意路径或媒体写入能力。

## 后果

Docker 自动创建的 `./data` 可直接承载 SQLite，不再需要 `chown`、named volume 或 Compose
用户覆盖。代价是媒体解析器或应用漏洞获得 UID 0 进程上下文；在未丢弃 capabilities、未启用
只读根的简化部署中影响更大。只读媒体 mount、容器隔离、认证和不可信网络隔离因此更重要。
既有非 root 双架构证据只保留为历史记录，不能证明新的 root 候选。

## 验证与复审

- 对应 fitness function：修订后的 `AF-012`。
- 证据：架构测试固定 Dockerfile/Compose 均无用户覆盖；release image/Compose smoke 使用
  root-owned 数据 bind 并验证数据库、health、媒体只读与优雅退出。
- 需要复审：Docker/Compose 支持可移植的 bind ownership 初始化，或产品重新接受非 root、
  named volume、权限初始化命令或特权降权启动器。
- 替代关系：替代 ADR-0009 和 Stage 5 Gate 文档中“发布镜像必须非 root”的部分；
  ADR-0009 的 `/library`、`openat2`、no-symlink 与 no-xdev 决策保持有效。
