# 智能媒体发现技术架构

## 文档状态

- 对应 feature：`FTR-INT-001`
- 目标：`POST-MVP-5` revision 2（A～E）
- 状态：A+B S0/S1 与 C/D/E S1R2 为 Go；S2A/S2B/S2C **Backend Ready / Release No-Go**
- 权威边界：[Frozen scope](../releases/POST-MVP-5-scope-r2.md)；
  [INT-S1](../gates/POST-MVP-5/int-s1-contract-ready.md)授权 A+B，
  [INT-S1R2](../gates/POST-MVP-5/int-s1r2-contract-ready.md)授权 C/D/E 后端；
  [CR-2026-022](../changes/CR-2026-022-s2-backend-release-gate-separation.md)把最终发行输入归入 S4。
- S1 capability/事务合同：[A+B S1 contract](intelligent-media-s1-contract.md)

## 设计结论

推荐保持 FolioPath 的单进程、单容器结构，在 Go 进程内通过窄接口调用本地推理运行时。托管模型、
embedding 和加速索引放在 `/app/data`；可选只读 `/models` 只作为审核模型包的离线来源或固定哈希
直接来源；原媒体仍只经 `internal/files` 安全打开。SQLite 保存可重建
embedding 与任务状态的权威派生记录，独立索引文件只作为可删除、可重建的查询加速层。

revision 1 已接受 ORT C API + SigLIP 1 float16-internal + SQLite float16 exact + CPU-only 作为 A+B
唯一候选路径；真实质量、双架构、完整资源和供应链仍须在 Backend/Release Gate 通过。人脸模型和
ANN 未被该决定接受；revision 2 的 C/D/E runtime、存储和事务合同已由 S1R2 冻结，E 的生产实现仍
由隐私/合法数据/候选许可准入失败关闭。

## 组件与所有权

```text
HTTP / API
  ├─ AI model service ─ built-in compatibility catalog / fixed /models:ro
  │      └─ managed model store ─────────── /app/data/models
  ├─ semantic service ── inference port ── local ONNX adapter
  │      ├─ embedding repository
  │      └─ vector search port ── SQLite exact adapter
  └─ jobs service ───── bounded admission / lease / retry
             │
             ├─ internal/files ── /library (read-only)
             ├─ SQLite ───────── /app/data/foliopath.db
             └─ model/index files /app/data/models, /app/data/ai-indexes
```

建议源码边界：

- `internal/semantic`：文本/图像向量语义、查询绑定、排名和模型 generation，并拥有 inference/index ports；
- `internal/aimodel`：唯一拥有兼容清单、映射目录扫描、manifest/hash/license/
  architecture 校验、托管发布、直接来源状态与 model generation；
- `internal/inference`：capability-owned port 的适配实现；候选子目录 `onnx`，不能反向拥有业务规则；
- `internal/jobs`：复用持久任务生命周期、公平性、取消和恢复，但不拥有语义/人脸状态机；
- `internal/store/sqlite`：实现上述 capability 定义的 repository；
- `internal/api`：只做认证、DTO、错误映射和 service 调用；
- `internal/app`：唯一组合模型、索引、任务和生命周期；
- `web/src/features/semantic-search` 消费已冻结合同；人工标签仍由既有 `curation` feature 拥有。

禁止：handler 直接读 SQLite/媒体/模型；为 AI 新建独立 worker 服务；把任意本地路径或模型下载
地址暴露给 API；让 inference adapter 自行下载、寻找或激活模型。

## 推理与模型生命周期

### 模型清单

每个模型必须有内建 manifest：

```text
model_id, purpose, semantic_version, sha256, source, license,
input_shape, preprocessing_version, output_schema_version,
embedding_dimension, supported_architectures
```

- 构建/安装只接受项目审核过的 manifest 和哈希；不执行模型附带代码。
- 模型缺失时能力显示 unavailable，不影响应用启动和普通浏览。
- 每次派生结果绑定 `model_id + version + transform_version + source_fingerprint`。
- 新 generation 先构建、校验、原子激活，再延迟清理旧 generation。
- INT-S0 候选可评估多语视觉-文本模型、YuNet/SFace 等许可清晰的模型；名称仅是 spike 候选，
  不是选型承诺。不得采用权重许可不允许产品分发的模型。

### 模型获取、离线导入与直接来源

模型获取有两条受控入口，不提供浏览服务器文件系统或粘贴下载 URL：

