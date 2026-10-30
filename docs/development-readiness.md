> 2026-10-30 当前删除修订已完成 DELETE-S4 并发布 v1.10.0，见 [正式记录](changes/RELEASE-2026-10-30-direct-deletion.md)。旧阶段 No-Go 为历史状态，默认只读和显式启用不变。

2026-10-30 / DELETE：S1 合同及本地 S2 后端门禁通过，允许 S3 页面开发，见 [S1](gates/POST-MVP-DELETE/delete-s1-contract-ready.md)、[S2](gates/POST-MVP-DELETE/delete-s2-backend-ready.md)。用户已接受内核写隔离，不支持时保持不可用；生产镜像、双原生、容量及真实可写部署仍等待 S4，不因 UI 接入自动启用。

2026-10-29 / ARCHIVE-S2：7z合同与资源边界先行，Backend Ready后已接入正式UI；本地Linux arm64镜像与浏览验收通过，后续v1.9.0双原生门禁和正式推广亦已通过。实际Gate见[CR-2026-056](changes/CR-2026-056-7z-image-browsing.md)。

2026-10-29 / POST-MVP ZIP-S1：用户已授权独立分支实现并在试用后要求合并发布，S0 以 ADR-0040、CR-2026-055、zip-source-contract 和加法 OpenAPI 为输入；先完成真实扫描/读取/存储回归，再开始正式 UI，最终合成 Linux 实例和四档宽度验证。v1.8.0 原生双架构发布门禁已通过，见[正式记录](changes/RELEASE-2026-10-29-v1.8.0.md)；Gate 状态见[变更记录](changes/CR-2026-055-zip-image-browsing.md)。

# FolioPath 开发就绪评审

CR-2026-052：消费已通过CR050的OCR管理Backend Ready与CR051检索合同，仅推进正式设置/任务UI与稳定Story；不新增API或模型行为，不自动推进原6010或质量/容量/发行Gate。

2026-10-26 / CR-2026-051 / ADR-0037：批准OCR全文索引/融合搜索backend切片，范围FR-OCR-005、FR-SEARCH-001～003。结果/文档/checkpoint事务、部署identity/源/库可见性、快照分页及片段合同先行；生产组件消费此合同，稳定Story为合成证据。验收明细记录在CR，不自动推广原6010/管理UI或质量/联合容量/发行Gate。

2026-10-26 / CR-2026-050 / ADR-0036：接受OCR runtime/显式管理backend子片：安全源读取/复核、现有WorkerPool/资源准入、计划handoff及库作用域认证/CSRF/CAS API。默认无部署OCR配置则无路由/worker，有配置也不自动开库或提交任务。此候选backend的真实模型Linux三进程同任务恢复、52项/race、17表保护与完整性、Go全量/集成/runtime和合同检查已通过。仅这一管理backend Gate可供后续设计消费；搜索/UI/6010部署及质量/容量/发行仍需独立验收。见[记录](changes/CR-2026-050-ocr-runtime-management.md)。

2026-10-26 / CR-2026-049 / ADR-0035：接受独立OCR源绑定结果/持久任务事务/分批清理的内部存储合同（FR-OCR-003）。仅允许实现与合成数据库验证；默认关闭，无API/运行worker/自动启动/搜索/UI/6010激活。runtime及公开管理合同需后续Gate。该内部子片最终固定镜像33项/race、Go全量/集成、runtime与17表保护验收已通过；仍非完整OCR Backend Ready。验证结果见[本轮记录](changes/CR-2026-049-ocr-durable-results.md)。

2026-10-26 / CR-2026-048：OCR producer与共享准入候选已通过隔离实际模型/Go消费者联验，包含缺OCRcache时旧功能继续工作、原生内存取消恢复与冻结身份/输出对照。只允许后续设计源代次/持久任务/事务/清理合同；**尚非全OCR Backend Ready或产品启用**。90°/版面、真实质量、联合容量/双架构/发行及原6010部署继续独立。[producer记录](changes/CR-2026-048-ocr-producer.md)。

