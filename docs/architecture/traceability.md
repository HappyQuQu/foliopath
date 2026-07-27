# FolioPath 需求—架构追踪

## 状态与使用方式

本文把当前已确认的需求族映射到架构责任、契约、数据、决策、风险、验证和交付阶段。它是设计与评审索引，不表示表中 API、前端、容器或全部测试已经实现。

状态来源：

- 目标版本与冻结范围以 [scope manifest](../releases/MVP-2026-07-23-scope.md) 为准，需求语义以[产品需求](../product-requirements.md)为准；
- 结构决策以[已接受 ADR](../adr/)为准；
- HTTP 结构以权威 [`api/openapi.yaml`](../../api/openapi.yaml) 为准；[API 设计说明](../api-design.md)
  只保留动机与实现参数，不能覆盖 wire 契约；
- 已有证据只以 [spike 报告](../spikes/)、[测试策略](../testing-strategy.md)和 Gate 记录明确链接的 scope 为准。

表中的“计划 API/流程”和“必需验证”是交付目标，不得据此宣称功能已可用。具体切片必须遵守[交付与架构治理](delivery-governance.md)。
Stage 0 Gate 已通过并只授权后端优先的 Stage 1：OpenAPI 已成为结构权威，FS-01～05 的
Stage 0 范围及供应链识别已有报告证据；这不把任何生产切片自动标为 `Backend Ready`，
也不授权跳过后端实现业务 UI。

## 功能需求追踪

