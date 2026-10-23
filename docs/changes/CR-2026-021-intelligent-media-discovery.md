# CR-2026-021：本地智能媒体发现

## 状态

- 状态：Approved for frozen revision 2（Slice A～E）；A+B S2A No-Go，C/D/E S1 extension pending
- 变更等级：C3（新用户能力、native 模型运行时、敏感人脸数据、派生向量持久化）
- 目标版本：`POST-MVP-5` revision 2 Frozen
- 日期：2026-08-18；模型获取补充：2026-08-24
- 分支：`aifeature`
- Scope：[POST-MVP-5 scope revision 2](../releases/POST-MVP-5-scope-r2.md)
- 被替代范围：[POST-MVP-5 scope revision 1](../releases/POST-MVP-5-scope.md)
- 历史草案：[POST-MVP-5 scope manifest proposal draft 1](../releases/POST-MVP-5-scope-proposal.md)

## 变更事件与负责人

用户要求在 MVP 冻结后独立评估 AI 标签建议、视频内容搜索、图片语义搜索和人脸归类。随后明确：

- 内容以人物套图为主，但不要求系统识别 Coser、模特或角色姓名；
- 只做人脸检测/相似归类，人物名称由用户建立；
- 顺序为后台先匿名聚类，再由用户建立人物库；匿名组可合并到人物，单个人脸也可归入人物；
- 不照抄 Immich 代码，需要独立、如实说明困难的可行方案。
- 模型管理需要支持界面下载、选择兼容版本，并照顾不能访问境外模型站点的部署：可扫描只读
  `/models` 映射，允许托管复制或在严格校验后直接读取。

产品负责人：产品用户。技术、安全、发布和各 capability owner 在 S0 前分配。

## 用户价值

- 跨目录用自然语言找人物构图、服装、场景和视觉概念；
- 用有限标签建议降低人工标注成本，但不牺牲手动标签可信度；
- 先聚类后命名，降低以人物为主的套图整理成本；
- 保持目录、原媒体和普通搜索在 AI 不可用时继续工作。

## 提议范围

以下是最初完整提议。2026-08-27 冻结的 revision 1 只纳入模型基础与图片语义搜索；标签建议、视频
语义和人脸人物库不在 revision 1，必须分别通过后续 scope revision 才能开工。

- 图片中英文语义搜索；
- 视频既有故事板代表帧的近似语义搜索；
- 从受控词表/现有标签计算建议，用户确认后转成人工标签；
- 图片人脸检测、embedding、匿名聚类、人物创建/命名、整组或单 face 归类、合并/拆分/排除；
- 按媒体库启停、覆盖率、任务状态、重建、清除和模型 generation。
- 从签名发行清单/部署镜像下载审核模型包，或从固定 `/models:ro` 离线发现；默认复制到
  `/app/data/models`，可选择固定哈希直接读取；只允许兼容模型版本切换。

明确不含 OCR、身份/姓名/角色自动识别、敏感属性推断、视频人脸追踪、云推理、自动接受标签、
自由 caption、内容审核、GPU、独立 worker、第二个部署单元、任意模型市场或任意下载 URL。

## 架构与发布影响

- 提议新增 `internal/semantic`、`internal/face` 与本地 inference adapter，但保持模块化单体。
- 提议在 SQLite 中增加可重建 embedding/face 派生数据和不可重建的人物人工关系；外部向量索引仅作缓存。
- 镜像可能增加 native inference runtime 与模型权重，必须重做 amd64/arm64、SBOM、许可证、漏洞、
  内存、备份/恢复和故障关闭验证。
- 人脸 embedding 按高敏感数据处理，改变隐私和清除/备份审查面；不改变 `/library` 只读边界。
- 新增可选管理员触发的模型下载出站边界和可选 `/models:ro` 模型来源；它不是媒体 mount，也不允许
  API 浏览路径。下载源、重定向、凭据、续传、临时空间、镜像运营和 SSRF 必须进入安全/发布 Gate。
- 国内镜像只有在项目或部署者真正提供签名清单约束的可达基础设施时才是能力；否则只承诺离线目录。
- 当前 OpenAPI、migration、产品需求、用户流程、UI 设计和部署文档均不在本记录中修改；S1 只有在
  scope 冻结和 S0 Go 后才更新这些权威合同。

## 成本与排期判断

粗估为单名熟悉现有代码库的工程师 5～8 个月，不含模型训练；其中 S0 spike 约 4～8 周。主要变量是
双架构 native 集成、向量索引、真实数据评测、人物纠错 UX 和发布供应链。若只有 1～2 个月，不应承诺
完整四项；可以只选择“图片语义搜索”作为更小版本，不能把剩余能力伪装成同一 MVP。

## 风险与回退

- 资源/吞吐不达标：降级为手动触发、夜间处理或只交付图片语义搜索；仍不达标则 No-Go。
- 向量索引不稳定：10 万档 exact scan 若达标则不用 ANN；否则停止该切片，不引入未经审查服务。
- 人脸误合并过高：降级到小组/相似 pair 建议，不提供整组创建人物。
- 模型权重许可不清：更换模型或不分发该能力；不以“仓库可下载”代替许可。
- 下载源或映射目录不可信：只接受 allowlist manifest/hash；托管复制失败保留现用模型；直接来源
  缺失/变化则模型 unavailable，不删除现有索引和人工状态，不自动选取其他模型。
- AI 故障：禁用该库、删除可重建派生结果，保留人物人工状态和全部核心浏览能力。

风险见 `R-024～R-030`；任务与证据见 [执行清单](../features/intelligent-media-discovery-task-list.md)。

## 批准结果

1. 产品用户在 2026-08-27 对推荐方案回复“继续”，据此接受 A+B 首版、C/D/E 后续独立切片、实例级
   人物默认、按库独立开关、派生数据默认清除、`/models:ro` 基线和 16/32 工程周停损。
2. ADR-0013 只对 A+B 接受 ORT C API、SigLIP 1 float16-internal、SQLite float16 exact 和单进程
   CPU-only 候选；最终质量、双架构和供应链仍由 S2/Release Gate 持有。
3. revision 1 不包含在线/国内镜像，因此不需要伪造运营 owner；新增在线源必须创建新 scope revision。
4. INT-S0 只对 A+B 为 Go并授权 S1 合同设计；生产 OpenAPI/migration/backend 必须等待 S1 接受，UI
   必须等待 Backend Evidence Ready。

## Revision 2 批准补充（2026-08-29）

1. 产品用户在要求“S2 都完成再汇报”后，对 A～E 是否全部纳入的确认问题明确回复“都纳入”；据此
   冻结 revision 2，并替代仅 A+B 的 revision 1。
2. 显式接受 scope-budget exception：S0 后工程上限由 16 周恢复为完整范围最多 32 单人工程周；没有
   用延后既有 A+B 换预算，也不允许削弱只读、安全、隐私、质量、双架构或供应链 Gate。
3. A+B 保持当前 S2A No-Go。C/D/E 只获得 S1 extension 合同授权；必须先冻结需求、数据、OpenAPI、
   安全/隐私、部署、测试和 traceability 并通过独立 Contract Ready，才能开始对应生产后端。
4. E 的产品纳入不构成隐私/合规批准。合法真实 face ground truth、core precision ≥99.5%、可商业分发
   模型/runtime 和具名隐私签署缺一时保持 No-Go；不得用合成数据或用户自行下载绕过。