1. **托管下载**：认证管理员从签名发行清单列出的 model/version/source 中选择。`internal/aimodel`
   检查配额与安全余量，以并发 1 写 `/app/data/models/.partial`，支持取消和受约束续传；完整包通过
   manifest、大小、SHA-256、目标架构和许可证标识后，原子发布到内容寻址的托管路径。
2. **固定映射**：部署者可把宿主机模型目录只读映射为 `/models`。服务只枚举该目录的直接文件，
   拒绝 symlink、子挂载、特殊文件和未知扩展；只有内建清单中完全匹配的包才显示“可用”。UI 永不
   接收或返回宿主机路径。

映射包默认复制到托管目录。管理员也可选择直接使用，以避免重复占用数百 MiB/数 GiB 空间，但仅在
`/models` 本身只读、文件为普通文件且持续匹配固定 SHA-256 时成立。服务记录 `storage_mode=direct`、
content hash 和 model generation；每次启动/加载前重新校验。源缺失、变为可写或哈希变化时，模型
进入 unavailable，不自动回退到另一个版本，不删除旧 embedding/index/people/manual curation。
由于生成 query embedding 也依赖同代文本模型，模型不可用时语义查询必须返回明确的
`model_unavailable`，不能假装旧索引仍可完整查询。

部署者镜像只能通过启动配置提供基址，且下载对象仍由签名发行清单固定；这解决镜像托管位置差异，
不允许镜像任意替换内容。若项目没有真正运营国内镜像，则文档和 UI 不能宣称存在国内下载加速，
离线 `/models:ro` 是受限网络环境的基线方案。

模型包在原型中使用 `.foliomodel` 名称，但这不是已冻结格式。S1 必须决定它是单一权重文件还是受限
容器；若使用归档，解析器必须限制总展开大小、文件数、压缩比和路径深度，拒绝绝对路径、`..`、
symlink/hardlink、device、重复文件名和附带可执行内容，并在隔离临时目录完成验证。签名算法、密钥轮换、
撤回清单、离线清单分发和过期策略同样必须由 S0/S1 冻结，不能只写“已签名”而没有可执行合同。
首个发布切片不支持 ONNX external-data：模型包可以包含 image/text graph、tokenizer 和声明文件等多个
独立清单文件，但每个 ONNX graph 必须内嵌全部 initializer。发行流水线负责解析并拒绝任何
external-data 引用，运行时只接受签名 allowlist 中固定 size/hash 的已审核 graph；runtime 自身的路径
检查只作纵深防御。若未来确需外置 tensor，必须以新的安全/格式决策重新打开，不能顺带放宽。

### 获取与激活状态机

```text
catalog candidate
  → downloading/scanning
  → verifying
  → installed(managed|direct)
  → available
  → building generation
  → active

任何阶段 → cancelled | failed | unavailable
```

- `failed` 的 operation 可以清理；`installed` 模型记录和 `active` generation 由 aimodel service 独占更新。
- 下载/导入成功不等于激活成功；模型 session smoke 和新 generation 验证通过后才能更新 active pointer。
- 删除未激活模型只删除托管副本；删除活动/旧 fallback 模型前必须先证明没有 session、operation、
  generation 或恢复引用。首版可以不提供删除模型操作，以降低错误回收风险。
- 直接来源不可“卸载”外部文件；FolioPath 只删除引用记录。任何删除动作都不修改 `/models`。

关键失败语义：

| 场景 | 状态 | 保留 | 用户可执行 |
| --- | --- | --- | --- |
| 下载中断/取消 | operation cancelled/failed | 现用模型与索引；可验证的 partial 按策略保留 | 继续或重试 |
| 下载错哈希/错架构/撤回 | package rejected | 现用模型与索引 | 更换审核来源/版本；不能强制加载 |
| 托管复制磁盘满 | import failed | 源文件、现用模型与索引 | 清理空间后重试 |
| direct 文件缺失/变化/可写 | model unavailable | DB 派生、索引文件、人物与人工标签 | 恢复同哈希只读文件或改用托管副本 |
| 新模型 session/build 失败 | activation failed | 旧 active model/generation | 查看脱敏原因、重试或选择其他审核版本 |
| 国内/部署镜像不可达 | source unavailable | 所有本地状态 | 选择其他审核源或 `/models:ro`；不回退任意 URL |

### 资源控制

- AI 默认并发 1，并受现有全局 background admission 约束；不能再建无界 goroutine 池。
- 模型 session 延迟加载；禁用全部 AI 或空闲回收后不常驻内存。
- 图片从 `internal/files` 打开并使用有界解码输入；首选现有安全缩放链得到模型输入，不能读取原路径。
- 每项推理有超时、进程/请求取消、输入像素和批量上限；批量大小由 spike 固定。
- 浏览派生任务、扫描和交互读取优先级高于 AI backfill；AI admission 满时只保留可合并意图。

