# FolioPath 文档索引

这里保存 FolioPath 在编码前与开发过程中的产品、交互和工程约束。用户已于 2026-07-23
确认 `RQ-001`～`RQ-014` 全部采用 A。Stage 0～4 已通过各自 Gate，单管理员认证、媒体库、
可靠扫描、浏览/缩略图、搜索、非模态预览与完整查看器均已接入真实产品纵向链。Stage 5
候选镜像、Compose、可信代理、恢复/失败关闭、原生双架构运行/升级和完整容量 Gate 已通过；
真实 Firefox/物理辅助功能、供应链处置、最终不可变 digest 与 Release Candidate Gate
仍未完成，当前候选不是稳定发布。

## 从哪里开始

| 读者 / 任务 | 首先阅读 | 继续阅读 |
| --- | --- | --- |
| 了解项目 | [项目 README](../README.md) | [产品需求](product-requirements.md)、[可行性研究](feasibility-study.md)、[路线图](roadmap.md) |
| 确认产品范围 | [MVP scope manifest](releases/MVP-2026-07-23-scope.md) | [需求确认清单](requirements-checklist.md)、[产品需求](product-requirements.md)、[用户流程](user-flows.md) |
| 设计界面 | [界面设计规范](ui-design.md) | [品牌标识规范](branding.md)、[前端架构](architecture/frontend.md)、[用户流程](user-flows.md)、[API 设计](api-design.md) |
| 开发后端或扫描器 | [后端开发清单](backend-task-list.md) | [模块边界](architecture/modules.md)、[目录与依赖约束](project-structure.md)、[数据模型](data-model.md)、[API 设计](api-design.md)、[安全模型](security.md) |
| 开发前端 | [前端开发清单](frontend-task-list.md) | [前端架构](architecture/frontend.md)、[界面设计规范](ui-design.md)、[用户流程](user-flows.md)、[API 设计](api-design.md) |
| 部署和运维 | [部署](deployment.md) | [安全模型](security.md)、[测试策略](testing-strategy.md) |
| 判断能否开工 | [开发就绪评审](development-readiness.md) | [S4 搜索 Backend Ready](gates/MVP-2026-07-23/s4-search-backend-ready.md)、[S4-002 搜索实现](gates/MVP-2026-07-23/s4-search-keyset.md)、[S4 搜索 Contract Ready](gates/MVP-2026-07-23/s4-search-contract-ready.md)、[S3-007 浏览/缩略图 Backend Ready](gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)、[S3-006 敌意媒体与资源安全](gates/MVP-2026-07-23/s3-media-resource-safety.md)、[S3-005 媒体任务与缓存保护](gates/MVP-2026-07-23/s3-media-jobs-cache.md)、[S3-004 媒体处理](gates/MVP-2026-07-23/s3-media-processing.md)、[S3-003 目录树与详情](gates/MVP-2026-07-23/s3-directory-tree.md)、[S3-002 Catalog keyset](gates/MVP-2026-07-23/s3-catalog-keyset.md)、[S3 浏览 Contract Ready](gates/MVP-2026-07-23/s3-browse-contract-ready.md)、[扫描 Backend Ready](gates/MVP-2026-07-23/s2-scan-backend-ready.md)、[媒体库 Backend Ready](gates/MVP-2026-07-23/s2-library-backend-ready.md)、[认证 Backend Ready](gates/MVP-2026-07-23/s1-auth-backend-ready.md)、[Stage 0 Gate](gates/MVP-2026-07-23/stage-0-current.md)、[可行性研究](feasibility-study.md)、[风险登记](risk-register.md) |
| 查看或更新任务 | [开发任务清单](task-list.md) | [路线图](roadmap.md)、[当前 Stage 0 Gate](gates/MVP-2026-07-23/stage-0-current.md)、[交付治理](architecture/delivery-governance.md) |
| 修改架构或范围 | [交付与架构治理](architecture/delivery-governance.md) | [系统架构档案](architecture/README.md)、[Agent 约束](../AGENTS.md)、[ADR](adr/README.md) |

## 产品与体验

