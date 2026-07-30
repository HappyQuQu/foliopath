# FolioPath 开发任务清单

## 先看这里：项目现在到哪了

一句话：**Stage 0～4 已 Integrated Done，Stage 5 发布加固正在进行；候选镜像、
安全 Compose 和原生双架构验收面已经建立；本机 arm64 与指定原生 amd64 服务器
已分别通过候选运行和升级/配对回滚，按操作者决定本轮不等待计费阻断的 amd64 CI。
全量媒体吞吐与缓存水位已完成；真实 Firefox/读屏/移动设备、供应链和 RC Gate
仍未完成。候选供应链扫描仍有 1 Critical / 8 High，是明确 No-Go。**

目前还没有可供用户使用的 FolioPath。开发工作现拆成独立的
[后端清单](backend-task-list.md)与[前端清单](frontend-task-list.md)；本文件只负责总进度、
交接点和共同发布任务。

当前首要切片是 [FTR-UIF-001 生产前端原型一致性](features/frontend-prototype-fidelity.md)，
已进入 MVP revision 4，S0 Architecture Ready 为 Go，S1 Contract Ready 尚未开始。完整执行
见[专用任务清单](features/frontend-prototype-fidelity-task-list.md)；账户、目录 q 和缓存清理
必须先到 Backend Evidence Ready，前端再接真实业务。

后续版本已经确认一个尚未开工的独立切片：
[FTR-VID-001 视频故事板悬停预览](features/video-storyboard-preview.md)。它不进入当前
MVP/RC，执行见[专用任务清单](features/video-storyboard-preview-task-list.md)，必须先后端
达到 `VSP-S2 Backend Evidence Ready`，再开始生产前端接入。`POST-MVP-1` scope 已冻结，
`VSP-S1 Contract Ready` 已 Go，当前正在执行 `VSP-106～113` 后端实现与证据。

另已确认 `POST-MVP-3` 后台任务中心方向，但 scope 尚未冻结。它不进入 MVP/RC，执行见
[FTR-OPS-001 开发任务清单](features/task-center-task-list.md)。当前只允许 S0 scope、
owner 与容量 spike，不得从本清单顺带修改 OpenAPI、migration、后端或前端。

| 阶段 | 用普通话解释 | 状态 | 完成后能看到什么 |
| --- | --- | --- | --- |
| 0. 开工准备 | 定需求、定架构、验证 Docker/SQLite/媒体处理是否可行 | ✅ 已完成 | 确认方案能做，风险有人负责 |
| 1. 后端运行与认证 | Go 服务、数据库启动流程和管理员认证 API | ✅ 后端已完成 | 服务可启动，认证后端达到 Backend Ready |
| 1F. 前端基础与登录 | React 应用壳、设计系统和认证页面 | ✅ Integrated Done | 可以初始化管理员、登录、恢复会话并退出 |
| 2. 媒体库和扫描 | 在设置中选择 `/library` 子目录，创建多个媒体库并扫描 | ✅ Integrated Done | 可以创建、改名、扫描、取消和安全移除媒体库 |
| 3. 文件夹浏览和缩略图 | 展示目录树、图片/视频缩略图和递归浏览 | ✅ Integrated Done | 可以按真实文件夹浏览并非模态预览媒体 |
| 4. 搜索和查看器 | 搜索、筛选、查看大图和播放兼容视频 | ✅ Integrated Done | MVP 的主要使用流程完整 |
| 5. 发布 | 安全、恢复、性能、双架构镜像和部署文档验收 | 🟡 `S5-002/003/004/005/008` 通过，`001/006/007` 阻断 RC | 才可以称为可发布版本 |

### 当前只做什么

当前首要 feature 的交付顺序：

```text
UIF-S0 架构 ✅
  → UIF-S1 账户 / 目录 q / 缓存合同
  → UIF-S2 后端证据
  → UIF-S3 共享壳与真实页面
  → UIF-S4 视觉 / E2E / 容量 / RC 重验
```

