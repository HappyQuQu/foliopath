# FolioPath 需求—架构追踪

2026-09-22 / CR-2026-030 / FR-INT-009～014 / POST-MVP-5：web AI 与共享应用壳在既有公开合同内
重构人物日常入口和智能设置层级。媒体浏览左侧“快速访问”直接进入人物待整理视图，人物工作区
保留同一导航与媒体库上下文；匿名组复用正式
`previewAssetIds` 和缩略图端点形成预览拼图；设置页保留逐库设置 owner，只移除嵌套高级维护入口，
图片、视频和人物维护仍复用既有页面与任务合同。无新 API、持久化、部署单元或人脸判断策略；
延迟的质量、容量与发行门禁不变。

2026-09-17 / POST-MVP-5 P6 G01/G03/G04 / CR-2026-030 / NFR-INT-004、006、007：最终七项工程检查完成，当前两个 arm64 镜像生成 SPDX SBOM 并执行 High/Critical 扫描，模型 revision/hash、许可/notices、provenance 和延期门禁均按实际状态复核。没有改变业务 owner、部署单元、API 或持久化；最终 Release No-Go，不以核对完成替代漏洞处置、模型授权、双架构、质量容量或外部签署。[证据](../evidence/int-001/p6-final-verification-2026-09-17.md)。

2026-09-17 / POST-MVP-5 P5 F03/F04/G02 / CR-2026-030 / NFR-INT-004、006：deployment/test/docs 负责正式 Compose 配对安装升级回退、运行说明和双服务安全验证。新增仅测试控制器与离线网络覆盖，不增加部署单元或产品 API。既有 `TestFacePrivacyProjectionRemainsClosed` 扩展检查 remoteml 日志边界；SQLite secure_delete、鉴权和 Linux openat2 回归保持原 owner。构建及实际镜像证据分别记录，不将 P5 写作 P6 发行批准。

2026-09-17 / CR-2026-030 / POST-MVP-5 P5 F02：deployment/test 复用根 Compose smoke 增加模型缓存与身份失败/恢复矩阵；模型可用性和代次准入仍由 app/aimodel/semantic/face 原 owner 决定，无新 API、持久化或默认阈值。测试独立运行于本地 Docker，UI 既有不可用门禁由 AISettingsPage 专项回归覆盖。[实施记录](../changes/P5-2026-09-17-deployment-delivery.md)。

2026-09-17 / CR-2026-030 / POST-MVP-5 P5 F01：deployment/test 拥有根 `compose.ai.yaml` 交付配置及合成 smoke；沿用 ADR-0019/0020 两服务边界。`TestAIComposeDeploymentBoundaries` 纳入 `make arch-check` 检查部署单元、核心不依赖 AI 健康、AI 私网/只读缓存/无媒体数据卷等静态约束；Docker smoke 检查解析后及实际运行配置。无新 API 或持久化合同。[实施记录](../changes/P5-2026-09-17-deployment-delivery.md)。

2026-09-17 / FR-INT-002 / NFR-INT-002、003 / P4：imagevips 语义源读取修复取消后仍继续读取的问题，保留单次源缓冲和原像素/字节预算，增加块间取消与连续无进展保护。无模型输出、空间身份或 API 变化；回归先失败后修复验证。[记录](../changes/FIX-2026-09-17-semantic-read-cancellation.md)。

2026-09-17 P4 升级补验：FR-INT-002、005，Owner app/semantic/测试；两版实际镜像、合成旧库通过新代次零继承完成数、两库回填、人工状态保持、重启和升级前备份配对回退。共享短视频降级基线须保持，不计完整质量。新增 opt-in `p4-semantic-upgrade-probe.mjs`，未改变生产合同。[证据](../evidence/int-001/p4-semantic-large-jpeg-2026-09-17.md)。

2026-09-17 / CR-2026-030 / FR-INT-002、005 / NFR-INT-002、003：P4 大 JPEG 兼容由 imagevips 负责有界解码，semantic 拥有版本化输入合同，app 将其绑定空间身份。公共 API/部署/持久化边界不变；合成回归与固定样本结果分别记录，未验证不关闭 Gate。[变更记录](../changes/FIX-2026-09-17-semantic-large-jpeg.md)。

2026-09-16 大图内存修复：FR-INT-009 / NFR-INT-002、003，POST-MVP-5 P4，Owner imagevips/app/测试。第一次 native open 前执行既有 JPEG shrink 策略，关闭 operation cache；不改变 input-v1、像素限制、API 或部署边界。合成 race 回归、固定副本序列及容量复验分别记录，36 项语义超限不得算通过；完整 Gate 保持开放。[修复与证据](../evidence/int-001/p4-face-memory-2026-09-16.md)。

2026-09-16 早前小样本独立验证：CR-2026-030 / NFR-INT-002、003 / P4，Owner app/测试；采样复用 internal/files 的 Linux 只读 openat2 边界，临时匿名副本与原媒体隔离，无新生产合同或部署单元。探针结果判定独立检查覆盖率，不以 settled job 冒充成功。当时本地通过、远端 OOM；最新修复见上条，完整 Gate 仍不关闭。[早前证据](../evidence/int-001/p4-small-sample-2026-09-16.md)。

2026-09-16 P4 原生测试：CR-2026-030 / NFR-INT-002、003 / E01、E03、E04，Owner app/测试。新增显式授权的 server222 维护测试入口、固定资源配置和 run label 清理；不是新增产品部署单元，不改变媒体/数据库边界。原服务停服后隔离构建测试，退出恢复同镜像并验证健康；原生功能与容量子项不替代完整质量/跨架构/长期验收。[维护证据](../evidence/int-001/p4-server222-native-2026-09-16.md)。

2026-09-16 P4 启动：CR-2026-030 / NFR-INT-002、003 / POST-MVP-5 / E01，Owner catalog/SQLite/测试；修正容量探针临时目录 UID 假设。既有 catalog 计数作为仓储查询计划提示，SQLite 分别优化稀疏/宽匹配和计数连接；无新计数缓存、表、公开合同或媒体边界。三组专项 race 及固定镜像 100k/10k 基础目录容量通过；联合 AI 负载、独立质量数据/原生双架构仍未完成。[变更](../changes/FIX-2026-09-16-sparse-catalog-search-capacity.md)、[执行记录](../evidence/int-001/p4-quality-capacity-2026-09-16.md)。

2026-09-16 最终全项复查：CR-2026-030 / FR-INT-007、009、011、013、016 / POST-MVP-5 S2/S3，Owner face/SQLite/app/API/web AI。在既有 owner 内修复人物路由切换/移出后的焦点和卡片本地化，无新合同、持久化或部署边界；复用共享 Button/Input 和 token。237 前端单测、选定后端 race、最终本地双容器完整业务/故障/恢复/备份回滚及三浏览器、32 张独立视觉比较通过，生成/类型/架构检查通过。**P2 6/6、P3 9/9；D04 关闭，18/34**。P4～P6 保留。[最终证据](../evidence/int-001/p2-p3-final-recheck-2026-09-16.md) 优先于以下历史状态。

2026-09-16 最新修复：CR-2026-030 / FR-INT-011、013 / POST-MVP-5 S2/S3，Owner face/SQLite/API/web AI。AssignFaceReview 增加可选配对源人物/版本，既有事务原子转移 anchor、双方版本与撤销快照，无新表或部署边界；前端沿用共享审核与查询 owner。最终 Docker race/业务、234 前端单测、完整三浏览器在线/离线通过；架构及生成检查通过。A05/D03 重新关闭，D04 保留，当前 **17/34**。[最新证据](../evidence/int-001/face-reassignment-2026-09-16.md) 优先于早前复查/验收结论。

2026-09-16 P3 修复：CR-2026-030 / FR-INT-009、010、016 / NFR-INT-002、006，POST-MVP-5 S2/S3，Owner face/settings、SQLite、web 共享 InlineStatus。修复服务恢复后同代次 awaiting_model 阻止正式重试；仍由既有设置 owner 及事务可用性/revision 检查拥有转换，disabled 不变。修复浅色普通提示对比度；无新部署、表或端点，OpenAPI 仅澄清重试语义。新增 migration 40 拒绝有损降级、真实双容器故障/审核/浏览器证据。[P3 执行记录](../evidence/int-001/p3-acceptance-2026-09-16.md)。质量/容量/发行门禁不变。

同轮补修：app 的部署 bootstrap 标记不再匹配当前配置的同用途 service 模型不可用，保留代次和数据；验证错误配置不留下假可用状态，恢复原配置可恢复。共享 GlobalHeader 修复带 query 的目标二次拼接，统一替换 q 并保留调用方范围。均为已有生命周期/搜索合同内修复，不新增 owner 或产品范围。

2026-09-16 P2 完成：CR-2026-030 / FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2，Owner face/SQLite/app/API/web。正式非空分析、人工审核、按库可见代次、精确人工重绑、有界清理与浏览器闭环通过。合同仅修复既有 action discriminator 和缺漏 error enum；无新端点/表/部署单元。门禁 owner 仍在 face；arch-check、合同、Docker race、全前端 226 单测和固定双容器 Chromium 通过。[证据](../evidence/int-001/face-business-p2-2026-09-16.md)。质量/容量/发布不关闭；以下同日记录为此前增量。

2026-09-16 P2/A05 增量：CR-2026-030 / FR-INT-009，POST-MVP-5 S2，Owner app/face/SQLite/API。可选 face 部署组合既有人物/匿名组查询与人工审核服务；生命周期由 app 数据库适配器管理，业务策略仍归 face。无新 API/数据合同。[验证与未完成边界](../evidence/int-001/face-people-composition-2026-09-16.md)；非空与浏览器验收未关闭。