## 语义检索设计

### 写入流程

1. catalog 发布资产 source fingerprint。
2. semantic service 以 keyset 小批登记 `semantic_image` 任务。
3. worker 经 `internal/files` 安全读取并生成视觉 embedding。
4. 在短事务中写 SQLite generation 行；不得持有事务做媒体读取或推理。
5. index builder 从已提交 embedding 增量构建加速索引，并以临时文件 + 原子 rename 发布。

视频不重新解码：读取已就绪故事板单元格，对每帧生成 embedding。查询时视频得分为代表帧最高相似度，
同时保存命中 frame ordinal；多帧聚合策略必须由评测决定，不能凭直觉平均。

### 查询流程

1. API 将文本、库/目录/type/filter 和分页参数交给 semantic service。
2. 文本模型生成 query embedding；查询文本不写日志。
3. 向量层返回有界候选 ID 和 score；catalog owner 再应用存在性、offline、scope 和类型过滤。
4. 使用 `score DESC, asset_id ASC` 稳定排序，cursor 绑定模型/index generation 与 query fingerprint。
5. generation 变化令旧 cursor 过期，返回稳定冲突错误；不混合两代分数。

向量引擎在 S0 比较三类方案：SQLite 内精确扫描、SQLite 向量扩展、独立 HNSW/ANN 文件。选择标准是
10 万档延迟/RSS、崩溃恢复、双架构构建、许可证和维护风险，不以“更先进”作为理由。SQLite 中的向量
行必须足以重建外部索引，外部索引永不成为唯一事实。

### AI 标签建议

受控词条使用同一文本 encoder 生成向量，按阈值与 Top-K 产生 suggestion。建议表只引用 `tag_id`、
asset、score 和 generation。接受建议必须调用 curation service 的既有并发控制；AI capability 不直接
写 `asset_tags`。首版不保存自由文本 caption，也不从图片生成任意新标签。

## 人脸聚类与人物库设计

### 流程

```text
asset → face detect → quality gate → face embedding → anonymous clustering
                                                     │
                      user creates person ◀─────────┤
                      user merges cluster/person ◀──┤
                      user assigns one face ◀───────┤
                      user moves/excludes face ◀────┘
```

检测框坐标使用归一化坐标并绑定源 fingerprint；UI 通过受保护的媒体/缩略图 API 绘制框，不持久化
人脸裁剪文件作为产品数据。若性能迫使缓存小裁剪，必须单独规定加密/清除/配额和备份语义。

聚类分两层：高置信核心簇可作为“建立人物”的候选；边缘样本只作为待审核建议。阈值不能直接照搬
模型示例，必须在以项目实际人物套图构建、获得授权且不入库的评测集上校准。已命名人物间只允许用户
发起事务合并；后台最多生成候选，不自动执行。

### 一致性规则

- `people`、manual assignment、manual exclusion、merge audit 是应用状态，随备份保存；
- observation、embedding、anonymous cluster 和 similarity edge 是派生状态，可清除重建；
- 人物合并在单一 SQLite 事务内迁移 assignment、保留 alias/audit，并增加 people revision；
- 单 face 归类引用 `face_id`，同时校验 source fingerprint；源变化导致旧 observation 失效；
- `cannot_link` exclusion 优先于模型边；manual assignment 固定目标 person，重聚类不能搬走；
- 删除媒体库级 AI 数据只删除派生行和该库 assignment；跨库人物若无成员可保留空人物，具体语义 S1 冻结。

## 数据模型草案

仅用于 S0 评审，S1 后才允许创建追加 migration。