- [产品需求](product-requirements.md)：愿景、用户、范围、需求编号和验收标准。
- [MVP scope manifest revision 2](releases/MVP-2026-07-23-scope-r2.md)：冻结版本、revision、精确需求/非目标/验收 ID；已合入 revision 不原地改写。
- [需求确认清单](requirements-checklist.md)：全部 14 项 A 方案的确认记录与未采用备选。
- [用户流程](user-flows.md)：创建媒体库、扫描、浏览、搜索、查看和异常恢复流程。
- [界面设计规范](ui-design.md)：信息架构、页面、组件、响应式、状态、可访问性和动效边界。
- [品牌标识规范](branding.md)：标识概念、唯一资产、尺寸、主题、可访问性和生产接入规则。
- [Apple 风格静态 UI 原型](../prototypes/apple-redesign/index.html)：登录、欢迎、浏览、搜索、查看器与设置页面的浅色/深色和响应式视觉验收基线；不进入生产 import graph。
- [CR-2026-001 非模态媒体预览](changes/CR-2026-001-non-modal-media-preview.md)：默认预览、固定与双击切换、完整查看器分层的确认记录。
- [CR-2026-002 经认证的局域网 HTTP](changes/CR-2026-002-authenticated-lan-http.md)：直接
  LAN 访问与外部 TLS/反向代理职责边界。
- [CR-2026-003 统一品牌标识](changes/CR-2026-003-brand-identity.md)：极简目录树标识、
  `BrandMark` 唯一所有权与生产入口的确认记录。
- [FIX-2026-07-29 管理员密码最低长度](changes/FIX-2026-07-29-admin-password-minimum.md)：
  首次 setup 调整为 8～128 个 Unicode 字符，保留 Argon2id、限流和会话安全边界。
- [FIX-2026-07-29 redesign 应用壳恢复](changes/FIX-2026-07-29-redesign-shell-restoration.md)：
  恢复 Apple redesign 的固定侧栏、底部导航、面包屑顶栏和精简浏览工具栏，同时保留刷新与
  媒体类型筛选能力。
- [FTR-VID-001 视频故事板悬停预览](features/video-storyboard-preview.md)：已确认的
  `Post-MVP/1` feature 规格；包含产品、交互、采样、架构、已冻结 API/data 合同、风险和验收。
- [FTR-VID-001 开发任务清单](features/video-storyboard-preview-task-list.md)：严格按
  Architecture/Contract → 后端 → Backend Ready → 前端 → Integrated Done 排列的执行清单。
- [VSP-002 视频故事板 spike](spikes/vsp-002-video-storyboard.md)：2 秒～2 小时合成视频的
  fast seek、顺序解码、sprite、资源与运行时能力证据。
- [POST-MVP-1 scope revision 1](releases/POST-MVP-1-scope.md)与
  [VSP-S0 Architecture Ready](gates/POST-MVP-1/vsp-s0-architecture-ready.md)：
  feature 冻结范围和允许进入 Contract Ready 的 Gate。
- [VSP-S2 Backend Evidence Ready](gates/POST-MVP-1/vsp-s2-backend-evidence-ready.md)：
  双架构生产 FFmpeg、真实认证纵向链、故障恢复和 100k/10k 后端容量通过，前端已获准。
- [VSP-S3 Consumer/UI Ready](gates/POST-MVP-1/vsp-s3-consumer-ui-ready.md)：
  生成 client、共享 hover/sprite、输入模式、组件工作台与三引擎前端容量通过，纵向集成已获准。
- [VSP-301 真实产品纵向链](gates/POST-MVP-1/vsp-301-product-vertical.md)：
  生产镜像真实登录、扫描、浏览/搜索 hover、预览焦点恢复与 cache repair 贯通。
- [VSP-302 目标平台与资源复验](gates/POST-MVP-1/vsp-302-target-platform.md)：
  原生 amd64/arm64 结构化证据与成对校验入口已建立，等待同一提交的远端矩阵签署。
- [VSP-303 发布文档与追踪收敛](gates/POST-MVP-1/vsp-303-documentation-convergence.md)：
  文档矩阵和签署条件已准备；依赖 VSP-302，当前不能关闭。
- [VSP-S4 Integrated Slice Done](gates/POST-MVP-1/vsp-s4-integrated-slice-done.md)：
  `VSP-AC-001～008` 聚合骨架已准备；AC-008 与前置 Gate 未完成，当前 No-Go。
- [POST-MVP-1 readiness 快照](releases/POST-MVP-1-readiness.json)：
  Gate、验收、R-018 与任务状态的失败关闭机器事实；`make storyboard-ready` 当前必须失败。
