# POST-MVP-5 Scope Manifest Proposal — Draft 1（已被替代）

- 状态：**Superseded by [revision 1 Frozen](POST-MVP-5-scope.md)**
- 日期：2026-08-27
- 产品显示标识：`Post-MVP/5`
- Change Record：[CR-2026-021](../changes/CR-2026-021-intelligent-media-discovery.md)
- Feature：[FTR-INT-001](../features/intelligent-media-discovery.md)
- Gate at draft time：[INT-S0 Architecture Ready](../gates/POST-MVP-5/int-s0-architecture-ready.md)（草案形成时 No-Go）
- 需求：`FR-INT-001～020`、`NFR-INT-001～010`
- 产品 Owner：产品用户
- Capability Owner：模型管理、推理运行时、语义检索、人脸/人物库、curation、media、release、
  security/privacy、frontend；具体人员仍须在冻结前签署
- 交付预算提案：S0 4～8 工程周；S0 后最多 32 单人工程周。超过预算必须按下述切片边界删减，
  不得通过降低只读、安全、隐私、质量或双架构 Gate 换范围
- S0 收口：[产品决定与外部阻塞](../gates/POST-MVP-5/int-s0-closeout-and-blockers.md)；本地技术探索
  已冻结，不再以增加合成测试延长 S0

本文件保留为 revision 1 冻结前的完整方案与替代项记录。当前范围以
[POST-MVP-5 scope revision 1](POST-MVP-5-scope.md) 为准；revision 1 只纳入 A+B，C/D/E 不在首版。

## 拟纳入范围

### Slice A：模型与派生任务基础

- `FR-INT-001`、`FR-INT-005`、`FR-INT-015～020`、`NFR-INT-001～005`、`NFR-INT-007～010`。
- 按媒体库启停；模型 availability/generation、覆盖率、失败、取消、重建和清除。
- 仅从项目签名发行清单或固定 `/models:ro` 获取审核模型；默认复制托管，严格校验后可直接读取。
- 无真实运营 owner 时不显示或承诺项目国内镜像；离线 `/models:ro` 是必须成立的受限网络基线。
- Slice A 只提供后续能力需要的基础，不作为无消费者的模型市场单独发布。

### Slice B：图片语义搜索（首个可独立交付切片）

- `FR-INT-002～003`、`FR-INT-005`、`NFR-INT-001～006`。
- 用固定模型 generation 为图片生成可重建 embedding，支持中文/英文自然语言查询。
- 语义查询与现有文件名搜索并列，不替代目录层级或字面搜索；保留库、目录、递归和媒体类型范围。
- 只有 Slice A 和 Slice B 的 S0/S1/S2 证据完成后才允许其生产 UI。

### Slice C：受控 AI 标签建议

- `FR-INT-006～008`。
- 只从管理员词表或既有标签产生候选；AI 建议与人工标签分离。
- 用户确认后必须通过既有 `internal/curation` owner 写入；不自动接受、不在线训练。
- 依赖 Slice B 的同代 embedding；可以整体删除而不影响图片语义搜索或人工标签。

### Slice D：视频代表帧语义搜索

- `FR-INT-004`。
- 只消费既有 4/10 格故事板和现有 `internal/media` admission；不创建第二套 FFmpeg/抽帧 owner。
- 返回视频和命中代表帧，不承诺声音、对白、字幕、动作时序或精确时间段理解。
- 100 个合法代表性视频的质量和联合负载未过 Gate 时，整个 Slice D 延期，不以图片结果冒充视频能力。

### Slice E：匿名人脸聚类与用户人物库

- `FR-INT-009～014`。
- 后台先检测并形成匿名 core/edge 建议；用户从匿名组建立并命名人物。
- 用户可把匿名 core 合并到人物、把单个 face 归入人物、拆分/移动/排除错误成员。
- 人工 assignment、exclusion、cannot-link 高于模型结果；不得自动合并两个已命名人物。
- 不识别或推测 Coser、模特、公众人物、真人身份或动漫角色姓名；名称完全由用户输入。
- Slice E 只有在合法真实 ground truth、核心簇 precision、隐私评审和可商业分发的 face runtime/权重
  全部通过后才纳入 Frozen revision 1。SFace hold 未解除且无合格替代时，必须把 Slice E 从
  revision 1 删除，而不是让用户自行下载来转移分发/隐私责任。

