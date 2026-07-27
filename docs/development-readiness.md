# FolioPath 开发就绪评审

## 当前结论

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
FTS/keyset、三种 scope、类型/mtime、离线、认证 HTTP 与真实 composition。当前进入
`S4-003` 容量/并发和 Backend Ready Gate；搜索前端仍不可接入。
这些结论不把前端、Stage 4～5 或发布标记为完成，也不授权共享预览或非回环监听。

当前仓库已有 `go.mod`/`.go-version`、Go 路径/媒体库/scanner/SQLite 实验代码、首个嵌入式
Goose migration、权威 `api/openapi.yaml`、确定性 TypeScript 类型生成、唯一 Web API client
边界、OpenAPI 摘要锁与语义兼容检查、契约/HTTP 边界/容量 harness、固定 Node/npm 工具链和
双架构 CI 工作流。`cmd/foliopath` 的最小进程入口、版本命令和退出码已经建立；
`internal/app` 已拥有唯一组合点、进程根取消、顺序启动、失败回滚、运行故障传播、反向关闭和
有界停机；启动配置已固定 `/library`、`/app/data`、单监听地址和认证前回环限制。正式应用已
接入 SQLite WAL、嵌入 migration、空数据目录准备、重复启动和迁移失败关闭，并由真实
composition root 集成测试和测试专用非 root 容器 smoke 覆盖。`internal/auth` 已实现
Argon2id 密码存储、Unicode 管理员身份规范化、单管理员原子初始化，以及高熵摘要化会话、
7 天绝对期限、重启恢复、退出撤销和 Cookie 策略，并经 SQLite adapter 接入 composition
root。五个认证 HTTP handler、同源 Origin、session-bound CSRF、业务 API 默认拒绝、
防缓存、受限 JSON 和按直连 peer 的有界限流已经接入真实 composition root，并通过认证
Backend Ready Gate；安全目录选择、媒体库生命周期、manual scan admission 与异步移除
handler 也已接入真实 composition root 并通过媒体库 Backend Ready Gate。生产扫描 worker
已消费 creation/startup/scheduled scan，扫描观察、取消和设置 HTTP 已接线并通过 Backend Ready；
当前还没有可信代理配置、
完整认证产品 UI、正式 Dockerfile、浏览器 E2E 或可发布镜像；
HTTP 运行边界已有服务端 request ID、统一安全 404/500、JSON 日志、panic 隔离、在途请求排空、
liveness/readiness 和受保护系统状态；数据库及 migration 成功后 readiness 才进入 ready，
系统状态已使用真实会话保护。隔离 FS-05 Dockerfile 只用于 Stage 0 probe。
原生 amd64/arm64 PR CI、runtime/recovery 与 SBOM/license jobs 已通过。
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
| 需求冻结 | MVP 范围、用户流程、验收标准、格式矩阵、日期语义、目标规模、认证/网络边界和明确非目标 | 已就绪；RQ-001～RQ-014 全部确认 A |
| 架构 | 运行拓扑、包边界、路径模型、扫描一致性和数据模型 | 基线已形成；启动配置、应用组合、HTTP listener/中间件、健康状态、数据库/migration 与生命周期已有单元测试 |
| API | `/api/v1` 资源、统一错误、游标、Range 与扫描任务语义；`api/openapi.yaml` 为唯一结构契约 | 权威契约、完整 Go 解析/结构/引用/pattern/语义测试、确定性 TypeScript 类型、唯一 client、摘要锁和真实 PR 基线语义比较已通过；认证、媒体库生命周期、安全目录选择与 manual scan admission handler 已实现，其余业务 handler 尚未实现 |
| UI/UX | 创建媒体库、扫描状态、目录浏览、递归浏览、查看器和异常恢复的可评审流程；前端分层、共享组件和响应式/无障碍要求 | 产品行为与目标前端架构已确认；代码 token、组件工作台、尺寸和移动抽屉细节待原型与脚手架验证 |
| 数据 | 首个 schema、迁移工具、外键/索引、generation 与任务恢复测试方案 | Goose migration 1～10 已执行；媒体库 revision/removal/idempotency、扫描与媒体 durable contract、typed settings、目录/资产自然名称 keyset、asset source fingerprint、thumbnail/cache deletion、搜索规范键/FTS 与 global revision 具备真实升级与约束测试；发布版本升级仍待 Release Gate |
| 测试 | 测试层次、合成 fixture、风险用例、CI 命令和发布门槛 | 原生双架构 Go/race、Web 契约、媒体、mount、runtime/recovery 与 SBOM CI 已通过；真实后端应用的组合/容器 smoke 已接线，尚无前端/浏览器产品 E2E 或最终发布容器验证 |
| 部署 | 单容器 Dockerfile/Compose、非 root 权限、健康检查、备份恢复和升级流程 | FS-05 probe 与真实应用测试镜像已验证目标运行模式；正式发布镜像、真实版本升级和发布签署未完成 |
| 安全 | 路径边界、媒体解析限制、同源策略、认证决策、依赖更新和日志脱敏 | FS-01 Stage 0 路径可行性范围通过；认证 Backend Gate 已覆盖密码、原子 setup、安全会话、CSRF/default-deny、直连 peer 限流、错误脱敏和依赖 audit。可信代理、非回环暴露和发布 volume/unmount 仍由 Stage 5 Gate 阻断 |

