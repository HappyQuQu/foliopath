# FolioPath 路线图

## 状态

路线图按依赖和可验证出口组织，不承诺发布日期。用户已于 2026-07-23 确认 RQ-001～RQ-014 全部采用 A；后续阶段按该基线执行，新的范围变化必须遵守[交付与架构治理](architecture/delivery-governance.md)。路线图阶段是实施顺序，不是发布版本；阶段 0～5 和发布门槛全部通过后才能称为稳定 MVP。
总进度和交接点见[开发任务清单](task-list.md)，具体执行分别见
[后端开发清单](backend-task-list.md)与[前端开发清单](frontend-task-list.md)；任务清单不能
改变本文阶段顺序或冻结 scope。

当前已通过 Stage 0 Gate；认证、媒体库管理与可靠扫描后端已经分别通过 Backend Ready，
Stage 3 目录与媒体浏览已通过 S3-001 Contract Ready，S3-002 catalog 排序/cursor 已完成，
后端当前进入 S3-003 目录树、详情与 breadcrumb 实现。生产前端仍按独立清单推进：

- [FS-02 SQLite/generation](spikes/fs-02-sqlite-generation.md) 已通过当前正确性 scope。
- [FS-01 路径边界](spikes/fs-01-path-boundary.md) 已通过 Darwin 与原生 Linux amd64/arm64 路径矩阵、
  Linux `openat2` 同/跨设备及 self-bind mount 拒绝和真实 HTTP test harness 子范围；生产
  handler/auth 转入首个受保护 API Backend Gate，发布 volume/unmount 转入 FS-05/Release
  Gate；FS-01 的 Stage 0 路径范围已通过。
- [FS-03 媒体矩阵](spikes/fs-03-media-matrix.md) 的原生双架构 govips/FFmpeg Stage 0 范围通过；
  生产任务、更多敌意输入、浏览器与最终发布门槛转入后续 Gate。
- [FS-04 目标容量](spikes/fs-04-capacity-baseline.md) 的 Linux/arm64 四核、4 GiB
  Stage 0 扫描/索引范围、Linux RSS 与三档趋势通过；代表性存储、媒体、FTS、HTTP 与前端
  容量按 S0-106 转入后续 Gate。[FS-05](spikes/fs-05-runtime-recovery.md) 双架构运行、
  恢复和失败关闭范围通过。
- Go 运行骨架、认证和 7 个媒体库管理 operation 已接入真实 composition root；权威
  OpenAPI、TypeScript/sqlc 生成、摘要锁、语义兼容检查和双架构 CI 工作流已经建立。
  媒体库 Backend Gate 已覆盖认证 HTTP、SQLite、路径/mount、并发/幂等、重启移除和原媒体
  不变。可靠扫描已通过 Backend Ready，React 产品前端和正式 Dockerfile仍未完成；
  不能把后端 Gate 扩张为阶段 2 Integrated Done 或可发布版本。

## 阶段 0：基线同步与可行性验证

目标：把高成本未知项变成明确决定或实测证据。

范围：

- 已完成：确认并同步[需求清单](requirements-checklist.md)中的认证、格式、扫描、隐藏目录、搜索、布局、目标规模等决策。
- 完成[可行性研究](feasibility-study.md)要求的路径/扫描、SQLite、媒体工具、Range 与前端虚拟化
  spike；FS-01 的 Stage 0 路径范围、FS-02 当前 scope 已完成。生产 HTTP/发布容器证据按
  [S0-105](gates/MVP-2026-07-23/s0-105-gate-order.md)由后续 Gate 强制。
- 把 spike 数据写回性能预算、支持矩阵与风险登记。
- 已以确认 PRD、用户流程和 UI 方向建立第一版权威 `api/openapi.yaml`；强制离线解析、
  引用/结构、ECMAScript pattern、AST/Schema 与 scanner/migration 选择性关键不变量检查
  （`queued`、`animated`、可空 `startedAt`）已通过；这不是完整领域实现一致性证明。
  Redocly 无结构错误，当前保留两条 health endpoint 4xx 规则 warning；TypeScript 生成类型、
  唯一 client、摘要锁、语义兼容入口与 PR 漂移工作流已建立；本地自比较和首次真实
  base-branch PR 比较均通过。

