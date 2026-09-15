# FolioPath 文档

最新（09-16）：[远程人脸激活调度](evidence/int-001/face-deployment-worker-2026-09-16.md)，独立运行时分支及 service 人脸模型迁移已实现；尚非主进程真实激活。

最新：[人脸安全图片适配](evidence/int-001/face-runtime-adapter-2026-09-15.md)，四格式真实离线零人脸及 face/faceonnx 回归通过；代次准入与主进程组合仍待完成。

最新：[Go 人脸服务传输](evidence/int-001/face-go-transport-2026-09-15.md)，共享并发但隔离模型身份，本地 Docker 合同与真实零人脸请求通过；主应用人脸任务仍待组合。

最新：[独立人脸服务合同与验证](evidence/int-001/face-service-contract-2026-09-15.md)，可选 buffalo_l 私有接口及本地 Docker 零人脸推理通过；主应用组合、非空识别和人物验收仍未完成。

最新：[跨库审核清理隔离](evidence/int-001/tag-review-cross-library-2026-09-15.md)，混合请求失效、另一库审核和正式标签保留的 SQLite 回归通过。

最新：[新词条编码请求取消恢复](evidence/int-001/tag-text-request-cancellation-2026-09-15.md)，本地实际服务等待取消及重试通过，明确非 ONNX 内核中断证据。

最新：[标签非空评分与提交取消](evidence/int-001/tag-job-processing-cancellation-2026-09-15.md)，合成向量验证提交边界，未扩展为真实推理验收。

最新：[标签生成实际等待取消](evidence/int-001/tag-job-runtime-cancellation-2026-09-15.md)，非空任务取消及恢复通过，正式标签保留。

最新：[标签生成取消边界](evidence/int-001/tag-job-cancellation-2026-09-15.md)，准入取消与终态优先修复通过，非空实际运行仍待复验。

最新：[非空自动标签审核闭环](evidence/int-001/tag-review-chain-2026-09-15.md)，实际模型建议→接受→清理成功，正式标签保留。

最新：[自动标签生成首次接入](evidence/int-001/tag-generation-wiring-2026-09-15.md)，实际任务成功 1/1；零建议不等于非空审核闭环完成。

最新缺口：[自动标签生成未接入主应用](evidence/int-001/tag-generation-gap-2026-09-15.md)，本地正式任务接口 404；审核清理测试不能代表自动标签功能完成。

最新：[标签审核清理实际镜像验收](evidence/int-001/tag-clear-runtime-2026-09-15.md)，新本地镜像取消及重试通过，非空批次仍属组合测试证据。

最新：[标签审核清理批次取消](evidence/int-001/tag-clear-batch-2026-09-15.md)，非空批次取消收尾及正式标签保留通过。

最新：[标签审核清理准入取消](evidence/int-001/tag-clear-admission-2026-09-15.md)，用户取消有界收尾、停机保留恢复；非空批次交错待补。

最新：[取消修复运行镜像回归](evidence/int-001/cancellation-runtime-refresh-2026-09-15.md)，图片/视频/扫描/清理四类本地实际链路通过。

最新：[清理批次取消修复](evidence/int-001/clear-batch-cancellation-2026-09-15.md)，部分/最终删除批次后取消均通过；不回滚此前已删除的派生数据。

最新：[图片处理器终态组合回归](evidence/int-001/image-processor-terminal-2026-09-15.md)，已提交向量保留、编码失败后取消通过；清理终态交错仍待修复。

最新：[图片终态取消优先](evidence/int-001/image-terminal-cancellation-2026-09-15.md)，修复后续成功/失败收尾对已接受取消的错误处理，Docker 回归通过。

最新：[扫描与清理实际混合取消验收](evidence/int-001/scan-clear-admission-runtime-2026-09-15.md)，新本地 Docker 镜像已验证取消及恢复，批次交错仍待覆盖。

最新：[扫描准入取消收尾](evidence/int-001/scan-admission-cancellation-2026-09-15.md)，复用扫描既有取消规则，保留可靠索引；实际混合场景仍待验证。

最新：[语义清理准入取消](evidence/int-001/clear-admission-cancellation-2026-09-15.md)，区分用户取消与停机，扫描未改动。