## 环境与工具链状态

已落地的实验基线：

- Go 版本已写入 `go.mod`，`.go-version` 为 1.26.4；CI 直接读取该文件，但尚需首次远端执行证明。
- SQLite 使用 `modernc.org/sqlite`，迁移使用 Goose；初始迁移通过 `go:embed` 运行。
- sqlc 固定为 `v1.31.1`；配置、媒体库 SQL 源和提交的生成包归属
  `internal/store/sqlite`，`make generate-check` 在临时目录重生成并比较。
- `Makefile` 当前提供 `fmt`、`fmt-check`、`arch-check`、`contract-check`、`generate`、
  `generate-check`、`web-check`、`openapi-lint`、`compatibility-check`、`lint`、`test`、
  `test-race`、`test-integration`、真实应用容器 `test-e2e` 和显式 `spike-capacity`；CI 已
  复用这些入口。
- `.node-version`、`packageManager` 和 lockfile 固定 Node 22.22.2/npm 10.9.7；strict
  TypeScript、生成文件漂移检查与 high-severity npm audit 已在本地通过。

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
生产 API 实现；真实 PR 基线兼容检查和原生双架构 CI 已通过，前端产品、浏览器和发布容器
检查仍未执行，不能用已有测试代替“完整测试”。

## 开发启动实施顺序

1. **记录已确认基线**：RQ-001～RQ-014 已全部采用 A，并同步 PRD、UX、API、数据、安全与交付文档。
2. **运行可行性 spike**：Stage 0 所需 FS-01～05 范围已经完成；完整产品/发布缺口已写入
   后续 Gate，见[可行性研究](feasibility-study.md)。
3. **Stage 1 先建立后端运行骨架**：创建 Go 入口与 `internal/app`、配置、生命周期、健康检查、
   认证基础边界、统一构建命令和基础 CI；不预建空业务包。
4. **契约先行**：首个 SQLite migration、`api/openapi.yaml`、TypeScript 类型/client、
   sqlc 查询/生成包、摘要锁、兼容性和确定性漂移检查已建立；相关 PR 仍须让 CI 实际执行
   同一生成入口，再实现首个业务 handler。
5. **建立前端基础而非业务替身**：只为已批准 S0/S1 切片按时间盒创建最小 React/Vite 应用壳、路由、token、共享原语和组件工作台；原型不进入生产 import graph，业务 feature 等待对应后端契约与集成证据。
6. **完成安全纵向切片**：只实现“列出允许目录 → 创建媒体库 → 安全扫描 → 索引一页媒体 → 返回缩略图状态”；先通过 Backend Ready，再实现对应 UI，全程使用合成 fixture。
7. **补齐故障路径**：验证重叠库、离线挂载、中断扫描、路径逃逸、损坏媒体和进程重启。
8. **容器化验收**：非 root 单容器运行，仅挂载 `/library:ro` 与 `/app/data`，演练健康检查、备份和恢复。

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