- [POST-MVP-1 发布说明草案](releases/POST-MVP-1-release-notes.md)：
  记录候选行为、升级/部署影响、已完成证据和发布前阻断；尚未发布。
- [CR-2026-004 视频故事板悬停预览](changes/CR-2026-004-video-storyboard-preview.md)：
  后续版本归属、范围影响和 Conditional Go 决定。
- [FTR-SCN-001 媒体库自动发现](features/automatic-library-discovery.md)：`Post-MVP/2`
  已冻结 feature；以 Linux 文件事件触发安全定向校准，并保留完整 generation 扫描作为
  正确性基线。WCH-S0 当前仅允许 spike 与 ADR 评审。
- [CR-2026-005 媒体库自动发现](changes/CR-2026-005-automatic-library-discovery.md)：
  记录自动近实时发现的用户确认、C3 架构影响、风险和 Gate 边界。
- [路线图](roadmap.md)：不承诺日期的实施阶段、依赖与阶段出口条件。
- [开发任务清单](task-list.md)：总进度、前后端交接点和共同发布任务。
- [后端开发清单](backend-task-list.md)：Go、API、SQLite、认证、扫描与媒体服务任务。
- [前端开发清单](frontend-task-list.md)：React、设计系统、页面、交互与浏览器测试任务。
- [术语表](glossary.md)：统一 `/library`、媒体库、目录、递归浏览、扫描和派生数据等词义。

## 可行性与开工准备

- [可行性研究](feasibility-study.md)：产品、技术、性能、媒体、SQLite、安全、运维、跨架构和许可证的条件 Go 结论。
- [FS-01 路径边界 spike](spikes/fs-01-path-boundary.md)：Darwin 与原生 Linux amd64/arm64 路径证据、
  Linux `openat2` 同/跨设备及 self-bind mount 拒绝和真实 HTTP harness；Stage 0 范围已通过，
  生产 handler/auth 与发布 volume/unmount 分别转入后续 Backend/Release Gate。
- [FS-02 SQLite 与扫描 generation spike](spikes/fs-02-sqlite-generation.md)：真实文件数据库、WAL、迁移、故障保留和原子清理的当前正确性证据。
- [FS-03 媒体矩阵 spike](spikes/fs-03-media-matrix.md)：合成格式矩阵、Darwin/arm64 与
  Linux/amd64 QEMU FFmpeg 探测、视频封面与损坏样本证据，以及 libvips/浏览器/原生双架构缺口。
- [FS-04 容量基线 spike](spikes/fs-04-capacity-baseline.md)：Linux/arm64 四核、4 GiB 目标
  数据档的扫描/索引结果、已修复瓶颈和仍待完整产品验证的边界。
- [FS-05 运行与恢复 spike](spikes/fs-05-runtime-recovery.md)：原生双架构镜像、非 root/
  只读边界、健康、退出、离线恢复、重复迁移和故障关闭证据。
- [WCH-001 Linux watcher spike](spikes/wch-001-linux-watcher.md)：`Post-MVP/2` 独立
  inotify 探针、双架构交叉编译和待补原生 Linux 事件/overflow/10k watches 证据。
- [供应链与许可证审查](supply-chain-review.md)：source/npm/image SPDX、FFmpeg codec/GPL
  组合与 Release Gate 未决项。
- [风险登记](risk-register.md)：概率、影响、触发信号、缓解、fallback、Owner 角色和发布阻断风险。
- [开发就绪评审](development-readiness.md)：开工门槛、阶段 0 顺序、Definition of Ready 与 Definition of Done。
- [路线图](roadmap.md)：从需求/spike 到发布安全的阶段依赖和出口条件。

## 工程与交付