最新：[图片等待资源取消实际验收](evidence/int-001/image-admission-runtime-2026-09-15.md)，混合脚本支持图片/视频双向取消。

最新：[图片回填准入取消收尾](evidence/int-001/image-admission-cancellation-2026-09-15.md)，复用通知及原 CAS，扫描状态机未修改。

最新：[实际混合任务等待期取消](evidence/int-001/mixed-admission-runtime-2026-09-15.md)，真实图片运行、视频取消和后续恢复通过。

最新：[视频资源等待期取消收尾](evidence/int-001/video-admission-cancellation-2026-09-15.md)，包含失败复现、修复及其他任务类型边界。

最新：[后台资源取消准入回归](evidence/int-001/background-admission-2026-09-15.md)，共享额度与取消检查通过，实际混合任务和等待期取消收尾待验收。

最新：[旧任务记录实际浏览器验收](evidence/int-001/missing-task-records-runtime-2026-09-15.md)，本地镜像、中英提示和刷新持久化通过。

最新：[旧任务记录提示修复](evidence/int-001/missing-task-records-2026-09-15.md)，明确不存在与临时失败分开，移除仅影响浏览器记录。

最新：[浏览器多库差异进度验收](evidence/int-001/multilib-browser-2026-09-15.md)，含切换/刷新及旧任务残留待修项。

最新：[多库视频调度与取消隔离](evidence/int-001/multilib-cancellation-2026-09-15.md)，含真实 Docker API、重试和按库检索验证。

最新：[视频运行中取消实际镜像验收](evidence/int-001/video-cancellation-runtime-2026-09-15.md)，两种取消路径及后续补齐已在本地浏览器验证。

最新：[视频取消收尾与跨库隔离回归](evidence/int-001/video-cancellation-2026-09-15.md)，修复取消与完成/失败竞态，保留停机恢复语义。

最新：[视频操作按钮与实际浏览器闭环](evidence/int-001/video-actions-ui-2026-09-15.md)，含补齐、确认重建和覆盖率真实性。

最新：[图片未记录到真实回填的实际镜像验收](evidence/int-001/image-unrecorded-runtime-2026-09-15.md)，新数据卷中验证未知→真实 1/1，图片与视频状态分离。

最新：[图片未记录状态](evidence/int-001/image-unrecorded-2026-09-15.md)，与已验收的离线保留进度区分。

最新：[图片模型离线不清零覆盖率](evidence/int-001/image-offline-coverage-2026-09-15.md)，包含实际离线/恢复和拒绝新任务验证。

最新：[视频状态实际镜像及自动恢复验收](evidence/int-001/video-status-runtime-2026-09-15.md)，含自动刷新修复及图片离线 0/0 的未完成项。

最新：[视频只读状态与界面接通](evidence/int-001/video-status-ui-2026-09-15.md)，含未知/离线/降级语义与本地 Docker 回归边界。

最新：[独立服务开关与图片进度范围修复](evidence/int-001/settings-service-toggle-2026-09-15.md)，含 Docker 回归与尚未接通的视频 UI 边界。

最新：[视频删除真实闭环和设置提示复核](evidence/int-001/video-removal-app-ui-2026-09-15.md)，包含人脸路由未接入的真实边界。

最新：[视频移除后的保守覆盖率](evidence/int-001/video-removal-2026-09-15.md)，明确区分待处理重算与逐资产失败归属。

最新：[视频源变化/新增候选](evidence/int-001/video-invalidation-2026-09-15.md)，删除与精确失败归属另列未完成。

最新：[图片新增/移除覆盖率](evidence/int-001/semantic-membership-2026-09-15.md)，与视频尚未完成的计数更新区分记录。

最新：[扫描后图片覆盖率修复](evidence/int-001/semantic-invalidation-2026-09-15.md)，含本地 Docker 过期/恢复闭环及未覆盖边界。

最新：[主进程图片/视频链路](evidence/int-001/app-chain-2026-09-15.md)，包含实际发现的路由缺陷、测试环境缺项及覆盖率待修项。