| 需求族 | 目标版本 | 范围状态 | Change Record | Capability / adapter | 计划 API 或用户流程 | 数据与派生状态 | 决策与约束 | 主要风险 | 必需验证 | Roadmap 阶段 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `FR-DEP-001～004` 部署与设置 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | `internal/app`、`internal/api`、`internal/webassets`、`internal/store/sqlite`；容器适配器 | 启动/初始化、`/health/live`、`/health/ready`、`/api/v1/status`、设置读写 | migrations、`settings`、运行版本与健康状态；Vite 产物为构建输出 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0009](../adr/0009-linux-openat2-single-media-root.md)、[架构](../architecture.md)、[部署](../deployment.md) | R-002、R-004、R-008、R-011、R-014、R-016 | liveness/readiness/status、真实 composition root 的空目录迁移、重复启动、取消关闭和迁移/数据目录失败关闭已有测试；测试专用非 root 容器验证固定只读媒体根、health、SIGTERM 和重复启动。仍需无后代挂载的正式发布镜像、备份/恢复和真实升级 | 1、5 |
| `FR-AUTH-001～004` 认证与会话 | `MVP-2026-07-23` | Frozen r1；[Contract Ready](../gates/MVP-2026-07-23/s1-auth-contract-ready.md)；[Backend Ready](../gates/MVP-2026-07-23/s1-auth-backend-ready.md)；[Integrated Done](../gates/MVP-2026-07-23/s1-auth-integrated-done.md) | `BASELINE-2026-07-23` | `internal/auth` 拥有规则；`internal/api` 与 SQLite adapter；`web/src/features/auth` 与 `web/src/lib/api/auth.ts` | 首次 setup、login、session、logout；受保护业务 API 与 CSRF 失败流 | `users` singleton、password verifier、`sessions` 摘要/期限/撤销/auth version；秘密不进入数据库明文、响应或日志 | [ADR-0005](../adr/0005-built-in-single-admin-auth.md)、[安全模型](../security.md) | R-010、R-011、R-016 | S1-101～106 已完成 Backend Ready；生产前端通过生成 client 的领域 adapter 实现 setup/login/session/logout、会话守卫、安全 readiness、简中/英文和通用账户页；一次性真实后端 Chromium E2E 覆盖四档宽度、主题、axe、创建/退出/过期返回/再登录，认证切片已 Integrated Done。Stage 5 仍负责可信代理、最终浏览器矩阵和网络发布 | 1（Integrated Done）、5（发布加固） |
| `FR-LIB-001～008` 媒体库管理 | `MVP-2026-07-23` | Frozen r1；[Backend Ready](../gates/MVP-2026-07-23/s2-library-backend-ready.md)；[Integrated Done](../gates/MVP-2026-07-23/s2-library-scan-integrated-done.md) | `BASELINE-2026-07-23`；[Stage 2 Architecture Ready](../gates/MVP-2026-07-23/stage-2-architecture-ready.md)；[Contract Ready](../gates/MVP-2026-07-23/s2-library-contract-ready.md) | `internal/library`；`internal/files` 和 SQLite adapter；`web/src/features/libraries` | 允许目录选择、媒体库创建/列表/详情/改名/移除、离线重试 | `libraries.revision`、`name_sort_key`、唯一 creation scan、`library_removals`、摘要化 `idempotency_records`；根不可变；创建三记录同事务 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0002](../adr/0002-library-path-model.md)、[ADR-0004](../adr/0004-library-root-immutable.md)、[ADR-0009](../adr/0009-linux-openat2-single-media-root.md)、[安全模型](../security.md) | R-002、R-003、R-012、R-016 | S2-002～007 已通过后端完整矩阵；前端 S2-201～208 通过生成 client adapter 连接真实列表、长路径幂等创建、ETag 改名、异步幂等移除、离线重试和扫描/设置页面。真实 Chromium 成功链与受控故障状态矩阵覆盖长内容、重复提交、键盘焦点、主题/语言、四档响应式和 axe，Stage 2 已 Integrated Done | 2（Integrated Done） |
| `FR-SCN-001～009` 扫描与索引 | `MVP-2026-07-23` | Frozen r1；[Backend Ready](../gates/MVP-2026-07-23/s2-scan-backend-ready.md)；[Integrated Done](../gates/MVP-2026-07-23/s2-library-scan-integrated-done.md)；[Contract Ready](../gates/MVP-2026-07-23/s2-scan-contract-ready.md)；[S2-102 worker](../gates/MVP-2026-07-23/s2-bounded-scan-worker.md)；[S2-103 目录计数](../gates/MVP-2026-07-23/s2-directory-counts.md)；[S2-104 媒体收敛](../gates/MVP-2026-07-23/s2-media-convergence.md)；[S2-105 故障恢复](../gates/MVP-2026-07-23/s2-scan-recovery.md)；[S2-106 容量并发](../gates/MVP-2026-07-23/s2-scan-capacity.md) | `BASELINE-2026-07-23`；[Stage 2 Architecture Ready](../gates/MVP-2026-07-23/stage-2-architecture-ready.md) | `internal/jobs` 拥有领取/租约/worker；`internal/scanner` 拥有 generation、admission、查询游标、取消、scheduler、扫描周期范围、容量常量与目录策略；`internal/settings` 只编排 typed setting 更新；`internal/thumbnail` 拥有缓存配额范围；`internal/media` 拥有候选格式与 source fingerprint；`internal/files` 与 SQLite adapter；`web/src/features/libraries` 和 `web/src/features/settings` | 创建/启动/定时/手动扫描、历史/详情轮询、协作取消、错误和跳过统计、计划设置 | `scan_runs`、`scan_issues`、`directories`、`assets`、`settings`、generation、source fingerprint；同库唯一 full scan 与 durable admission | [ADR-0003](../adr/0003-scan-consistency.md)、[数据模型](../data-model.md) | R-003、R-004、R-005、R-013、R-016 | S2-101～107 已完成全部冻结后端 operation；前端 S2-204～208 连接详情轮询、取消、重试、可靠索引保留提示及 ETag 设置更新。真实成功链加受控 running/cancelled/failed/offline 契约状态通过 Chromium、axe、长内容和响应式矩阵，Stage 2 已 Integrated Done | 2（Integrated Done）；容量证据在 0/FS-04 与 S2-106 |
| `FR-BRW-001～009` 导航与浏览 | `MVP-2026-07-23` | Frozen r1；[S3-001 Contract Ready](../gates/MVP-2026-07-23/s3-browse-contract-ready.md)；[S3-002 keyset](../gates/MVP-2026-07-23/s3-catalog-keyset.md)；[S3-003 目录树](../gates/MVP-2026-07-23/s3-directory-tree.md)；[S3-007 Backend Ready](../gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)；[S3-101 前端目录导航](../gates/MVP-2026-07-23/s3-frontend-directory-navigation.md)；[S3-102 前端浏览范围](../gates/MVP-2026-07-23/s3-frontend-browse-scope.md)；[S3-103 前端媒体集合](../gates/MVP-2026-07-23/s3-frontend-media-collection.md)；[S3-104 前端浏览状态](../gates/MVP-2026-07-23/s3-frontend-browse-states.md)；[S3-105 前端媒体预览](../gates/MVP-2026-07-23/s3-frontend-media-preview.md)；[S3-106 固定预览交互](../gates/MVP-2026-07-23/s3-frontend-pinned-preview.md)；[S3-107 前端容量预算](../gates/MVP-2026-07-23/s3-frontend-capacity.md)；[Stage 3 Integrated Done](../gates/MVP-2026-07-23/s3-browse-integrated-done.md) | `BASELINE-2026-07-23` | `internal/catalog` 拥有 root/目录/资产 query normalization、排序、cursor payload、breadcrumb 与拓扑验证；`internal/cursor` 提供 token 机制；SQLite/API adapter；`web/src/features/browse` 拥有目录导航、浏览 URL codec、query、有界 pending 刷新策略、预览选择/固定状态机与 UI，`web/src/lib/api/catalog.ts` 是生成 client 的领域 adapter；共享 `MediaCollection` 拥有 MediaCard、布局、虚拟窗口、容量预算、焦点恢复、skeleton 和分页错误，`MediaPreview` 拥有非模态图片/原生视频/信息/导航/固定/Escape/调宽，`AsyncState` 拥有空/错误/离线语义 | 媒体库切换、目录树/面包屑、当前/递归浏览、布局与排序、URL 恢复、非模态预览选择/固定/双击/关闭恢复 | `directories`、`assets`、`libraries.current_generation`、migration 7 `natural_name_key`/浏览索引、直接/递归计数；客户端只保留 URL/显示偏好/瞬时选择与预览状态 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[界面设计](../ui-design.md)、[S3 浏览契约](../api-design.md#s3-浏览契约) | R-005、R-012、R-013、R-015、R-016 | S3-001～007 已完成后端与真实 composition；S3-101～108 已通过生成 client 接入真实导航、direct/recursive keyset、模式默认排序、来源返回、ready WebP、自适应 grid、记忆 masonry、TanStack Virtual 有界 DOM、完整浏览状态矩阵、共享图片/原生视频非模态预览、固定交互，以及 100k DOM/cursor/虚拟滚动/播放资源/焦点预算。真实成功链、受控状态 Chromium、稳定浅深主题 axe、响应式 E2E、原型同状态对照与 100k Chromium 主档共同形成 Integrated Done 证据 | 3（Integrated Done） |
| `FR-SRH-001～004` 搜索过滤排序 | `MVP-2026-07-23` | Frozen r1；[S4-001 Contract Ready](../gates/MVP-2026-07-23/s4-search-contract-ready.md)；[S4-002 Implemented](../gates/MVP-2026-07-23/s4-search-keyset.md)；[S4-003 Backend Ready](../gates/MVP-2026-07-23/s4-search-backend-ready.md)；[S4-004 Frontend Done](../gates/MVP-2026-07-23/s4-frontend-search.md) | `BASELINE-2026-07-23` | `internal/catalog` 拥有 search profile、scope、排序与 cursor；SQLite FTS adapter；`web` search feature 的 URL/query owner；共享 `SearchInput` 与 `MediaCollection` | 当前目录（可递归）/当前库/全部库搜索，类型和 filesystem mtime 半开区间过滤，结果查看/返回 | migration 10 的 `assets.search_*_key`、外部内容 FTS5 `asset_search`、`catalog_search_state.revision`；库内 generation 与跨库 revision cursor；URL 保存搜索指纹，TanStack Query 保存游标页面 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[数据模型](../data-model.md)、[S4 搜索契约](../api-design.md#s4-搜索契约) | R-005、R-012、R-016 | S4-002 已实现 NFKC/full fold、trigram candidate + 精确字面 AND、短词/标点 fallback、三种 scope、kind/mtime、稳定 keyset、offline、认证 HTTP、旧库回填与 composition；S4-003 完成 100k/10k 后端主档；S4-004 已连接真实库内/全局 operation、稳定 URL、类型/日期/排序、虚拟结果及 empty/offline/error 状态，非模态预览和查看器仍留给 S4-005～009 | 4（Backend Ready；S4-004 前端搜索 Done；预览/查看器待完成） |
| `FR-MED-001～008` 缩略图、查看器与视频 | `MVP-2026-07-23` | Frozen r1；[S3-004 媒体处理](../gates/MVP-2026-07-23/s3-media-processing.md)；[S3-005 媒体任务/缓存](../gates/MVP-2026-07-23/s3-media-jobs-cache.md)；[S3-006 资源安全](../gates/MVP-2026-07-23/s3-media-resource-safety.md)；[S3-007 Backend Ready](../gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)；[S3-105 前端媒体预览](../gates/MVP-2026-07-23/s3-frontend-media-preview.md)；[S3-106 固定预览交互](../gates/MVP-2026-07-23/s3-frontend-pinned-preview.md)；[Stage 3 Integrated Done](../gates/MVP-2026-07-23/s3-browse-integrated-done.md) | [CR-2026-001](../changes/CR-2026-001-non-modal-media-preview.md) | `internal/media` 拥有结果/错误/fingerprint/资源上限；`internal/thumbnail` 拥有派生键、交付状态和缓存策略；`internal/jobs` 拥有 worker/lease；`internal/app` 拥有 native lifecycle；`internal/files`、SQLite/cache、govips、FFmpeg adapter；`web/src/components/patterns/MediaPreview` 是共享 preview owner | 资产详情、thumbnail、原内容/Range；非模态预览、完整图片查看器、原生视频、不兼容/损坏状态、缓存设置 | migration 8 `assets`/`thumbnails`；migration 9 `media_jobs`/fairness/`cache_deletions`；source fingerprint/transform version；缓存文件在 `/app/data/cache` | [ADR-0001](../adr/0001-go-react-sqlite.md)、[安全模型](../security.md)、[数据模型](../data-model.md) | R-006、R-007、R-008、R-009、R-014、R-016 | S3-004～007 已实现 production adapter、原子 cache→DB、2-worker/3-attempt durable queue、公平、失效/LRU、资源边界及认证 202/200/304/409/422 thumbnail；Stage 3 已接入同源认证原内容 URL，并把共享图片/原生视频预览、基本信息、导航、固定/双击/Escape/单活动媒体纳入浏览 Integrated Done。完整 Range/codec/离线/删除状态、全屏查看器和发布矩阵仍待 Stage 4～5 | 0/FS-01/03；3（thumbnail/preview Integrated Done）、4（完整内容/查看器） |
| `FR-UI-001～007` 界面与可访问性 | `MVP-2026-07-23` | Frozen r1 | [CR-2026-001](../changes/CR-2026-001-non-modal-media-preview.md) | `web` 的 auth/libraries/browse/search/viewer/settings feature；统一 API client | 欢迎/认证、设置、扫描状态、浏览/搜索/预览/查看器；桌面侧栏、移动抽屉、中英与主题 | 服务端状态不复制为独立事实；URL 保存导航状态，`settings` 保存受控偏好 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[用户流程](../user-flows.md)、[界面设计](../ui-design.md) | R-005、R-010、R-015、R-016 | loading/empty/error/offline/cancel、预览选择/固定/双击、键盘/焦点/读屏、主题/语言/reduced-motion、响应式、虚拟化、关键 E2E | 1（认证 UI）、3、4；5（发布加固） |

### Stage 4 媒体内容 Backend Ready

- `FR-MED-004～006` / `NFR-SAFE-001` / `NFR-SEC-001` / `NFR-PRIV-001`：
  [S4-005 原媒体内容实现](../gates/MVP-2026-07-23/s4-media-content.md)和
  [S4-005B Backend Ready](../gates/MVP-2026-07-23/s4-media-content-backend-ready.md)已完成。
  `internal/media` capability、SQLite、`internal/files`、认证 HTTP route 与 composition
  贯通；GET/HEAD/条件请求/单 Range/416、16-stream admission、取消、poisoned path、源变化/
  缺失/offline 和错误脱敏均有自动证据。冻结 content operation 可交接前端；查看器、浏览器
  播放矩阵与发布 volume/network 仍由产品集成和 Stage 5 Gate 约束。

## 非功能需求追踪

| 需求族 | 目标版本 | 范围状态 | Change Record | 架构落点 | 关键响应或约束 | 决策与风险 | 必需证据 | 适用阶段 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `NFR-SAFE-001` 原文件安全 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | `internal/files` 单一媒体访问边界；容器 `/library:ro`；服务能力不提供写原件接口 | 创建、扫描、浏览、查看、取消和移除媒体库都不能改变原媒体 | ADR-0002、ADR-0009；R-002、R-003 | [FS-01](../spikes/fs-01-path-boundary.md) 验证系统调用边界；[S2-006](../gates/MVP-2026-07-23/s2-library-removal-invariance.md) 已以生产认证删除链路证明媒体树逐项、逐字节不变并以 fitness test 固定 removal 无媒体能力。仍需正式只读发布挂载 | 0、2、5 |
| `NFR-SEC-001～002` 路径与网络安全；`NFR-PRIV-001` 信息披露 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | `internal/files`、`internal/auth`、API middleware、错误/日志适配器、反向代理信任配置 | 逃逸与后代 mount crossing 失败关闭；无有效会话不返回业务数据；错误不泄露路径、SQL、stderr、Cookie 或令牌 | ADR-0002、ADR-0005、ADR-0009；R-002、R-010、R-012、R-016 | 服务端 request ID、统一安全 404/500、panic/HTTP runtime 日志脱敏已有单元测试；认证/CSRF/限流已通过认证 Backend Gate；媒体库 Backend Gate 已验证生产 path/create/remove handler、openat2 mount/TOCTOU/ABA、权限、错误脱敏和原媒体不变。扫描与媒体读取 handler 仍须逐切片验证，可信代理、发布 volume/unmount 和网络暴露由 Stage 5 强制 | 0、1（认证 Backend Ready）、2（媒体库 Backend Ready）、5 |
| `NFR-REL-001` 扫描与恢复一致性 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | scanner generation 状态机、SQLite 短事务、restart-safe jobs、原子缓存落盘 | 只有完整成功扫描可清理；崩溃、取消、离线和部分错误保留可靠状态并可收敛 | ADR-0003；R-003、R-004、R-011、R-013 | FS-02 generation 故障矩阵、FS-05 离线恢复/满盘/损坏失败关闭与 S2-105 production startup/late-lease 恢复已通过；正式应用强杀、真实升级和在线备份仍待 Gate | 0、2、5 |
| `NFR-PERF-001～002` 资源与容量 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | 有界队列/并发、串行批量写、keyset、虚拟化、缓存水位、全局工作调度 | 四核/4 GiB、约 10 万媒体/1 万目录为主验收档；扫描时浏览仍可用，发布预算由代表性设备 Gate 固定 | ADR-0001、ADR-0003；R-005、R-009、R-013、R-015 | [FS-04](../spikes/fs-04-capacity-baseline.md)与[S2-106](../gates/MVP-2026-07-23/s2-scan-capacity.md)已验证扫描/索引；[S4-003](../gates/MVP-2026-07-23/s4-search-backend-ready.md)已在 100k/10k macOS 与 Linux 2 CPU/4 GiB 档验证扫描并发搜索、FTS/短词/全局/keyset、取消、rebuild、RSS 与 DB 预算；[S3-107](../gates/MVP-2026-07-23/s3-frontend-capacity.md)已固定前端 100k 有界 DOM、cursor 请求、虚拟滚动、播放资源和焦点预算。代表性发布设备 FPS/RSS 与存储仍按 [S0-106](../gates/MVP-2026-07-23/s0-106-capacity-gate-order.md)留给 Stage 5 | 0/FS-04、2～5 |
| `NFR-ACC-001` 可访问性 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | 语义 HTML、DOM 顺序、焦点管理、状态文案、主题与 reduced-motion token | 核心流程键盘可完成，状态不只依赖颜色，目标 WCAG 2.2 AA | UI 设计；R-015、R-016 | 键盘/焦点、读屏、对比度、缩放、forced-colors/reduced-motion 和关键 E2E | 1、3、4、5 |
| `NFR-COMP-001` 平台兼容 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | Debian slim 镜像、Go/CGO、libvips/FFmpeg、浏览器兼容层 | 承诺的 linux/amd64、linux/arm64 和主流浏览器行为必须由同一 fixture 验证 | ADR-0001；R-007、R-008、R-014 | FS-03 双架构媒体与 FS-05 双架构 runtime/SBOM 已通过；浏览器播放、UI E2E 和最终 digest 仍待 Gate | 0/FS-03/05、5 |
| `NFR-OPS-001` 可运维性 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | `internal/app` 配置/生命周期、SQLite WAL、health、日志/指标、migration、备份恢复、缓存清理 | 本地可靠文件系统；安全启动/退出/升级/恢复；缓存和磁盘问题不能破坏配置或原件 | ADR-0001；R-004、R-009、R-011、R-016 | `internal/app`/`internal/api` 单元测试已覆盖固定根、安全监听、JSON 日志、health/readiness/status、数据库 → HTTP → ready 启动、空目录/重复 migration、迁移与数据目录失败关闭、失败回滚、HTTP 排空、反向关闭和停机期限；FS-05 probe 已验证 PID 1 退出与离线恢复；在线备份和真实升级仍待 Gate | 1、5 |

## 追踪维护规则

1. FR/NFR 编号一经进入已确认基线不得复用。语义改变使用新的需求或变更记录，并标出替代关系。
2. 一个需求可映射多个 capability，但必须指定唯一的业务规则 Owner；adapter 不能成为业务语义来源。
3. API、schema、风险或测试尚不存在时标记“计划/未实现”，不能留空或写成已通过。
4. 每个纵向切片在 Gate S0/S1 更新计划映射，在 Gate S4 把测试计划替换为证据链接。
5. 新增风险先进入[风险登记](../risk-register.md)，不能只在实现 PR 中描述。
6. 新增部署单元、外部服务、持久化系统、信任边界或不可逆 migration 必须链接新的 ADR。
7. 需求延期到下一版本时保留映射和历史，不删除记录来制造“从未承诺”的假象。
8. 发现本表与权威 PRD/ADR/OpenAPI/migration 冲突时停止相关切片，先修复来源和映射。

## 新增需求 / Change Record 模板

```markdown
# CR-YYYY-NNN：<标题>

## 状态

- 状态：Proposed | Confirmed | Rejected | Superseded
- 变更等级：C0 | C1 | C2 | C3 | C4
- 目标版本：MVP-2026-07-23 | Post-MVP/<版本>
- Scope revision / 范围状态：
- Change Record ID / 基线事件：
- 提出日期：
- 产品负责人：
- 架构负责人：
- Capability Owner：

## 用户问题与价值

- 用户 / JTBD：
- 当前问题及证据：
- 为什么必须进入目标版本：

## 范围

- 新增或改变的 FR/NFR：
- 明确包含：
- 明确不包含：
- 被替代/延期的现有范围：
- Scope-budget exception（如有）：理由、批准人、复审条件

## 架构影响

- Capability 与依赖方向：
- API / 用户流程：
- 数据 / migration / 派生状态：
- 安全、隐私与信任边界：
- 性能、容量与并发：
- 部署、升级、备份、恢复与观测：
- 平台、依赖、许可证与供应链：
- ADR：N/A（理由）| 新 ADR 链接

## 质量属性场景

- 刺激：
- 环境：
- 系统响应：
- 可测结果：

## 风险与验证

- 风险 ID / 新风险：
- Fallback / 回滚：
- 正常、边界、失败、恢复测试：
- Fixture 与目标环境：
- 验收证据要求：

## Gate 影响与决定

- 需要重跑的阶段/切片 Gate：
- 产品决定：
- 架构决定：
- 安全/数据评审：
- 最终结论：Go | Conditional Go | No-Go
```

模板字段不得无故删除；不适用项写 `N/A` 及理由。确认后应同步 PRD、本文映射、roadmap、API/数据/安全/测试文档，并在需要时先接受 ADR，再进入实施。
