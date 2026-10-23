# POST-MVP-5 revision 2 C+D+E capability contract

- 状态：**Accepted product/capability contract for INT-114～116**
- 日期：2026-08-29
- Scope：[POST-MVP-5 revision 2](../releases/POST-MVP-5-scope-r2.md)
- Gate：[INT-S1R2 Contract Ready](../gates/POST-MVP-5/int-s1r2-contract-ready.md)（整体仍 Pending）
- 需求：`FR-INT-004`、`FR-INT-006～014`、`NFR-INT-001～010`

本文只接受 capability 行为和状态机。SQLite、OpenAPI、安全/隐私/部署/测试合同分别由
`INT-117～119` 接受；在 `INT-120` Gate Go 前不得进入 production import graph。

## Canonical owners

| Policy / transition | 唯一 owner | 只允许的依赖 |
| --- | --- | --- |
| 受控词表 snapshot、suggestion 计算/失效/review decision | `internal/semantic` 的后续公开服务 | semantic generation；curation 只通过公开 port |
| 接受建议后写人工标签 | 现有 `internal/curation` | suggestion owner 提交 tag ID + expected curation revision |
| 故事板计划、帧文件和 source fingerprint | 现有 `internal/thumbnail` / `internal/media` | semantic 只读消费，不抽第二遍帧 |
| 视频 frame embedding、video score/cursor/coverage | `internal/semantic` | storyboard source port + generation-bound encoder |
| face observation、匿名组、cluster generation | 后续 `internal/face` | safe media source、generation-bound face runtime |
| person、manual assignment/exclusion/cannot-link、merge/split/undo | 后续 `internal/face` application-state service | SQLite port；不得由 runtime/cluster adapter直接写 |
| HTTP/DTO/error/auth/CSRF/rate-limit | `internal/api` | 仅调用 capability services |

不得把 suggestion review 写入 curation 私有表、把 face assignment 写入 semantic embedding 表，或让 API/
SQLite adapter 复制下列状态转换。

## C：受控标签建议

### 输入与身份

- 词表是有 revision 的有界 snapshot，条目只引用现有 tag ID 或管理员显式创建的受控 tag；禁止模型产生
  任意自由文本、同义词扩写或从文件名偷偷补词。
- suggestion 身份为 `library_id + asset_id + semantic_generation_id + vocabulary_revision + tag_id`；
  source fingerprint、generation 或词表 revision 变化即失效，不跨代复用 confidence。
- 每资产最多保存 Top-5，confidence 必须有限且在 `[0,1]`；同分按 tag ID 稳定排序。

### 派生状态机

```text
absent → pending → invalidated
             ├─ accept ──→ reviewed(accepted)
             └─ dismiss ─→ reviewed(dismissed)
```

- `pending` suggestion/confidence 是可重建派生状态；重建可替换它，但不得触碰 review decision。
- `accept` 必须在单个 application use case 中复核 suggestion、asset、tag、generation/vocabulary 和 curation
  ETag，再调用 curation owner。人工标签提交成功后才记录 accepted；任一步失败都不得出现半接受。
- `dismissed` 是用户应用状态，不参与训练、不改变词表、不被模型升级静默删除，并抑制相同 asset/tag
  再次展示。只有显式“清除建议复核记录”二次确认才删除；普通 AI disable/derived clear 保留它。
- 已经是人工标签的 pending suggestion 不展示；移除人工标签不会自动删除既有 accepted audit，也不会
  自动重新接受。
- 批量复核最多 100 个 suggestion ID，逐项返回 accepted/dismissed/conflict；事务不得跨 100 个资产持锁，
  但同一 suggestion + curation 写入必须原子或通过 capability-owned compensating failure 保持未 reviewed。

### 查询与失败

- 默认只查询当前库，可按 pending/accepted/dismissed 和 tag 过滤；稳定顺序为
  `confidence DESC, suggestion_id ASC`，reviewed 列表为 `reviewed_at DESC, suggestion_id ASC`。
- 模型 unavailable、库 offline 或 coverage 不完整时保留人工标签与 review decision，只停止新 suggestion；
  客户端必须显示 coverage，不能把未处理当作“没有建议”。

## D：视频代表帧语义搜索

### 输入合同

- Eligible 只包括 catalog 支持的 MP4/MOV/MKV，且存在现有 owner 发布的完整 active storyboard plan。
- 优先消费完整 10-frame plan；没有完整 10-frame 时可消费完整 4-frame fallback。不得把不同 plan/version
  的帧拼接成一次 embedding，也不得由 semantic 直接调用 FFmpeg。
- 每帧身份绑定 `asset_id + source_fingerprint + storyboard_plan/version + ordinal + semantic_generation_id`。
  任一绑定变化使对应 frame embedding stale；原故事板与视频始终只读。

### 排名合同

- 每帧使用与图片相同 generation/dimension/normalization 的 embedding；query 与每帧计算 cosine score。
- 视频分数固定为该 active plan 的最高 frame score；命中帧是最高分 frame。帧同分取较小 ordinal，视频
  同分按 asset ID 升序。该 max/best-frame 规则同时拥有 pagination tuple，不另存无法解释的聚合向量。