2026-10-25 / CR-2026-047：独立OCR安全imagevips输入子片已实现并在本地arm64原生Docker验证，允许下一步审查服务producer/共享调度注册合同；不是全OCR Backend Ready。现有scheduler源码属于旧runtime身份，不能为OCR注册无声重标CLIP/tag/face，或创建第二native准入owner。源指纹/代次、持久任务/事务、结果清理及搜索/UI仍待接受与实施。[输入证据](changes/CR-2026-047-ocr-image-input.md)。

2026-10-25 / CR-2026-046 / ADR-0032：接受OCR独立身份/结果合同与Go传输消费者，仅传输子片可交付。严格预配身份、错误分类、共享准入/取消/截止与隔离Docker证据见[记录](changes/CR-2026-046-ocr-transport.md)。**这不是OCR生产Backend Ready**；AI服务注册、internal/files/imagevips输入合同、源指纹/代次、任务/事务、存储/清理和搜索/UI尚待后续门槛，禁止先行启用。90°方向和真实质量继续未通过。

2026-10-25 / CR-2026-045 OCR：横排离线推理S0已取得可复跑证据，允许继续设计独立OCR内部合同与源指纹/代次、持久化及任务清理语义。**生产Backend Ready仍未签署，OCR搜索/UI保持未接入**；90°旋转图0/4，需先明确旋转/版面范围与质量门槛，不能将进程ready当识别准确。完整记录见[OCR S0](changes/CR-2026-045-ocr-feasibility.md)。

2026-10-23 PNG 语义兼容切片：ADR-0028 接受显式版本化输入合同，Go/native 与真实模型合成 Docker
已通过，允许候选交付；原库部署验收尚未完成。用户授权当前任务终态、完整停写备份后再切换及重算图片语义。
不授权中断现有任务、重标旧向量、升级标签策略或跳过质量/容量/发行门禁。[切片证据](changes/FIX-2026-10-23-semantic-large-png.md)。

2026-09-14 ADR-0019/CR-2026-030 接受独立 AI 服务方向，目前通过本地离线协议 spike。
旧 INT Backend Ready 不自动覆盖新 HTTP/模型代次合同；允许继续合同设计与隔离验证，
正式主应用和 UI 集成仍须先补齐新切片 Backend Ready。
已接受[传输子片合同](architecture/independent-ml-transport-contract.md)，允许独立 adapter
实现及隔离测试，不扩展至模型身份准入、持久化或主应用组合。

## 当前结论

[FTR-INT-001 本地智能媒体发现](features/intelligent-media-discovery.md)是独立的
[`POST-MVP-5` revision 2](releases/POST-MVP-5-scope-r2.md)，不继承 MVP 的 Stage 0 结论。其
[INT-S0 Architecture Ready](gates/POST-MVP-5/int-s0-architecture-ready.md)只对 A 模型基础 + B 图片语义
搜索为 **Go**，[INT-S1](gates/POST-MVP-5/int-s1-contract-ready.md)也已 Go，并曾授权
`INT-201～216` S2A 后端实现；[INT-S2A](gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)、
[INT-S2B](gates/POST-MVP-5/int-s2b-backend-evidence-ready.md) 与
[INT-S2C](gates/POST-MVP-5/int-s2c-privacy-ready.md) 均已签署 **Backend Ready / Release No-Go**，
[INT-S3 Consumer/UI Ready](gates/POST-MVP-5/int-s3-consumer-ui-ready.md) 也已完成真实合同消费者。
按 [CR-2026-022](changes/CR-2026-022-s2-backend-release-gate-separation.md)，最终模型、governed 质量、
隐私签署、native 双架构、4 GiB 完整进程和供应链由 S4 持有；
S4 未 Go 时 reviewed catalog、face composition 与发行 UI 继续失败关闭。revision 2 仍只承诺
`/models:ro`，不包含国内或在线镜像；S3 对齐记录见
[CR-2026-023](changes/CR-2026-023-s3-offline-model-ui-contract-alignment.md)。