| 概念表 | 关键字段 | 分类 |
| --- | --- | --- |
| `ai_models` | purpose、version、hash、license、state、storage_mode、source_kind、installed_at | 配置/运行状态 |
| `ai_model_operations` | kind=download/import/verify、model/version、state、bytes、error_code、lease | 可清理运维状态 |
| `ai_library_settings` | library_id、semantic_enabled、face_enabled、generation | 应用状态 |
| `semantic_embeddings` | library_id、asset_id、model_generation、vector、source_fingerprint | 可重建派生 |
| `video_frame_embeddings` | asset_id、storyboard_version、frame_ordinal、vector | 可重建派生 |
| `ai_tag_suggestions` | asset_id、tag_id、score、generation、state | 可重建建议 |
| `face_observations` | face_id、asset_id、box、quality、model_generation、fingerprint | 可重建派生 |
| `face_embeddings` | face_id、vector、generation | 可重建派生 |
| `face_clusters` / `face_cluster_members` | cluster generation、face_id、confidence、role | 可重建派生 |
| `people` | person_id、display_name、revision、created_at | 应用状态 |
| `person_face_assignments` | person_id、face_id、source=manual/confirmed_cluster | 应用状态 |
| `face_exclusions` | face_a、face_b 或 face/person constraint、reason | 应用状态 |
| `person_merge_events` | source、target、actor、timestamp、revision | 审计应用状态 |
| `ai_index_generations` | kind、model generation、build state、checksum、activated_at | 可重建控制状态 |

向量可采用定长 little-endian float32 或量化表示，但格式必须有 schema version 和维度 CHECK。未通过
容量 spike 前不冻结 blob 格式，也不允许把 SQLite extension 当隐含部署依赖。

## API 草案

实际路径/字段以未来 `api/openapi.yaml` 为准。S0 仅冻结资源语义：

- `GET/PATCH /api/v1/libraries/{libraryId}/ai-settings`：启停与状态；PATCH 使用 ETag/If-Match；
- `GET /api/v1/ai-models`：返回审核模型、已安装版本、来源类型、空间和可用状态，不返回文件路径/URL；
- `POST /api/v1/ai-model-downloads`：只接受 manifest 中的 opaque model/version/source ID；
- `GET/DELETE /api/v1/ai-model-downloads/{operationId}`：读取进度或取消，不暴露上游地址；
- `POST /api/v1/ai-model-scan`：扫描固定 `/models`，返回兼容状态和 opaque candidate ID；
- `POST /api/v1/ai-model-imports`：以 candidate ID 选择 `managed_copy|direct_readonly`；不接受路径；
- `POST /api/v1/ai-model-activations`：选择已安装兼容 generation，并登记安全重建/切换；
- `GET /api/v1/semantic-search`：`q`、scope、kind、cursor；与现有字面搜索为不同 operation；
- `GET /api/v1/assets/{assetId}/ai-tag-suggestions`、`POST .../{tagId}/accept|ignore`；
- `GET /api/v1/face-clusters`、`GET /api/v1/face-clusters/{clusterId}`；
- `GET/POST/PATCH /api/v1/people`、`GET /api/v1/people/{personId}/assets`；
- `POST /api/v1/people/{personId}/faces`：单 face 人工归入；
- `POST /api/v1/people/{personId}/merge-cluster`、`POST /api/v1/people/{personId}/merge-person`；
- `POST /api/v1/faces/{faceId}/move`、`POST /api/v1/faces/{faceId}/exclude`；
- `POST /api/v1/libraries/{libraryId}/ai-rebuilds`、`DELETE /api/v1/libraries/{libraryId}/ai-data`。

所有写操作要求管理员认证、CSRF 和 revision/precondition；批量操作有硬上限并单事务或明确 partial
result。错误必须区分 model unavailable、index building、generation conflict、stale face、offline、
resource exhausted、invalid scope、model source unavailable、hash mismatch、insufficient storage 和
unsupported model，且不泄露路径、下载 URL、凭据、embedding、模型内部输出或人脸裁剪。

## 任务、恢复与删除

建议 job kinds：`ai_model_download`、`ai_model_import`、`ai_model_verify`、`semantic_image`、
`semantic_video_frames`、`face_analyze`、`face_recluster`、`ai_index_rebuild`。媒体分析任务幂等键包含
library/asset/kind/model generation/source fingerprint；模型任务幂等键包含 operation/model/version/hash。

- 扫描发现新增/变化资产只登记意图；删除资产通过 FK/capability cleanup 删除派生关联；
- 重建采用 parent run + keyset 小批 admission，不能一次登记 10 万行到活跃队列；
- 取消停止后续 admission 并协作取消当前推理；已安全提交批次保留；
- 崩溃后 lease 到期重取；模型或索引不兼容标记 blocked，不无限重试；
- 清除 AI 数据先停该库 admission、等待/取消活跃任务，再删除 SQLite 派生和索引目录；原媒体不变；
- 数据库备份包含人物应用状态和 embedding；外部索引可排除在备份外并在恢复后重建。
- 下载和复制先预留安全空间，写 `.partial` 后校验并原子发布；取消、崩溃和磁盘满只清理本 operation
  临时文件，不删除现用模型。续传必须绑定同一 manifest、ETag/长度与已验证分块，来源变化则重新开始。