2026-09-16 P1 复查修复：FR-INT-009 / NFR-INT-002、004，CR-2026-030，POST-MVP-5 S2，Owner app/remoteml/face。激活在共享槽满时可取消等待；身份匹配的服务端 busy 使用既有有界恢复。修正先前覆盖遗漏，Docker 专项和最终镜像探针通过，不新增状态机或 API。[证据](../evidence/int-001/face-busy-repair-2026-09-16.md)。

2026-09-16 P1 完成：CR-2026-030 / FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2，Owner face/app/SQLite/API。主进程组合有界分析/清理 worker 与共享资源预算，启用既有每库控制合同；事务内可用性检查、取消/停机区分及专属恢复 owner 已落实。Docker 正式任务与强杀后租约恢复通过；P2 读写人物服务与非空质量未验收。[证据](../evidence/int-001/face-worker-p1-2026-09-16.md)。以下同日条目保留为较早增量记录。

2026-09-16 CR-2026-030 / FR-INT-009 / NFR-INT-004，POST-MVP-5 S2，Owner face/SQLite：A03 新增部署代次绑定处理器，复用原 JobProcessor 状态机，按完整指纹/维度/活动状态/可用性在推理前后检查；SQLite 单条关联查询提供内部身份合同。Docker 回归与 arch-check 通过，无 API/表结构变更；主进程组合及最终事务边界尚待补齐。[证据](../evidence/int-001/face-generation-binding-2026-09-16.md)。

2026-09-16 CR-2026-030 / FR-INT-009 / NFR-INT-002、004，Owner app/aimodel/face：可选部署 face 配置、共享准入/重试、身份绑定事务在 run.go 实例化。真实本地镜像双模型激活、停服恢复/重启通过，A02 关闭；A03 分析 worker 未组合。[证据](../evidence/int-001/face-startup-2026-09-16.md)。

2026-09-16 CR-2026-030 / FR-INT-009 / NFR-INT-004：face 完整部署身份、SQLite 原子激活提交已验证，app 独立身份/提交适配实现；无新表或部署单元。P1/A01 关闭，A02 配置/启动待接通。[合同与证据](../evidence/int-001/face-deployment-commit-2026-09-16.md)。

2026-09-16 CR-2026-030 / FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2：aimodel 原激活 worker 增加独立服务人脸分支；SQLite migration 40 扩展内部模型记录，旧文件提交拒绝 service。[合同与验证](../evidence/int-001/face-deployment-worker-2026-09-16.md)。app 真实身份/阈值准入和身份绑定代次提交尚未实现。

2026-09-15 CR-2026-030 / FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2：imagevips 解码→remoteml FaceRuntime→face.Candidate，质量启发式由 face 唯一持有、旧 faceonnx 复用。无阈值调整。
[Docker 竞态与真实四格式证据](../evidence/int-001/face-runtime-adapter-2026-09-15.md)，架构检查通过；app 人脸代次与任务仍未组合。

2026-09-15 CR-2026-030 / FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2：remoteml 新增 face-v1 客户端，复用 HTTP/并发/身份读取实现，不复用 CLIP 身份；app 尚未组合。
[13 项 Docker 竞态及真实零人脸传输证据](../evidence/int-001/face-go-transport-2026-09-15.md)。架构检查通过；face 的质量、代次和聚类归属不变。

2026-09-15 CR-2026-030 / FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2：inference 新增独立人脸身份、可选固定缓存、有界私有协议；face 仍拥有质量/聚类，app 尚未组合。
[合同与 Docker 证据](../evidence/int-001/face-service-contract-2026-09-15.md)。13 项合同/缓存测试及真实零人脸通过，架构检查通过；不代表非空识别或发行验收。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016 / NFR-INT-007：跨库混合审核账本清理后取消与另一库隔离回归通过；
[合成持久化证据](../evidence/int-001/tag-review-cross-library-2026-09-15.md)。Owner semantic/SQLite/curation，无生产契约变化。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016 / NFR-INT-007：新词表文本编码请求在 AI 暂停期间取消及恢复通过，Owner app/semantic/remoteml；
[实际证据边界](../evidence/int-001/tag-text-request-cancellation-2026-09-15.md)，本轮不改生产合同。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016 / NFR-INT-007：标签非空处理器在提交前/后取消及停机边界通过，Owner semantic/SQLite；
[确定性四场景证据](../evidence/int-001/tag-job-processing-cancellation-2026-09-15.md)。本轮无生产变化。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016 / NFR-INT-007：新本地镜像非空标签任务等待取消、恢复及 curation 保留通过；
[实际证据与未覆盖交错](../evidence/int-001/tag-job-runtime-cancellation-2026-09-15.md)。Owner app/semantic/resourcecontrol，无合同变化。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016 / NFR-INT-007：标签生成处理器有界取消收尾及 SQLite 已接受取消优先，Owner semantic/SQLite；
[Docker 边界回归](../evidence/int-001/tag-job-cancellation-2026-09-15.md)。原合同不变，非空实际运行矩阵待补。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016：正式 HTTP 非空建议接受、幂等重放及清理保留 curation 验收通过；
[本地证据和未覆盖范围](../evidence/int-001/tag-review-chain-2026-09-15.md)。Owner app/semantic/curation，无生产合同变更。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016：app 组合标签生成服务、单 worker 与共享后台额度，复用同代次 BoundEncoder；
[本地首次任务及未校准阈值边界](../evidence/int-001/tag-generation-wiring-2026-09-15.md)。Owner app/semantic，合同沿用，非空端到端未完成。