当前首要切片 [FTR-UIF-001 生产前端原型一致性](features/frontend-prototype-fidelity.md)已通过
[UIF-S0 Architecture Ready](gates/MVP-2026-07-23/uif-s0-architecture-ready.md)与
[UIF-S2 Backend Evidence Ready](gates/MVP-2026-07-23/uif-s2-backend-evidence-ready.md)、
[UIF-S3 Consumer/UI Ready](gates/MVP-2026-07-23/uif-s3-consumer-ui-ready.md)，授权
`UIF-401～408` 逐页比较、真实纵向集成和发布前复验；其中 `UIF-401～403` 已完成，
当前进入 `UIF-404` 浏览器与可访问性复验。账户维护、直接子目录 `q`
和缓存清理已通过生成 client 接入；完整执行顺序见
[专用任务清单](features/frontend-prototype-fidelity-task-list.md)。

项目已通过 **Stage 0 开发就绪 Gate**，并完成 Stage 1 的 Go 运行骨架与单管理员认证后端。
[认证 Backend Evidence Ready](gates/MVP-2026-07-23/s1-auth-backend-ready.md)结论为 `Go`，
允许认证产品 UI 连接真实 API。[媒体库 Backend
Ready](gates/MVP-2026-07-23/s2-library-backend-ready.md)也已结论为 `Go`，允许前端为
7 个媒体库管理 operation 建立真实 client adapter；[可靠扫描 Backend
Ready](gates/MVP-2026-07-23/s2-scan-backend-ready.md)也已结论为 `Go`，允许前端接入
扫描历史、详情轮询、手动请求、取消和计划设置。[S3-001 目录与媒体浏览 Contract
Ready](gates/MVP-2026-07-23/s3-browse-contract-ready.md)已通过；[S3-002 Catalog 排序与
游标](gates/MVP-2026-07-23/s3-catalog-keyset.md)和[S3-003 目录树与
详情](gates/MVP-2026-07-23/s3-directory-tree.md)和[S3-004 媒体处理
基础](gates/MVP-2026-07-23/s3-media-processing.md)、[S3-005 媒体任务与缓存
保护](gates/MVP-2026-07-23/s3-media-jobs-cache.md)、[S3-006 敌意媒体与资源
安全](gates/MVP-2026-07-23/s3-media-resource-safety.md)和[S3-007 浏览与缩略图 Backend
Ready](gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)已完成，允许前端接入
资产分页、详情和缩略图。[S4-001 搜索 Contract
Ready](gates/MVP-2026-07-23/s4-search-contract-ready.md)已冻结 search profile v1；
[S4-002 搜索实现](gates/MVP-2026-07-23/s4-search-keyset.md)已完成 migration 10、
FTS/keyset、三种 scope、类型/mtime、离线、认证 HTTP 与真实 composition；[S4-003
搜索 Backend Ready](gates/MVP-2026-07-23/s4-search-backend-ready.md)已通过 100k、
扫描并发、取消和 rebuild Gate，允许前端接入冻结的搜索 operation。[S4-005B 原媒体内容
Backend Ready](gates/MVP-2026-07-23/s4-media-content-backend-ready.md)也已通过真实认证、
Range、取消、Linux path boundary 和离线/变化源矩阵，允许前端接入冻结 content operation。
这些结论不把前端、Stage 5 或发布标记为完成，也不授权共享预览或非回环监听。