- 直接模型在启动、激活和 session 冷加载前复核只读属性与 SHA-256；失效只改变 model availability，
  不触发派生数据清理或模型自动替换。

## 安全、隐私与供应链

- 人脸 embedding 属敏感生物特征派生数据，即使不用于确认真实身份也按高敏感度处理；默认关闭并明确告知。
- 只有管理员可启停、命名、合并、导出诊断或清除；诊断包不得含 query、向量、框对应裁剪或姓名列表。
- `/library` 仍只读，全部打开经过 `internal/files` 的 Linux `openat2` 边界；模型不能访问任意路径。
- `/models` 若存在必须是独立、只读、固定容器路径；它不是第二个媒体 mount，不能被媒体扫描/API
  浏览，也不能包含 symlink、嵌套 mount 或特殊文件。其安全打开由 model file boundary 独立拥有。
- 不允许网络推理。运行时默认无出站依赖；唯一可选出站流量是管理员明确触发的模型包下载，且只到
  签名清单/部署配置中的审核源。模型来源、哈希、许可证和获取方式进入 SBOM/provenance。
- 下载客户端禁止重定向到非审核 origin、回环、link-local、RFC1918 或 Unix socket；是否允许部署者
  内网镜像必须由明确配置和威胁模型冻结，不能由请求参数绕过，防止 SSRF/凭据泄露。
- 模型代码、运行时、向量引擎与权重分别审查许可证；代码许可不代表权重可再分发。
- ONNX 等模型文件视为不可信输入，只装载内建 allowlist/hash；解析崩溃与 OOM 是发布阻断风险。
- native runtime 的最终镜像必须显式保留 ELF SONAME 闭包、包元数据、许可证/notices 和独立 SBOM
  component；不得把“builder 可链接”当作 final-stage 可加载证明。若最小闭包仍有 Critical/High，只有
  修复基座或安全 owner 审批的 VEX/可达性结论可以处置，发行脚本不得仅按发行版 `no-dsa` 自动忽略。
- 人物删除、人脸数据清除、库移除、备份/恢复和模型升级必须有真实集成测试。

## 部署影响

保持一个镜像和一个 HTTP 进程，但会增加 native runtime、模型目录、可选出站下载、临时空间、启动/
按需加载路径和双架构供应链面。这已达到 ADR 门槛并在提议 ADR-0013 中记录。CPU-only 是首个候选
基线；GPU、独立 worker、新容器和任意模型卷仍不在方案。部署提案新增可选 `/models:ro` 映射与可选
审核镜像配置；`/library` 单一媒体挂载、`/app/data` 可写应用数据和单容器拓扑均不改变。

模型不默认打入应用镜像，也不进入应用数据备份；托管模型可由 manifest/hash 重新取得，直接模型由
操作员恢复相同 `/models:ro` 内容。备份保存已选 model ID/version/hash/storage mode 与人物人工状态。
恢复后模型缺失时应用可启动、普通浏览可用，AI 状态为 unavailable，不能静默选择不同模型。

## 验证策略

- 单元：向量规范化/排序、cursor 绑定、阈值、人工约束、合并事务、generation 和错误映射；
- 集成：SQLite migration/FK/cascade、任务重启/取消、索引损坏重建、模型缺失、源变化、offline；
- 安全：路径越界、external-data 拒绝、恶意模型、畸形图片、OOM/超时、CSRF、API
  脱敏、AI 数据清除和原媒体 hash 不变；
- 获取：下载取消/续传/重定向/来源变化/错哈希/磁盘满；`/models:ro` symlink/嵌套 mount/可写/
  源消失/替换；托管复制和直接使用在重启、升级、备份恢复后的失败关闭；
- 质量：授权的项目代表性套图评测集，人物身份按 ground truth 分组；结果只提交汇总和 fixture 生成说明；
- 容量：4 CPU/4 GiB、100k 媒体/10k 目录、浏览并发、冷/热查询、重启续跑、DB/index/模型空间；
- 跨架构：最终镜像在原生 linux/amd64 与 arm64 跑相同模型、fixture 和误差容忍规则；
- 前端：URL 恢复、覆盖率/错误状态、多人 face 选择、合并/拆分/撤销、键盘、双语、移动端和 axe。

具体命令、数据集和判定见 [INT-001 spike](../spikes/int-001-ai-feasibility.md)与
[任务清单](../features/intelligent-media-discovery-task-list.md)。
