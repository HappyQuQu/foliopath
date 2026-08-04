# FolioPath 需求—架构追踪

## 状态与使用方式

本文把当前已确认的需求族映射到架构责任、契约、数据、决策、风险、验证和交付阶段。它是设计与评审索引，不表示表中 API、前端、容器或全部测试已经实现。

状态来源：

- 目标版本与冻结范围以当前 [scope revision 4](../releases/MVP-2026-07-23-scope-r4.md) 为准，需求语义以[产品需求](../product-requirements.md)为准；
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
| `FR-DEP-001～004` 部署与设置 | `MVP-2026-07-23` | Frozen r1；[CR-2026-013 root runtime](../changes/CR-2026-013-root-runtime-bind-data.md)重新打开身份候选；既有 S5-001/002/004 历史证据保留 | `BASELINE-2026-07-23`、`CR-2026-013` | `internal/app`、`internal/api`、`internal/webassets`、`internal/store/sqlite`；根 `Dockerfile`、`compose.yaml` 与容器适配器 | 启动/初始化、嵌入 SPA、`/health/live`、`/health/ready`、`/api/v1/status`、设置读写 | migrations、`settings`、运行版本与健康状态；Vite 产物仅在构建期嵌入 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0009](../adr/0009-linux-openat2-single-media-root.md)、[ADR-0012](../adr/0012-root-runtime-bind-data.md)、[部署](../deployment.md) | R-002、R-004、R-008、R-011、R-014、R-016、R-022 | 历史非 root 双架构证据不证明当前身份；root 候选必须以 Docker 自动创建的 root-owned `/app/data` bind 复验数据库、只读根/媒体、health、SIGTERM、Compose/代理、恢复与供应链 | 1、5 |
| `FR-AUTH-001～004`、`FR-DEP-005` 认证与 LAN 部署 | `MVP-2026-07-23` | Frozen r2；认证 Integrated Done；LAN HTTP implemented；密码最低长度可用性修复 | [CR-2026-002](../changes/CR-2026-002-authenticated-lan-http.md)；[FIX-2026-07-29](../changes/FIX-2026-07-29-admin-password-minimum.md) | `internal/auth` 拥有规则；`internal/api` 拥有 transport/HTTP；`internal/app` 拥有监听配置；Compose 拥有宿主绑定 | 首次 setup（8～128 Unicode 字符密码）、login、session、logout；受保护业务 API；受信 LAN HTTP 与可选严格 HTTPS 代理 | 认证数据不变；transport 不持久化，未受信转发头清除 | [ADR-0005](../adr/0005-built-in-single-admin-auth.md)、[ADR-0010](../adr/0010-authenticated-lan-http.md)、[安全模型](../security.md) | R-010、R-011、R-016 | 8 字符 ASCII/中文接受、7 字符拒绝、前后端 Unicode 计数一致；Argon2id、限流、错误脱敏、session/CSRF 回归保持。直连 HTTP 使用实际 peer/Host，显式可信代理配置继续严格校验 HTTPS 并拒绝旁路 | 1（Integrated Done）、5 |
| `FR-AUTH-005`、`FR-UI-009` 管理中心独立功能页与账户维护 | `MVP-2026-07-23` | Frozen r3；[UIF-S4 Integrated Slice Done](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md) Go | [CR-2026-006](../changes/CR-2026-006-management-center-pages.md) | `internal/auth` 拥有当前密码验证、密码哈希与其他会话撤销；`internal/settings` 拥有受控偏好；既有 library/scanner/thumbnail capability 保持规则所有权；`web` settings routes 只通过生成 client/adapter 组合 | 独立 general/libraries/storage/account 路由；账户资料/密码；媒体库、扫描、缓存 operation 复用 | 单管理员显示名称/密码哈希/会话；settings revision；原型 local state 不进入生产 import graph | ADR-0005、ADR-0007；[用户流程第 8 节](../user-flows.md)；[界面设计](../ui-design.md)；[UIF 当前状态](../releases/MVP-2026-07-23-uif-integration-status.md)；[UIF-408 evidence](../evidence/uif-408/README.md) | R-010、R-012、R-015、R-016 | 账户 OpenAPI/事务/ETag、四独立路由、generated client、真实改名/改密/退出/重登、四档逐页视觉、浏览器/可访问性、容量与完整仓库验证通过；物理辅助功能仍由 Stage 5 持有 | `UIF-S4 Integrated Done → Stage 5 No-Go` |
| `FR-BRW-010`、`FR-UI-010`、`NFR-UIF-001` 生产前端原型一致性 | `MVP-2026-07-23` | Frozen r4；[UIF-S0](../gates/MVP-2026-07-23/uif-s0-architecture-ready.md)、[S1](../gates/MVP-2026-07-23/uif-s1-contract-ready.md)、[S2](../gates/MVP-2026-07-23/uif-s2-backend-evidence-ready.md)、[S3](../gates/MVP-2026-07-23/uif-s3-consumer-ui-ready.md)、[S4](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md) Go | [CR-2026-009](../changes/CR-2026-009-frontend-prototype-fidelity.md) | `internal/auth` 账户维护；`internal/catalog` 目录 q/cursor；`internal/thumbnail` 缓存摘要/清理；SQLite/API adapter；`web` 唯一 token、壳、媒体集合和预览 owner | account profile/password；direct-directory q；cache summary/cleanup；全局 Header、BrowseShell、ManagementShell、Search 无侧栏、四档视觉 Gate | migration 13 已实现 users revision、directory search key、singleton cleanup state，并复用 sessions/idempotency/cache；视觉 reference 不进入生产 import graph | [FTR-UIF-001](../features/frontend-prototype-fidelity.md)；[开发清单](../features/frontend-prototype-fidelity-task-list.md)；[UIF 当前状态](../releases/MVP-2026-07-23-uif-integration-status.md)；[UIF-408 evidence](../evidence/uif-408/README.md)；ADR-0005/0007；无需新 ADR | R-010、R-012、R-015、R-016、R-021 | 逐页比较、Linux 基线、真实纵向链、三引擎/可访问性、100k/10k 容量、完整仓库验证、跨文档收敛和 12 页 × 4 断点复核均有实际证据；受影响 Stage 5 检查已重跑，独立发布阻断保持 No-Go | `UIF-S4 Integrated Done → Stage 5 No-Go` |
| `FR-LIB-001～008` 媒体库管理 | `MVP-2026-07-23` | Frozen r1；[Backend Ready](../gates/MVP-2026-07-23/s2-library-backend-ready.md)；[Integrated Done](../gates/MVP-2026-07-23/s2-library-scan-integrated-done.md) | `BASELINE-2026-07-23`；[Stage 2 Architecture Ready](../gates/MVP-2026-07-23/stage-2-architecture-ready.md)；[Contract Ready](../gates/MVP-2026-07-23/s2-library-contract-ready.md) | `internal/library`；`internal/files` 和 SQLite adapter；`web/src/features/libraries` | 允许目录选择、媒体库创建/列表/详情/改名/移除、离线重试 | `libraries.revision`、`name_sort_key`、唯一 creation scan、`library_removals`、摘要化 `idempotency_records`；根不可变；创建三记录同事务 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0002](../adr/0002-library-path-model.md)、[ADR-0004](../adr/0004-library-root-immutable.md)、[ADR-0009](../adr/0009-linux-openat2-single-media-root.md)、[安全模型](../security.md) | R-002、R-003、R-012、R-016 | S2-002～007 已通过后端完整矩阵；前端 S2-201～208 通过生成 client adapter 连接真实列表、长路径幂等创建、ETag 改名、异步幂等移除、离线重试和扫描/设置页面。真实 Chromium 成功链与受控故障状态矩阵覆盖长内容、重复提交、键盘焦点、主题/语言、四档响应式和 axe，Stage 2 已 Integrated Done | 2（Integrated Done） |
| `FR-SCN-001～009` 扫描与索引 | `MVP-2026-07-23` | Frozen r1；[Backend Ready](../gates/MVP-2026-07-23/s2-scan-backend-ready.md)；[Integrated Done](../gates/MVP-2026-07-23/s2-library-scan-integrated-done.md)；[Contract Ready](../gates/MVP-2026-07-23/s2-scan-contract-ready.md)；[S2-102 worker](../gates/MVP-2026-07-23/s2-bounded-scan-worker.md)；[S2-103 目录计数](../gates/MVP-2026-07-23/s2-directory-counts.md)；[S2-104 媒体收敛](../gates/MVP-2026-07-23/s2-media-convergence.md)；[S2-105 故障恢复](../gates/MVP-2026-07-23/s2-scan-recovery.md)；[S2-106 容量并发](../gates/MVP-2026-07-23/s2-scan-capacity.md) | `BASELINE-2026-07-23`；[Stage 2 Architecture Ready](../gates/MVP-2026-07-23/stage-2-architecture-ready.md) | `internal/jobs` 拥有领取/租约/worker；`internal/scanner` 拥有 generation、admission、查询游标、取消、scheduler、扫描周期范围、容量常量与目录策略；`internal/settings` 只编排 typed setting 更新；`internal/thumbnail` 拥有缓存配额范围；`internal/media` 拥有候选格式与 source fingerprint；`internal/files` 与 SQLite adapter；`web/src/features/libraries` 和 `web/src/features/settings` | 创建/启动/定时/手动扫描、历史/详情轮询、协作取消、错误和跳过统计、计划设置 | `scan_runs`、`scan_issues`、`directories`、`assets`、`settings`、generation、source fingerprint；同库唯一 full scan 与 durable admission | [ADR-0003](../adr/0003-scan-consistency.md)、[数据模型](../data-model.md) | R-003、R-004、R-005、R-013、R-016 | S2-101～107 已完成全部冻结后端 operation；前端 S2-204～208 连接详情轮询、取消、重试、可靠索引保留提示及 ETag 设置更新。真实成功链加受控 running/cancelled/failed/offline 契约状态通过 Chromium、axe、长内容和响应式矩阵，Stage 2 已 Integrated Done | 2（Integrated Done）；容量证据在 0/FS-04 与 S2-106 |
| `FR-BRW-001～009` 导航与浏览 | `MVP-2026-07-23` | Frozen r1；[S3-001 Contract Ready](../gates/MVP-2026-07-23/s3-browse-contract-ready.md)；[S3-002 keyset](../gates/MVP-2026-07-23/s3-catalog-keyset.md)；[S3-003 目录树](../gates/MVP-2026-07-23/s3-directory-tree.md)；[S3-007 Backend Ready](../gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)；[S3-101 前端目录导航](../gates/MVP-2026-07-23/s3-frontend-directory-navigation.md)；[S3-102 前端浏览范围](../gates/MVP-2026-07-23/s3-frontend-browse-scope.md)；[S3-103 前端媒体集合](../gates/MVP-2026-07-23/s3-frontend-media-collection.md)；[S3-104 前端浏览状态](../gates/MVP-2026-07-23/s3-frontend-browse-states.md)；[S3-105 前端媒体预览](../gates/MVP-2026-07-23/s3-frontend-media-preview.md)；[S3-106 固定预览交互](../gates/MVP-2026-07-23/s3-frontend-pinned-preview.md)；[S3-107 前端容量预算](../gates/MVP-2026-07-23/s3-frontend-capacity.md)；[Stage 3 Integrated Done](../gates/MVP-2026-07-23/s3-browse-integrated-done.md) | `BASELINE-2026-07-23` | `internal/catalog` 拥有 root/目录/资产 query normalization、排序、cursor payload、breadcrumb 与拓扑验证；`internal/cursor` 提供 token 机制；SQLite/API adapter；`web/src/features/browse` 拥有目录导航、浏览 URL codec、query、有界 pending 刷新策略、预览选择/固定状态机与 UI，`web/src/lib/api/catalog.ts` 是生成 client 的领域 adapter；共享 `MediaCollection` 拥有 MediaCard、布局、虚拟窗口、容量预算、焦点恢复、skeleton 和分页错误，`MediaPreview` 拥有非模态图片/原生视频/信息/导航/固定/Escape/调宽，`AsyncState` 拥有空/错误/离线语义 | 媒体库切换、目录树/面包屑、当前/递归浏览、布局与排序、URL 恢复、非模态预览选择/固定/双击/关闭恢复 | `directories`、`assets`、`libraries.current_generation`、migration 7 `natural_name_key`/浏览索引、直接/递归计数；客户端只保留 URL/显示偏好/瞬时选择与预览状态 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[界面设计](../ui-design.md)、[S3 浏览契约](../api-design.md#s3-浏览契约) | R-005、R-012、R-013、R-015、R-016 | S3-001～007 已完成后端与真实 composition；S3-101～108 已通过生成 client 接入真实导航、direct/recursive keyset、模式默认排序、来源返回、ready WebP、自适应 grid、记忆 masonry、TanStack Virtual 有界 DOM、完整浏览状态矩阵、共享图片/原生视频非模态预览、固定交互，以及 100k DOM/cursor/虚拟滚动/播放资源/焦点预算。真实成功链、受控状态 Chromium、稳定浅深主题 axe、响应式 E2E、原型同状态对照与 100k Chromium 主档共同形成 Integrated Done 证据 | 3（Integrated Done） |
| `FR-SRH-001～004` 搜索过滤排序 | `MVP-2026-07-23` | Frozen r1；[S4-001 Contract Ready](../gates/MVP-2026-07-23/s4-search-contract-ready.md)；[S4-002 Implemented](../gates/MVP-2026-07-23/s4-search-keyset.md)；[S4-003 Backend Ready](../gates/MVP-2026-07-23/s4-search-backend-ready.md)；[S4-004 Frontend Done](../gates/MVP-2026-07-23/s4-frontend-search.md)；[S4-005 Search Preview Done](../gates/MVP-2026-07-23/s4-frontend-search-preview.md)；[S4-006 Viewer Done](../gates/MVP-2026-07-23/s4-frontend-media-viewer.md)；[Stage 4 Integrated Done](../gates/MVP-2026-07-23/s4-search-media-integrated-done.md) | `BASELINE-2026-07-23` | `internal/catalog` 拥有 search profile、scope、排序与 cursor；SQLite FTS adapter；`web` search feature 的 URL/query owner；共享 `SearchInput`、`MediaCollection`、`MediaPreview` 与 `MediaViewer` | 当前目录（可递归）/当前库/全部库搜索，类型和 filesystem mtime 半开区间过滤，结果预览/查看/返回 | migration 10 的 `assets.search_*_key`、外部内容 FTS5 `asset_search`、`catalog_search_state.revision`；库内 generation 与跨库 revision cursor；URL 保存搜索指纹，TanStack Query 保存游标页面，路由瞬时状态仅保存已载入结果 ID 序列 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[数据模型](../data-model.md)、[S4 搜索契约](../api-design.md#s4-搜索契约) | R-005、R-012、R-016 | S4-002～003 已完成搜索实现、100k/10k 主档与 Backend Ready；S4-004～009 已连接真实搜索、共享非模态预览和完整查看器，并以一次性真实后端验证查询 URL、固定预览筛选保留、进入查看器、返回查询与卡片焦点恢复。Stage 4 Integrated Done，最终浏览器/真机与发布证据归 Stage 5 | 4（Integrated Done） |
| `FR-MED-001～008` 缩略图、查看器与视频 | `MVP-2026-07-23` | Frozen r1；[S3-004 媒体处理](../gates/MVP-2026-07-23/s3-media-processing.md)；[S3-005 媒体任务/缓存](../gates/MVP-2026-07-23/s3-media-jobs-cache.md)；[S3-006 资源安全](../gates/MVP-2026-07-23/s3-media-resource-safety.md)；[S3-007 Backend Ready](../gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)；[S3-105 前端媒体预览](../gates/MVP-2026-07-23/s3-frontend-media-preview.md)；[S3-106 固定预览交互](../gates/MVP-2026-07-23/s3-frontend-pinned-preview.md)；[Stage 3 Integrated Done](../gates/MVP-2026-07-23/s3-browse-integrated-done.md)；[S4-005B Content Backend Ready](../gates/MVP-2026-07-23/s4-media-content-backend-ready.md)；[S4-006 Viewer Done](../gates/MVP-2026-07-23/s4-frontend-media-viewer.md)；[S4-007 Media Strategy Done](../gates/MVP-2026-07-23/s4-frontend-media-strategy.md)；[S4-008 Media Matrix Done](../gates/MVP-2026-07-23/s4-frontend-media-matrix.md)；[Stage 4 Integrated Done](../gates/MVP-2026-07-23/s4-search-media-integrated-done.md) | [CR-2026-001](../changes/CR-2026-001-non-modal-media-preview.md) | `internal/media` 拥有结果/错误/fingerprint/资源上限；`internal/thumbnail` 拥有派生键、交付状态和缓存策略；`internal/jobs` 拥有 worker/lease；`internal/app` 拥有 native lifecycle；`internal/files`、SQLite/cache、govips、FFmpeg adapter；`web/src/lib/media/availability.ts` 是媒体可用性呈现策略 owner；共享 `MediaPreview` 是非模态 preview owner，`MediaViewer` 是完整查看 chrome owner，`web/src/features/media` 组合详情 query 与路由 | 资产详情、thumbnail、原内容/Range；非模态预览、完整图片/GIF 查看器、原生视频 poster、不兼容/损坏/离线/删除状态、缓存设置 | migration 8 `assets`/`thumbnails`；migration 9 `media_jobs`/fairness/`cache_deletions`；source fingerprint/transform version；缓存文件在 `/app/data/cache`；查看器仅缓存服务端详情与瞬时 ID 序列 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[安全模型](../security.md)、[数据模型](../data-model.md) | R-006、R-007、R-008、R-009、R-014、R-016 | S3-004～007 已实现缩略图/媒体 processing 与认证交付；S4-005B 完成原内容 GET/HEAD/Range、admission、取消和错误脱敏；S4-006～008 完成查看器、GIF/视频、唯一降级策略、Chromium 桌面/Pixel 5 触摸、真实 `206 Range` 和 codec/offline/deleted 矩阵；S4-009 以真实搜索纵向链完成 Stage 4 Integrated Done。Firefox、Safari/WebKit 与物理设备仍归 Stage 5 | 0/FS-01/03；3（thumbnail/preview Integrated Done）、4（search/media Integrated Done） |
| `FR-MED-009～011`、`FR-UI-008` 视频故事板悬停预览 | `POST-MVP-1` | Frozen r1；[VSP-S2 Backend Evidence Ready](../gates/POST-MVP-1/vsp-s2-backend-evidence-ready.md) Go；[VSP-S3 Consumer/UI Ready](../gates/POST-MVP-1/vsp-s3-consumer-ui-ready.md) Go；[VSP-301 Product Vertical](../gates/POST-MVP-1/vsp-301-product-vertical.md) Done；[VSP-302 Target Platform](../gates/POST-MVP-1/vsp-302-target-platform.md) Pending | [CR-2026-004](../changes/CR-2026-004-video-storyboard-preview.md) | `internal/thumbnail` 拥有 variant/采样/派生/缓存；`internal/media` FFmpeg adapter；`internal/jobs` 调度；`internal/files` 安全打开；SQLite/cache/API adapter；共享 `MediaCollection` 和唯一 availability adapter | authenticated `thumbnail?variant=storyboard` 已冻结并实现；poster 后低优先级生成；桌面 fine-pointer hover 按需加载；touch/键盘/reduced-motion 回退 poster | migration 11 已实现 replacement tables、4/10 帧 layout CHECK、priority、source fingerprint/transform version、原子发布、统一 LRU 和 128 项 bounded admission；Linux 100k/10k、10% 视频档通过 | [FTR-VID-001](../features/video-storyboard-preview.md)；复用 ADR-0001/0006/0007/0009；若改变 job 一致性/所有权则先新增 ADR | R-006、R-009、R-013、R-015、R-016、R-018 | [VSP-002](../spikes/vsp-002-video-storyboard.md)验证 2s～2h fast seek/sprite；OpenAPI、生成 client、双架构 FFmpeg runtime、生产镜像 API/cache repair、故障矩阵、race 和 Linux 四核/4 GiB 容量均通过；共享 hover/input/reduced-motion、组件工作台、六种浏览器/输入 profile、100-video 与 100k 三浏览器容量通过；生产镜像真实登录、浏览/搜索 hover、预览焦点恢复已贯通；[readiness 快照](../releases/POST-MVP-1-readiness.json)及 fitness test 以证据锚点校验 Gate/AC/风险/任务并保持 No-Go；原生双架构结构化证据入口已建立，当前远端 runner 在分配前受账户计费阻断，模拟 amd64 按安全边界失败关闭，不能替代 VSP-302 | `VSP-S0 Done → S1 Done → S2 Done → S3 Done → VSP-301 Done → VSP-302 Pending → VSP-303/304` |
| `FR-MED-013` 扫描后派生媒体进度 | `POST-MVP-1` | Frozen r5；只读纵向切片实现 | [CR-2026-014](../changes/CR-2026-014-derived-media-progress.md) | `internal/thumbnail` 拥有聚合语义；SQLite 读取现有 durable job；API/Web adapter 与生成 client | `GET /libraries/{libraryId}/media-processing`；状态页把 scan、thumbnail/poster、storyboard 分开；active 时有界轮询 | 无 migration；只读取 `assets`/`media_jobs`，不创建通用 operation run | 复用 ADR-0001/0006/0007/0009；R-005/R-013/R-015/R-016 | SQLite/API/service unit、OpenAPI/生成、Web type/unit、架构与契约检查 | `POST-MVP-1 r5 maintenance slice` |
| `FR-MED-012` AVI、`FR-BRW-011` 文件大小排序 | `POST-MVP-1` | Frozen r2；实现与本地验证完成 | [CR-2026-010](../changes/CR-2026-010-avi-and-size-sort.md) | `internal/media` 统一格式注册和 FFmpeg adapter；`internal/catalog` 排序/cursor owner；SQLite/API adapter；浏览与搜索 URL/UI owner | `.avi` 索引、poster/storyboard、原内容 Range；`sort=size&order=asc|desc`；默认排序不变 | migration 14 保留 catalog/FTS/派生外键并追加 AVI CHECK 与 size indexes；复用 `assets.size_bytes`、source fingerprint、既有派生任务与缓存 | 复用 ADR-0001/0006/0007/0009；AVI 不承诺浏览器直放；R-006/R-007/R-012 | 格式/MIME、真实 FFmpeg AVI、超限与损坏、同大小跨页 size keyset、OpenAPI/生成、URL/UI 双语、完整 Go/Web/集成与容器 smoke 已通过 | `Post-MVP/1 optimization implemented` |
| `FR-BRW-012` 选中目录媒体类型数量 | `POST-MVP-1` | Frozen r3；实现与本地验证完成 | [CR-2026-011](../changes/CR-2026-011-directory-media-counts.md) | `internal/catalog` 聚合语义；SQLite adapter；AssetPage OpenAPI；浏览类型控件 | direct/recursive/q 范围的 all/images/videos，kind 与分页不改变统计基准 | 复用 assets/FTS，无 migration | R-005/R-012/R-015；不访问文件系统 | SQLite 范围/类型统计、生成 client、浏览控件数字与可访问名称、Go/Web 回归 | `Post-MVP/1 optimization implemented` |
| `FR-SET-001`、`NFR-PERF-004` NAS 资源模式 | `POST-MVP-1` | Frozen r4；实现与本地验证完成 | [CR-2026-012](../changes/CR-2026-012-nas-resource-profiles.md) | `internal/resourcecontrol` 拥有实例级 profile/许可；`internal/settings` 持久化与编排；`internal/app` 启动恢复与 processor 组合；API/SQLite/Web adapter | 配置的扫描页签选择 NAS 友好/均衡/性能；后台 1/2/4，共享覆盖 full scan/reconcile/media；内容读取 4/8/16 | migration 16 `settings.resource_profile`；强 ETag/If-Match；durable jobs 与 active work 不变 | 复用 ADR-0001/0006/0007/0009；R-004/R-006/R-009/R-013/R-016；不提高既有硬上限 | 动态收缩/扩展、取消、SQLite CHECK/default、设置 HTTP、OpenAPI/生成、Web type/test/Storybook、Go/架构/生成检查 | `Post-MVP/1 optimization implemented` |
| `FR-SCN-010～014` 媒体库自动发现 | `POST-MVP-2` | Frozen r3；WCH-S0 Go；[WCH-S1 Contract Ready](../gates/POST-MVP-2/wch-s1-contract-ready.md) Go；[WCH-S2 Backend Evidence Ready](../gates/POST-MVP-2/wch-s2-backend-evidence-ready.md)发布 No-Go、有限授权 S3 本地 UI | [CR-2026-005](../changes/CR-2026-005-automatic-library-discovery.md) | `internal/scanner` 拥有事件合并、定向范围、清理资格和 content revision；`internal/files` 实现 Linux watch/安全重开；`internal/jobs` durable lease；SQLite/API adapter；Web catalog-state consumer 统一条件检查与 active query 刷新 | 自动发现设置/状态；新增、修改、删除、rename 的定向校准；媒体库状态与刷新；可见相关页面 5 秒 ETag 条件检查；不推送 | migration 12 专用 `catalog_reconcile_jobs` requested/claimed 水位；library/global `content_revision`；`cache_deletions`；settings 开关；完整扫描/定向任务按库互斥 | [FTR-SCN-001](../features/automatic-library-discovery.md)；ADR-0003/0009；[ADR-0011](../adr/0011-linux-inotify-hints-and-anchored-reconciliation.md)已接受 | R-002、R-003、R-005、R-013、R-016、R-019 | 后端 Linux/arm64 证据矩阵已完成；revision 3 增加 full-scan cache tombstone、自动发现状态/刷新和 5 秒可见页面条件检查。S2 发布仍需原生 amd64 | `WCH-S0 Done → S1 Done → S2 release No-Go / S3 local authorized → S4` |
| `FR-OPS-001～011`、`NFR-OPS-002～004` 后台任务、日志、版本与消息中心 | `POST-MVP-3` revision 1 | Scope Frozen；失败诊断/恢复、系统事件与 release-info 纵向切片 In Delivery；统一 parent task Pending | [CR-2026-008](../changes/CR-2026-008-task-center.md)、[CR-2026-015](../changes/CR-2026-015-operations-observability-and-updates.md)、[关于页直接展示更新内容](../changes/FIX-2026-08-04-inline-release-notes.md) | `internal/thumbnail` 拥有 derived failure/恢复及结构化 attempt；`internal/systemlog` 拥有有界脱敏系统事件；`internal/releaseinfo` 拥有版本比较、缓存和有界官方 Release 正文；scanner/jobs 保持原状态 owner；SQLite/API/Web adapter 与 Header 通知投影 | 配置合并；媒体库扫描记录/处理结果及行内尝试详情；完整扫描/失败恢复分离；系统日志；关于页内联版本正文/更新；可确认的聚合全局消息 | 复用 `scan_runs`/`media_jobs`/assets/libraries；migration 17 追加最多 5,000 条 `system_events`；migration 18 每 job 最多保留 10 条无原始 stderr 的 `media_job_attempts`；浏览器保存 completed acknowledgement 与 latest failure watermark；后续 `operation_runs` 仍由 parent task S1 冻结 | FTR-OPS-001、POST-MVP-3 scope r1；复用 ADR-0001/0006/0007/0009 | R-003、R-006、R-009、R-011、R-013、R-015、R-016、R-020 | Go/SQLite/HTTP tests 覆盖失败分页、结构化 attempts、transient/permanent 恢复、系统事件分级/脱敏/保留；OpenAPI/generated client、Release Markdown 安全文本投影、媒体库记录/行内诊断、系统日志/关于/消息消费者已接入，完整 evidence 与 parent run 容量仍待收敛 | `OPS diagnostics S1→S4；parent task S0→S4` |
| `FR-UI-001～007` 界面与可访问性 | `MVP-2026-07-23` | Frozen r1；[S4-005 搜索预览 Done](../gates/MVP-2026-07-23/s4-frontend-search-preview.md)；[S4-006 Viewer Done](../gates/MVP-2026-07-23/s4-frontend-media-viewer.md)；[S4-007 Media Strategy Done](../gates/MVP-2026-07-23/s4-frontend-media-strategy.md)；[S4-008 Media Matrix Done](../gates/MVP-2026-07-23/s4-frontend-media-matrix.md)；[Stage 4 Integrated Done](../gates/MVP-2026-07-23/s4-search-media-integrated-done.md) | [CR-2026-001](../changes/CR-2026-001-non-modal-media-preview.md)；[CR-2026-003](../changes/CR-2026-003-brand-identity.md)；[redesign 应用壳恢复](../changes/FIX-2026-07-29-redesign-shell-restoration.md)；[原型品牌对齐](../changes/FIX-2026-07-30-prototype-brand-alignment.md)；[Header 快速语言切换](../changes/FIX-2026-07-31-header-locale-toggle.md) | `web` 的 auth/libraries/browse/search/media/settings feature；统一 API client；共享 `BrandMark`、`LocaleToggle`、`AsyncState`、`InlineStatus`、`FormField`、`Button`、`Toast`、`MediaCollection`、`MediaPreview`、`MediaViewer`、`MediaAvailabilityState` 与 `useMediaPreviewController` | 欢迎/认证、设置、扫描状态、浏览/搜索/预览/查看器；统一品牌入口；桌面固定侧栏和底部上下文导航、移动抽屉、中英与主题；设置与 Header 双入口语言切换；UIF-316 八类状态工作台矩阵；UIF-317 locale/theme/四档/input 矩阵 | 服务端状态不复制为独立事实；URL 保存导航状态，`settings` 保存受控偏好；`LocaleProvider` 唯一拥有语言偏好和即时切换；固定预览保存当前媒体快照，查看器瞬时状态只保存已载入 ID 序列和安全返回位置；品牌 SVG 是无业务状态的静态资产 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0007](../adr/0007-shared-frontend-system.md)、[用户流程](../user-flows.md)、[界面设计](../ui-design.md)、[品牌标识](../branding.md) | R-005、R-010、R-015、R-016 | loading/empty/offline/error/conflict/cancel/pending/success 由唯一共享组件语义和真实领域状态覆盖；工作台接入真实 locale/provider 和主题 token，并保存 2×2×4 确定性视觉矩阵；真实设置页笛卡尔 E2E 检查 lang/theme/overflow/axe/reduced-motion；共享 locale toggle、应用壳和设置页回归固定持久化、按钮顺序、`html[lang]` 与双入口同步；既有纵向链和媒体矩阵覆盖键盘焦点返回、Pixel 5 触摸、三引擎与 forced-colors；其余证据包括预览/查看器、媒体降级、原型同视口对照、虚拟化和关键 E2E | 1（认证 UI/品牌原语）、3、4（Integrated Done）；5（发布浏览器/真机加固） |

### Stage 3 浏览例行修复

- [FIX-2026-07-29 浏览媒体类型三态筛选](../changes/FIX-2026-07-29-browse-media-filter.md)
  继续归 `FR-BRW-004～005` 与既有 S3 浏览合同所有；`web/src/features/browse` 唯一拥有
  `kind=all|image|video` URL 语义、query key 与工具栏组合，catalog adapter 复用已冻结
  `kind` 参数。证据覆盖 URL 恢复、`image,animated` 分组、`video` 参数、cursor 重置及
  三态可访问控件；无 API、数据、信任边界或部署变化。

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
| `NFR-SEC-001～002` 路径与网络安全；`NFR-PRIV-001` 信息披露 | `MVP-2026-07-23` | Frozen r2；S5-003 代理安全 + CR-2026-002 LAN HTTP | `BASELINE-2026-07-23`、`CR-2026-002` | `internal/files`、`internal/auth`、API transport/middleware、错误/日志适配器、可选反向代理信任配置 | 逃逸与 mount crossing 失败关闭；无有效会话不返回业务数据；直连 LAN 转发头不能改变安全判断；代理模式严格验证 | ADR-0002、ADR-0005、ADR-0009、ADR-0010；R-002、R-010、R-012、R-016 | request ID、脱敏错误、认证/CSRF/限流、直连 LAN transport、可选严格 HTTPS 代理及 openat2 边界测试；发布 volume 与 LAN 实机拓扑继续由 Stage 5 验证 | 0、1～5 |
| `NFR-REL-001` 扫描与恢复一致性 | `MVP-2026-07-23` | Frozen r1 | `BASELINE-2026-07-23` | scanner generation 状态机、SQLite 短事务、restart-safe jobs、原子缓存落盘 | 只有完整成功扫描可清理；崩溃、取消、离线和部分错误保留可靠状态并可收敛 | ADR-0003；R-003、R-004、R-011、R-013 | FS-02 generation 故障矩阵、FS-05 离线恢复/满盘/损坏失败关闭与 S2-105 production startup/late-lease 恢复已通过；S5-004 又以原生双架构不同不可变候选完成强杀恢复、向前升级和配对数据回滚。在线备份不在当前自动化承诺内 | 0、2、5 |
| `NFR-REL-002`、`NFR-PERF-003` watcher 一致性与资源边界 | `POST-MVP-2` | Frozen r3；WCH-S0/S1 Go；S2 发布 No-Go；S3 Conditional Go | [CR-2026-005](../changes/CR-2026-005-automatic-library-discovery.md) | scanner 定向校准与 content revision、files Linux watcher、jobs/SQLite durable admission、thumbnail durable cache deletion、Web 有界条件检查与显式刷新 | 事件不是清理资格；丢失/重复/乱序/overflow 后完整扫描收敛；每目录 watch 和所有队列/刷新有界；相关可见页面最多每 5 秒一次 catalog-state 条件请求，`304` 不重取 catalog | ADR-0003/0009/0011；R-003、R-005、R-013、R-019 | WCH-001 已验证 Linux/arm64 10k watch 与真实 overflow；生产实现固定 32,768 watch、8,192 event、4,096 dirty、2 global/1 per-library 并发，通过 Linux/arm64 纵向链、全应用回归、100k 进程内 burst 有界 overflow、dirty 硬上限、双 worker 跨库 claim、跨 SQLite 重开与独立领取进程强杀恢复、受控内核 ENOSPC 回滚和真实 nested mount/unmount 恢复；四核/4 GiB 的 100k/10k 档进一步覆盖 10,001 watch 注册且进程 FD 不按目录增长、100 个跨目录增量发布、P95 与进程写调用预算；S3 覆盖显式刷新、目录返回重取、cursor 首页面裁剪、full-scan cache tombstone 与 ETag 条件检查。S2 发布仍需原生 amd64 | `WCH-S0～S4` |
| `NFR-PERF-001～002` 资源与容量 | `MVP-2026-07-23` | Frozen r1；[S5-005 候选容量](../gates/MVP-2026-07-23/s5-release-capacity-candidate.md)；[S5-005C 原生 amd64 真实媒体](../gates/MVP-2026-07-23/s5-native-amd64-real-media.md)；[S5-005D 原生 amd64 目标容量](../gates/MVP-2026-07-23/s5-native-amd64-capacity.md)；[UIF-405 revalidation](../evidence/uif-405/README.md) passed | `BASELINE-2026-07-23` | 有界队列/并发、串行批量写、keyset、虚拟化、缓存水位、全局工作调度 | 四核/4 GiB、约 10 万媒体/1 万目录为主验收档；扫描时浏览仍可用，发布预算由代表性设备 Gate 固定 | ADR-0001、ADR-0003；R-005、R-009、R-013、R-015 | [FS-04](../spikes/fs-04-capacity-baseline.md)、[S2-106](../gates/MVP-2026-07-23/s2-scan-capacity.md)、[S4-003](../gates/MVP-2026-07-23/s4-search-backend-ready.md)和[S3-107](../gates/MVP-2026-07-23/s3-frontend-capacity.md)已固定扫描、搜索与前端有界 DOM。S5-005 在本机 arm64 与指定原生 amd64 通过 100k/10k 扫描、认证查询、重扫、取消/offline、100k 全量派生、cache 90%→80% 水位、持续健康和三引擎 FPS/RSS；UIF-405 又在最新共享集合上复验三引擎 100k 滚动均挂载 60 项并通过 FPS/RSS，后端完整 10k/100k 扫描期 2,353 次浏览/搜索并发且 0 budget violation | 0/FS-04、2～5（S5-005 Passed） |
| `NFR-ACC-001` 可访问性 | `MVP-2026-07-23` | Frozen r1；[S5-006A 浏览器质量候选](../gates/MVP-2026-07-23/s5-browser-quality-candidate.md)；[UIF-404 automated applicability](../evidence/uif-404/README.md) passed | `BASELINE-2026-07-23` | 语义 HTML、DOM 顺序、焦点管理、状态文案、主题与 reduced-motion token | 核心流程键盘可完成，状态不只依赖颜色，目标 WCAG 2.2 AA | UI 设计；R-015、R-016 | Chromium/Firefox/WebKit 共同验证查看器焦点、键盘、降级状态、overflow 与 axe serious/critical；移动 Chromium touch、Chrome forced-colors 和 storyboard reduced-motion 通过；Chrome 原生 200% 及 Firefox 153.0.1 核心链/原生 200%/400% 已在物理 Mac 通过。自动化和缩放证据不冒充读屏、物理 OS 高对比或触摸签署；这些项目、移动设备、Safari 缩放和最终视觉仍由 S5-006B 阻断 | 1、3、4、5 |
| `NFR-UIF-001` 生产页面原型一致性 | `MVP-2026-07-23` | Frozen r4；[UIF-S4 Integrated Slice Done](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md) Go；Phase 3 `UIF-301～318` 与 Phase 4 `UIF-401～408` 完成 | [CR-2026-009](../changes/CR-2026-009-frontend-prototype-fidelity.md) | 中央 token、共享壳、机器校验 reference manifest、Linux-owned visual regression、真实浏览器纵向链 | 四档同状态比较无 P0/P1/P2；主要区域几何偏差不超过 2px；基线更新可审计；真实链不修改原媒体 | R-015、R-016、R-021 | [当前 UIF 集成状态](../releases/MVP-2026-07-23-uif-integration-status.md)逐项聚合 UIF-401～408：12 页共同 1280 与 12 页 × 4 断点原型/生产比较、独立双语双主题矩阵、11 张 Linux 基线、真实纵向链、浏览器/可访问性、100k/10k、完整仓库验证和文档收敛；受影响 Stage 5 复验通过当前合同 | `UIF-S4 Integrated Done → Stage 5 No-Go` |
| `NFR-COMP-001` 平台兼容 | `MVP-2026-07-23` | Frozen r1；[S5-006A 浏览器质量候选](../gates/MVP-2026-07-23/s5-browser-quality-candidate.md)；[S5-007A 候选供应链](../gates/MVP-2026-07-23/s5-supply-chain-candidate.md)；[S5-007C 最小媒体运行时](../gates/MVP-2026-07-23/s5-minimal-media-runtime.md)；[S5-007D 最小 FFmpeg 运行时](../gates/MVP-2026-07-23/s5-minimal-ffmpeg-runtime.md)；[S5-007E 内建健康检查](../gates/MVP-2026-07-23/s5-built-in-healthcheck-runtime.md)；[S5-007F 无 shell 最小运行时](../gates/MVP-2026-07-23/s5-distroless-runtime.md)；[S5-007G 修复来源 GLib](../gates/MVP-2026-07-23/s5-patched-glib-runtime.md)；[S5-008 发布文档](../gates/MVP-2026-07-23/s5-release-documentation.md) | `BASELINE-2026-07-23` | Debian-family 构建层、distroless final stage、Go/CGO、最小 libvips/FFmpeg、内建健康检查、浏览器兼容层 | 承诺的 linux/amd64、linux/arm64 和主流浏览器行为必须由同一 fixture 验证 | ADR-0001；R-007、R-008、R-014、R-017 | FS-03 与 S5-002 双架构媒体/runtime 已通过；S5-006A 已把 Chromium/Firefox/WebKit 稳定状态和 Linux 视觉回归接入 CI，Safari 26.5.2 真机链、Chrome normal/forced-colors、Chrome 原生 200% 及 Firefox 153.0.1 核心链/原生 200%/400% 已通过；候选已建立确定性 SPDX、双架构扫描/notices 和 provenance 入口，修复来源 GLib 的本机 arm64 候选达到 `0 Critical / 0 High`。最终原生双架构 digest、全阻断复扫、安全/合规决定、读屏/物理触控/移动设备、Safari 缩放和最终视觉签署仍待 Gate | 0/FS-03/05、5 |
| `NFR-OPS-001` 可运维性 | `MVP-2026-07-23` | Frozen r1；[S5-008 发布文档](../gates/MVP-2026-07-23/s5-release-documentation.md)；[S5-009A 当前 RC No-Go](../gates/MVP-2026-07-23/s5-release-candidate-current.md) | `BASELINE-2026-07-23` | `internal/app` 配置/生命周期、SQLite WAL、health、日志/指标、migration、备份恢复、缓存清理 | 本地可靠文件系统；安全启动/退出/升级/恢复；缓存和磁盘问题不能破坏配置或原件 | ADR-0001；R-004、R-009、R-011、R-016 | `internal/app`/`internal/api` 已覆盖固定根、安全监听、日志、health、迁移失败关闭、HTTP 排空与停机；S5-004A 已验证候选离线恢复、强杀/WAL、满盘和损坏 DB，S5-004B 已在原生双架构验证不同不可变候选间的向前升级和配对回滚，S5-008 已固定 runbook；S5-009A 以机器快照聚合八 Gate/八风险并让 promotion 失败关闭 | 1、5 |

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
