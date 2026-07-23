# FolioPath 开发任务清单

## 用途与执行规则

本文是 `MVP-2026-07-23` 的可执行工作清单，不新增产品范围。产品边界以
[冻结 scope manifest](releases/MVP-2026-07-23-scope.md)为准，实施顺序以
[路线图](roadmap.md)为准，架构与交付门禁以
[交付与架构治理](architecture/delivery-governance.md)为准。

当前里程碑：**Stage 0 / Conditional Go**。在 Stage 0 Gate 复审通过前，只允许完成本节列出的
可行性、契约、CI、容器和架构护栏任务，不开始生产 feature handler 或业务 UI。

执行约束：

- 严格按阶段和依赖顺序工作；同一时刻只推进一个主要后端切片。
- 每个产品切片依次经过 `Architecture Ready → Contract Ready → Backend Ready →
  Frontend Ready → Integrated Done`。
- 后端接口、失败语义和集成证据完成后，前端才通过生成 client 消费该能力。
- 任务只有在“实现、自动化验证、文档/风险同步”全部完成后才能勾选。
- 新需求默认进入 MVP 之后；不得直接追加到本清单。要进入冻结 MVP，必须先走正式范围变更。
- `[x]` 表示已有仓库证据，`[ ]` 表示尚未完成；“工作流已定义”不等于 CI 已通过。

## 当前关键路径

```text
Stage 0 剩余证据
  → Stage 0 Gate 复审
  → Go 可运行基础与认证后端
  → 媒体库与可靠扫描后端
  → 媒体库设置 UI
  → 文件夹浏览与缩略图
  → 搜索与查看器
  → 发布加固与稳定 MVP
```

## Stage 0：关闭可行性与工程基础条件

### 已完成基线

- [x] `S0-001` 确认 RQ-001～RQ-014 全部采用 A，并冻结 MVP scope revision 1。
- [x] `S0-002` 建立系统上下文、模块所有权、前后端架构、交付治理和 fitness functions。
- [x] `S0-003` 接受模块化 Go 单体、SQLite、单一 `/library`、后端优先和共享前端系统 ADR。
- [x] `S0-004` 建立权威 `api/openapi.yaml`、离线契约测试和 Redocly 交叉检查。
- [x] `S0-005` 建立确定性 TypeScript 类型、唯一 Web API client、摘要锁和语义兼容入口。
- [x] `S0-006` 建立 Go 架构测试、契约测试、单元/集成/race 测试和统一 `Makefile` 入口。
- [x] `S0-007` 建立固定 Node/npm/Go 工具链、lockfile 和 high-severity npm audit。
- [x] `S0-008` 定义原生 linux/amd64 与 linux/arm64 Go、媒体和 mount-boundary CI jobs。
- [x] `S0-009` FS-02 当前 SQLite/generation 正确性范围通过。
- [x] `S0-010` FS-01 原生 Linux amd64/arm64 `openat2` mount 边界与真实 HTTP harness
  的 Stage 0 可行性范围通过。
- [x] `S0-011` FS-03 Darwin/arm64 与 Linux/amd64 QEMU 合成 FFmpeg 子范围通过。
- [x] `S0-012` FS-04 Linux/arm64 目标容量扫描/索引子范围通过。

### 当前待办