## 明确非目标

- `POST-MVP-5-NG-001`：自动识别/猜测 Coser、模特、公众人物、真实身份或动漫角色姓名。
- `POST-MVP-5-NG-002`：年龄、性别、种族、情绪等敏感属性推断，人脸登录、监控或外部人物库。
- `POST-MVP-5-NG-003`：OCR、自由 caption、内容审核、NSFW 分类、生成/编辑图片。
- `POST-MVP-5-NG-004`：视频人脸追踪、声音/字幕理解、动作时序和精确时间段检索。
- `POST-MVP-5-NG-005`：云推理、任意模型市场、任意下载 URL、模型附带代码/插件/脚本。
- `POST-MVP-5-NG-006`：GPU、独立 AI worker、第二容器、Redis、外部数据库或新部署单元。
- `POST-MVP-5-NG-007`：自动接受标签、自动合并已命名人物、让模型覆盖人工关系。
- `POST-MVP-5-NG-008`：修改、移动、重命名、删除原媒体或把 AI metadata 写回媒体/sidecar。

## 交付顺序与预算停损

顺序按依赖冻结，不表示价值排名：

1. S0 只完成许可/隐私/质量/容量/runtime/vector/model-source 证据和实际纳入切片决定。
2. S1 先冻结 Slice A+B 的 OpenAPI、data、错误、任务和删除/升级语义。
3. S2 先交付 A+B 后端证据；B 未通过不得开始 C/D/E 生产实现。
4. C、D、E 各自拥有独立 S1～S4 Gate，可以延期或删除，不能相互伪造完成状态。
5. 总投入到达 32 单人工程周仍未进入集成时，保留已通过的最小完整切片，删除未开始或未过 Gate 的
   后续切片；不得留下生产 mock、隐藏入口或半套 schema。

推荐的预算复核点：S0 完成、A+B S2 完成、每个 C/D/E 开始前、任一真实质量/双架构/供应链 Gate
失败时。精确排期只有 owners 和实际候选冻结后才有意义。

## 合同与 canonical owner

| 范围 | Canonical owner | S1 权威合同 |
| --- | --- | --- |
| 模型 catalog/download/install/activate | `internal/aimodel`（提议） | OpenAPI、model manifest、operation 状态、错误和 data model |
| inference session/admission/error map | capability-owned port + adapter（提议） | Go interface、资源上限、取消/超时、稳定错误 |
| 图片/视频 embedding 与语义 query | `internal/semantic`（提议） | OpenAPI、query/cursor、generation、SQLite schema |
| 故事板抽取 | 现有 `internal/media` / `internal/thumbnail` | 复用现有合同，不新增第二实现 |
| AI 标签接受 | 现有 `internal/curation` | 接受动作调用既有标签服务；建议 schema 由 semantic 拥有 |
| face observation/cluster/person/manual constraint | `internal/face`（提议） | OpenAPI、事务、删除/备份/升级与 privacy contract |
| HTTP/DTO/middleware | `internal/api` | `api/openapi.yaml`，仅在对应 S1 Go 后修改 |
| composition/lifecycle | `internal/app` | 单进程启动/停止/资源模式；不得增加部署单元 |

上述提议包名只有 ADR-0013 接受且 S0 Go 后才能建立；当前不得创建 production package、migration
或 endpoint。

## Frozen revision 1 的硬条件

- 产品用户明确接受本草案的范围、非目标、预算停损、默认关闭和人脸高敏感边界。
- 明确 revision 1 实际包含 B/C/D/E 中哪些切片；不允许用“以后补”保留没有 owner、Gate 和删除
  fallback 的范围。
