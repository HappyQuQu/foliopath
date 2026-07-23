# FolioPath 开发就绪评审

## 当前结论

项目目前 **尚未达到功能开发就绪**。阶段 0 已完成需求确认；FS-01 的 Linux/arm64
`openat2` mount 边界和 HTTP test harness 子范围、FS-02 当前正确性范围已通过，FS-03/04
取得局部证据，但 FS-01/03/04 仍为 Conditional。只可继续剩余 spike、契约/生成护栏与不扩大
信任边界的实验脚手架；产品功能开发仍未获准。

当前仓库已有 `go.mod`/`.go-version`、Go 路径/媒体库/scanner/SQLite 实验代码、首个嵌入式
Goose migration、权威 `api/openapi.yaml`、确定性 TypeScript 类型生成、唯一 Web API client
边界、OpenAPI 摘要锁与语义兼容检查、契约/HTTP 边界/容量 harness、固定 Node/npm 工具链和
双架构 CI 工作流。仍无 `cmd/foliopath` 可启动进程、`internal/app` 组装、生产 HTTP handler、
React 产品应用、Dockerfile、浏览器 E2E 或可发布镜像；CI 工作流也尚未实际运行。现有代码是
spike 与契约工程证据，不是已经可运行的产品能力。

用户已于 2026-07-23 确认[需求确认清单](requirements-checklist.md)中的 RQ-001～RQ-014
全部采用 A。产品冻结门槛已经满足，[系统架构档案](architecture/README.md)也已形成目标
视图、所有权、交付治理和 fitness function 基线；权威 OpenAPI、强制离线解析/结构/引用/
ECMAScript pattern 验证、选择性跨源关键不变量检查（`queued`、`animated`、可空
`startedAt`）与外部 lint 已经存在；这不代表完整 ScanRun 领域实现。项目仍因 FS-01 的生产
handler/认证错误边界/发布挂载/Linux amd64 证据、
FS-03/04 剩余门槛、FS-05、首次原生双架构 CI 证据、完整脚手架、容器与发布级验证缺失而
未达到功能开发就绪。架构约束变更仍必须遵循 ADR 流程。

## 开发启动门槛

| 领域 | 开始实现前所需产物 | 当前判断 |
| --- | --- | --- |
| 需求冻结 | MVP 范围、用户流程、验收标准、格式矩阵、日期语义、目标规模、认证/网络边界和明确非目标 | 已就绪；RQ-001～RQ-014 全部确认 A |
| 架构 | 运行拓扑、包边界、路径模型、扫描一致性和数据模型 | 基线已形成，后端部分边界已有实验代码；应用组装与运行拓扑未验证 |
| API | `/api/v1` 资源、统一错误、游标、Range 与扫描任务语义；`api/openapi.yaml` 为唯一结构契约 | 权威契约、完整 Go 解析/结构/引用/pattern/语义测试、确定性 TypeScript 类型、唯一 client、摘要锁和语义兼容入口已就绪；PR 基线比较工作流尚未运行，生产 handler 未就绪 |
| UI/UX | 创建媒体库、扫描状态、目录浏览、递归浏览、查看器和异常恢复的可评审流程；前端分层、共享组件和响应式/无障碍要求 | 产品行为与目标前端架构已确认；代码 token、组件工作台、尺寸和移动抽屉细节待原型与脚手架验证 |
| 数据 | 首个 schema、迁移工具、外键/索引、generation 与任务恢复测试方案 | 首个 schema/Goose/WAL/generation 已有真实文件数据库测试；备份恢复、升级、磁盘满和容量演练缺失 |
| 测试 | 测试层次、合成 fixture、风险用例、CI 命令和发布门槛 | Go 单元/契约/集成/race、Web 契约生成/typecheck/audit、真实 HTTP harness、FS-03 fixture 和显式容量档已可运行；CI 已定义但未执行，尚无生产 HTTP、前端/浏览器 E2E 或完整发布容器验证 |
| 部署 | 单容器 Dockerfile/Compose、非 root 权限、健康检查、备份恢复和升级流程 | 尚无镜像或演练证据 |
| 安全 | 路径边界、媒体解析限制、同源策略、认证决策、依赖更新和日志脱敏 | FS-01 Darwin/Linux arm64、openat2 mount 与 HTTP harness 子范围已验证但仍为 Conditional；生产 handler、认证/错误 envelope、只读发布 volume、运行期 unmount、Linux/amd64、媒体解析和认证控制未完成 |