当前仓库已有 `go.mod`/`.go-version`、Go 路径/媒体库/scanner/SQLite 实验代码、首个嵌入式
Goose migration、权威 `api/openapi.yaml`、确定性 TypeScript 类型生成、唯一 Web API client
边界、OpenAPI 摘要锁与语义兼容检查、契约/HTTP 边界/容量 harness、固定 Node/npm 工具链和
统一本地验证入口。自动测试 CI 已关闭；`cmd/foliopath` 的最小进程入口、版本命令和退出码已经建立；
`internal/app` 已拥有唯一组合点、进程根取消、顺序启动、失败回滚、运行故障传播、反向关闭和
有界停机；启动配置已固定 `/library`、`/app/data`、单监听地址和认证前回环限制。正式应用已
接入 SQLite WAL、嵌入 migration、空数据目录准备、重复启动和迁移失败关闭，并由真实
composition root 集成测试和测试专用容器 smoke 覆盖。root runtime 变更按 ADR-0012
等待当前候选复验。`internal/auth` 已实现
Argon2id 密码存储、Unicode 管理员身份规范化、单管理员原子初始化，以及高熵摘要化会话、
7 天绝对期限、重启恢复、退出撤销和 Cookie 策略，并经 SQLite adapter 接入 composition
root。五个认证 HTTP handler、同源 Origin、session-bound CSRF、业务 API 默认拒绝、
防缓存、受限 JSON 和按直连 peer 的有界限流已经接入真实 composition root，并通过认证
Backend Ready Gate；安全目录选择、媒体库生命周期、manual scan admission 与异步移除
handler 也已接入真实 composition root 并通过媒体库 Backend Ready Gate。生产扫描 worker
已消费 creation/startup/scheduled scan，扫描观察、取消和设置 HTTP 已接线并通过 Backend Ready；
Stage 1～4 认证产品 UI 与 Chromium E2E、Stage 5 可信代理边界、生产候选 Dockerfile、
安全 Compose 和 Docker Hub 双架构发布入口已经建立；当前仍没有可发布镜像，但本机原生
arm64 与操作者指定的原生 amd64 服务器候选矩阵已经审阅通过；S5-006A 已建立
Firefox/WebKit 稳定状态与 Linux 视觉回归，真实 Firefox 核心链及原生 200%/400%
缩放也已通过；其余物理辅助功能/真机与 RC Gate 尚未完成；S5-009A 的统一 RC 审计当前
明确为 No-Go；
HTTP 运行边界已有服务端 request ID、统一安全 404/500、JSON 日志、panic 隔离、在途请求排空、
liveness/readiness 和受保护系统状态；数据库及 migration 成功后 readiness 才进入 ready，
系统状态已使用真实会话保护。隔离 FS-05 Dockerfile 只用于 Stage 0 probe。
历史原生 amd64/arm64、runtime/recovery 与 SBOM/license 结果作为原提交证据保留。
认证、媒体库管理和可靠扫描是已通过各自 Gate 的正式后端能力；静态原型仍不是可用产品。
不得把媒体库 Gate 扩张为扫描、浏览、
缩略图、前端集成或发布完成。

用户已于 2026-07-23 确认[需求确认清单](requirements-checklist.md)中的 RQ-001～RQ-014
全部采用 A。产品冻结门槛已经满足，[系统架构档案](architecture/README.md)也已形成目标
视图、所有权、交付治理和 fitness function 基线；权威 OpenAPI、强制离线解析/结构/引用/
ECMAScript pattern 验证、选择性跨源关键不变量检查（`queued`、`animated`、可空
`startedAt`）与外部 lint 已经存在；这不代表完整 ScanRun 领域实现。FS-03/04 的完整产品
范围与发布级验证已被分配到对应后续 Gate，不反向阻断 Stage 1。架构约束变更仍必须遵循
ADR 流程。

## 开发启动门槛