2026-09-15 CR-2026-030 / FR-INT-004 / FR-INT-016：核查确认独立 AI 主应用缺少标签生成服务/worker 组合，不能标为生产已交付；
[代码与 HTTP 证据](../evidence/int-001/tag-generation-gap-2026-09-15.md)。Owner app/semantic，非空正式审核链路需先补此依赖。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：新镜像正式 API 验证标签审核清理等待取消及重试，Owner semantic/SQLite/resourcecontrol/app；
[实际链路证据边界](../evidence/int-001/tag-clear-runtime-2026-09-15.md)。无合同变化。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：标签审核清理终态优先处理已接受取消，Owner semantic/SQLite；
[非空批次及正式标签保留证据](../evidence/int-001/tag-clear-batch-2026-09-15.md)。接口与数据结构未变。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：标签审核清理复用资源取消通知与有界收尾，SQLite 要求已接受取消，Owner semantic/SQLite；
[Docker 回归和边界](../evidence/int-001/tag-clear-admission-2026-09-15.md)。原清理合同不变。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：包含终态修复的新本地镜像通过四类受控混合取消恢复，Owner semantic/scanner/resourcecontrol/app；
[实际链路与精确交错证据边界](../evidence/int-001/cancellation-runtime-refresh-2026-09-15.md)。合同未变。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：语义清理终态事务优先处理已接受取消，Owner semantic/SQLite；
[部分及最终批次 Docker 复现与修复](../evidence/int-001/clear-batch-cancellation-2026-09-15.md)。原删除边界、领取版本及合同不变。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：真实图片处理器与 SQLite 确定性终态取消组合通过，Owner semantic/SQLite；
[证据与清理终态待修边界](../evidence/int-001/image-processor-terminal-2026-09-15.md)。本轮无生产合同变化。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：SQLite 图片终态事务优先处理已持久化取消，原领取版本校验保留；
[先失败后通过的 Docker 回归](../evidence/int-001/image-terminal-cancellation-2026-09-15.md)。Owner semantic/SQLite，无 API 或数据结构变更。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007，扫描关联 FR-SCN-004/008：新运行镜像完成扫描/语义清理受控混合等待取消；
[实际 HTTP 证据及恢复边界](../evidence/int-001/scan-clear-admission-runtime-2026-09-15.md)。Owner scanner/semantic/resourcecontrol/app，未变更合同。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：扫描入口及资源准入取消交回 scanner.abort，SQLite 原事务保持索引与代次；
[Docker 组合和既有扫描回归](../evidence/int-001/scan-admission-cancellation-2026-09-15.md)。未改变 API 或状态归属，实际 worker 混合等待验收待补。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：ClearProcessor 复用取消通知，SQLite 要求 cancelling 才接受 cancelled；
[组合回归与保留边界](../evidence/int-001/clear-admission-cancellation-2026-09-15.md)。扫描独立状态机不变。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：图片取消清理接入的新运行镜像完成受控混合任务验证；
[实际 HTTP 证据与范围](../evidence/int-001/image-admission-runtime-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：图片回填复用资源准入取消通知，semantic 有界清理、SQLite 终态 CAS 保持原归属；
[组合回归及剩余项](../evidence/int-001/image-admission-cancellation-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：实际双容器、额度 1 的图片/视频混合任务验证视频取消和恢复；
[证据及可观测边界](../evidence/int-001/mixed-admission-runtime-2026-09-15.md)。不代表所有任务类型或容量验收。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：resourcecontrol 可选准入取消通知，视频能力复用有界终态清理，SQLite CAS 不变；
[组合测试证据](../evidence/int-001/video-admission-cancellation-2026-09-15.md)。其他任务类型不视为已修复。

2026-09-15 CR-2026-030 / FR-INT-016 / NFR-INT-007：resourcecontrol 保持唯一预算 owner，补齐已取消请求准入拒绝；
[Docker 竞态回归及未验证边界](../evidence/int-001/background-admission-2026-09-15.md)，不更改并发数或任务状态机。

2026-09-15 CR-2026-030 / FR-INT-016：旧记录不存在提示、本地单条移除及刷新持久化已在实际本地镜像验证；
[浏览器证据](../evidence/int-001/missing-task-records-runtime-2026-09-15.md)。无服务端边界变化。

2026-09-15 CR-2026-030 / FR-INT-016：前端消费明确任务不存在合同，复用 preferences owner 移除本地 ID，无服务端副作用。
[Docker 回归及边界](../evidence/int-001/missing-task-records-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-004/016：实际浏览器验证多库选择、不同覆盖率、URL 刷新和 beta 补齐目标；
[证据与旧任务残留问题](../evidence/int-001/multilib-browser-2026-09-15.md)。不视为全部任务页验收。

2026-09-15 CR-2026-030 / FR-INT-004/016：全局单视频 worker 多库排队、取消隔离、重试与按库检索已在独立本地 Docker 验证；
[证据](../evidence/int-001/multilib-cancellation-2026-09-15.md)。不更改并发预算或发行 Gate。

2026-09-15 CR-2026-030 / FR-INT-004/016：实际本地镜像及浏览器验证取消收尾、
最终提交竞态和取消后补齐，历史操作保留；[证据](../evidence/int-001/video-cancellation-runtime-2026-09-15.md)。不代表多库并发/发行验收完成。

2026-09-15 CR-2026-030 / FR-INT-004/016：semantic 视频 worker 有界取消收尾，
SQLite 既有终态事务保持已接受取消优先，非用户取消保留租约恢复。
[本地 Docker 交错与跨库隔离证据](../evidence/int-001/video-cancellation-2026-09-15.md)；不扩大发行 Gate。

2026-09-15 CR-2026-030 / FR-INT-004/016：前端消费既有视频任务合同，复用确认、任务追踪/取消及失效规则。
Docker 38 项回归、实际浏览器补齐/重建通过。[证据与边界](../evidence/int-001/video-actions-ui-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-005/015/016：在独立本地 Docker 数据卷验证
coverageRecorded 从缺记录到真实模型回填成功，总览/管理页及图片/视频隔离显示通过。
[实际证据](../evidence/int-001/image-unrecorded-runtime-2026-09-15.md)，不视为全发行完成。

2026-09-15 CR-2026-030 / FR-INT-005/015/016：Settings 增加 coverageRecorded 只读字段，
SQLite 判断记录存在，API 约束未知 complete=false，前端消费生成合同；无新持久化或任务状态机。
[Docker 证据与实际新代次验收边界](../evidence/int-001/image-unrecorded-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-005/015/016：SQLite 图片设置读取合并为单快照，
覆盖率不依赖模型健康，但供任务准入的代次可用性语义不变。
[本地 Docker 与浏览器证据](../evidence/int-001/image-offline-coverage-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-004/015/016：视频状态实际运行镜像/浏览器、AI 离线恢复验证通过；
canonical queries 补充模型和图片静止状态刷新，33 项 Docker 前端测试与架构检查通过。
[记录及图片离线 0/0 未完成项](../evidence/int-001/video-status-runtime-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-004/006/007/016：semantic 拥有视频状态窄服务，SQLite
读取同一快照，API 暴露已认证只读合同，app 组合；前端消费生成客户端，不借用搜索推理。
Docker SQLite/HTTP/合同及前端定向回归通过。[证据](../evidence/int-001/video-status-ui-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-001/002/005/015/016：前端 ai 复用 AIModel.storageMode
修复独立服务开关误依赖旧运行时，标明图片覆盖率范围；无新协议或状态机。
Docker 页面测试 19/19、构建及架构检查通过。[证据](../evidence/int-001/settings-service-toggle-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-006/007/015/016：SQLite 成功扫描事务核对视频移除
覆盖率，复用完整计划谓词；未知汇总退回 pending，历史操作不改写。
[证据](../evidence/int-001/video-removal-2026-09-15.md)，API 说明/生成源及回归同步。

2026-09-15 CR-2026-030 / FR-INT-006/007/015/016：SQLite 扫描事务同步视频新增和
失效进度；完整计划仍由既有候选谓词拥有，未复制判定。[记录](../evidence/int-001/video-invalidation-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-005/015/016：图片新增/移除计数由 SQLite 原扫描
事务维护，删除汇总仅允许成功扫描路径调用；不改原媒体。[记录](../evidence/int-001/semantic-membership-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-005/015/016：SQLite 在扫描原事务中同步图片语义
进度失效，不复制 HTTP 状态机；仓储竞态及本地真实闭环通过。
[证据](../evidence/int-001/semantic-invalidation-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-001/002/005/006/007：api 语义设置路由修正为权威
OpenAPI 的 PATCH，HTTP 竞态通过；本地实际运行镜像的图片/视频全链路及重启通过。
[证据](../evidence/int-001/app-chain-2026-09-15.md)，源指纹变更后 coverage 一致性待修。

2026-09-15 最新 CR-2026-030 / FR-INT-017/020 / NFR-INT-007/009/010，POST-MVP-5：
app 拥有部署解析及启动组合，原 aimodel worker 和 SQLite 事务仍是激活唯一 owner；
run.go 已实例化共享图片/文本/视频 BoundEncoder 和后台核验。旧子片“未实例化”被
本条替代。证据：[本地主进程启动](../evidence/int-001/deployment-startup-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-017/020 / NFR-INT-007/009/010：aimodel 复用激活调度；
app 组合共享身份客户端和 semantic 提交合同，remoteml 身份复核计入既有两请求预算。
本地 Docker 相关竞态通过；run.go 未实例化，不代表应用接通。[证据](../evidence/int-001/deployment-worker-adapter-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-017/020 / NFR-INT-007/009/010：semantic 拥有部署激活
提交合同，SQLite 复用原唯一激活事务与空间绑定 helper；不增加状态机，失败保留旧索引。
本地 Docker 7 种场景和真实图片/视频链路通过，app 未调用；[证据](../evidence/int-001/deployment-activation-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-017/020 / NFR-INT-007/009/010：aimodel 拥有 service
记录验证/内部注册，SQLite migration 39 保留引用并约束类型；旧文件刷新跳过、旧激活拒绝。
OpenAPI 响应 enum 扩展，安装请求不扩展；[证据](../evidence/int-001/service-model-record-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-019/020 / NFR-INT-007/009/010：semantic.VideoProcessor
统一消费流式编码器，旧构造器复用既有桥接；完整计划/事务所有权不变，app 未切换。
本地 Docker 单元、视频 SQLite 竞态及真实四帧链路通过；[证据](../evidence/int-001/remote-video-stream-2026-09-15.md)。

2026-09-15 ADR-0020 / CR-2026-030 / FR-INT-017/019/020 / NFR-INT-007/009/010：
`services/inference` 拥有专用服务与固定快照；remoteml 拥有独立预期身份校验和响应绑定。
候选本地 Docker 单元/竞态及真实链路通过，生产 app 组合未完成；
[证据](../evidence/int-001/owned-ml-image-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-017/019/020 / NFR-INT-007/009/010：SQLite 测试适配
以 `remotemlprobe` 标签组合真实 HTTP、imagevips、BoundEncoder、既有回填任务和持久化检索；
本地 Docker 竞态通过，3 张合成图片写入后数据库重开检索成功。仅测试身份，无部署准入、
主应用组合或质量承诺；[证据](../evidence/int-001/independent-ml-persisted-chain-2026-09-15.md)。

2026-09-15 CR-2026-030 / FR-INT-017/019/020 / NFR-INT-007/009/010：semantic 统一拥有旧
SigLIP 代次准入，app 图片/文本工厂消费；新增状态/维度/空间矩阵和工厂边界回归，
本地 Docker 竞态通过。属于兼容 Gate 修补，不开放生产接入；[证据](../evidence/int-001/independent-ml-legacy-boundary-2026-09-15.md)。

2026-09-15 CR-2026-030 本地 Docker test profile 已覆盖 native/semantic 竞态与 8 项 SQLite
代次/回填测试、真实离线向量 probe；SQLite 全包首次 5 分钟超时，随后不缩减范围以 15 分钟预算完整复跑通过。
详见 [Docker 回归](../evidence/int-001/local-docker-regression-2026-09-15.md)；发行 Gate 不变。

2026-09-15 / CR-2026-030：semantic 回填统一 stream 编码入口，legacy bridge 保留旧 SigLIP。
SQLite 真实任务成功/提交前退役零写入测试通过，见 [证据](../evidence/int-001/independent-ml-stream-backfill-2026-09-15.md)。
新 app 组合尚未启用，事务与队列所有权未变化。

CR-2026-030 共同准入子片：semantic BoundEncoder 为文本/图片共用空间与状态检查 owner，
SQLite 提供持久化 GenerationRuntime，transport 不持有准入策略。
[单元/SQLite 组合证据](../evidence/int-001/independent-ml-bound-encoder-2026-09-14.md)通过；主应用替换仍待完成。

CR-2026-030 / FR-INT-017/019/020、NFR-INT-007/009/010 / POST-MVP-5：
semantic GenerationSpaceRepository → SQLite semantic_space / migration 38 实现一次性代次绑定，
保留既有 write transaction 所有权。旧代次不补指纹；`semantic_space_test.go` 覆盖幂等、
冲突、维度、旧状态、已有向量拒绝、独立数据库连接读取和删除级联。新查询/回填组合仍未完成。

CR-2026-030 图片入口增量：opt-in `TestEncodedImageServiceProbe` 已在 Linux/arm64 将
PNG/JPEG/GIF/WebP 合成 stream 经 imagevips 编码、Go HTTP 送达真实固定 ML，向量验证通过。
[证据](../evidence/int-001/independent-ml-encoded-chain-2026-09-14.md)不替代 app/index/generation 集成 Gate。

CR-2026-030 / POST-MVP-5 / FR-INT-017/019/020、NFR-INT-007/009/010：
[编码图片入口合同](independent-ml-image-contract.md)由 semantic 拥有，imagevips 实现；
接受已打开流，不解析路径，独立于 SigLIP 张量。尚未接入 app，不改变发行 Gate。

CR-2026-030 隔离缓存诊断：`spikes/int001-remote-ml/cache-audit.mjs` 校验完整本地基线，
5 项合成测试与真实只读缓存/固定镜像离线复测通过；[证据](../evidence/int-001/independent-ml-cache-audit-2026-09-14.md)。
不作为 app 生产文件边界或服务身份认证，不改变模型代次准入 Gate。

2026-09-14 / CR-2026-030 / POST-MVP-5 / FR-INT-017/019/020、NFR-INT-007/009/010：
semantic 单独拥有[向量空间身份与代次兼容合同](independent-ml-generation-contract.md)，
实现于 `space_identity.go`；字段变化、非法描述、旧代次拒绝有合成单元/竞态证据。
这是纯检查，不是已接入的运行时防线；模型缓存事实核验、持久化与 app 组合仍待完成。

2026-09-14 POST-MVP-5 / FR-INT-017/019/020、NFR-INT-007/009/010：
[ADR-0019](../adr/0019-independent-machine-learning-service.md) 接受独立 AI 容器方向。
app 仍持有任务/数据/媒体边界，inference 拥有内部服务适配。隔离协议探针位于
`spikes/int001-remote-ml`，Linux/arm64 离线 HTTP 测试通过；生产协议、代次和 UI 迁移未完成。
Go HTTP adapter 位于 `internal/inference/remoteml`，遵循
[传输子片合同](independent-ml-transport-contract.md)，单元/竞态覆盖故障与资源边界；
不读媒体、不写索引，不冒用旧 ImageEncoder 接口。模型代次/身份准入 Gate 保持未完成。
Go adapter 的真实离线容器合成联调也已通过，见
[传输证据](../evidence/int-001/independent-ml-go-transport-2026-09-14.md)。
对应用户选定的设置原型位于 `spikes/int001-ui-settings`，仅复用共享 UI，不定义新 API
或生产路由。浏览器视觉/交互检查通过，模型身份与后端 Gate 不因此变更。
正式 `AISettingsPage` 已按 CR-2026-030 的 UI-only 范围应用方案一：仅重排既有
S2A/S2C/S3 契约消费者，保留独立 ETag/CSRF 和兼容维护入口。197 项前端测试与
桌面/手机复核通过，见 [复核记录](../../design-qa.md)；新独立服务接入仍未完成。

2026-09-10 FR-INT-017/019/020 / POST-MVP-5，CR-2026-026/027：
aimodel 安装与激活 owner、files managed publisher、ONNX runtime、SQLite commit 的
真实候选隔离集成通过；测试为 `semantic_candidate_activation_test.go`，详情见
[证据](../evidence/int-001/siglip-download-source-check-2026-09-10.md)。
不替代生产下载源、正式 runtime 闭包或质量 Gate。

FR-INT-017/019、NFR-INT-007/009 的管理员代理设置见 [CR-2026-029](../changes/CR-2026-029-admin-model-proxy-settings.md) / ADR-0018。
aimodel 拥有模式/优先级，downloadhttp 拥有 URL/固定探测；SQLite 迁移 00037 使用 write gate + revision。
Backend Ready：持久化集成已通过；API/前端回归记录见 CR。范围为 POST-MVP-5，发行 Gate 不变。

FR-INT-017/019、NFR-INT-007/009 的显式下载代理扩展见
[CR-2026-028](../changes/CR-2026-028-model-download-proxy.md) / ADR-0017：app 配置、downloadhttp
唯一出口策略 owner；配置/CONNECT/TLS/目标拒绝/race 回归及 server222 固定文件 probe 已通过。
代理模式 DNS/IP enforcement 委托部署代理，不复用入站代理信任配置；未改变发布 Gate。

2026-09-10 FR-INT-017/019 模型获取跟进：隔离 SigLIP ABI 转换与 native candidate 测试证据见
[源核对](../evidence/int-001/siglip-download-source-check-2026-09-10.md)。仅补充开发验证，未改变
reviewed Catalog、runtime 合同或 S4 Gate。脚本留在 spikes，native 测试需显式 build tag 与精确 fixture。

FR-INT-017/019/020 的 POST-MVP-5 预设展示扩展见
[CR-2026-027](../changes/CR-2026-027-builtin-model-presets.md)：aimodel 拥有候选元数据，API 映射，
前端消费生成合同；候选不会进入可安装 Catalog。回归覆盖预设隔离、HTTP 元数据及 UI 不提供虚假下载。
此项不代表默认模型下载、激活或 S4 发布验收完成。

## 状态与使用方式

本文把当前已确认的需求族映射到架构责任、契约、数据、决策、风险、验证和交付阶段。它是设计与评审索引，不表示表中 API、前端、容器或全部测试已经实现。

状态来源：

- 目标版本与冻结范围以当前 [scope revision 4](../releases/MVP-2026-07-23-scope-r4.md) 为准，需求语义以[产品需求](../product-requirements.md)为准；
- 结构决策以[已接受 ADR](../adr/)为准；
- HTTP 结构以权威 [`api/openapi.yaml`](../../api/openapi.yaml) 为准；[API 设计说明](../api-design.md)
  只保留动机与实现参数，不能覆盖 wire 契约；
- 已有证据只以 [spike 报告](../spikes/)、[测试策略](../testing-strategy.md)和 Gate 记录明确链接的 scope 为准。

表中的“计划 API/流程”和“必需验证”是交付目标，不得据此宣称功能已可用。具体切片必须遵守[交付与架构治理](delivery-governance.md)。
`POST-MVP-5` 的本地技术探索已于 2026-08-27
[收口](../gates/POST-MVP-5/int-s0-closeout-and-blockers.md)；[revision 2](../releases/POST-MVP-5-scope-r2.md)
现冻结 A～E。INT-S0/INT-S1 对 A+B 为 Go，C/D/E 的 INT-S1R2 为 Go；按 CR-2026-022，S2A/S2B/S2C
均已签署 **Backend Ready / Release No-Go**。最终模型、governed 质量、native 双架构、联合容量、供应链和
隐私/合规发布签署由 S4 持有；S3 可实现合同消费者，但未获 S4 Go 的 UI 不得进入发行路径。
Stage 0 Gate 已通过并只授权后端优先的 Stage 1：OpenAPI 已成为结构权威，FS-01～05 的
Stage 0 范围及供应链识别已有报告证据；Backend Ready 结论只来自各 S2 Gate，不授权跳过后端实现或
把 release No-Go 的业务 UI 公开发布。

`FR-MED-001/004/005` 的 2026-08-06 可靠性回归见
[大视频探测与失败诊断纠偏](../changes/FIX-2026-08-06-large-video-probe-diagnostics.md)：
视频源上限为 1 TiB，probe/poster 各有 60 秒预算，超限、缺少 moov、不支持解码器、明确
损坏与暂时性工具故障使用不同结构化原因；无 migration 或媒体内容变更。

## 功能需求追踪

2026-09-08 的 POST-MVP-5 扩展 [CR-2026-026](../changes/CR-2026-026-ai-model-download-and-ui.md)
对应 FR-INT-015/017/019/020、NFR-INT-007/009/010：`aimodel` 唯一拥有下载目录与准入，
`aimodel/downloadhttp` 为受控出站适配器，`files` 读取配置并复用 managed store；SQLite migration 36
提供最近任务索引，API/OpenAPI → domain adapter → AI settings 为消费者。复用安装 worker/operation owner，
不增加进程或状态机；ADR-0016 允许部署管理员固定目录，空配置仍不联网。Architecture fitness 验证
composition 只能读取固定配置并保留 face/Release No-Go；下载/集成/UI 测试链接见 CR。
阶段为后端与 UI 实现扩展，不将历史发布 Gate 改为 Go；真实包下载源、推理镜像和最终验收仍不在完成声明中。

2026-09-09 的[部署一致性修复](../changes/FIX-2026-09-09-aifeature-deployment-integrity.md)归属
`internal/webassets` / 构建部署，关联 FR-DEP-001～004、上述 FR-INT 与 NFR-REL-001。
`scripts/verify-deployed-web.mjs` 及 3 项 Node 回归测试建立“新 dist → 镜像内嵌 SPA → 线上字节”校验；
候选和 server222 正式端点已通过。健康检查不再作为前端版本的唯一证据，S3 / S4 Gate 状态不变。

| 需求族 | 目标版本 | 范围状态 | Change Record | Capability / adapter | 计划 API 或用户流程 | 数据与派生状态 | 决策与约束 | 主要风险 | 必需验证 | Roadmap 阶段 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `FR-DEP-001～004` 部署与设置 | `MVP-2026-07-23` | Frozen r1；[CR-2026-013 root runtime](../changes/CR-2026-013-root-runtime-bind-data.md)重新打开身份候选；既有 S5-001/002/004 历史证据保留 | `BASELINE-2026-07-23`、`CR-2026-013` | `internal/app`、`internal/api`、`internal/webassets`、`internal/store/sqlite`；根 `Dockerfile`、`compose.yaml` 与容器适配器 | 启动/初始化、嵌入 SPA、`/health/live`、`/health/ready`、`/api/v1/status`、设置读写 | migrations、`settings`、运行版本与健康状态；Vite 产物仅在构建期嵌入 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0009](../adr/0009-linux-openat2-single-media-root.md)、[ADR-0012](../adr/0012-root-runtime-bind-data.md)、[部署](../deployment.md) | R-002、R-004、R-008、R-011、R-014、R-016、R-022 | 历史非 root 双架构证据不证明当前身份；root 候选必须以 Docker 自动创建的 root-owned `/app/data` bind 复验数据库、只读根/媒体、health、SIGTERM、Compose/代理、恢复与供应链 | 1、5 |
| `FR-AUTH-001～004`、`FR-DEP-005` 认证与 LAN 部署 | `MVP-2026-07-23` | Frozen r2；认证 Integrated Done；LAN HTTP implemented；密码最低长度可用性修复 | [CR-2026-002](../changes/CR-2026-002-authenticated-lan-http.md)；[FIX-2026-07-29](../changes/FIX-2026-07-29-admin-password-minimum.md) | `internal/auth` 拥有规则；`internal/api` 拥有 transport/HTTP；`internal/app` 拥有监听配置；Compose 拥有宿主绑定 | 首次 setup（8～128 Unicode 字符密码）、login、session、logout；受保护业务 API；受信 LAN HTTP 与可选严格 HTTPS 代理 | 认证数据不变；transport 不持久化，未受信转发头清除 | [ADR-0005](../adr/0005-built-in-single-admin-auth.md)、[ADR-0010](../adr/0010-authenticated-lan-http.md)、[安全模型](../security.md) | R-010、R-011、R-016 | 8 字符 ASCII/中文接受、7 字符拒绝、前后端 Unicode 计数一致；Argon2id、限流、错误脱敏、session/CSRF 回归保持。直连 HTTP 使用实际 peer/Host，显式可信代理配置继续严格校验 HTTPS 并拒绝旁路 | 1（Integrated Done）、5 |
| `FR-AUTH-005`、`FR-UI-009` 管理中心独立功能页与账户维护 | `MVP-2026-07-23` | Frozen r3；[UIF-S4 Integrated Slice Done](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md) Go | [CR-2026-006](../changes/CR-2026-006-management-center-pages.md) | `internal/auth` 拥有当前密码验证、密码哈希与其他会话撤销；`internal/settings` 拥有受控偏好；既有 library/scanner/thumbnail capability 保持规则所有权；`web` settings routes 只通过生成 client/adapter 组合 | 独立 general/libraries/storage/account 路由；账户资料/密码；媒体库、扫描、缓存 operation 复用 | 单管理员显示名称/密码哈希/会话；settings revision；原型 local state 不进入生产 import graph | ADR-0005、ADR-0007；[用户流程第 8 节](../user-flows.md)；[界面设计](../ui-design.md)；[UIF 当前状态](../releases/MVP-2026-07-23-uif-integration-status.md)；[UIF-408 evidence](../evidence/uif-408/README.md) | R-010、R-012、R-015、R-016 | 账户 OpenAPI/事务/ETag、四独立路由、generated client、真实改名/改密/退出/重登、四档逐页视觉、浏览器/可访问性、容量与完整仓库验证通过；物理辅助功能仍由 Stage 5 持有 | `UIF-S4 Integrated Done → Stage 5 No-Go` |
| `FR-BRW-010`、`FR-UI-010`、`NFR-UIF-001` 生产前端原型一致性 | `MVP-2026-07-23` | Frozen r4；[UIF-S0](../gates/MVP-2026-07-23/uif-s0-architecture-ready.md)、[S1](../gates/MVP-2026-07-23/uif-s1-contract-ready.md)、[S2](../gates/MVP-2026-07-23/uif-s2-backend-evidence-ready.md)、[S3](../gates/MVP-2026-07-23/uif-s3-consumer-ui-ready.md)、[S4](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md) Go | [CR-2026-009](../changes/CR-2026-009-frontend-prototype-fidelity.md) | `internal/auth` 账户维护；`internal/catalog` 目录 q/cursor；`internal/thumbnail` 缓存摘要/清理；SQLite/API adapter；`web` 唯一 token、壳、媒体集合和预览 owner | account profile/password；direct-directory q；cache summary/cleanup；全局 Header、BrowseShell、ManagementShell、Search 无侧栏、四档视觉 Gate | migration 13 已实现 users revision、directory search key、singleton cleanup state，并复用 sessions/idempotency/cache；视觉 reference 不进入生产 import graph | [FTR-UIF-001](../features/frontend-prototype-fidelity.md)；[开发清单](../features/frontend-prototype-fidelity-task-list.md)；[UIF 当前状态](../releases/MVP-2026-07-23-uif-integration-status.md)；[UIF-408 evidence](../evidence/uif-408/README.md)；ADR-0005/0007；无需新 ADR | R-010、R-012、R-015、R-016、R-021 | 逐页比较、Linux 基线、真实纵向链、三引擎/可访问性、100k/10k 容量、完整仓库验证、跨文档收敛和 12 页 × 4 断点复核均有实际证据；受影响 Stage 5 检查已重跑，独立发布阻断保持 No-Go | `UIF-S4 Integrated Done → Stage 5 No-Go` |
| `FR-LIB-001～008` 媒体库管理 | `MVP-2026-07-23` | Frozen r1；[Backend Ready](../gates/MVP-2026-07-23/s2-library-backend-ready.md)；[Integrated Done](../gates/MVP-2026-07-23/s2-library-scan-integrated-done.md) | `BASELINE-2026-07-23`；[Stage 2 Architecture Ready](../gates/MVP-2026-07-23/stage-2-architecture-ready.md)；[Contract Ready](../gates/MVP-2026-07-23/s2-library-contract-ready.md) | `internal/library`；`internal/files` 和 SQLite adapter；`web/src/features/libraries` | 允许目录选择、媒体库创建/列表/详情/改名/移除、离线重试 | `libraries.revision`、`name_sort_key`、唯一 creation scan、`library_removals`、摘要化 `idempotency_records`；根不可变；创建三记录同事务 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0002](../adr/0002-library-path-model.md)、[ADR-0004](../adr/0004-library-root-immutable.md)、[ADR-0009](../adr/0009-linux-openat2-single-media-root.md)、[安全模型](../security.md) | R-002、R-003、R-012、R-016 | S2-002～007 已通过后端完整矩阵；前端 S2-201～208 通过生成 client adapter 连接真实列表、长路径幂等创建、ETag 改名、异步幂等移除、离线重试和扫描/设置页面。真实 Chromium 成功链与受控故障状态矩阵覆盖长内容、重复提交、键盘焦点、主题/语言、四档响应式和 axe，Stage 2 已 Integrated Done | 2（Integrated Done） |
| `FR-SCN-001～009` 扫描与索引 | `MVP-2026-07-23` | Frozen r1；[Backend Ready](../gates/MVP-2026-07-23/s2-scan-backend-ready.md)；[Integrated Done](../gates/MVP-2026-07-23/s2-library-scan-integrated-done.md)；[Contract Ready](../gates/MVP-2026-07-23/s2-scan-contract-ready.md)；[S2-102 worker](../gates/MVP-2026-07-23/s2-bounded-scan-worker.md)；[S2-103 目录计数](../gates/MVP-2026-07-23/s2-directory-counts.md)；[S2-104 媒体收敛](../gates/MVP-2026-07-23/s2-media-convergence.md)；[S2-105 故障恢复](../gates/MVP-2026-07-23/s2-scan-recovery.md)；[S2-106 容量并发](../gates/MVP-2026-07-23/s2-scan-capacity.md) | `BASELINE-2026-07-23`；[Stage 2 Architecture Ready](../gates/MVP-2026-07-23/stage-2-architecture-ready.md) | `internal/jobs` 拥有领取/租约/worker；`internal/scanner` 拥有 generation、admission、查询游标、取消、scheduler、扫描周期范围、容量常量与目录策略；`internal/settings` 只编排 typed setting 更新；`internal/thumbnail` 拥有缓存配额范围；`internal/media` 拥有候选格式与 source fingerprint；`internal/files` 与 SQLite adapter；`web/src/features/libraries` 和 `web/src/features/settings` | 创建/启动/定时/手动扫描、历史/详情轮询、协作取消、错误和跳过统计、计划设置 | `scan_runs`、`scan_issues`、`directories`、`assets`、`settings`、generation、source fingerprint；同库唯一 full scan 与 durable admission | [ADR-0003](../adr/0003-scan-consistency.md)、[数据模型](../data-model.md) | R-003、R-004、R-005、R-013、R-016 | S2-101～107 已完成全部冻结后端 operation；前端 S2-204～208 连接详情轮询、取消、重试、可靠索引保留提示及 ETag 设置更新。真实成功链加受控 running/cancelled/failed/offline 契约状态通过 Chromium、axe、长内容和响应式矩阵，Stage 2 已 Integrated Done | 2（Integrated Done）；容量证据在 0/FS-04 与 S2-106 |
| `FR-BRW-001～009` 导航与浏览 | `MVP-2026-07-23` | Frozen r1；[S3-001 Contract Ready](../gates/MVP-2026-07-23/s3-browse-contract-ready.md)；[S3-002 keyset](../gates/MVP-2026-07-23/s3-catalog-keyset.md)；[S3-003 目录树](../gates/MVP-2026-07-23/s3-directory-tree.md)；[S3-007 Backend Ready](../gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)；[S3-101 前端目录导航](../gates/MVP-2026-07-23/s3-frontend-directory-navigation.md)；[S3-102 前端浏览范围](../gates/MVP-2026-07-23/s3-frontend-browse-scope.md)；[S3-103 前端媒体集合](../gates/MVP-2026-07-23/s3-frontend-media-collection.md)；[S3-104 前端浏览状态](../gates/MVP-2026-07-23/s3-frontend-browse-states.md)；[S3-105 前端媒体预览](../gates/MVP-2026-07-23/s3-frontend-media-preview.md)；[S3-106 固定预览交互](../gates/MVP-2026-07-23/s3-frontend-pinned-preview.md)；[S3-107 前端容量预算](../gates/MVP-2026-07-23/s3-frontend-capacity.md)；[Stage 3 Integrated Done](../gates/MVP-2026-07-23/s3-browse-integrated-done.md) | `BASELINE-2026-07-23` | `internal/catalog` 拥有 root/目录/资产 query normalization、排序、cursor payload、breadcrumb 与拓扑验证；`internal/cursor` 提供 token 机制；SQLite/API adapter；`web/src/features/browse` 拥有目录导航、浏览 URL codec、query、有界 pending 刷新策略、预览选择/固定状态机与 UI，`web/src/lib/api/catalog.ts` 是生成 client 的领域 adapter；共享 `MediaCollection` 拥有 MediaCard、布局、虚拟窗口、容量预算、焦点恢复、skeleton 和分页错误，`MediaPreview` 拥有非模态图片/原生视频/信息/导航/固定/Escape/调宽，`AsyncState` 拥有空/错误/离线语义 | 媒体库切换、目录树/面包屑、当前/递归浏览、布局与排序、URL 恢复、非模态预览选择/固定/双击/关闭恢复 | `directories`、`assets`、`libraries.current_generation`、migration 7 `natural_name_key`/浏览索引、直接/递归计数；客户端只保留 URL/显示偏好/瞬时选择与预览状态 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[界面设计](../ui-design.md)、[S3 浏览契约](../api-design.md#s3-浏览契约) | R-005、R-012、R-013、R-015、R-016 | S3-001～007 已完成后端与真实 composition；S3-101～108 已通过生成 client 接入真实导航、direct/recursive keyset、模式默认排序、来源返回、ready WebP、自适应 grid、记忆 masonry、TanStack Virtual 有界 DOM、完整浏览状态矩阵、共享图片/原生视频非模态预览、固定交互，以及 100k DOM/cursor/虚拟滚动/播放资源/焦点预算。真实成功链、受控状态 Chromium、稳定浅深主题 axe、响应式 E2E、原型同状态对照与 100k Chromium 主档共同形成 Integrated Done 证据 | 3（Integrated Done） |
| `FR-SRH-001～004` 搜索过滤排序 | `MVP-2026-07-23` | Frozen r1；[S4-001 Contract Ready](../gates/MVP-2026-07-23/s4-search-contract-ready.md)；[S4-002 Implemented](../gates/MVP-2026-07-23/s4-search-keyset.md)；[S4-003 Backend Ready](../gates/MVP-2026-07-23/s4-search-backend-ready.md)；[S4-004 Frontend Done](../gates/MVP-2026-07-23/s4-frontend-search.md)；[S4-005 Search Preview Done](../gates/MVP-2026-07-23/s4-frontend-search-preview.md)；[S4-006 Viewer Done](../gates/MVP-2026-07-23/s4-frontend-media-viewer.md)；[Stage 4 Integrated Done](../gates/MVP-2026-07-23/s4-search-media-integrated-done.md)；[2026-09-01 容量维护](../gates/MVP-2026-07-23/s4-search-capacity-regression-2026-08-31.md)本地通过、native 验收待完成 | `BASELINE-2026-07-23` | `internal/catalog` 拥有 search profile、scope、排序与 cursor；SQLite FTS adapter 以 sort/scope 索引有序扫描并执行精确 FTS membership；`web` search feature 的 URL/query owner；共享 `SearchInput`、`MediaCollection`、`MediaPreview` 与 `MediaViewer` | 当前目录（可递归）/当前库/全部库搜索，类型和 filesystem mtime 半开区间过滤，结果预览/查看/返回 | migration 10 的 `assets.search_*_key`、外部内容 FTS5 `asset_search`、`catalog_search_state.revision`；库内 generation 与跨库 revision cursor；URL 保存搜索指纹，TanStack Query 保存游标页面，路由瞬时状态仅保存已载入结果 ID 序列 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[数据模型](../data-model.md)、[S4 搜索契约](../api-design.md#s4-搜索契约) | R-005、R-012、R-016 | S4-002～003 已完成搜索实现、100k/10k 主档与 Backend Ready；2026-09-01 有序扫描维护在两库完整矩阵保持逐 ID 等价，本机 arm64 production keyset P95 133.637 ms、零预算违规；native 双架构复验待完成。S4-004～009 已连接真实搜索、共享非模态预览和完整查看器，并以一次性真实后端验证查询 URL、固定预览筛选保留、进入查看器、返回查询与卡片焦点恢复。Stage 4 Integrated Done，最终浏览器/真机与发布证据归 Stage 5 | 4（Integrated Done） |
| `FR-MED-001～008` 缩略图、查看器与视频 | `MVP-2026-07-23` | Frozen r1；[S3-004 媒体处理](../gates/MVP-2026-07-23/s3-media-processing.md)；[S3-005 媒体任务/缓存](../gates/MVP-2026-07-23/s3-media-jobs-cache.md)；[S3-006 资源安全](../gates/MVP-2026-07-23/s3-media-resource-safety.md)；[S3-007 Backend Ready](../gates/MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)；[S3-105 前端媒体预览](../gates/MVP-2026-07-23/s3-frontend-media-preview.md)；[S3-106 固定预览交互](../gates/MVP-2026-07-23/s3-frontend-pinned-preview.md)；[Stage 3 Integrated Done](../gates/MVP-2026-07-23/s3-browse-integrated-done.md)；[S4-005B Content Backend Ready](../gates/MVP-2026-07-23/s4-media-content-backend-ready.md)；[S4-006 Viewer Done](../gates/MVP-2026-07-23/s4-frontend-media-viewer.md)；[S4-007 Media Strategy Done](../gates/MVP-2026-07-23/s4-frontend-media-strategy.md)；[S4-008 Media Matrix Done](../gates/MVP-2026-07-23/s4-frontend-media-matrix.md)；[Stage 4 Integrated Done](../gates/MVP-2026-07-23/s4-search-media-integrated-done.md) | [CR-2026-001](../changes/CR-2026-001-non-modal-media-preview.md)；[浏览器拥有视频播放能力判断](../changes/FIX-2026-08-04-browser-owned-video-playback.md) | `internal/media` 拥有探测事实/错误/fingerprint/资源上限，不替客户端断言平台相关解码能力；`internal/thumbnail` 拥有派生键、交付状态和缓存策略；`internal/jobs` 拥有 worker/lease；`internal/app` 拥有 native lifecycle；`internal/files`、SQLite/cache、govips、FFmpeg adapter；`web/src/lib/media/availability.ts` 是媒体可用性呈现策略 owner；共享 `MediaPreview` 是非模态 preview owner，`MediaViewer` 是完整查看 chrome owner，`web/src/features/media` 组合详情 query 与路由 | 资产详情、thumbnail、原内容/Range；非模态预览、完整图片/GIF 查看器、原生视频 poster；浏览器先尝试原内容并在真实 media error 后降级；损坏/离线/删除状态、缓存设置 | migration 8 `assets`/`thumbnails`；migration 9 `media_jobs`/fairness/`cache_deletions`；source fingerprint/transform version；遗留 `unsupported_codec` 不预先阻断且无需 migration；缓存文件在 `/app/data/cache`；查看器仅缓存服务端详情与瞬时 ID 序列 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[安全模型](../security.md)、[数据模型](../data-model.md) | R-006、R-007、R-008、R-009、R-014、R-016 | S3-004～007 已实现缩略图/媒体 processing 与认证交付；S4-005B 完成原内容 GET/HEAD/Range、admission、取消和错误脱敏；S4-006～008 完成查看器、GIF/视频、唯一降级策略、Chromium 桌面/Pixel 5 触摸、真实 `206 Range` 和 codec/offline/deleted 矩阵；2026-08-04 回归进一步固定非 H.264 为 unknown、遗留状态仍创建 video、只有原生 error 后提示；S4-009 以真实搜索纵向链完成 Stage 4 Integrated Done。Firefox、Safari/WebKit 与物理设备仍归 Stage 5 | 0/FS-01/03；3（thumbnail/preview Integrated Done）、4（search/media Integrated Done） |
| `FR-MED-009～011`、`FR-UI-008` 视频故事板悬停预览 | `POST-MVP-1` | Frozen r1；[VSP-S2 Backend Evidence Ready](../gates/POST-MVP-1/vsp-s2-backend-evidence-ready.md) Go；[VSP-S3 Consumer/UI Ready](../gates/POST-MVP-1/vsp-s3-consumer-ui-ready.md) Go；[VSP-301 Product Vertical](../gates/POST-MVP-1/vsp-301-product-vertical.md) Done；[VSP-302 Target Platform](../gates/POST-MVP-1/vsp-302-target-platform.md) Pending | [CR-2026-004](../changes/CR-2026-004-video-storyboard-preview.md)；[自适应处理预算修复](../changes/FIX-2026-08-06-adaptive-storyboard-budget.md) | `internal/thumbnail` 拥有 variant/采样/派生/缓存与 10→4 帧降级；`internal/media` 拥有 FFmpeg adapter 与处理预算；`internal/jobs` 调度；`internal/files` 安全打开；SQLite/cache/API adapter；共享 `MediaCollection` 和唯一 availability adapter | authenticated `thumbnail?variant=storyboard` 已冻结并实现；poster 后低优先级生成；桌面 fine-pointer hover 按需加载；touch/键盘/reduced-motion 回退 poster | migration 11 已实现 replacement tables、4/10 帧 layout CHECK、priority、source fingerprint/transform version、原子发布、统一 LRU 和 128 项 bounded admission；自适应预算不修改 schema 或 transform version；Linux 100k/10k、10% 视频档通过 | [FTR-VID-001](../features/video-storyboard-preview.md)；复用 ADR-0001/0006/0007/0009；若改变 job 一致性/所有权则先新增 ADR | R-006、R-009、R-013、R-015、R-016、R-018 | [VSP-002](../spikes/vsp-002-video-storyboard.md)验证 2s～2h fast seek/sprite；OpenAPI、生成 client、双架构 FFmpeg runtime、生产镜像 API/cache repair、故障矩阵、race 和 Linux 四核/4 GiB 容量均通过；2026-08-06 回归固定 1080p/4K 预算、request 级取消、同 job 10→4 帧降级与预算耗尽终态；共享 hover/input/reduced-motion、组件工作台、六种浏览器/输入 profile、100-video 与 100k 三浏览器容量通过；生产镜像真实登录、浏览/搜索 hover、预览焦点恢复已贯通；[readiness 快照](../releases/POST-MVP-1-readiness.json)及 fitness test 以证据锚点校验 Gate/AC/风险/任务并保持 No-Go；原生双架构结构化证据入口已建立，当前远端 runner 在分配前受账户计费阻断，模拟 amd64 按安全边界失败关闭，不能替代 VSP-302 | `VSP-S0 Done → S1 Done → S2 Done → S3 Done → VSP-301 Done → VSP-302 Pending → VSP-303/304` |
| `FR-MED-013` 扫描后派生媒体进度 | `POST-MVP-1` | Frozen r5；只读纵向切片实现 | [CR-2026-014](../changes/CR-2026-014-derived-media-progress.md) | `internal/thumbnail` 拥有聚合语义；SQLite 读取现有 durable job；API/Web adapter 与生成 client | `GET /libraries/{libraryId}/media-processing`；状态页把 scan、thumbnail/poster、storyboard 分开；active 时有界轮询 | 无 migration；只读取 `assets`/`media_jobs`，不创建通用 operation run | 复用 ADR-0001/0006/0007/0009；R-005/R-013/R-015/R-016 | SQLite/API/service unit、OpenAPI/生成、Web type/unit、架构与契约检查 | `POST-MVP-1 r5 maintenance slice` |
| `FR-MED-012` AVI、`FR-BRW-011` 文件大小排序 | `POST-MVP-1` | Frozen r2；实现与本地验证完成 | [CR-2026-010](../changes/CR-2026-010-avi-and-size-sort.md) | `internal/media` 统一格式注册和 FFmpeg adapter；`internal/catalog` 排序/cursor owner；SQLite/API adapter；浏览与搜索 URL/UI owner | `.avi` 索引、poster/storyboard、原内容 Range；`sort=size&order=asc|desc`；默认排序不变 | migration 14 保留 catalog/FTS/派生外键并追加 AVI CHECK 与 size indexes；复用 `assets.size_bytes`、source fingerprint、既有派生任务与缓存 | 复用 ADR-0001/0006/0007/0009；AVI 不承诺浏览器直放；R-006/R-007/R-012 | 格式/MIME、真实 FFmpeg AVI、超限与损坏、同大小跨页 size keyset、OpenAPI/生成、URL/UI 双语、完整 Go/Web/集成与容器 smoke 已通过 | `Post-MVP/1 optimization implemented` |
| `FR-MED-014～015` 视频预览自动播放与默认静音偏好 | `POST-MVP-1` | Frozen r9；实现与本地验证完成 | [CR-2026-016](../changes/CR-2026-016-video-preview-autoplay.md)；[CR-2026-019](../changes/CR-2026-019-video-preview-default-mute.md) | `web/src/lib/storage/preferences.ts` 拥有两个独立偏好的默认与持久化；共享 `useMediaPreviewController` 读取；共享 `MediaPreview` 消费；通用设置编辑 | autoplay 与 muted 默认均开启且可独立关闭；带声音 autoplay 服从浏览器策略；完整查看器与 storyboard 不变 | 复用 `foliopath.preferences.v1`，无 API、migration 或服务器状态 | 复用 ADR-0001/0007；R-015/R-016；浏览器阻止带声音 autoplay 时保留原生手动播放 | 两项偏好默认/写入、设置保存、共享 video 属性、Browse/Search 与类型/架构/生成检查 | `Post-MVP/1 revision 9 implemented` |
| `FR-BRW-013` 默认排序与媒体布局 | `POST-MVP-1` | Frozen r7；实现与本地验证完成 | [CR-2026-017](../changes/CR-2026-017-default-media-presentation.md) | `web/src/lib/storage/preferences.ts` 拥有排序/布局偏好；Browse/Search URL codec 拥有可导航排序；共享 `MediaCollection` 拥有 grid/masonry | contextual 或六种字段/方向默认；grid/masonry；显式 URL 优先并规范化 | 复用 `foliopath.preferences.v1`；无 API、migration 或服务器状态 | 复用 ADR-0001/0007；R-005/R-015/R-016；不得让偏好破坏复制 URL | 偏好读写、设置联合保存、Browse/Search codec 优先级/规范化、页面与类型/架构/生成检查 | `Post-MVP/1 revision 7 implemented` |
| `FR-BRW-012` 选中目录媒体类型数量 | `POST-MVP-1` | Frozen r3；实现与本地验证完成 | [CR-2026-011](../changes/CR-2026-011-directory-media-counts.md) | `internal/catalog` 聚合语义；SQLite adapter；AssetPage OpenAPI；浏览类型控件 | direct/recursive/q 范围的 all/images/videos，kind 与分页不改变统计基准 | 复用 assets/FTS，无 migration | R-005/R-012/R-015；不访问文件系统 | SQLite 范围/类型统计、生成 client、浏览控件数字与可访问名称、Go/Web 回归 | `Post-MVP/1 optimization implemented` |
| `FR-SET-001`、`NFR-PERF-004` 资源并发设置 | `POST-MVP-1` | Frozen r8；实现与本地验证完成 | [CR-2026-018](../changes/CR-2026-018-explicit-resource-limits.md)，替代 [CR-2026-012](../changes/CR-2026-012-nas-resource-profiles.md) 的三档交互 | `internal/resourcecontrol` 拥有实例级数值限制/许可；`internal/settings` 持久化与编排；`internal/app` 启动恢复与 processor 组合；API/SQLite/Web adapter | 配置的扫描页签直接设置后台 1～4（共享 full scan/reconcile/media）和内容读取 1～16；默认 2/8 | migration 20 `background_concurrency`/`content_read_concurrency`，从 migration 16 profile 等价迁移；强 ETag/If-Match；durable jobs 与 active work 不变 | 复用 ADR-0001/0006/0007/0009；R-004/R-006/R-009/R-013/R-016；不提高既有硬上限 | 动态收缩/扩展、取消、SQLite CHECK/default、设置 HTTP、OpenAPI/生成、Web 数字表单与保存测试、Go/架构/生成检查 | `Post-MVP/1 revision 8 implemented` |
| `FR-SCN-010～014` 媒体库自动发现 | `POST-MVP-2` | Frozen r3；WCH-S0 Go；[WCH-S1 Contract Ready](../gates/POST-MVP-2/wch-s1-contract-ready.md) Go；[WCH-S2 Backend Evidence Ready](../gates/POST-MVP-2/wch-s2-backend-evidence-ready.md)发布 No-Go、有限授权 S3 本地 UI | [CR-2026-005](../changes/CR-2026-005-automatic-library-discovery.md) | `internal/scanner` 拥有事件合并、定向范围、清理资格和 content revision；`internal/files` 实现 Linux watch/安全重开；`internal/jobs` durable lease；SQLite/API adapter；Web catalog-state consumer 统一条件检查与 active query 刷新 | 自动发现设置/状态；新增、修改、删除、rename 的定向校准；媒体库状态与刷新；可见相关页面 5 秒 ETag 条件检查；不推送 | migration 12 专用 `catalog_reconcile_jobs` requested/claimed 水位；library/global `content_revision`；`cache_deletions`；settings 开关；完整扫描/定向任务按库互斥 | [FTR-SCN-001](../features/automatic-library-discovery.md)；ADR-0003/0009；[ADR-0011](../adr/0011-linux-inotify-hints-and-anchored-reconciliation.md)已接受 | R-002、R-003、R-005、R-013、R-016、R-019 | 后端 Linux/arm64 证据矩阵已完成；revision 3 增加 full-scan cache tombstone、自动发现状态/刷新和 5 秒可见页面条件检查。S2 发布仍需原生 amd64 | `WCH-S0 Done → S1 Done → S2 release No-Go / S3 local authorized → S4` |
| `FR-OPS-001～011`、`NFR-OPS-002～004` 后台任务、日志、版本与消息中心 | `POST-MVP-3` revision 1 | Scope Frozen；失败诊断/恢复、系统事件与 release-info 纵向切片 In Delivery；统一 parent task Pending | [CR-2026-008](../changes/CR-2026-008-task-center.md)、[CR-2026-015](../changes/CR-2026-015-operations-observability-and-updates.md)、[关于页直接展示更新内容](../changes/FIX-2026-08-04-inline-release-notes.md) | `internal/thumbnail` 拥有 derived failure/恢复及结构化 attempt；`internal/systemlog` 拥有有界脱敏系统事件；`internal/releaseinfo` 拥有版本比较、缓存和有界官方 Release 正文；scanner/jobs 保持原状态 owner；SQLite/API/Web adapter 与 Header 通知投影 | 配置合并；媒体库扫描记录/处理结果及行内尝试详情；完整扫描/失败恢复分离；系统日志；关于页内联版本正文/更新；可确认的聚合全局消息 | 复用 `scan_runs`/`media_jobs`/assets/libraries；migration 17 追加最多 5,000 条 `system_events`；migration 18 每 job 最多保留 10 条无原始 stderr 的 `media_job_attempts`；浏览器保存 completed acknowledgement 与 latest failure watermark；后续 `operation_runs` 仍由 parent task S1 冻结 | FTR-OPS-001、POST-MVP-3 scope r1；复用 ADR-0001/0006/0007/0009 | R-003、R-006、R-009、R-011、R-013、R-015、R-016、R-020 | Go/SQLite/HTTP tests 覆盖失败分页、结构化 attempts、transient/permanent 恢复、系统事件分级/脱敏/保留；OpenAPI/generated client、Release Markdown 安全文本投影、媒体库记录/行内诊断、系统日志/关于/消息消费者已接入，完整 evidence 与 parent run 容量仍待收敛 | `OPS diagnostics S1→S4；parent task S0→S4` |
| `FR-CUR-001～007`、`NFR-CUR-001～003` 收藏与手动标签 | `POST-MVP-4` revision 1 | Scope Frozen；[CUR-S3 Consumer UI Ready](../gates/POST-MVP-4/cur-s3-consumer-ui-ready.md) Go；[CUR-S4](../gates/POST-MVP-4/cur-s4-integrated-slice-current.md) Conditional Go | [CR-2026-020](../changes/CR-2026-020-favorites-and-tags.md) | `internal/curation` 规范化、状态、revision、cursor；SQLite/API adapter；`web/src/features/curation` consumer | 收藏切换/列表；标签词表、单资产替换和标签媒体页；全部/单库范围 | migration 21 `curation_state`、`asset_favorites`、`tags`、`asset_tags`；复合 FK cascade、ETag 与稳定 keyset | [FTR-CUR-001](../features/favorites-and-tags.md)；复用 ADR-0001/0006/0007 | R-005、R-012、R-015、R-016、R-023 | Unicode/幂等/precondition/cursor/cascade、OpenAPI/generated client、HTTP/CSRF、真实 composition、原媒体不变、TS/前端行为/Storybook、Docker E2E 与 dependency audit 通过；目标浏览器/真实触摸人工签署待解除 | `CUR-S0 Done → S1 Done → S2 Go → S3 Go → S4 Conditional Go` |
| `FR-INT-001～020`、`NFR-INT-001～010` 本地智能媒体发现 | `POST-MVP-5` r2 | Frozen r2；S2A/S2B/S2C Backend Ready；[INT-S3 Consumer/UI Ready](../gates/POST-MVP-5/int-s3-consumer-ui-ready.md)；[INT-S4 Current](../gates/POST-MVP-5/int-s4-current.md) 3/11 / Release No-Go | [CR-2026-021](../changes/CR-2026-021-intelligent-media-discovery.md)；[CR-2026-022](../changes/CR-2026-022-s2-backend-release-gate-separation.md)；[CR-2026-023](../changes/CR-2026-023-s3-offline-model-ui-contract-alignment.md)；ADR-0014/0015 | 后端：`internal/semantic`、`internal/aimodel`、`internal/face` 与 inference/media/SQLite/API/app adapters；消费者：`web/src/features/ai`、`web/src/features/search`；共享媒体集合/预览/查看器 | 离线模型管理、图片/视频语义搜索、标签审核、匿名组/人物/逐脸人工复核；禁止身份推断、自动标签、云/GPU/新部署单元 | migrations 22～35；生成客户端；URL/query key；浏览器只保留偏好、瞬时选择和 opaque operation ID | revision 2 scope；A+B/C+D+E capability；model package；`api/openapi.yaml`；data/security/deployment/testing；ADR-0013/0014/0015 | R-024～R-030 全部适用且开放 | S3 消费与可访问性矩阵完成；S4 已关闭 SQLite 人物恢复/派生重建、损坏/盘满失败关闭、原媒体只读/path attack 和用户/运维文档。仍缺最终模型、governed quality、native paired image、联合容量、最终浏览器/真机、供应链与发布签署 | `Frozen r2 A～E → S2 Backend Ready → S3 Consumer/UI Ready → S4 3/11 / Release No-Go` |
| `FR-UI-001～007` 界面与可访问性 | `MVP-2026-07-23` | Frozen r1；[S4-005 搜索预览 Done](../gates/MVP-2026-07-23/s4-frontend-search-preview.md)；[S4-006 Viewer Done](../gates/MVP-2026-07-23/s4-frontend-media-viewer.md)；[S4-007 Media Strategy Done](../gates/MVP-2026-07-23/s4-frontend-media-strategy.md)；[S4-008 Media Matrix Done](../gates/MVP-2026-07-23/s4-frontend-media-matrix.md)；[Stage 4 Integrated Done](../gates/MVP-2026-07-23/s4-search-media-integrated-done.md) | [CR-2026-001](../changes/CR-2026-001-non-modal-media-preview.md)；[CR-2026-003](../changes/CR-2026-003-brand-identity.md)；[redesign 应用壳恢复](../changes/FIX-2026-07-29-redesign-shell-restoration.md)；[原型品牌对齐](../changes/FIX-2026-07-30-prototype-brand-alignment.md)；[Header 快速语言切换](../changes/FIX-2026-07-31-header-locale-toggle.md) | `web` 的 auth/libraries/browse/search/media/settings feature；统一 API client；共享 `BrandMark`、`LocaleToggle`、`AsyncState`、`InlineStatus`、`FormField`、`Button`、`Toast`、`MediaCollection`、`MediaPreview`、`MediaViewer`、`MediaAvailabilityState` 与 `useMediaPreviewController` | 欢迎/认证、设置、扫描状态、浏览/搜索/预览/查看器；统一品牌入口；桌面固定侧栏和底部上下文导航、移动抽屉、中英与主题；设置与 Header 双入口语言切换；UIF-316 八类状态工作台矩阵；UIF-317 locale/theme/四档/input 矩阵 | 服务端状态不复制为独立事实；URL 保存导航状态，`settings` 保存受控偏好；`LocaleProvider` 唯一拥有语言偏好和即时切换；固定预览保存当前媒体快照，查看器瞬时状态只保存已载入 ID 序列和安全返回位置；品牌 SVG 是无业务状态的静态资产 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[ADR-0007](../adr/0007-shared-frontend-system.md)、[用户流程](../user-flows.md)、[界面设计](../ui-design.md)、[品牌标识](../branding.md) | R-005、R-010、R-015、R-016 | loading/empty/offline/error/conflict/cancel/pending/success 由唯一共享组件语义和真实领域状态覆盖；工作台接入真实 locale/provider 和主题 token，并保存 2×2×4 确定性视觉矩阵；真实设置页笛卡尔 E2E 检查 lang/theme/overflow/axe/reduced-motion；共享 locale toggle、应用壳和设置页回归固定持久化、按钮顺序、`html[lang]` 与双入口同步；既有纵向链和媒体矩阵覆盖键盘焦点返回、Pixel 5 触摸、三引擎与 forced-colors；其余证据包括预览/查看器、媒体降级、原型同视口对照、虚拟化和关键 E2E | 1（认证 UI/品牌原语）、3、4（Integrated Done）；5（发布浏览器/真机加固） |

### 2026-08-12 媒体处理例行修复

- [媒体处理韧性与诊断纠偏](../changes/FIX-2026-08-12-media-processing-resilience.md)归属
  `FR-MED-001/004/005/009～011`、`FR-OPS-008`，复用 `S3-004/006/007` 与 `VSP-301`，
  并更新 `R-006/R-018` 的缓解证据。真实格式核验、JPEG 独立 shrink-on-load 预算、稳定诊断、
  工具有效产物判定、故事板邻帧/降级预算和 external libdav1d 均留在既有 capability owner。
  该修复不改变 API/OpenAPI、schema、已发布 migration、transform version、事务或模块边界；
  部署后通过现有“补齐缺失”有界重排失败/缺失派生项，真实损坏仍保持永久失败。
- [JPEG 有界容错与 MPEG-TS 派生兼容](../changes/FIX-2026-08-12-tolerant-jpeg-mpegts-derivation.md)
  继续归属 `FR-MED-001/004～006/009～011`、`FR-OPS-008` 和同一 capability owner。只有
  ≤100 MP 的真实 JPEG 命中窄 allowlist 时可单次容错，成功为 ready/succeeded 并仅写入
  内部 attempt warning；MPEG-TS 只为既有扩展名候选提供 derive-only 兼容并保持 playback
  `unknown`。无新 API 字段、schema、migration 或 transform bump；`.ts` 支持、浏览器直放、
  remux/转码和可见 degraded 状态均不在范围。证据覆盖窄 JPEG 正反例、attempt 本轮语义、
  最终镜像 MPEG-TS 派生与 `foliopath-ffmpeg 7.1.5-4` 双架构供应链门槛。

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