2026-09-15 最新：[主进程部署配置与离线恢复](evidence/int-001/deployment-startup-2026-09-15.md)。run.go 已接入；下列独立子片的“尚未接入”为当时状态，不再代表当前组合进度。发行、完整产品链路仍待验收。

2026-09-15：[共享激活 worker 与 app 适配验证](evidence/int-001/deployment-worker-adapter-2026-09-15.md)，独立适配通过，正式启动配置尚未接入。

2026-09-15：[服务代次原子激活](evidence/int-001/deployment-activation-2026-09-15.md)事务及真实推理链路已验证；主应用调度未接入。

2026-09-15：[service 模型记录及迁移验证](evidence/int-001/service-model-record-2026-09-15.md)，仅记录与隔离合同完成，主应用激活仍未接入。

2026-09-15：[视频流式编码与真实四帧持久化检索](evidence/int-001/remote-video-stream-2026-09-15.md)已验证；主应用配置与服务代次激活仍待完成。

方向已接受：[ADR-0020 专用 AI 镜像](adr/0020-ai-runtime-delivery-proposal.md)，[内部服务合同](architecture/independent-ml-service-contract.md)。

2026-09-15：[真实离线 AI 回填、数据库重开与检索](evidence/int-001/independent-ml-persisted-chain-2026-09-15.md)已在本地 Docker 通过；生产部署准入仍待完成。

2026-09-15：[旧运行时空间隔离修补](evidence/int-001/independent-ml-legacy-boundary-2026-09-15.md)，本地 Docker 回归通过，独立 AI 主应用接入仍未完成。

SQLite 全包竞态已在本地 Docker 完整复验通过（首次超时后改为 15 分钟预算）；[完整记录](evidence/int-001/local-docker-regression-2026-09-15.md)。主应用独立 AI 接入仍待完成。

本地 Docker 测试入口与本轮结果：[2026-09-15 Docker 回归](evidence/int-001/local-docker-regression-2026-09-15.md)。

2026-09-15：[流式回填与 SQLite 事务提交](evidence/int-001/independent-ml-stream-backfill-2026-09-15.md)已验证；app 仍未切换独立 AI。

独立 AI：[查询/图片共同代次校验](evidence/int-001/independent-ml-bound-encoder-2026-09-14.md)已通过单元与真实 SQLite 组合测试，app/worker 尚未切换。

独立 AI 数据增量：[空间绑定持久化合同](architecture/independent-ml-generation-contract.md)与 migration 38 已实现；不代表新查询/回填已接通。

2026-09-14：[四格式安全编码到真实离线 ML 链路](evidence/int-001/independent-ml-encoded-chain-2026-09-14.md)通过，主应用代次/查询/回填仍未接入。

独立 AI 图片入口：[流式输入与编码合同](architecture/independent-ml-image-contract.md)。独立 imagevips 适配不代表查询/回填或代次迁移完成。

2026-09-14：[独立 ML 缓存/容器复核](evidence/int-001/independent-ml-cache-audit-2026-09-14.md)，仅本地基线和隔离推理，不代表生产身份准入。

独立 AI 增量：[代次兼容性合同](architecture/independent-ml-generation-contract.md)已实现纯检查及合成测试；模型事实核验和生产接入仍待完成。

2026-09-14 正式智能设置已按方案一重排既有接口消费者；[视觉与测试记录](../design-qa.md)。不代表新独立 AI 服务已接入。

2026-09-14 AI 新方向：[ADR-0019](adr/0019-independent-machine-learning-service.md) /
[CR-2026-030](changes/CR-2026-030-independent-machine-learning-service.md)。独立服务本地探针已验证；
生产接入未完成。模型商店旧路线暂停，历史交付状态不能当成新架构验收。

内部通信子片：[独立 ML 传输合同](architecture/independent-ml-transport-contract.md)，
Go adapter 位于 `internal/inference/remoteml`；尚未接入主应用代次。
验证记录：[Go 传输子片](evidence/int-001/independent-ml-go-transport-2026-09-14.md)。
选定的 AI 设置[方案一交互原型](../spikes/int001-ui-settings/README.md)已本地运行，
与生产代码导入图隔离；参见其[视觉检查](../spikes/int001-ui-settings/design-qa.md)。