前端壳、token、共享组件、reference manifest 和契约 fixture 可以在 S1/S2 同期推进；真实
业务 API 集成必须等待对应 `Backend Evidence Ready`。既有 Stage 1～4 完成证据继续有效，
但不能代替本 feature 的新增合同和一致性 Gate。

### 编号怎么读

- `S1-001`：第一个数字是阶段，后三位是该阶段任务编号。
- `[x]`：已经有代码、测试或评审证据，不只是“讨论过”。
- `[ ]`：尚未完成。
- `Backend Ready`：后端接口、失败处理和测试都通过，前端才可以开始接入。
- `Gate`：阶段验收点；未通过时不能跳到后面的功能。

如果只想看进度，看本节即可。下面是开发和审计使用的详细任务，不需要逐条阅读。

## 用途与执行规则

本文是 `MVP-2026-07-23` 的可执行工作清单，不新增产品范围。产品边界以
[冻结 scope manifest](releases/MVP-2026-07-23-scope.md)为准，实施顺序以
[路线图](roadmap.md)为准，架构与交付门禁以
[交付与架构治理](architecture/delivery-governance.md)为准。

当前共同里程碑：**Stage 5 / 发布加固**。媒体库 `S2-001～007` 已通过 Backend Ready，
扫描 `S2-101～107` 已完成契约、worker、全部可读目录/计数、媒体候选、fingerprint、
增量收敛、故障恢复、容量并发、观察/取消 HTTP 和定时设置，并通过 Backend Ready；
`S3-001` 已固定 root、breadcrumb、direct/recursive scope、排序 tuple、generation cursor
和 offline 语义；`S3-002` 已实现 migration 7、自然排序、严格 keyset、query fingerprint
与取消传播；`S3-003` 已实现目录树、root detail、完整 breadcrumb 和真实认证 HTTP；
`S3-004` 已实现媒体探测、缩略图/视频封面、派生状态和原子缓存发布；`S3-005` 已实现
durable 媒体任务、fingerprint 失效、LRU 与磁盘保护；`S3-006` 已完成敌意媒体、真实
磁盘满和并发资源矩阵；`S3-007` 已接入资产页/详情与认证缩略图并通过 Backend Ready。
`S4-003` 已以 100k/10k 主档验证扫描并发搜索、FTS/短词/全局/keyset、取消和 rebuild；
Stage 4 已通过 Integrated Done 并授权进入 Stage 5。`S5-001A` 已建立嵌入真实 SPA 的
候选镜像基础，`S5-003` 已完成可信代理与网络安全边界；`S5-001B/S5-002` 已建立安全
Compose、媒体/代理 smoke 与原生双架构 CI；`S5-002A/S5-005C` 又在原生 amd64
服务器通过完整候选 smoke 和 31,899 项真实 ZFS 媒体验证。`S5-005A` 已建立完整候选
容量自动化并通过本机 100k/10k 档；指定原生 amd64 服务器又完成 100k 全量媒体吞吐、
cache 90%→80% 水位、持续健康与只读哨兵验证，`S5-005` 已关闭。`S5-007A` 已建立候选
供应链自动化；按操作者决定，本轮 amd64 以指定原生服务器证据替代计费阻断的 CI。
真实 Firefox/物理辅助功能签署、最终镜像 provenance 与漏洞处置尚未通过。

执行约束：

- 严格按阶段和依赖顺序工作；同一时刻只推进一个主要后端切片。
- 每个产品切片依次经过 `Architecture Ready → Contract Ready → Backend Ready →
  Frontend Ready → Integrated Done`。
- 后端和前端使用独立任务清单、代码所有权和 PR；只有 OpenAPI、fixture、Gate 与 E2E 是交接面。
- 后端接口、失败语义和集成证据完成后，前端才通过生成 client 消费该能力。
- 任务只有在“实现、自动化验证、文档/风险同步”全部完成后才能勾选。
- 新需求默认进入 MVP 之后；不得直接追加到本清单。要进入冻结 MVP，必须先走正式范围变更。
- `[x]` 表示已有仓库证据，`[ ]` 表示尚未完成；“工作流已定义”不等于 CI 已通过。