- 返回 video asset opaque ID、有限 score、命中 frame ordinal/timestamp 与 plan size；不返回 frame 路径、
  原始 vector、FFmpeg 输出或内部 cache key。
- 100-video 合法质量集若达不到 Top-20 ≥80%，整个 D 保持 No-Go；不得在 Gate 后临时切换 mean/max 以
  调参。改变聚合规则需要 transform version、合同修订和全量重建。

### 部分失败与覆盖率

- 单帧失败时当前 plan 标记 degraded，不把缺帧 plan 宣称 ready；仍可保存已完成 frame embedding 作为
  可重建进度，但不进入搜索结果。
- 10-frame 失败且独立的完整 4-frame plan 已由 storyboard owner 发布时，可明确降级到 4-frame；降级计入
  coverage/diagnostic，不在一次查询中混用。
- video source 变化、offline、storyboard eviction、模型切代或任务取消保留最后可靠 active generation，
  新 generation 未 ready 前不删除旧结果。

## E：匿名人脸聚类与人物库

### 数据分类与身份

- `face observation` 包含 opaque face ID、库/资产绑定、generation、非可逆质量/位置投影和敏感 embedding；
  crop 仅可作为有界可重建 cache，不进入 SQLite/API/日志/诊断。
- observation、embedding、anonymous cluster 是高敏感可重建派生状态；person name、manual assignment、
  exclusion、cannot-link 和用户操作审计是不可重建应用状态。
- face ID 不跨 source fingerprint 或 detector generation 稳定；人工关系通过 capability-owned observation
  lineage/reconciliation 迁移，不能以 bbox 顺序猜测同一 face。

### 匿名 cluster 状态

```text
unassigned observation
  → edge suggestion（不能整组建人物）
  → anonymous core（达到冻结 precision 才可整组操作）
  → assigned to person（仅用户动作）
  → excluded / moved（用户约束优先）
```

- 后台只能创建/更新匿名 core/edge，不能创建名称或把 observation 自动写入已命名 person。
- core component 采用确定性 smallest-ID anchor coherence：每个 core 成员都必须与 component 的稳定
  anchor 达到冻结 core similarity。仅凭 `A≈B`、`B≈C` 不得推导 `A≈C` 或把两个身份经 bridge face
  传递合并；未通过 anchor coherence 的候选最多作为 edge 逐项确认。
- core 只有合法真实 ground truth 证明 precision ≥99.5% 才允许“整组创建/并入人物”；否则产品降级为
  pair/小组逐项确认。edge 永远要求逐项确认。
- cluster generation 变化不得覆盖 manual assignment、exclusion 或 cannot-link；受约束 observation 在新代
  聚类前先应用约束。

### 人物与人工操作

- person 是实例级 opaque entity，名称允许重名、NFC 规范化、1～100 Unicode code points，不作为唯一键。
- 从 anonymous core 创建人物、整组并入人物、单 face 归入、移动、拆分、排除和 cannot-link 都要求
  expected person/cluster revision；全部为单 owner 短事务并写有界审计事件。
- 一张多人图片按 face ID 操作，绝不把整个 asset 隐式归给人物。
- 两个已命名 person 永不由后台自动合并。用户显式 merge 必须预览冲突；cannot-link 冲突时整个 merge
  失败，不产生部分成员或空人物。源 person 只有成员/别名审计成功迁移后才 tombstone。
- undo 只针对最近一次仍未被后续冲突修改的人工操作，使用记录的 expected revisions 反向提交；不是任意
  历史回放。若 revision 已变化返回 conflict，不猜测恢复。

### 删除、禁用与搜索

- 按库禁用停止检测/聚类并卸载 face session；默认清除只删除 observation/crop/embedding/anonymous
  cluster 等派生数据，保留 person、manual assignment lineage、exclusion、cannot-link 和审计。
- “清除人物关系”是独立危险操作，必须列明库范围、影响计数和不可恢复内容并二次确认；绝不删除原媒体。
- 人物搜索只返回具有有效 manual/confirmed face assignment 的资产；匿名相似度不能冒充人物归属。
- API/日志/诊断不得返回 embedding、crop bytes/path、精确 bbox、模型原始错误、查询原文或人物名之外的
  推测身份属性。

## Cross-capability invariants

- 所有写入要求认证管理员、CSRF、If-Match 或 idempotency；所有列表使用稳定 keyset，不使用 OFFSET。
- 后台全局有界并低于浏览派生任务；禁用或 offline 不解释为空，不清理最后可靠派生状态。
- C/D/E 的生成、重建、清除和取消复用公共 operation 投影，但各 capability 拥有自己的 durable queue/
  state machine；不得用通用 operation 表代替业务事务。
- 所有媒体读取继续只经 `internal/files`；没有 API 接受 path、URL、vector、crop 或 arbitrary model ID。