## 环境与工具链状态

已落地的实验基线：

- Go 版本已写入 `go.mod`，`.go-version` 为 1.26.4；CI 直接读取该文件，但尚需首次远端执行证明。
- SQLite 使用 `modernc.org/sqlite`，迁移使用 Goose；初始迁移通过 `go:embed` 运行。
- `Makefile` 当前提供 `fmt`、`fmt-check`、`arch-check`、`contract-check`、`generate`、
  `generate-check`、`web-check`、`openapi-lint`、`compatibility-check`、`lint`、`test`、
  `test-race`、`test-integration` 和显式 `spike-capacity`；CI 已复用这些入口。
- `.node-version`、`packageManager` 和 lockfile 固定 Node 22.22.2/npm 10.9.7；strict
  TypeScript、生成文件漂移检查与 high-severity npm audit 已在本地通过。

仍待阶段 1 固定或补齐：

- `sqlc` 已由 ADR-0001 接受；其配置、SQL 源查询、生成输出位置和确定性 `generate-check` 尚未建立。当前只有手写 migration 与 spike store。
- 系统依赖固定 libvips、FFmpeg/ffprobe 版本及构建选项，并记录 amd64/arm64 差异。
- Docker Buildx 构建发布镜像；SQLite CLI 可用于诊断，但应用不得依赖宿主机已安装 SQLite。
- 补上与 `AGENTS.md` 对齐的 `test-e2e`；`generate-check` 当前只覆盖 OpenAPI TypeScript
  产物，后续还要纳入 `sqlc`。
- 前端在首个业务 feature 前补齐 React/Vite、import/token lint、组件测试、axe、Storybook
  构建和聚焦视觉回归门禁。

当前可声称的证据只限各报告明确写出的环境与子范围，详见
[FS-01](spikes/fs-01-path-boundary.md)、[FS-02](spikes/fs-02-sqlite-generation.md)、
[FS-03](spikes/fs-03-media-matrix.md) 与 [FS-04](spikes/fs-04-capacity-baseline.md)。
权威 OpenAPI、生成 TypeScript client 基础与真实 HTTP test harness 是契约/测试证据，不是
生产 API 实现；PR 基线兼容检查和原生双架构 CI 尚未实际运行，前端产品、浏览器和发布容器
检查也未执行，不能用已有测试代替“完整测试”。

## 开发启动实施顺序

1. **记录已确认基线**：RQ-001～RQ-014 已全部采用 A，并同步 PRD、UX、API、数据、安全与交付文档。
2. **运行可行性 spike**：FS-01 Linux/openat2 与 HTTP harness 子范围、FS-02 当前 scope
   已通过；FS-03/04 已取得局部证据但保持 Conditional；继续完成 FS-01、FS-03/04
   剩余门槛、跨架构镜像与恢复，见
   [可行性研究](feasibility-study.md)。
3. **进入路线图阶段 1 后先建立后端运行骨架**：创建 Go 入口与 `internal/app`、配置、生命周期、健康检查、认证基础边界、统一构建命令和基础 CI；不预建空业务包。
4. **契约先行**：首个 SQLite migration、`api/openapi.yaml`、TypeScript 类型/client、
   摘要锁、兼容性和确定性漂移检查已建立；仍需建立 `sqlc` 生成方案并让 PR 基线比较在 CI
   实际通过，再实现首个 handler。
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
- [可行性研究](feasibility-study.md)中的必做 spike 有可复现实验和结论，Conditional Go 已转为允许进入 MVP 实施，或范围已明确缩减。当前 FS-01、FS-03、FS-04 仍为 Conditional，FS-05 未完成，因此本项未满足。
- [风险登记册](risk-register.md)中的阶段 0 阻断风险已有 owner、验证计划和可接受 fallback。
- OpenAPI 与迁移输入已建立；UI 原型、生成流程和完整 fixture 的阶段 1 输入仍需补齐，不能通过正式实现猜需求。

## 首个安全纵向切片出场条件

- 最小脚手架可在干净环境通过统一检查命令。
- 首个安全纵向切片和关键故障用例可自动复现。
- [风险登记册](risk-register.md)中的发布阻断风险已有验证结果、有效缓解或明确 No-Go 决策。
- 后续功能可以依据稳定 API、迁移、测试和容器基线逐项交付，而不需要重建核心边界。