- [x] `S0-101` 在真实 PR 上运行首次 CI，并保存原生 amd64/arm64 job 结果。
  - 依赖：当前变更形成可审查分支和 PR。
  - 完成证据：[PR #1 CI run 29985018814](https://github.com/HappyQuQu/foliopath/actions/runs/29985018814)
    的 Go、race、Web contract、media matrix、mount boundary 7 个 jobs 全绿。
- [x] `S0-102` 验证真实 base branch → 当前 OpenAPI 的语义兼容检查。
  - 依赖：`S0-101`。
  - 完成证据：PR #1 的 Generated web contract job 对真实 base branch 执行 `oasdiff`
    成功；工具配置使用 `--fail-on WARN` 阻断检测到的破坏性变化。
- [ ] `S0-103` 完成 FS-03 libvips/govips 图片链路 spike。
  - 范围：JPEG/PNG/WebP/GIF 元数据、方向、色彩、透明度、动画和缩略图限制。
  - 完成证据：固定依赖、合成 fixture、超时/损坏输入结果写回 FS-03。
- [ ] `S0-104` 在原生 linux/amd64 与 linux/arm64 环境运行相同媒体 fixture。
  - 依赖：`S0-101`、`S0-103`。
  - 当前证据：合成 FFmpeg/webp 子矩阵已在 PR #1 两架构通过。
  - 完成证据：加入 `S0-103` 的 libvips/govips 后，两架构完整 codec、失败行为和依赖版本对比记录。
- [x] `S0-105` 解决 Stage 0 与 FS-01“生产 handler 证据”的门禁顺序。
  - 当前冲突：Stage 0 禁止生产 feature handler，但 FS-01 完整关闭条件包含生产 handler。
  - 完成证据：[S0-105 Gate allocation record](gates/MVP-2026-07-23/s0-105-gate-order.md)
    将生产 HTTP/auth 证据分配到首次受保护 API Backend Gate，将发布挂载/运行期故障分配到
    FS-05/Release Gate；要求保持强制。
- [ ] `S0-106` 完成 FS-04 剩余的代表性存储、RSS、FTS/keyset 和趋势预算。
  - 完成证据：可重复环境、预算、结果与超限 fallback 写回 FS-04 和风险登记。
- [ ] `S0-107` 完成 FS-05 双架构镜像与恢复 spike。
  - 范围：多阶段 Debian slim、非 root、`/library:ro`、`/app/data`、健康检查、优雅停机、
    SQLite 备份/恢复、升级和磁盘满。
  - 完成证据：两架构 smoke、恢复演练和失败注入报告。
- [ ] `S0-108` 生成 Go/npm/镜像 SBOM，审查 libvips、FFmpeg codec 与许可证。
  - 依赖：`S0-103`、`S0-107`。
  - 完成证据：依赖清单、构建选项、许可证结论和未决风险。
- [ ] `S0-109` 复审风险 R-002～R-016，并更新 owner、状态和 fallback。
- [ ] `S0-110` 复审 Stage 0 Gate。
  - 依赖：`S0-101`～`S0-109` 完成，或每个未完成项有正式缩减范围/延期 Gate。
  - 完成证据：Gate 结论从 `Conditional Go` 更新为允许进入 Stage 1，或明确 `No-Go`。

## Stage 1：后端优先的可运行基础

### Go 运行骨架

- [ ] `S1-001` 创建最小 `cmd/foliopath`，只负责参数解析、启动和退出码。
- [ ] `S1-002` 创建 `internal/app` composition root，集中组装依赖和生命周期。
- [ ] `S1-003` 实现经过验证的配置加载；固定 `/library`、`/app/data` 和单 HTTP 端口边界。
- [ ] `S1-004` 实现结构化日志、request ID、错误脱敏和优雅停机。
- [ ] `S1-005` 实现 `/health/live`、`/health/ready` 与 `/api/v1/status`。
- [ ] `S1-006` 从空数据目录启动并执行嵌入 migration；验证重复启动和迁移失败行为。
- [ ] `S1-007` 建立 `sqlc` 配置、查询源、生成目录和确定性 `generate-check`。
- [ ] `S1-008` 增加运行应用的集成测试、取消测试和最小容器 smoke。

### 单管理员认证后端

- [ ] `S1-101` 固定初始化、登录、会话、退出和 CSRF 的 OpenAPI/数据契约。
- [ ] `S1-102` 实现管理员首次初始化状态机和密码安全存储。
- [ ] `S1-103` 实现安全 Cookie 会话、过期、轮换与退出失效。
- [ ] `S1-104` 实现 CSRF、防缓存和全部业务 API 默认拒绝未认证访问。
- [ ] `S1-105` 覆盖错误脱敏、重复初始化、错误密码、过期会话和并发请求测试。
- [ ] `S1-106` 记录认证切片 `Backend Ready` Gate。

### 最小前端基础

- [ ] `S1-201` 仅在认证后端达到 `Backend Ready` 后建立 React/Vite 应用壳。
- [ ] `S1-202` 建立唯一 token、主题、排版、焦点和 reduced-motion 基础。
- [ ] `S1-203` 建立唯一 Button、Input、Dialog、Toast、FormField 等共享原语及组件工作台。
- [ ] `S1-204` 实现初始化、登录和退出 UI，只通过生成 client/领域 adapter 访问 API。
- [ ] `S1-205` 加入组件测试、axe、键盘、主题和响应式验证。
- [ ] `S1-206` 完成认证纵向切片 E2E，并记录 `Integrated Done` Gate。

## Stage 2：媒体库与可靠扫描

### 媒体库后端

- [ ] `S2-001` 固定目录选择、创建、列表、详情、改名和移除的契约与失败语义。
- [ ] `S2-002` 实现 `/library` 安全目录枚举；所有 I/O 只经过 `internal/files`。
- [ ] `S2-003` 实现唯一名称、相对根、不可变根和重叠根校验。
- [ ] `S2-004` 实现媒体库创建、改名、离线状态、重试和只删除派生数据的移除。
- [ ] `S2-005` 覆盖 traversal、symlink、nested mount、TOCTOU、重叠、离线和权限失败。
- [ ] `S2-006` 证明移除媒体库前后 fixture 原文件逐字节不变。
- [ ] `S2-007` 记录媒体库后端 `Backend Ready` Gate。

### 扫描后端

- [ ] `S2-101` 固定扫描创建、状态、取消、issues 和默认 24 小时计划的契约。
- [ ] `S2-102` 完成有界 generation 扫描服务和全局任务队列。
- [ ] `S2-103` 索引全部可读目录，包括空目录，并维护直接/递归计数。
- [ ] `S2-104` 实现媒体候选识别、fingerprint、增量 upsert 和成功后陈旧清理。
- [ ] `S2-105` 实现取消、离线、权限失败、重启恢复和失败保留旧索引。
- [ ] `S2-106` 覆盖并发扫描、队列上限、深目录、损坏拓扑和容量回归。
- [ ] `S2-107` 记录扫描后端 `Backend Ready` Gate。

### 媒体库设置 UI

- [ ] `S2-201` 实现设置中的媒体库列表、状态、空态和错误态。
- [ ] `S2-202` 实现“选择 `/library` 下安全目录 → 命名 → 创建 → 自动扫描”流程。
- [ ] `S2-203` 实现改名、移除确认、离线重试和扫描状态/取消。
- [ ] `S2-204` 验证键盘、移动端、长路径、中英文、加载和故障恢复。
- [ ] `S2-205` 完成媒体库/扫描纵向切片 E2E 与 `Integrated Done` Gate。

## Stage 3：目录浏览、缩略图与递归视图

### 后端

- [ ] `S3-001` 固定目录树、面包屑、当前目录和递归媒体列表的游标契约。
- [ ] `S3-002` 实现稳定排序、opaque cursor、查询指纹和请求取消。
- [ ] `S3-003` 实现包含空目录及直接/递归计数的目录树。
- [ ] `S3-004` 实现 govips/FFmpeg 媒体探测、缩略图/视频封面和损坏状态。
- [ ] `S3-005` 实现有界媒体任务队列、fingerprint 失效、默认 10 GiB LRU 和磁盘余量保护。
- [ ] `S3-006` 覆盖损坏媒体、像素炸弹、超时、取消、磁盘满和并发限制。
- [ ] `S3-007` 记录浏览/缩略图后端 `Backend Ready` Gate。

### 前端

- [ ] `S3-101` 实现桌面侧栏、移动抽屉、目录树、面包屑和可复制直达 URL。
- [ ] `S3-102` 实现当前目录/递归模式及对应稳定 URL 状态。
- [ ] `S3-103` 实现默认自适应网格、可记忆瀑布流和统一虚拟化集合。
- [ ] `S3-104` 实现 skeleton、空、错误、离线、缩略图 pending/failed 状态。
- [ ] `S3-105` 验证十万媒体规模下 DOM、请求数量、滚动与焦点恢复预算。
- [ ] `S3-106` 完成核心浏览 E2E 与 `Integrated Done` Gate。

## Stage 4：搜索与媒体查看器

- [ ] `S4-001` 固定文件名、类型、日期、路径和三种搜索范围的契约。
- [ ] `S4-002` 实现 SQLite FTS/keyset 搜索、稳定排序、取消和离线语义。
- [ ] `S4-003` 完成搜索后端正确性、容量和 `Backend Ready` Gate。
- [ ] `S4-004` 实现统一 SearchInput、filter、结果列表、URL 状态和无结果恢复。
- [ ] `S4-005` 固定资产详情、原内容、HEAD、单 Range、条件请求和 416 契约。
- [ ] `S4-006` 实现图片查看、GIF 策略、原生视频、封面及不可播放/损坏状态。
- [ ] `S4-007` 验证目标桌面/移动浏览器、键盘、触摸、Range 和错误降级。
- [ ] `S4-008` 完成搜索/查看器 E2E 与 `Integrated Done` Gate。

## Stage 5：发布加固

- [ ] `S5-001` 完成非 root、只读媒体、最小权限和默认安全配置的最终镜像。
- [ ] `S5-002` 完成 linux/amd64 与 linux/arm64 构建、启动、健康和媒体矩阵。
- [ ] `S5-003` 完成反向代理信任、Secure Cookie、CSRF、限流和安全响应头验证。
- [ ] `S5-004` 完成备份、恢复、升级、回滚、强杀、磁盘满和缓存重建演练。
- [ ] `S5-005` 完成 10 万媒体/1 万目录/4 核/4 GiB 的完整产品容量验收。
- [ ] `S5-006` 完成全量单元、race、集成、组件、axe、浏览器 E2E 和视觉回归。
- [ ] `S5-007` 生成最终 SBOM，完成依赖漏洞、许可证和 FFmpeg 构建配置审查。
- [ ] `S5-008` 校对 README、Compose、部署、备份、支持格式和已知限制。
- [ ] `S5-009` 关闭或正式接受发布阻断风险，形成 Release Candidate Gate。
- [ ] `S5-010` 完成稳定 MVP 发布 Gate；只有此项通过后才能宣称 FolioPath 可发布。

## 明确不进入当前 MVP

以下内容不是待办，不得在上述任务中顺带实现：

- 上传、移动、改名、编辑或删除原始媒体；
- 相册、时间线、地图、AI 分类、重复检测、收藏、评分或分享链接；
- HEIC/HEIF、AVIF、RAW、SVG 缩略图；
- 视频转码、文件系统 watcher、多用户或细粒度权限；
- Redis、PostgreSQL、独立 worker、微服务、多实例共享写入；
- 每个媒体库单独配置 Docker volume，或在 `/library` 下使用后代嵌套挂载。

这些能力若未来进入计划，必须创建新的版本 scope 或正式 Change Record，不直接扩充本清单。

## 维护方式

完成任务时同时：

1. 勾选对应任务，并链接实际测试、Gate、报告或 PR；
2. 更新受影响的 OpenAPI、migration、ADR、风险和用户文档；
3. 记录实际执行命令、平台及未执行项；
4. 若任务拆分，子任务沿用原 ID 前缀，不改变其验收结果；
5. 若任务取消或延期，不删除历史，写明目标版本、原因和替代 Gate。