- 每个纳入切片有具名或可执行角色 owner、S1～S4 Gate、验收 ID 和 fallback。
- 每个真实数据、native amd64、隐私、许可、供应链和容量外部条件有责任角色、最迟 Gate 与失败时删除
  哪个切片的明确记录；不能用“继续研究”作为 fallback。
- 在线发行源/镜像有真实运营 owner、签名 key/checkpoint/rotation/revocation 和可用性方案；否则
  revision 1 只承诺 `/models:ro` 离线安装。
- ADR-0013 接受或明确保持 Proposed；`R-024～R-030` 的严重项有 owner、最迟 Gate 和可执行 fallback。

Frozen scope 只决定做什么和何时停止，不等于生产开发或发布授权。以下验证归入对应切片 Gate，
不再通过扩张 S0 合成测试提前完成：

- 图片 1,000 张、视频 100 个与人脸 ground truth 的来源、许可、隐私和质量证据；
- 原生 Linux/amd64 与 arm64 的 runtime/数值、最终镜像、SBOM/provenance、漏洞与 notices；
- 4 CPU/4 GiB、100k 完整进程容量；
- SFace hold 解除或经审查替代。E 开工 Gate 未通过时直接删除 E；
- 既有 catalog keyset 债务的独立维护 Gate 与原预算复测。

## S1 文档与合同影响清单

下表只列明 S0 Go 后必须更新的权威源；当前不得提前写入生产合同。

| S1 task | 权威文件 | 必须冻结的内容 |
| --- | --- | --- |
| `INT-101` | `docs/product-requirements.md`、`docs/user-flows.md`、`docs/ui-design.md`、根 `README.md` | 实际纳入 FR/NFR、非目标、导航、状态、响应式、可访问性、用户告知 |
| `INT-102` | `docs/architecture/intelligent-media-discovery.md`、`docs/adr/0013-local-ai-runtime-and-derived-vector-index.md`、`AGENTS.md`（仅 accepted decision 需要时） | capability/interface/error/admission owner、依赖方向、runtime/vector/model 决定 |
| `INT-103～104` | `docs/data-model.md`、只追加 `migrations/` source、`internal/store/sqlite` schema tests | 派生/人工状态、generation/revision/FK/cascade/retention/backup、升级失败语义 |
| `INT-105` | `docs/security.md` | 生物特征分类、模型供应链、日志/诊断、清除、无网络和模型文件边界 |
| `INT-106` | `docs/deployment.md`、根 `README.md` | `/models:ro`、`/app/data/models`、空间、权限、备份/恢复、双架构、可选出站 |
| `INT-107` | `docs/testing-strategy.md`、`docs/feasibility-study.md`、`docs/risk-register.md` | fixture 合法性/隐私、质量阈值、100k/4 GiB、native 双架构、供应链 Gate |
| `INT-108～109` | `api/openapi.yaml`、生成 Go/TypeScript client 与 digest | settings/status/search/suggestion/cluster/person/model operation、error/ETag/cursor |
| `INT-110～112` | `docs/api-design.md`、`docs/deployment.md`、feature decision tables | 并发/stale/generation/partial failure、opaque ID、source/mirror/proxy/quota |
| `INT-113` | `docs/architecture/traceability.md`、本清单的 Frozen successor、S1 Gate | requirement→owner→contract→evidence 映射和 scope revision 一致性 |

生产实现后还必须同步 `docs/features/intelligent-media-discovery-task-list.md`、对应 S2～S4 Gate 与 evidence
索引；这些文件跟踪事实，不替代上述权威合同。

## 当前判断

**本节是冻结前的历史判断，已被 revision 1 替代。** 当时本地技术探索已经收口，阻塞是产品用户尚未确认
`DEC-INT-001～005` 和实际 owners。推荐 revision 1 先冻结 A+B；C/D/E 作为可删除后续切片，只有各自
外部准入条件成立才开工。代表性真实质量、native amd64、最终安全/合规与完整进程容量继续作为对应
切片 Gate，不再通过追加 S0 合成测试追赶。当前状态以 [revision 1](POST-MVP-5-scope.md) 和
[INT-S0](../gates/POST-MVP-5/int-s0-architecture-ready.md) 为准。