## 当前关键路径

```text
后端：运行基础 → 认证 → 媒体库/扫描 → 浏览/缩略图 → 搜索/内容
                    │              │              │
                    ▼              ▼              ▼
前端：应用壳/原语 → 登录界面 → 媒体库界面 → 浏览界面 → 搜索/查看器
                    └──────────── Integrated Done ────────────┘
                                      │
                                      ▼
                              发布加固与稳定 MVP
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

### Stage 0 关闭记录

- [x] `S0-101` 在真实 PR 上运行首次 CI，并保存原生 amd64/arm64 job 结果。
  - 依赖：当前变更形成可审查分支和 PR。
  - 完成证据：[PR #1 CI run 29985018814](https://github.com/HappyQuQu/foliopath/actions/runs/29985018814)
    的 Go、race、Web contract、media matrix、mount boundary 7 个 jobs 全绿。
- [x] `S0-102` 验证真实 base branch → 当前 OpenAPI 的语义兼容检查。
  - 依赖：`S0-101`。
  - 完成证据：PR #1 的 Generated web contract job 对真实 base branch 执行 `oasdiff`
    成功；工具配置使用 `--fail-on WARN` 阻断检测到的破坏性变化。
- [x] `S0-103` 完成 FS-03 libvips/govips 图片链路 spike。
  - 范围：JPEG/PNG/WebP/GIF 元数据、方向、色彩、透明度、动画和缩略图限制。
  - 完成证据：[PR #3 CI run 29987783802](https://github.com/HappyQuQu/foliopath/actions/runs/29987783802)
    的隔离 govips 2.18.0/libvips 8.15.1 fixture 在原生 amd64/arm64 均通过；覆盖
    JPEG/PNG/WebP、GIF 页数/首帧策略、方向、alpha、有界缩略图和截断 PNG 拒绝。
- [x] `S0-104` 在原生 linux/amd64 与 linux/arm64 环境运行相同媒体 fixture。
  - 依赖：`S0-101`、`S0-103`。
  - 完成证据：PR #1 的 FFmpeg/webp 矩阵及 PR #3 的 libvips/govips 矩阵在两个原生
    runner 运行相同 fixture；PR #3 两边均为 libvips 8.15.1、测试高水位 355.94 KiB，
    codec 与截断输入行为一致。
- [x] `S0-105` 解决 Stage 0 与 FS-01“生产 handler 证据”的门禁顺序。
  - 当前冲突：Stage 0 禁止生产 feature handler，但 FS-01 完整关闭条件包含生产 handler。
  - 完成证据：[S0-105 Gate allocation record](gates/MVP-2026-07-23/s0-105-gate-order.md)
    将生产 HTTP/auth 证据分配到首次受保护 API Backend Gate，将发布挂载/运行期故障分配到
    FS-05/Release Gate；要求保持强制。
- [x] `S0-106` 关闭 FS-04 的 Stage 0 容量可行性范围并分配后续证据。
  - 完成证据：[S0-106 容量证据 Gate 分配记录](gates/MVP-2026-07-23/s0-106-capacity-gate-order.md)
    及 FS-04 的 Linux RSS、三档趋势、`stage0-comparable-v1` 暂定回归预算和 fallback。
  - 后续强制条件：代表性存储和最终镜像由 Performance/Release Gate 阻断；生产媒体队列、
    FTS/keyset、HTTP 和前端并发分别由对应 Backend/UI Gate 阻断。
- [x] `S0-107` 完成 FS-05 双架构镜像与恢复 spike。
  - 范围：多阶段 Debian slim、非 root、`/library:ro`、`/app/data`、健康检查、优雅停机、
    SQLite 备份/恢复、升级和磁盘满。
  - 完成证据：[FS-05 报告](spikes/fs-05-runtime-recovery.md)与
    [原生双架构 CI](https://github.com/HappyQuQu/foliopath/actions/runs/29990148384)；
    两边相同 fixture 验证非 root/只读、health、退出、恢复、重复迁移和故障关闭。
- [x] `S0-108` 生成 Go/npm/镜像 SBOM，审查 libvips、FFmpeg codec 与许可证。
  - 依赖：`S0-103`、`S0-107`。
  - 完成证据：[供应链与许可证审查](supply-chain-review.md)及
    [SBOM/license CI](https://github.com/HappyQuQu/foliopath/actions/runs/29990480565)。
- [x] `S0-109` 复审风险 R-002～R-016，并更新 owner、状态和 fallback。
  - 完成证据：[S0-109 风险复审](gates/MVP-2026-07-23/s0-109-risk-review.md)逐项分配最迟 Gate。
- [x] `S0-110` 复审 Stage 0 Gate。
  - 依赖：`S0-101`～`S0-109` 完成，或每个未完成项有正式缩减范围/延期 Gate。
  - 完成证据：[Stage 0 Gate](gates/MVP-2026-07-23/stage-0-current.md)结论为
    `Go — 允许进入 Stage 1`，并保留后端优先、风险和 Release Gate 约束。

## 独立开发清单

### 后端

[查看后端开发清单](backend-task-list.md)

- 当前：媒体库 `S2-001～007` Backend Ready 已通过；扫描 `S2-101` Contract Ready
  已通过，进入 `S2-102` 有界 generation 扫描服务和全局任务队列。
- 完成范围：Go/API/SQLite/认证/文件安全/扫描/媒体处理。
- 交付前端：评审后的 OpenAPI、契约 fixture、可启动服务和 `Backend Ready` 记录。

### 前端

[查看前端开发清单](frontend-task-list.md)

- 当前：完整静态原型已作为设计基线确认；生产前端 Stage 1 已完成应用壳、主题/语言、
  共享原语、安全启动故障、真实认证切片和自动浏览器 E2E，并通过
  [认证 Integrated Done Gate](gates/MVP-2026-07-23/s1-auth-integrated-done.md)；
  Stage 2 已完成真实媒体库列表、三步创建、改名、异步安全移除、扫描详情/取消/重试、
  扫描周期/缓存配额设置和完整验收矩阵，并通过
  [Integrated Done Gate](gates/MVP-2026-07-23/s2-library-scan-integrated-done.md)；
  Stage 3 已完成真实目录/递归浏览、虚拟缩略图集合、完整状态矩阵、非模态预览和
  100k 容量验收，并通过
  [Integrated Done Gate](gates/MVP-2026-07-23/s3-browse-integrated-done.md)；
  下一步进入 Stage 4 搜索与完整查看器界面。
- 完成范围：React、设计系统、页面、URL/Query 状态、无障碍与浏览器测试。
- 不等待后端的工作：应用壳、token、共享原语、Story/契约状态。
- 必须等待后端的工作：真实 API 提交、业务集成与 Integrated E2E。

## Stage 5：发布加固

- [ ] `[后端]` `S5-001` 完成非 root、只读媒体、最小权限和默认安全配置的最终镜像。
  - [x] `S5-001A` 建立嵌入真实 SPA 的候选镜像，以 UID/GID 65532、只读根、
    `cap_drop: ALL`、`no-new-privileges`、`/library:ro` 和持久 `/app/data` 通过
    linux/arm64 本地 smoke；证据见 [发布镜像基础](gates/MVP-2026-07-23/s5-release-image-foundation.md)。
  - [x] `S5-001B` 在 `S5-003` 网络边界通过后固定可部署 Compose 与最终默认配置，并在
    原生双架构重复完整 smoke。
    - [x] 根 `compose.yaml`、`.env.example`、媒体/代理/Compose smoke 与原生双架构 CI
      已建立；linux/arm64 本机通过。证据和未满足条件见
      [Compose 与候选镜像矩阵](gates/MVP-2026-07-23/s5-compose-candidate-matrix.md)。
    - [x] `S5-001C` 每个原生 job 生成绑定 release、commit、architecture、digest、
      size、run ID/attempt 和完整 smoke 结果的 JSON artifact；成对校验拒绝缺失架构、
      不同 commit/run 或非 passed 结果。
    - [x] 当前候选已在本机原生 linux/arm64 与操作者指定的原生 linux/amd64 服务器通过；
      按操作者决定，本轮不以计费阻断的 amd64 CI job 作为完成条件。
- [x] `[后端]` `S5-002` 完成 linux/amd64 与 linux/arm64 构建、启动、健康和媒体矩阵。
  - [x] `S5-002A` 在原生 linux/amd64 服务器重复完整镜像、媒体、Compose/代理和恢复
    smoke；按操作者决定，本轮 amd64 以指定原生服务器证据为准，不等待计费阻断的 CI。
    两架构候选返回相同的嵌入 SPA asset 标识并具有相同运行包闭包。证据见
    [原生 amd64 与真实媒体验证](gates/MVP-2026-07-23/s5-native-amd64-real-media.md)。
- [x] `[后端]` `S5-003` 完成反向代理信任、Secure Cookie、CSRF、限流和安全响应头验证。
  - 完成证据：[可信代理与发布 HTTP 安全 Gate](gates/MVP-2026-07-23/s5-trusted-proxy-security.md)。
  - 非回环监听要求显式可信 CIDR；单跳 HTTPS authority/client transport 是 Origin、
    Secure Cookie、HSTS 和认证限流的唯一输入，伪造/链式/混用头失败关闭。
- [x] `[后端]` `S5-004` 完成备份、恢复、升级、回滚、强杀、磁盘满和缓存重建演练。
  - [x] `S5-004A` 候选镜像已验证真实管理员状态的离线备份/恢复、cache/tmp 重建、
    同版本 migration 幂等、`SIGKILL` 后 WAL 恢复、数据盘满和损坏数据库失败关闭；
    见 [恢复与失败关闭候选演练](gates/MVP-2026-07-23/s5-recovery-failure-smoke.md)。
  - [x] `S5-004B` 使用两个架构各自不同的真实前一候选不可变 image ID 完成向前升级，
    并验证回滚必须配对恢复升级前数据备份；arm64 本机与用户指定的原生 amd64 服务器
    均通过同一自动化入口。Billing 阻止的 CI 不被冒充为运行证据。
- [x] `[共同]` `S5-005` 完成 10 万媒体/1 万目录/4 核/4 GiB 的完整产品容量验收。
  - [x] `S5-005A` 建立真实候选容器、独立有效媒体 fixture、认证 HTTP、后台缩略图和
    只读哨兵校验；本机 linux/arm64 100k/10k 档通过，并接入原生双架构 CI artifact。
  - [x] `S5-005B` 审阅原生双架构结果，在代表性 NAS/存储和目标浏览器记录扫描、
    API p95、RSS/FPS、媒体吞吐与缓存增长，并冻结发布预算。
  - [x] `S5-005C` 在 4 核/4 GiB 原生 amd64、本地 ZFS 的 31,899 项真实媒体树完成
    只读扫描、认证浏览和视频封面抽样；该诊断不替代 100k/10k 与浏览器签署。
  - [x] `S5-005D` 在同一规格原生 amd64 完成精确 100k/10k、首次/无变化/后台任务并发
    重扫、认证查询、取消、offline 与只读哨兵；后端目标预算通过。按操作者决定，本轮
    指定原生 amd64 服务器与本机 arm64 结果构成容量架构证据，不等待计费阻断的 amd64 CI。
  - [x] 现有 `S5-005B` 范围内已用 100k Storybook 主档记录 Chromium/Firefox/WebKit
    长列表 FPS、帧间隔、进程树 RSS 与有界 DOM，并冻结自动化预算；真实浏览器/物理设备
    签署仍归 `S5-006B`。
- [ ] `[共同]` `S5-006` 完成单元、race、集成、组件、axe、浏览器 E2E 和视觉回归。
  - [x] `S5-006A` 建立 Chromium 日常纵向链、Firefox/WebKit 稳定状态矩阵和固定
    Linux Chromium 视觉基线；本机三引擎功能与独立 Linux 视觉复跑通过，证据见
    [浏览器质量候选 Gate](gates/MVP-2026-07-23/s5-browser-quality-candidate.md)。
  - [ ] `S5-006B` Safari 26.5.2 真机纵向链、Chrome 150 normal/forced-colors 及三引擎
    100k 性能已通过；仍须在真实 Firefox 与代表性物理设备完成读屏、缩放、触摸和视觉签署。
- [ ] `[后端]` `S5-007` 生成最终 SBOM，完成依赖漏洞、许可证和 FFmpeg 构建配置审查。
  - [x] `S5-007A` 建立候选 source/npm/image SPDX、固定 digest 漏洞扫描、CI artifact、
    修复可用时阻断策略，并升级 Go 1.26.5、`x/image` v0.44.0 与 Debian 13 候选基础；
    证据见 [候选镜像供应链 Gate](gates/MVP-2026-07-23/s5-supply-chain-candidate.md)。
  - [x] `S5-007C` 以固定源码、最小 loader 的 libvips 生产包替换通用 Debian
    `libvips42` 闭包，完整媒体/Compose/恢复 smoke 通过；发现由 15 Critical /
    136 High 降为 8 / 73，证据见
    [最小媒体运行时切片](gates/MVP-2026-07-23/s5-minimal-media-runtime.md)。
  - [x] `S5-007D` 以固定源码、关闭网络和最小组件的 FFmpeg 生产包替换 Debian
    通用闭包，完整媒体/Compose/恢复 smoke 通过；发现进一步降至 5 Critical /
    44 High，证据见
    [最小 FFmpeg 运行时切片](gates/MVP-2026-07-23/s5-minimal-ffmpeg-runtime.md)。
  - [x] `S5-007E` 以内建、固定 loopback 的 readiness probe 替换生产 `curl`
    健康检查闭包，完整媒体/Compose/恢复 smoke 通过；发现进一步降至 5 Critical /
    30 High，证据见
    [内建健康检查运行时切片](gates/MVP-2026-07-23/s5-built-in-healthcheck-runtime.md)。
  - [x] `S5-007F` 以固定 digest 的 Debian 13 distroless final stage 替换通用
    Debian runtime，保留可扫描的包状态与许可证元数据；无 shell 生产镜像通过完整
    release smoke 和快速容量复验，发现进一步降至 1 Critical / 14 High，证据见
    [无 shell 最小运行时切片](gates/MVP-2026-07-23/s5-distroless-runtime.md)。
  - [ ] `S5-007B` 处置或逐项正式接受候选扫描的 1 条 Critical / 8 条 High，
    在最终双架构 digest 上执行全阻断策略，并完成 notices、provenance 与许可证签署。
- [x] `[共同]` `S5-008` 校对 README、Compose、部署、备份、支持格式和已知限制。
  - 完成证据：[发布文档 Gate](gates/MVP-2026-07-23/s5-release-documentation.md)。
    `make release-docs-check` 固定 candidate 警告、Compose 安全默认、格式、备份/回滚和限制披露。
- [ ] `[共同]` `S5-009` 关闭或正式接受发布阻断风险，形成 Release Candidate Gate。
  - [x] `S5-009A` 完成首次统一 RC 审计、机器可读 readiness 快照和失败关闭检查；当前结论
    为 No-Go，证据见 [Release Candidate 当前判断](gates/MVP-2026-07-23/s5-release-candidate-current.md)。
  - [ ] 所有前置 Gate 通过，发布阻断风险逐项关闭或有时限地正式接受，人工审阅
    `make release-ready` 的 Go 结果后形成 RC。
- [ ] `[共同]` `S5-010` 完成稳定 MVP 发布 Gate；只有此项通过后才能宣称 FolioPath 可发布。

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