| 领域 | 开始实现前所需产物 | 当前判断 |
| --- | --- | --- |
| `FTR-UIF-001` | revision 4、feature/CR、owner、API/data/security 影响、视觉真相、验收与 Gate | S0/S1/S2/S3 Go；生产消费者和 UI 矩阵通过，进入 S4 逐页集成与发布前复验 |
| 需求冻结 | MVP 范围、用户流程、验收标准、格式矩阵、日期语义、目标规模、认证/网络边界和明确非目标 | 已就绪；RQ-001～RQ-014 全部确认 A |
| 架构 | 运行拓扑、包边界、路径模型、扫描一致性和数据模型 | 基线已形成；启动配置、应用组合、HTTP listener/中间件、健康状态、数据库/migration 与生命周期已有单元测试 |
| API | `/api/v1` 资源、统一错误、游标、Range 与扫描任务语义；`api/openapi.yaml` 为唯一结构契约 | 权威契约、完整 Go 解析/结构/引用/pattern/语义测试、确定性 TypeScript 类型、唯一 client、摘要锁和真实 PR 基线语义比较已通过；冻结 MVP 的认证、媒体库/扫描、浏览/搜索、设置、资产详情/缩略图/原内容 handler 已接入真实 composition |
| UI/UX | 创建媒体库、扫描状态、目录浏览、递归浏览、非模态预览、查看器和异常恢复的可评审流程；前端分层、共享组件和响应式/无障碍要求 | 2026-07-24 完整静态原型已覆盖 15 个界面/状态；生产 token、共享组件、虚拟化、浅/深主题、390～1440px、axe、Chromium 桌面/移动与 Stage 1～4 纵向 E2E 已通过各自 Integrated Done Gate。S5-006A 又建立 Firefox/WebKit 查看器稳定状态和固定 Linux 视觉基线，Safari 26.5.2、Chrome 151 原生 200% 及 Firefox 153.0.1 核心链/原生 200%/400% 已通过；物理读屏/触控/移动设备、Safari 缩放与最终视觉签署仍待 S5-006B |
| 数据 | 首个 schema、迁移工具、外键/索引、generation 与任务恢复测试方案 | Goose migration 1～10 已执行；媒体库 revision/removal/idempotency、扫描与媒体 durable contract、typed settings、目录/资产自然名称 keyset、asset source fingerprint、thumbnail/cache deletion、搜索规范键/FTS 与 global revision 具备真实升级与约束测试；发布版本升级仍待 Release Gate |
| 测试 | 测试层次、合成 fixture、风险用例和发布门槛 | 原生双架构 Go/race、Web 契约、媒体、mount、runtime/recovery 与 Stage 0 SBOM CI 已通过；候选镜像/Compose 同入口已在本机原生 arm64 与指定原生 amd64 服务器通过。S5-005 的真实候选 100k/10k、100k 全量媒体、cache 水位和三引擎 100k FPS/RSS 预算已通过；S5-006A 的 Firefox/WebKit 稳定状态与 Linux 视觉基线、Safari 26.5.2 真机链、Chrome normal/forced-colors、Chrome 原生 200% 及 Firefox 153.0.1 核心链/原生 200%/400% 已通过；S5-007G 的修复来源 GLib 将本机 arm64 候选扫描降至 `0 Critical / 0 High`。物理读屏/触控/移动设备、Safari 缩放、最终原生双架构供应链复扫及正式安全/合规签署仍阻断 |
| 部署 | 单容器 Dockerfile/Compose、root bind-data 兼容、健康检查、备份恢复和升级流程 | 历史非 root 候选的只读根/媒体、health、媒体工具、代理、恢复和容量证据保留；ADR-0012 已把当前镜像改为 root，以支持 Docker 自动创建的 `./data`，因此当前身份、Compose 和双架构镜像 smoke 必须重跑。最终不可变 digest、供应链和发布签署未完成 |
| 安全 | 路径边界、媒体解析限制、同源策略、认证决策、依赖更新和日志脱敏 | FS-01 Stage 0 路径范围通过；S5-003 已覆盖可信代理 transport；候选 Compose 已验证回环发布端口、只读根、最小权限和代理失败关闭。发布 volume/unmount 仍由后续 Stage 5 Gate 阻断 |

## 环境与工具链状态

已落地的实验基线：

- Go 版本已写入 `go.mod`，`.go-version` 为 1.26.5；本地工具和候选镜像读取该文件。
  Stage 5 候选镜像固定到 Go 1.26.5 trixie digest。
- SQLite 使用 `modernc.org/sqlite`，迁移使用 Goose；初始迁移通过 `go:embed` 运行。
- sqlc 固定为 `v1.31.1`；配置、媒体库 SQL 源和提交的生成包归属
  `internal/store/sqlite`，`make generate-check` 在临时目录重生成并比较。