最新模型代理界面扩展：[CR-2026-029](changes/CR-2026-029-admin-model-proxy-settings.md)；属于 POST-MVP-5，不代表已部署。

本目录保存 FolioPath 的产品基线、架构约束、交付计划和验收证据。文档按用途分层：
先从本页找到当前权威入口，再进入专题索引；不要从文件数量或日期推断文档是否仍然有效。

当前状态：MVP 与后续视频预览、自动发现、运维可观测性、收藏/标签切片已有生产实现；各版本
是否达到稳定发布仍分别由对应 Gate 决定，不能由实例部署替代。MVP 冻结范围是
[MVP scope revision 4](releases/MVP-2026-07-23-scope-r4.md)，最新已实现公共合同基线为
[`POST-MVP-4`](releases/POST-MVP-4-scope.md)。本轮综合更新见
[2026-08-13 更新说明草案](releases/2026-08-13-integrated-update-notes.md)。

## 首先阅读

2026-09-08：[模型下载与智能界面扩展 CR-2026-026](changes/CR-2026-026-ai-model-download-and-ui.md)
实现了部署目录驱动的界面下载、后台进度/取消/重试与智能概览。默认下载源仍未配置，当前发行镜像
仍无 AI 推理组件；该扩展不把历史 S4 未验证项转为通过。配置见[模型下载目录](deployment.md#模型下载目录)。

| 目的 | 当前入口 | 说明 |
| --- | --- | --- |
| 了解产品 | [项目 README](../README.md) | 面向用户的能力、部署和限制 |
| 确认产品范围 | [产品需求](product-requirements.md) | 需求语义；冻结范围以 scope manifest 为准 |
| 了解系统结构 | [架构索引](architecture/README.md) | 模块、运行视图、前端边界、追踪和适配度检查 |
| 查看 API | [OpenAPI](../api/openapi.yaml) | 已实现 API 的权威合同 |
| 设计或修改界面 | [界面设计规范](ui-design.md) | 导航、组件、响应式、可访问性和动效 |
| 部署或运维 | [部署](deployment.md) | 容器、卷、权限、备份、恢复和升级 |
| 查看用户更新 | [更新日志](../CHANGELOG.md) | 自动维护的用户可见新增、改进、修复和注意事项 |
| 查看当前工作 | [开发任务清单](task-list.md) | 当前进度、阻断和下一步 |
| 判断能否发布 | [版本与 readiness](releases/README.md) | scope、候选状态和发布说明 |

## 按文档类型浏览

| 类型 | 索引 | 用途 |
| --- | --- | --- |
| 架构 | [architecture/](architecture/README.md) | 当前系统结构、模块所有权和治理 |
| ADR | [adr/](adr/README.md) | 只追加的架构决策及替代关系 |
| Feature | [features/](features/README.md) | 独立用户能力的范围、合同和交付阶段 |
| 变更记录 | [changes/](changes/README.md) | Change Record 与已批准 slice 内修复 |
| Gate | [gates/](gates/README.md) | Architecture、Contract、Backend、UI 和 Release 判断 |
| 版本 | [releases/](releases/README.md) | 冻结 scope、readiness 和发布说明 |
| 验收证据 | [evidence/](evidence/README.md) | 可复核的视觉、浏览器和容量证据 |
| Spike | [spikes/](spikes/README.md) | 可行性实验；不能代替生产 Gate |

当前另有一套不影响冻结 MVP 的 [FTR-INT-001 本地智能媒体发现提案](features/intelligent-media-discovery.md)：
已形成[技术架构](architecture/intelligent-media-discovery.md)、[spike 计划](spikes/int-001-ai-feasibility.md)
和[任务清单](features/intelligent-media-discovery-task-list.md)，并补充审核下载源与 `/models:ro` 离线
模型获取方案；`POST-MVP-5` [revision 2](releases/POST-MVP-5-scope-r2.md) 已冻结 A～E 完整范围；
INT-S0 与 [INT-S1](gates/POST-MVP-5/int-s1-contract-ready.md)仍只对 A+B 为 Go；
[INT-S2A](gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)、
[INT-S2B](gates/POST-MVP-5/int-s2b-backend-evidence-ready.md) 与
[INT-S2C](gates/POST-MVP-5/int-s2c-privacy-ready.md) 均为 Backend Ready / Release No-Go；
[INT-S3](gates/POST-MVP-5/int-s3-consumer-ui-ready.md) 已为 Consumer/UI Ready / Release No-Go；
[INT-S4 current](gates/POST-MVP-5/int-s4-current.md) 已验证完成恢复、原件只读与发布文档 3/11；产品用户按
[CR-2026-025](changes/CR-2026-025-int-s4-unverified-closure.md)将其余 8/11 未验证关闭，S4 终态为
Closed Unverified / Release No-Go。按
[CR-2026-022](changes/CR-2026-022-s2-backend-release-gate-separation.md)，最终真实质量、native 双架构、
联合容量、供应链与隐私/合规签署由 S4 持有。国内镜像仍没有承诺；S4 Go 前 reviewed catalog、face
composition 与发行 UI 继续失败关闭。

## 当前权威文档

### 产品与体验

- [产品需求](product-requirements.md)
- [需求确认记录](requirements-checklist.md)
- [用户流程](user-flows.md)
- [界面设计规范](ui-design.md)
- [品牌标识规范](branding.md)
- [路线图](roadmap.md)
- [术语表](glossary.md)

### 工程与运行

- [系统架构](architecture.md)
- [目录与依赖约束](project-structure.md)
- [数据模型](data-model.md)
- [API 设计说明](api-design.md)：解释资源边界；wire 合同以 OpenAPI 为准
- [安全模型](security.md)
- [部署](deployment.md)
- [测试策略](testing-strategy.md)
- [供应链与许可证审查](supply-chain-review.md)

### 计划与风险

- [开发任务清单](task-list.md)
- [后端开发清单](backend-task-list.md)
- [前端开发清单](frontend-task-list.md)
- [开发就绪评审](development-readiness.md)
- [可行性研究](feasibility-study.md)
- [风险登记](risk-register.md)

已完成计划仍可作为交付记录，但不应优先于当前产品、架构、OpenAPI、migration 或 Gate
事实。计划状态发生变化时，应更新计划本身或对应专题索引，避免创建新的平行总清单。

## 历史文件如何处理

- scope revision、ADR、Change Record、Gate 和发布证据是审计历史，采用只追加策略。
- 被替代的文件保留原路径，并在文件头或所属索引中指向当前替代项。
- 已完成任务清单保留任务与证据映射；后续工作进入新的 feature 或版本清单。
- `docs/evidence/` 和 `web/qa/` 中相同内容可能代表不同验收状态，不能仅按哈希去重。
- 只有未被任何当前或历史记录引用、且不承担审计含义的生成物或草图才可删除。

## 状态含义

- **已接受 / 已确认**：当前实现必须遵守。
- **冻结**：只追加新 revision，不原地改写范围。
- **Go**：适用检查已经实际执行并通过。
- **Conditional Go**：只允许记录中明确列出的有限下一步。
- **No-Go / Pending**：不得宣称切片或版本完成。
- **提案 / 未来**：不授权进入当前生产实现。
- **已被替代**：保留历史，但实现应遵循其替代文件。

发生冲突时，依次检查冻结 scope、已接受 ADR、OpenAPI/migration、当前 Gate 和产品需求；
不要静默选择其中一份，也不要通过删除旧记录消除冲突。

## 变更同步表

| 变更类型 | 至少同步更新 |
| --- | --- |
| 用户可见功能、格式或配置 | `README.md`、产品需求、相关用户流程、Change Record |
| 媒体库路径或删除语义 | 产品需求、架构、数据模型、安全模型，必要时新增 ADR |
| API 路径、字段或错误 | `api/openapi.yaml`、生成客户端、相关合同测试 |
| 页面、组件、响应式或动效 | 界面设计规范、相关用户流程、视觉证据 |
| 数据表、索引或迁移语义 | 数据模型、migration，必要时新增 ADR |
| Docker、权限、备份或升级 | README、部署、安全模型 |
| 测试门槛或支持矩阵 | 测试策略、readiness、README |
| 可行性假设、spike 或风险状态 | 可行性研究、风险登记、开发就绪评审 |