出口条件：

- 没有会改变数据模型或部署拓扑的未决 MVP 问题。
- 所有 No-Go 条件均已排除，或有明确缩减范围的 fallback。
- [开发就绪清单](development-readiness.md)达到阶段 1 的 Definition of Ready。

## 阶段 1：可运行基础与工程护栏

目标：建立可重复构建、迁移、测试和发布的最小纵向骨架。

范围：

- 优先建立 Go 运行骨架：应用生命周期、配置、结构化日志、请求 ID、liveness/readiness 和优雅停机。
- SQLite 连接、WAL 配置和 Goose 嵌入迁移（已有 FS-02 实验实现），以及待建立的 `sqlc` 确定性生成检查。
- 复用阶段 0 已建立的 OpenAPI 源、统一错误、TypeScript 生成类型、唯一 client、摘要锁、
  语义兼容与确定性漂移检查；在首个 handler 前补齐 Go/SQL 生成，并让真实 PR 基线比较通过。
- 建立管理员初始化、登录/会话校验、退出和 CSRF 的基础后端边界；阶段 5 再完成限流、代理与发布加固。
- React/Vite 只先建立已批准阶段 1/首个安全切片直接需要的最小应用壳、路由、token、共享原语和组件工作台，并设置时间盒；可丢弃原型不进入生产 import graph。认证后端达到 Backend Ready 后完成初始化、登录和退出 UI。其他业务 feature 必须等待各自的 Backend Ready Gate。
- 扩展现有统一开发命令、架构/生成检查和基础 CI，使其覆盖运行应用、`sqlc` 与 E2E。
- 最小多阶段 Docker 镜像，非 root、amd64/arm64 构建验证。
- 合成 fixture、临时文件系统和首批路径安全测试；当前已有运行时临时目录、Linux/arm64
  openat2/mount 与 HTTP harness，仍需许可明确的完整媒体 fixture、Linux/amd64 和双架构
  发布容器矩阵。

出口条件：

- 空数据目录可以启动、迁移、服务前端并优雅退出。
- 管理员初始化、登录、会话校验、退出、CSRF 和未认证业务拒绝完成 S4 证据；阶段 5 只做发布安全加固。
- 本地与 CI 的格式、生成、lint、单元和最小容器 smoke test 通过。
- 没有依赖真实媒体或开发机绝对路径的测试。

## 阶段 2：媒体库与可靠扫描

目标：完成最核心的数据入口，并证明不破坏原文件和旧索引。

范围：

- `/library` 安全目录选择器、唯一名称媒体库的创建/列表/重命名/删除；根路径不可原地修改。
- 真实路径边界、symlink 规则、重叠库检测和只读语义。
- generation 完整扫描、协作取消、任务恢复、错误摘要、创建/启动扫描，以及默认 24 小时可配置计划扫描。
- 所有可读目录及直接/递归计数、系统派生目录跳过统计、固定媒体类型识别、元数据探测和离线恢复。
- 后端 OpenAPI、能力服务和故障集成证据通过后，再实现设置页与首次创建媒体库流程。

出口条件：

- 新增、修改、删除、重命名、权限失败、断连和中断测试通过。
- 失败/取消/离线扫描无法触发陈旧记录清理。
- 删除媒体库后 fixture 原文件逐字节不变，配置和派生状态可清理。
- 目标规模扫描 spike 满足已确认资源预算，或已启用接受的降级方案。

## 阶段 3：文件夹浏览与缩略图体验

目标：交付 FolioPath 的核心“文件夹就是相册”浏览闭环。

范围：

- 先完成 catalog/thumbnail 后端契约、游标与缓存任务证据，再实现媒体库切换、包含空目录及计数的目录树、面包屑和直达 URL。
- 当前目录与递归浏览、稳定游标；普通目录自然名称升序，递归视图修改时间倒序。
- JPEG/PNG/WebP/GIF 缩略图、MP4/MOV/MKV 视频封面、有界队列、缓存失效和默认 10 GiB LRU 配额。
- 默认自适应网格、可记忆瀑布流、虚拟滚动、骨架/空/错误/离线状态。
- 桌面固定侧栏、移动目录抽屉、中英界面、键盘和 reduced-motion/contrast 支持。