- `Makefile` 当前提供 `fmt`、`fmt-check`、`arch-check`、`release-docs-check`、
  `release-readiness-check`、失败关闭的 `release-ready`、`contract-check`、`generate`、
  `generate-check`、`web-check`、`openapi-lint`、`compatibility-check`、`lint`、`test`、
  `test-race`、`test-integration`、真实应用容器 `test-e2e` 和显式 `spike-capacity`；
  合并与发布前在本地运行适用入口。
- `.node-version`、`packageManager` 和 lockfile 固定 Node 24.22.1 LTS/npm 11.19.1，CI 与
  Docker 构建阶段同步，官方 bookworm-slim 镜像固定多架构 digest。
  [工具链维护记录](changes/FIX-2026-10-23-node24-lts.md) 分开保留 main 与 aifeature 的历史证据：
  main 当时的 strict TypeScript、生成检查和 high-severity audit 通过，并完成原生双架构发布；
  aifeature 本机/Docker 均 67/414 测试通过，types/生成/构建通过，但当时 high/critical 依赖审计失败。
  两者均不自动关闭合并版本的综合检查、安全或发行 Gate；旧 Node 22 证据也不冒充 Node 24 执行。

仍待阶段 1 固定或补齐：

- 系统依赖固定 libvips、FFmpeg/ffprobe 版本及构建选项，并记录 amd64/arm64 差异。
- Docker Buildx 构建发布镜像；SQLite CLI 可用于诊断，但应用不得依赖宿主机已安装 SQLite。
- 为产品 UI 增加浏览器 E2E；现有 `test-e2e` 只覆盖真实后端进程的测试专用容器，
  `generate-check` 已覆盖 sqlc 与 OpenAPI TypeScript 产物。
- 前端在首个业务 feature 前补齐 React/Vite、import/token lint、组件测试、axe、Storybook
  构建和聚焦视觉回归门禁。

当前可声称的证据只限各报告明确写出的环境与子范围，详见
[FS-01](spikes/fs-01-path-boundary.md)、[FS-02](spikes/fs-02-sqlite-generation.md)、
[FS-03](spikes/fs-03-media-matrix.md)、[FS-04](spikes/fs-04-capacity-baseline.md) 与
[FS-05](spikes/fs-05-runtime-recovery.md)。
权威 OpenAPI、生成 TypeScript client 基础与真实 HTTP test harness 是契约/测试证据，不是
生产 API 实现；历史真实 PR 基线兼容检查和原生双架构结果已保留，前端产品、浏览器和发布容器
检查仍未执行，不能用已有测试代替“完整测试”。

## 开发启动实施顺序

1. **记录已确认基线**：RQ-001～RQ-014 已全部采用 A，并同步 PRD、UX、API、数据、安全与交付文档。
2. **运行可行性 spike**：Stage 0 所需 FS-01～05 范围已经完成；完整产品/发布缺口已写入
   后续 Gate，见[可行性研究](feasibility-study.md)。
3. **Stage 1 先建立后端运行骨架**：创建 Go 入口与 `internal/app`、配置、生命周期、健康检查、
   认证基础边界和统一构建命令；不预建空业务包。
4. **契约先行**：首个 SQLite migration、`api/openapi.yaml`、TypeScript 类型/client、
   sqlc 查询/生成包、摘要锁、兼容性和确定性漂移检查已建立；相关改动须在本地实际执行
   同一生成入口，再实现首个业务 handler。
5. **建立前端基础而非业务替身**：只为已批准 S0/S1 切片按时间盒创建最小 React/Vite 应用壳、路由、token、共享原语和组件工作台；原型不进入生产 import graph，业务 feature 等待对应后端契约与集成证据。
6. **完成安全纵向切片**：只实现“列出允许目录 → 创建媒体库 → 安全扫描 → 索引一页媒体 → 返回缩略图状态”；先通过 Backend Ready，再实现对应 UI，全程使用合成 fixture。
7. **补齐故障路径**：验证重叠库、离线挂载、中断扫描、路径逃逸、损坏媒体和进程重启。
8. **容器化验收**：root 单容器运行，仅挂载 `/library:ro` 与 `/app/data`，以 Docker 自动创建的数据 bind 演练健康检查、备份和恢复。

