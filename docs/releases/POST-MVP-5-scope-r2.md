# POST-MVP-5 Scope Manifest — Revision 2

- 状态：**Frozen scope / S1R2 contract accepted**
- 日期：2026-08-29
- 替代：[revision 1](POST-MVP-5-scope.md)
- Change Record：[CR-2026-021](../changes/CR-2026-021-intelligent-media-discovery.md)
- Feature：[FTR-INT-001](../features/intelligent-media-discovery.md)
- 当前 Gate：A+B 的 `INT-S2A` 保持 **No-Go**；C+D 的 S2B 已授权实现但 Backend Gate No-Go；E 的
  S2C 仍因隐私/合法数据/候选许可准入为 No-Go
- 需求：`FR-INT-001～020`、`NFR-INT-001～010`
- 产品 Owner：产品用户（2026-08-29 明确决定“都纳入”）
- 工程预算：接受 scope-budget exception，将 S0 后上限由 revision 1 的 16 单人工程周恢复为完整范围
  32 单人工程周；不以删除安全、隐私、质量、双架构或供应链 Gate 换取范围

## 变更理由与成本例外

Revision 1 为控制风险只冻结 A 模型基础与 B 图片语义搜索。产品用户现明确要求把任务清单中的 S2A、
S2B 和 S2C 全部纳入，即同时纳入原提案的 C 受控标签建议、D 视频代表帧语义搜索和 E 匿名人脸聚类/
人物库。

本次没有延期同等成本的既有 A+B 项，而是显式接受最多 32 单人工程周的 scope-budget exception。
安全不变量不参与交换：任一切片未达到自己的合同、合法数据、隐私、质量、native 双架构、容量或供应链
Gate 时，该切片保持 No-Go；“已纳入范围”不等于“获准提前写生产代码”或“必须降低门槛上线”。

## 冻结范围

### A：模型与派生任务基础

继承 revision 1：按库默认关闭；固定 `/models:ro` 离线包；managed/direct 严格校验；generation、任务、
覆盖率、取消、重建、清除和 availability；派生状态只写 `/app/data`，原媒体只读。在线/国内镜像仍不在
范围，除非另有具名运营 owner、签名 catalog 和撤回/轮换合同。

### B：图片语义搜索

继承 revision 1：JPEG/PNG/WebP/GIF 图片的中英文自然语言查询、库/目录/递归/类型 scope、稳定 cursor、
generation/coverage 冲突和失败关闭；查询、向量、路径及 native 原始错误不进入 API、日志或诊断。

### C：受控 AI 标签建议

1. 候选只能来自管理员受控词表或现有人工标签；不生成自由文本标签。
2. suggestion 是可重建派生状态，与人工标签分表、分 revision、分权限；模型不得直接写人工标签。
3. 接受建议必须调用现有 `internal/curation` owner，并遵守 tag revision、上限、幂等和 cascade。
4. 支持逐条接受、拒绝和批量复核；不自动接受、不在线训练、不因模型升级覆盖人工决定。
5. 依赖与图片相同的 active semantic generation；generation/source/词表变化使旧建议失效。

### D：视频代表帧语义搜索

1. 只消费现有 `internal/media` / `internal/thumbnail` 生成的 4/10 格故事板，不建立第二套 FFmpeg owner。
2. 每帧 embedding 与 source/storyboard/semantic generation 绑定；部分失败、降级和重建可观察且可取消。
3. 结果返回视频资产与命中代表帧信息；排名策略必须经合法 100 视频集冻结。
4. 不承诺音频、对白、字幕、动作时序、视频人脸追踪或精确时间段理解。

### E：匿名人脸聚类与用户人物库

1. 按库默认关闭；后台先生成 face observation 和匿名 core/edge 建议，名称只由用户输入。
2. 人物是实例级应用状态；observation、匿名组和资产关系按库隔离，跨库关系只能由用户显式建立。
3. 人工 assignment、exclusion、cannot-link 永远高于模型结果；不得自动合并两个已命名人物。
4. 支持从匿名 core 创建人物、整组并入、单 face 归类、拆分/移动/排除和可审计撤销。
5. face crop/embedding 是高敏感可重建派生数据；人物名和人工关系是不可重建应用状态，清除、备份、
   恢复和删除必须分别处理并二次确认。
6. 没有隐私批准的合法真实 ground truth、核心簇 precision 证据及可商业分发模型/runtime 前，E 的
   Backend Evidence Gate 必须保持 No-Go。

## 非目标

- 自动识别、猜测或建议真人、公众人物、Coser、模特或动漫角色姓名。
- 年龄、性别、种族、情绪等敏感属性推断；人脸登录、监控、外部人物库和视频人脸追踪。
- OCR、自由 caption、内容审核、NSFW 分类、生成/编辑图片、音频/字幕/动作理解。
- AI 标签自动写入或自动接受；自动合并已命名人物；模型覆盖人工关系。
- 云推理、GPU、任意模型市场/URL、独立 worker/容器、Redis、外部数据库或新部署单元。
- 修改、移动、重命名、删除原媒体，或把 AI 状态写回媒体/sidecar。

## 交付顺序与授权

1. A+B 保留已有 S1 合同和 S2A No-Go；不得因 revision 2 静默接通 production search/UI。
2. C+D+E 先完成 `INT-114～120` S1 extension，并经 Contract Ready Gate 接受；在此前只允许文档、合同、
   schema 设计和必要隔离 spike，不允许 production migration、handler、worker 或 UI。
3. S2B 只在 C+D 合同通过后实施；S2C 只在 E 合同、隐私准入和候选许可准入通过后实施。
4. 每个 S2 Gate 独立失败关闭；后续切片不得以 mock、隐藏入口或空实现冒充完成。
5. 所有 S2 均 Go 后才允许统一进入 S3 消费者/UI；S4 仍持有完整纵向、目标容量和发布签署。

## Gate 与失败回退

| 切片 | Backend Gate 必需输入 | 失败回退 |
| --- | --- | --- |
| A+B | accepted tokenizer/runtime、最终审核模型、合法 1,000 图片集、native 双架构、100k×768、最终供应链 | 删除 B，保留核心浏览与可删除的 A 状态 |
| C | 受控词表、建议 precision、人工标签 owner 纵向、清除/升级一致性 | 删除建议能力，保留人工标签 |
| D | 合法 100 视频集、4/10 帧策略、双架构 FFmpeg/embedding、联合负载 | 删除视频语义，保留现有故事板预览 |
| E | 隐私签署、合法 ground truth、core precision ≥99.5%、可分发模型、人工约束/备份恢复、双架构容量 | 降为 pair/小组建议；仍失败则删除 E |

## 验收 ID

- 继承 `POST-MVP-5-AC-001～006`。
- `POST-MVP-5-AC-007`：标签建议只来自受控词表，接受动作通过 curation，人工标签不被模型覆盖。
- `POST-MVP-5-AC-008`：视频搜索只复用现有故事板，在合法 100 视频集达到冻结质量门槛。
- `POST-MVP-5-AC-009`：人脸默认关闭、匿名先行、名称用户输入，人工约束优先且 core precision 达标。
- `POST-MVP-5-AC-010`：face 派生数据与人物应用状态按合同清除、备份、恢复，诊断/API 不泄露敏感值。
- `POST-MVP-5-AC-011`：A～E 在原生 amd64/arm64、4 CPU/4 GiB、100k 联合负载和最终供应链 Gate 通过。

本 manifest 冻结“要做什么”，不宣称实现或外部证据已经完成。