- [架构](architecture.md)：运行拓扑、技术栈、代码布局和核心流程。
- [系统架构档案](architecture/README.md)：系统级视图、模块所有权、范围/交付治理、需求追踪、前端架构与可执行门禁的入口。
- [系统上下文与运行视图](architecture/system-context.md)：Current/Target 状态、C4、部署/信任边界和关键时序。
- [模块、数据与运行时边界](architecture/modules.md)：capability、adapter、事务、任务、错误、配置与并发的唯一所有权。
- [交付与架构治理](architecture/delivery-governance.md)：MVP scope lock、变更分级、版本、后端优先切片与 Gate。
- [需求—架构追踪](architecture/traceability.md)：FR/NFR 到 capability、契约、数据、风险、测试和阶段的映射。
- [前端架构](architecture/frontend.md)：层次、设计系统、组件/模式复用、状态与前端质量门禁。
- [架构适配度检查](architecture/fitness-functions.md)：已执行与计划中的架构自动化检查。
- [Gate 记录](gates/README.md)：切片、阶段和版本的 Go/Conditional/No-Go 决策与证据。
- [版本范围清单](releases/README.md)：冻结 scope manifest 与后续 revision/版本关系。
- [目录与依赖约束](project-structure.md)：后端、前端、生成代码和测试文件的放置规则。
- [数据模型](data-model.md)：SQLite 领域对象、索引、扫描一致性和迁移语义。
- [权威 OpenAPI 契约](../api/openapi.yaml)：`/api/v1` 请求、响应、认证、错误、分页与 Range 的结构化事实来源。
- [API 设计与契约说明](api-design.md)：资源边界、已固定 wire 决策和仍待实现的内部参数。
- [安全模型](security.md)：路径、网络、媒体解析、容器和日志的信任边界。
- [部署](deployment.md)：Stage 5 候选单容器配置、权限、格式、备份、恢复、升级和已知限制。
- [测试策略](testing-strategy.md)：测试层次、风险覆盖、测试数据和发布门槛。
- [Agent 约束](../AGENTS.md)：代码边界、不可破坏的产品约束和修改规则。

## 架构决策记录

- [ADR-0001：采用 Go、React 与 SQLite 的模块化单体架构](adr/0001-go-react-sqlite.md)
- [ADR-0002：以单一允许根目录管理多个媒体库](adr/0002-library-path-model.md)
- [ADR-0003：使用扫描代次保证索引最终一致性](adr/0003-scan-consistency.md)
- [ADR-0004：MVP 媒体库根路径创建后不可变](adr/0004-library-root-immutable.md)
- [ADR-0005：稳定版内建单管理员认证](adr/0005-built-in-single-admin-auth.md)
- [ADR-0006：契约驱动、切片内后端优先交付](adr/0006-contract-driven-backend-first.md)
- [ADR-0007：单一共享前端设计系统](adr/0007-shared-frontend-system.md)
- [ADR-0008：统一应用组合根并分离纯路径策略](adr/0008-composition-root-and-path-policy.md)
- [ADR-0009：Linux `openat2` 与单一媒体根挂载](adr/0009-linux-openat2-single-media-root.md)
- [ADR-0010：经认证的局域网 HTTP 与可选外部 TLS](adr/0010-authenticated-lan-http.md)

新的架构决策使用连续编号和 [ADR 模板](adr/template.md)。已接受 ADR 不直接改写；方向改变时新增 ADR，并将旧记录标记为被替代。

## 状态与优先级

文档使用以下含义：

- **已接受 / 已确认**：当前实现必须遵守；改变架构约束需要 ADR，改变产品基线需要需求确认。
- **MVP 计划**：首个可用版本的目标，但在实现并验证前不能描述为已经可用。
- **提案**：不改变已确认产品范围的实现或视觉细节，仍需原型、spike 或评审收敛。
- **未来**：明确不属于 MVP，不应为其预先增加部署或架构复杂度。

发生冲突时不要静默选择其中一份文档。先检查已接受 ADR 和已确认需求，再同步修改所有受影响文档；不确定时把问题加入[需求确认清单](requirements-checklist.md)。

## 变更同步表

| 变更类型 | 至少同步更新 |
| --- | --- |
| 用户可见功能、格式或配置 | `README.md`、产品需求、相关用户流程 |
| 媒体库路径或删除语义 | 产品需求、架构、数据模型、安全模型，必要时新增 ADR |
| API 路径、字段或错误 | API 设计；开始编码后以 `api/openapi.yaml` 为准 |
| 页面、组件、响应式或动效 | 界面设计规范、相关用户流程 |
| 数据表、索引或迁移语义 | 数据模型，必要时新增 ADR |
| Docker、权限、备份或升级 | README、部署、安全模型 |
| 测试门槛或支持矩阵 | 测试策略、README |
| 可行性假设、spike 或风险状态 | 可行性研究、风险登记、开发就绪评审 |