在纵向切片通过前，不并行铺开搜索、分享、EXIF、watcher 或其他未来功能。

## Definition of Ready

一个实施项只有满足以下条件才可进入开发：

- 对应需求有稳定编号、用户价值、范围和可观察验收标准。
- 已在[需求—架构追踪](architecture/traceability.md)中明确目标版本、roadmap 阶段、capability owner、契约、风险和证据。
- 所有产品/安全未决项已确认，或明确标为不属于本项并有负责人。
- API、数据迁移、UI 状态和错误语义的影响已识别；需要时契约先更新。
- 依赖项、外部工具、目标平台和所需 fixture 可获得且许可证清晰。
- 正常、边界、失败和恢复测试已列出，性能敏感项有已确认的测量环境与目标。
- 不违反 `AGENTS.md`、安全模型或已接受 ADR；如违反，替代 ADR 已先获接受。
- 没有为已有规则、状态机、API adapter、Query/URL/error owner 或共享 UI 语义创建第二套实现。

缺少其中任一项时，工作保持“提案/待细化”，不得用实现细节替代产品决策。

## Definition of Done

一个变更只有满足适用条件才算完成：

- 实现与已确认验收标准一致，且不修改原始媒体、不越过路径边界。
- 数据库变化有只追加迁移，API 变化同步 OpenAPI，生成代码由源重新生成且无漂移。
- 单元、集成和必要的端到端/容器测试通过；执行过的命令与未执行原因准确报告。
- 适用的 `make arch-check`、契约/生成漂移、依赖方向和组件/token 门禁通过。
- 路径、扫描、并发、媒体解析、错误脱敏和恢复等风险用例得到覆盖。
- 前端包含加载、空、错误、离线和键盘操作状态，且大列表继续使用分页与虚拟化。
- 新共享组件进入唯一设计系统所有者和组件工作台；已有语义通过受控 variant 扩展而不是复制。
- 用户可见行为、部署、安全、数据或架构发生变化时，相应 README/docs/ADR 已同步。
- 没有提交数据库、缓存、日志、Vite 产物或真实用户媒体；依赖和 fixture 许可证可追溯。
- 发布相关变更在目标架构容器中通过启动、健康、升级、备份与恢复门槛。

## 阶段 0 出场条件

- P0 需求已经确认；MVP 范围、格式、认证、容量和验收口径没有架构级歧义。
- [可行性研究](feasibility-study.md)中的 Stage 0 必做 spike 有可复现实验和结论；完整产品/
  发布缺口已有 Owner、Fallback 和后续阻断 Gate。此项已满足。
- [风险登记册](risk-register.md)中的阶段 0 阻断风险已有 owner、验证计划和可接受 fallback。
- OpenAPI 与迁移输入已建立；UI 原型、生成流程和完整 fixture 的阶段 1 输入仍需补齐，不能通过正式实现猜需求。

## 首个安全纵向切片出场条件

- 最小脚手架可在干净环境通过统一检查命令。
- 首个安全纵向切片和关键故障用例可自动复现。
- [风险登记册](risk-register.md)中的发布阻断风险已有验证结果、有效缓解或明确 No-Go 决策。
- 后续功能可以依据稳定 API、迁移、测试和容器基线逐项交付，而不需要重建核心边界。

## CR-2026-053：自动AI部署 Gate

用户2026-10-27授权，ADR-0039及固定profile/进度合同已记录；配置、状态后端测试通过后接入既有服务设置UI。开发实现不等于发行Ready；两架构同digest镜像、空缓存真实下载及四项推理、安装恢复证据按变更记录追踪，未执行项保持未完成。