出口条件：

- 浏览刷新后恢复媒体库、目录和递归状态；长滚动无无界 DOM 增长。
- 缩略图损坏或源文件变化可以重建，不影响媒体索引。
- 扫描与生成缩略图并行时，API 和 UI 仍满足已确认交互预算。
- 关键流程完成可访问性检查，无阻断级键盘或读屏问题。

## 阶段 4：查看器、搜索与视频

目标：完成日常发现和查看能力闭环，为开发预览提供完整核心体验。

范围：

- 图片查看器的适应视口、缩放/平移、1:1、前后导航、全屏、基本信息和焦点恢复；不含完整 EXIF、显式下载按钮或移动滑动手势。
- 浏览器兼容视频的原文件 Range 播放与视频封面。
- 文件名/路径 FTS 搜索，默认当前媒体库并可切换当前目录（可递归）与全部媒体库；类型/修改时间过滤和稳定排序。
- 不支持/损坏媒体的清晰状态与重试边界。
- 根据确认范围加入必要元数据字段；不扩展到 AI、转码或文件管理。

出口条件：

- 图片与视频 E2E、Range/取消、搜索分页和返回滚动位置通过。
- 支持格式矩阵在双架构镜像上以 fixture 验证，并与文档一致。
- 浏览器不兼容编码的行为明确，不出现无限加载或错误承诺。

## 阶段 5：发布安全与运维闭环

目标：把功能构建提升为可安全安装、升级和恢复的版本。

范围：

- 完成并加固阶段 1 已建立的单管理员初始化、登录、会话、退出和 CSRF；补齐限流、代理信任、安全审计和恢复，不提供匿名局域网模式。
- Compose、固定 UID/GID、健康检查、版本信息和升级说明。
- 数据库备份/恢复、迁移失败、磁盘已满和挂载断连演练。
- SBOM、许可证、依赖/镜像安全扫描和多架构发布流水线。
- 文档与实际镜像、环境变量、端口、格式和限制最终核对。

出口条件：

- [测试策略](testing-strategy.md)中的发布门槛全部通过。
- 没有未处置的高影响安全或数据完整性风险。
- 从空环境部署、从备份恢复和从上一版本升级均可重复成功。
- README 不再包含镜像、端口或命令占位符。

## MVP 之后

以下方向需要独立需求和必要 ADR，不提前把复杂度放入 MVP：

- 文件系统 watcher（只作增量提示，完整扫描仍是正确性基线）。
- SVG、HEIC/HEIF、AVIF、RAW 等扩展格式、视频转码和更丰富 EXIF。
- 收藏、评分、历史、时间线、地图与重复检测。
- 分享链接、多用户和细粒度授权。
- 上传、备份、文件整理或任何写入原媒体的能力。
- 多实例、外部数据库、独立 worker 或分布式队列。

## 跨阶段原则

- 每阶段优先完成一条可验证的纵向路径，不同时铺开所有包和页面。
- 每条切片遵循 S0 Architecture Ready → S1 Contract Ready → S2 Backend Ready → S3 Frontend Ready → S4 Integrated Slice Done；Release Ready 只在版本级统一判断，前端业务实现不反向发明后端契约。
- 当前 MVP 之外的新能力默认进入后续版本；加入冻结 MVP 必须有 scope trade-off 或明确的 scope-budget exception。
- 每项规则、事务、任务状态机、错误映射和共享 UI 语义只有一个所有者，重复实现先重构所有权再继续功能。
- 架构或产品基线改变时先更新 ADR/PRD，再调整任务，不让代码成为唯一事实。
- 每项范围都需要成功、空、载入、错误、离线/取消和权限边界中的适用状态。
- 性能数字来自固定环境的可重复测量，不用猜测值代替验收标准。
- 发现无法满足 Go/No-Go 条件时先缩减格式、规模或非核心功能，不牺牲只读原件与路径安全。
