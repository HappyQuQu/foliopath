# FolioPath 数据模型

CR-2026-051 / ADR-0037 / migration47：ocr_search_documents是ocr_asset_results的复合外键派生文档，独立rowid用于external-content FTS5 trigram索引；正文NFKC/casefold复用catalog。OCR提交结果、文档与checkpoint同事务，结果更新先移除旧文档再重建，删除/清理/资产库删除级联维护FTS。Open有界每批100条回填已存ready结果，含空白，不触发推理。全文索引/结果/设置/库状态和源相关字段触发同事务catalog_search_state revision，原有search cursor不继续混用旧证据。OCR关闭不删除正文；确认clear才删除结果与全文文档，历史保留策略仍未完成。

2026-10-26 / ADR-0036：迁移00046在OCR任务增加next_attempt_ms（busy背压延后）和独立ocr_job_handoffs（精确claim/尝试/最新checkpoint与计数，最多10收据/任务）。不改旧analysis_job_handoffs或ai_model_operations。计划handoff由原WorkerPool收尾调用，普通取消不退attempt；库取消优先，源失败/服务失败分开。当前模型/源统计单查询，empty⊆ready、stale⊆pending，元数据保留策略仍独立待验收。

2026-10-26 / ADR-0035 / CR-2026-049：迁移00045新增独立ocr_generations（完整身份JSON/指纹不可变）、ocr_library_settings（默认未配置即关闭、CAS revision与epoch/clearing）、ocr_jobs/ocr_job_requests（每库一个活动任务、上界/keyset/checkpoint/lease/fence/尝试上限/幂等），以及ocr_asset_results（复合库/代次/asset外键，当前源指纹、有界NFC行/凸框/准备尺寸、ready或固定源失败）。结果与checkpoint同事务；清理先隔离再每批≤500删除各代次正文。旧结果换代保留但不混入当前读取；相同身份重新启用保留进度。没有全文索引、公共接口或后台启动；详见[持久合同](architecture/ocr-persistence-contract.md)。

2026-10-26 / CR-2026-048：无迁移、OCR结果持久化/任务/checkpoint或代次激活。producer只返回本次有界文本/框和compatibility指纹；独立OCR权重/cache/runtime/input/output身份不重标旧向量/标签/人脸。缺OCRcache或模型失败不创建源错误行，取消不写应用状态；原件、人工标签和删除抑制不变。下一步源指纹/事务/清理另需接受。

2026-10-25 / CR-2026-047：无迁移/持久化。PreparedImage只返回内存PNG、归正/缩放后宽高和input contract hash，不猜测源指纹/代次，不生成OCR进度。buffer固定本次header/native读取的字节，但未来任务仍须复核源指纹并按fenced事务提交；不能把本片输出当成已入库索引或更新旧向量/标签/人物。

2026-10-25 / CR-2026-046：无迁移、结果表、OCR配置或任务写入。internal/ocr.Result仅是一次响应的有界内存值，Identity.Fingerprint是独立部署兼容指纹，不是数据库generation或持久记录。空数组不生成占位索引，故障不写源失败标记；没有旧向量/标签/人物代次转换。后续持久化须先接受源变化失效、fenced checkpoint与删除清理合同，不能复用本片临时值冒充全库进度。

2026-10-25 / CR-2026-044：无迁移。预检在单ReadOnly事务读取双active人物版本、来源全部anchor数量及现有禁止关系/重复lineage归属，引用原maxBatchSize边界。GetPersonMergeCheck与实际MergePeople通过同一readPersonMergeConflictCounts查询拥有者读取约束，写事务仍重新校验。照片示例只投影ready库中bound且有current_face_id的现存资产，跨库按各自opaque库/媒体ID映射；DISTINCT排序LIMIT3，不读取原文件或持久化新“允许”状态，预检不写审计/版本/缓存文件。

2026-10-25 / CR-2026-043：无迁移。cannot_link仍由原SQLite审核写事务持有face_cannot_links、审计和undo JSON快照；写入先重放，再在同事务核对检测lineage、两位active人物/版本及bound anchor，规范排序anchor pair。新快照附加两位人物版本及linkAnchors ID/版本；撤销先匹配这些后置状态再删除原link版本，拒绝后续改变。旧快照保留其实际link-only合同、无历史guard推断。单图人脸与cannotLinkFaceIds在同一有界只读事务投影，仅映射当前同图已归类脸，不持久化第二份关系。

2026-10-25 / CR-2026-042：无迁移/新存储。后端仍先重放face_audit_events，再核对原来源/目标版本；合并源已tombstone或移出后无照片不妨碍同键重放。新增专项核对split/非空merge重放不重复增版本且undo仍可恢复，移动沿用已有重放回归。前端恢复仅当前PersonPanel的单个内存意图，不持久化、不复活人物、不删除审计或原件；收起提示不取消已提交动作。

2026-10-25 / CR-2026-041：无迁移或持久化变更。撤销继续由face_review_undo_snapshots及SQLite原审核事务校验关联/人物/排除等后置版本，以补偿动作提交并保留原审计；同键先重放，不复活旧快照或静默覆盖后续人工状态。Web本页只保留一份成功回执及文字上下文，不写localStorage、不构造历史数据库；dismiss不删除服务端记录，创建/改名不归入face undo。

2026-10-25 / CR-2026-040：无迁移。排除继续由SQLite既有fenced审核事务拥有：先检查幂等重放，再读取检测lineage及current_face_id的人工归属，与请求来源/人物修订比较，匹配后才删除关联、递增人物修订并写排除/审核/undo快照。检测修订不随人工归类变化，不能单独作为归属锁；来源省略只断言当前未归类。冲突不得留下审核事件或排除记录，合法undo及已完成请求重放保持原语义；未扩展到模型代次/源文件变化的全局并发策略。

2026-10-25 / CR-2026-039：无迁移。Person.previewAssetId是只读派生字段；复用PersonAsset的personAvailablePhotoJoins/personBoundPhotoPredicate，按ready库内bound且current_face_id非空的照片，以mtime_ns DESC/id DESC选1项；无项返回null。人物的既有bound人脸/不同媒体计数不变，离线库关联仍保留计数，预览不冒充全部关联在线。解除最新照片全部脸的关联后可回退较早照片；无剩余照片时为空预览而不删除人物名称。字段不写入人工封面，也不改原件/模型/审核事务。

2026-10-25 / CR-2026-038：无数据库迁移。faceCurrentLineageJoins是SQLite人脸当前归类/排除投影的唯一共享连接定义，供单图、分组目录及详情计数使用。目录读取在同一只读事务中流式生成当前活动build的摘要、全范围计数和limit+1页面；保留最多一组/4预览及有界页面，不加载全库到内存。SHA-256摘要覆盖库/设置/覆盖修订、build及所选类型各人脸的修订与当前anchor/person/exclusion身份，绑定游标，避免相同总数下换脸漏检。每页扫描该类型成员，耗时随成员数增长；未宣称大容量性能通过。分组持久化历史和人工写入事务完全不变。


ADR-0031融合搜索不新增持久化状态/迁移，复用catalog的规范化键、source_fingerprint、当前semantic generation与向量。
来源与融合排名在有界只读遍历中计算；内部水合证据摘要只用于当前响应的一致性检查，不写入数据库或公共游标。
人工标签、删除抑制、语义/人物代次和后台任务均不因搜索请求改变。

ADR-0029不增加迁移：`tag.image_preprocessing`的canonical contract绑定标签model_identity，
缺省hash不变；新profile使旧run不计新覆盖。仍使用每资产单行`autotag_asset_runs`最新尝试，不承诺保留逐身份历史；
旧operation与部署前完整备份提供核对依据。原子curation合并不预清空旧标签，人工来源和移除抑制不受换代影响，
语义/人脸表与身份不变。见[记录](changes/FIX-2026-10-24-autotag-png.md)。

ADR-0028 不增加表或迁移：可选 PNG v4 输入合同进入 semantic `PreprocessingSHA256`，通过既有激活事务
生成新代次、退役旧代次。旧 embeddings 保留原 generation，新代次覆盖从 0 开始；不复制/重标向量。
仅语义身份变更，face/tag identity 与人工整理不变。缺省 v3 无迁移；模型配置回退不保证重激活旧 generation，
需匹配完整停写备份。语义视频/候选建议也绑定 generation，不能把旧结果视为新输入版本已完成。
见[兼容性与实测](changes/FIX-2026-10-23-semantic-large-png.md)。

ADR-0027 / migration 44：`analysis_job_handoffs` 按 kind/job/claim 唯一记录计划停机交接前的
attempt/checkpoint/completed；operation 删除级联，每个 operation 最多保留最近 10 条。
在同一 fenced 写事务中记录审计、交回 job/operation 为 queued 并撤销该次 claim 的一次 attempt charge，
不清零历史、不修改 checkpoint、模型/代次、标签/人工关系或已提交向量。重复/旧 fence、取消和终态不重排。
离线首升仅对停写精确快照做有界整批交接，不对历史失败任务补造计划停机事实。

CR-2026-033 无新表或迁移：标签 name/normalized_name、ID、人工来源和抑制记录保持原状。
`Tag.localization` 是 curation 固定类别 JSON 加可验证 AI 关联的只读投影，不是持久改名；全局
没有 AI 或存在人工关联时不翻译。单图按关联来源返回元数据。SQL 在 keyset 分页之前展开有限别名并保持
库/种类约束；人工中文/英文同义保护在原有 fenced AI 合并事务内查询，不把别名数当作关联数量。
浏览器偏好使用原有 storage owner，不写 `/app/data` 或触发模型任务。

2026-10-23 / FR-INT-009/016 / migration 43：face_asset_failures 保存 generation/library/asset 主键、当前失败 job、源指纹、
固定原因码和时间；与现有 checkpoint 同一 fenced 短事务提交。成功删该图失败标记，源变化不将旧错误当作当前源错误。
按最新 job 隔离当前尝试展示，旧未恢复标记仍参与 missing 选择，不能被旧成功 result 屏蔽；库/资产/任务/generation 删除级联，
derived-clear 有界删除且不增加原公开影响计数。最近任务/进度/诊断在同一读快照读取；历史 coverage 无原因不会补造明细。

2026-09-16 A05 归属修复：重新归类复用 person_face_anchors 的同一 ID，不删除历史引用。现有审核事务内校验/推进旧人物和目标人物 revision，更新归属并写 undo 快照；快照新增可选 beforePersonId，沿用 sourcePersonId/sourcePersonRevision 记录旧人物的提交后版本。旧快照缺省 owner 时使用原 payload.personId。撤销同时校验双方版本和 anchor，不覆盖后续编辑；cannot-link 转移失败保留原状态。无新表或迁移。[证据](evidence/int-001/face-reassignment-2026-09-16.md)。

P3 恢复补修不新增表：同代次等待模型的库经既有设置 revision/可用性事务重入 building，再创建新任务；旧失败任务和幂等决定保留。部署配置不再匹配的旧 service 模型只改变 availability，不退役/删除原代次、进度、人物或人工关系。迁移 40 有 service 人脸记录时降级拒绝，不能用删记录方式绕过。

2026-09-16 P2（FR-INT-009 / CR-2026-030，Owner face/SQLite）：无新增表。新任务可将库的目标代次推进至当前部署代次，禁止在该库已有活动分析任务或清理中重定向。读取使用已发布 `active_cluster_build_id` 的代次，不以正在构建的目标代次隐藏旧结果。重建按有界 keyset 批次核对人工 anchor/exclusion 的库、资产、源指纹和量化框；仅唯一精确匹配重新绑定，源变化留待人工复核，不用 embedding 猜测身份。手工清理先分批移除本库 undo snapshot，再按引用叶节点删除审核事件，避免 RESTRICT 外键阻止清理；保留全局人物名称与其他库状态。

2026-09-16 P1：service 人脸代次在任务准入、进度提交、聚类可见切换及成功终态事务内复核模型可用性；不新增表。通用 AI 操作启动恢复不再处理四类人脸操作，由 face_analysis_jobs/face_clear_jobs 的既有租约和有界尝试负责恢复，避免操作行被提前失败而任务行仍可运行。

2026-09-16：远程人脸代次通过 `model_id` 关联 service 模型的完整兼容指纹（运行时/缓存/输入与质量规则/组件/阈值 profile），同时核对 FaceGeneration 组件与 profile hash。注册模型的 content_hash/source_identity 必须一致；沿用原激活事务，不新增表。失败不退役旧代次，成功也不提前切换各库可见聚类或 CLIP 指针。见 [提交证据](evidence/int-001/face-deployment-commit-2026-09-16.md)。

2026-09-16：migration 40 允许 `ai_models.storage_mode=service` 的人脸模型；继续要求非 installed 状态和 source_identity=content_hash。只新增迁移，不修改 39。旧本地人脸激活提交拒绝 service，后续须由身份绑定入口提交；不因此自动创建人脸代次。回退 39 遇到人脸 service 记录会由约束失败关闭，不能删除记录强行回退。

2026-09-15 视频删除覆盖率：只有成功扫描可按剩余完整当前故事板重算汇总。
ready 不高于删除前计数，避免全量重建提前复用旧结果；无法归属到具体资产的旧
degraded/failed/stale 汇总重置为待处理差额，历史操作记录不变。无新表/迁移。
这不是按资产记录失败结果，精确归属仍未实现。

服务代次激活（2026-09-15）在既有激活事务内创建 building 行、绑定服务空间、转 active，
旧 active 退役、全局活动指针与操作成功一起提交。身份必须匹配 service 模型 content_hash，
维度与模型可用性 revision 均复核；无额外 migration。旧向量保留，失败整体回滚。
已有文件激活提交不能用于 service；此能力尚未由 app 调度。

2026-09-15 migration 39：ai_models.storage_mode 增加 service，使用与 migration 35 相同的
事务表重建流程保留旧列、行和引用。服务模型仅 semantic_image_text、available/unavailable，
source_identity 必须等于 content_hash（已核验空间指纹），不能存路径/URL。
package_size_bytes 为部署固定模型字节量，不代表主应用本地磁盘占用。
普通文件安装/刷新/激活不拥有此类型；内部注册不自动创建或切换代次。
带 service 行的降级拒绝并保留数据；不承诺旧版本可直接打开新数据，回退使用升级前备份。

POST-MVP-5 / ADR-0018：迁移 00037 新增 model_proxy_settings singleton（id=1）。mode 为 inherit/direct/custom，
proxy_url 只在 custom 非空，revision 每次保存递增；同一 SQLite 写入门串行，旧 revision 返回冲突。
配置是 /app/data 应用状态，不属于媒体库；重启不覆盖，不存代理凭据。

## 原则

- 文件系统决定媒体是否存在以及目录如何组织。
- SQLite 保存媒体库配置、可重建索引、扫描状态和应用元数据。
- 所有目录、媒体、缩略图和扫描记录都必须归属于 `library_id`。
- 数据库不保存宿主机绝对路径，也不保存缩略图 BLOB。
- 媒体首版逻辑身份为 `library_id + normalized_relative_path`。

## 核心表

### 模型下载历史扩展（migration 36）

CR-2026-026 复用 `ai_model_install_requests` 与 `ai_model_operations`，不新增下载任务状态机或保存权重 BLOB。
目录 ID 作为 `candidate_id`，排队候选固定 manifest 与 `download:<contentHash>` 来源标识；网络 URL
只存在部署配置，不进入公共任务 DTO。migration 36 添加 `(candidate_id, created_at_ms DESC,
operation_id DESC)` 索引，支持稳定读取每个目录项最近任务。托管安装的 completed/total 字段以字节计，
进度最多每 500ms 提交一次（初始/最终除外），终态保留进度，历史可跨重启读取。相同幂等键重放，
另一键遇到同模型活动下载返回冲突；取消/失败后新键重新下载。运行中任务的恢复仍由既有 operation owner
负责，不承诺断点续传。文件发布、注册、激活保持独立步骤，已有模型与原媒体不因下载失败被删除。

### `libraries`

- `id`：不透明稳定 ID。
- `name`：用户设置的非空 NFC 显示名称，去除首尾空白后最多 128 个 Unicode code point，
  且不允许控制字符。
- `name_key`：对显示名称执行 NFKC 后再做 Unicode full case folding 的实例唯一比较键；
  不用于替换或重写用户看到的显示名称。
- `name_sort_key`：服务端生成的 locale-neutral、numeric-aware 自然排序派生键；只用于
  `(name_sort_key, name, id)` keyset 分页，不参与名称唯一性，也不包含路径。
- `root_rel_path`：相对于 `/library` 的规范化根路径；空字符串唯一表示 `/library` 本身。
- `status`：`pending`、`scanning`、`ready`、`offline` 或 `error`。
- `current_generation`：最近一次成功完整扫描的代次。
- `revision`：任一公开 Library 表示字段变化时递增的正整数，用于生成强 ETag。
- `created_at`、`updated_at`。

`root_rel_path` 必须是最长 4096 个 Unicode code point 的规范相对路径并保持唯一。空值表示
`/library`，因此会与任意其他库重叠。业务层在 SQLite immediate 写事务中按路径组件拒绝
相同、祖先和后代重叠，不能用裸字符串前缀判断。MVP 允许更新 `name`，但
`root_rel_path` 创建后不可修改；数据库 trigger 同样阻止直接更新。更换根路径通过移除
媒体库并重新创建完成，详见 [ADR-0004](adr/0004-library-root-immutable.md)。

### `directories`

- `id`、`library_id`、`parent_id`。
- `relative_path`：相对于媒体库根目录。
- `name`。
- `natural_name_key`：与媒体自然文件名排序使用同一经过验证的规则。
- `last_seen_generation`。
- `direct_asset_count`、`recursive_asset_count`：成功 finalize 时维护的必需、可重建聚合统计；扫描中可保持上次可靠值。
- 可选的目录修改时间和直接子目录数；是否持久化由查询/容量证据决定。

唯一约束为 `(library_id, relative_path)`。`parent_id` 用于目录树和面包屑，根目录的相对路径使用统一的空路径表示。
根目录数据库行的 `name` 可以为空，但 API 必须由 catalog capability 将其映射为当前媒体库显示名；
不得由 HTTP handler 或 SQLite adapter 各自发明根节点表示。migration 7 已追加
`natural_name_key` 及目录 keyset 索引，没有修改已发布的初始 migration。

### `assets`

- `id`、`library_id`、`directory_id`。
- `relative_path`、`name`。
- `natural_name_key`：服务端定义并经 fixture 验证的自然文件名排序键。
- `kind`：图片、动画图片或视频。
- `mime_type`、`size_bytes`、`mtime_ns`。
- `width`、`height`、`duration_ms` 等可空媒体属性。
- `source_fingerprint`：migration 6 起为非空版本化
  `v1:<size_bytes>:<mtime_ns>`，用于派生数据失效；它不是内容哈希或去重身份。
- `probe_status`、`probe_error_code`。
- `playback_status`：`playable`、`unsupported_codec`、`unknown` 或 `not_applicable`。
- `last_seen_generation`。

唯一约束为 `(library_id, relative_path)`。重命名在首版表现为新路径新增，并在成功完整扫描后清理旧路径；不承诺依赖 inode 自动识别跨路径移动。
名称序 v2 使用从 `relative_path` 与 `name` 派生的来源文件夹路径作为第一排序项：库内为
`(directory_path, natural_name_key, name, relative_path, id)`，跨库再以 `library_id` 为首项；
修改时间序使用 `(mtime_ns, id)`。`directory_path` 只在查询时派生，不新增持久化权威字段；
migration 19 为完整 v2 tuple 增加表达式索引。
Post-MVP/1 的文件大小序使用 `(size_bytes, id)`；migration 14 在不修改历史 migration
的前提下将 `avi` 加入 `media_format` CHECK，并增加目录、媒体库与全局大小排序索引。
`OFFSET` 不是容量档下的可接受实现。migration 8 追加媒体属性和 probe/playback 状态；
已有资产回填为待探测状态，不把未知值伪装成处理成功。

### `thumbnails`

- `library_id`、`asset_id`、`variant`。
- `source_fingerprint`、`transform_version`。
- `cache_rel_path`：相对于 `/app/data/cache`。
- `status`：`pending`、`ready` 或 `failed`；失败只保存稳定 `error_code`。
- `width`、`height`、`byte_size`、`created_at`、`last_accessed_at`。

唯一约束为 `(library_id, asset_id, variant)`，并以复合外键保证 asset 属于同一媒体库。
缓存键必须包含源指纹和变换版本。每库派生文件固定放在
`libraries/lib_<library-id>/` 子树内，使 removal worker 无需接触 `/library` 即可幂等清理。
文件先原子落盘，再提交可用状态；数据库不得把不存在或未完成的缓存文件标记为可用。
migration 8 已实现该表和 ready/failed 状态约束；durable media job、LRU 与访问时间刷新
由 migration 9 和 `internal/thumbnail` 缓存策略实现。S3-007 已在 ready/304 HTTP 命中时
刷新访问时间；若数据库 ready 文件缺失或长度不符，会在短事务中撤销 thumbnail ready、
重置 asset probe 并把同 fingerprint media job 归零重排，不在请求线程同步生成。

### `media_jobs`

- `id`、`library_id`、`asset_id`、唯一 `variant=grid`。
- `source_fingerprint`：领取和终止提交都必须匹配，旧 worker 不能覆盖新源。
- `status`：`queued`、`running`、`succeeded` 或 `failed`。
- `available_at_ms`、heartbeat、lease、attempt（最多 3 次）和稳定 `last_error_code`。
- `created_at_ms`、可空 start/finish 时间。

唯一约束为 `(asset_id, variant)`，复合外键保证同库归属。`media_job_library_state` 只保存
跨库公平领取游标；它不是任务事实来源。`cache_deletions` 保存 fingerprint 失效后必须
幂等清理的相对缓存路径，绝不保存或删除原媒体路径。

`POST-MVP-3` 首个失败诊断切片不新增媒体结果副本：日志列表由 `media_jobs` 联结 `assets` 和
`libraries` 的既有终态事实生成，并按 `media_jobs.id DESC` keyset 分页。手动恢复在短写事务中
把最多 256 个 transient failed row 重置为 queued；`invalid_media` 和 `unsupported_media`
保持终态。Header 最近完成的清除状态只存在当前浏览器 localStorage，不属于可靠任务历史。

### `media_job_attempts`

Migration 18 为诊断启用后的派生任务追加有界尝试事实：`job_id`、`attempt_number`、outcome、
失败 stage、稳定 reason code、处理 tool、可空 exit code、duration 与 finish time。每次任务
结束在同一事务内更新 `media_jobs` 并写入尝试；每个 job 只保留最近 10 条。表不保存 media path、
命令参数、stderr、SQL 或 stack，媒体相对路径仍由查询时联结 canonical asset 获得。

migration 17 追加 `system_events` 作为系统级运维事件事实：保存 `info`/`warning`/`error`、
稳定模块与事件码、request ID、HTTP method/route pattern/status/duration 和发生时间。每次追加
后把实例历史裁剪为最新 5,000 条；列表按 `id DESC` keyset 分页。表中不保存请求参数、任意
错误文本、媒体/宿主路径、header、SQL、stack 或 subprocess 输出。扫描历史仍由 `scan_runs`
拥有，逐媒体处理结果仍由 `media_jobs` 拥有。

### Post-MVP `storyboard` 派生

[FTR-VID-001](features/video-storyboard-preview.md)已通过只向前 migration 11 让
`thumbnails` 和 `media_jobs` 支持 `variant=storyboard`；历史 migration 8 与 9 未修改。

已实现数据语义：

- 保留 `(asset_id, variant)` 唯一身份、source fingerprint 和 transform version；
- storyboard ready 状态保存实际
  `frame_count/columns/rows/cell_width/cell_height`，grid 行不伪造这些值；
- grid/poster claim 优先于 storyboard，同时维持现有跨库公平、lease、有限重试和取消；
- 历史视频使用有界分批 admission，不在一个长事务或内存批次中创建全部任务；
- storyboard 独立参与 LRU，淘汰它不影响 grid poster；
- 文件发布、源变化 CAS、cache missing、library removal 和重启恢复继续沿用现有派生一致性。

精确 schema、CHECK、claim tuple 与索引已由
[`VSP-S1 Contract Ready`](gates/POST-MVP-1/vsp-s1-contract-ready.md)冻结，并由只向前
`migrations/00011_video_storyboards.sql` 实现。fresh schema、migration 10 upgrade、旧 running
lease 保留、安全 downgrade/fail-closed 与 `integrity_check` 已有自动测试；
`VSP-S2` 的 Linux 100k/10k、10% 视频容量档已通过。

### `scan_runs`

- `id`、`library_id`、`generation`。
- `status`：`queued`、`running`、`succeeded`、`failed`、`cancelled`、`offline` 或 `interrupted`。
- 创建时间，以及可空的开始、结束和心跳时间；排队任务尚无开始时间。
- 已发现目录数、媒体数、错误数和可安全展示的错误摘要。
- 跳过目录/文件数、取消请求时间和安全取消原因。
- `revision`、`phase`、`processed_assets`、分离的 skipped directory/file 与 error 计数，
  以及 issues 是否截断。
- `available_at_ms`、heartbeat、lease、attempt count 共同构成 restart-safe durable
  admission；进程内 channel 只负责唤醒。

同一媒体库最多有一个 `queued` 或 `running` 的完整扫描。失败代次不能执行媒体库级陈旧记录清理。
创建媒体库时，库记录、唯一 `library_created` queued scan 与创建幂等记录在同一个短事务
提交；提交后才唤醒 worker。

### `scan_issues`

- 归属于 `scan_run_id`，只保存稳定 code、正的聚合 count、可空且有界的媒体库相对示例
  路径和创建时间。
- 每个 scan 最多 50 个聚合组；超额由 `scan_runs.issues_truncated` 表示。
- 不保存本地化 message、errno、stack、媒体工具 stderr、宿主机或容器绝对路径。

### `library_removals`

- `id`、原媒体库 ID 和安全名称快照。
- `status`：`queued`、`running`、`succeeded` 或 `failed`。
- `revision`、创建/开始/完成时间和可空安全错误码。

每个媒体库最多有一个 queued/running removal。terminal removal 不随 `libraries` 删除而
级联，确保配置和派生状态清理后仍可轮询；它不保存媒体库根、宿主机路径或任何原媒体删除
指令。

### `idempotency_records`

- `operation`、32 字节 `key_hash` 和 32 字节规范请求 `request_hash`。
- 结果类型/ID、创建与过期时间；保留期至少 24 小时。

唯一键为 `(operation, key_hash)`。数据库不保存 `Idempotency-Key` 明文或原始请求体；相同
key/不同 request hash 是稳定冲突，相同逻辑结果已被删除时不得重新执行原操作。

### `media_jobs`

- `id`、`library_id`、`asset_id`、`kind`。
- `source_fingerprint`、`status`、`attempts`、`available_at`。
- `error_code`、时间戳。

任务必须幂等。进程重启时，超时的 `running` 任务可以安全返回队列；源指纹已变化的任务应被丢弃或替换。

### `users`

- `id`、固定且唯一的 `singleton_key=1`、原始规范化登录名 `username`、唯一比较键
  `username_key` 和显示名。
- setup 后存储的 `username` 使用 Unicode NFKC；`username_key` 是对 NFKC 值执行
  Unicode full case folding 的结果。认证只比较 `username_key`。
- 密码只保存 `password_hash` verifier、`password_scheme` 和 `password_parameters`，
  不保存明文密码。当前 verifier 为严格 Argon2id v19：64 MiB、3 次迭代、4 lanes、
  16 字节随机 salt、32 字节派生 key。
- `auth_version`、`created_at_ms`、`updated_at_ms`、`password_changed_at_ms` 和可空
  `disabled_at_ms`。

MVP 只允许创建一个管理员。首次初始化已通过 `internal/auth` 状态机、SQLite 写事务和
singleton 约束防止并发创建多个账号；应用重启后从数据库恢复 setup 状态。是否允许未来
多用户不能通过绕过该约束提前实现。认证边界见
[ADR-0005](adr/0005-built-in-single-admin-auth.md)。

### `sessions`

- `id`、`user_id`、唯一 32 字节 `token_hash` 和 32 字节 `csrf_token_hash`；数据库不保存
  浏览器持有的明文令牌。
- `created_at_ms`、`last_seen_at_ms`、`expires_at_ms` 和可空 `revoked_at_ms`，由检查约束
  保证正的绝对期限与时间顺序。
- `auth_version` 必须与管理员当前认证版本匹配，用于改密、退出和安全事件后的整体撤销；
  删除管理员会级联删除 session。

当前实现每次认证产生两个独立 32-byte 随机秘密。HttpOnly Cookie 是认证秘密与 CSRF
秘密组成的 64-byte 不透明值；页面只取得 CSRF 部分，因此不能由 CSRF 值还原 Cookie。
`token_hash` 是整个 Cookie 的 SHA-256，`csrf_token_hash` 是 CSRF 部分的 SHA-256。
会话固定 7 天绝对期限且不滑动续期；重新认证整体轮换，退出写入撤销时间。
过期/撤销记录在 24 小时宽限后由创建新会话的写事务清理。数据库不得保存可直接复用的
明文令牌，这一不可回放边界不能被后续 handler 或 middleware 改写。

### `settings`

使用固定 `singleton_key=1` 的 typed row，只保存 schema 已知的应用级配置，包括默认
24 小时完整扫描周期（允许 1～8760 小时或 null 关闭）、默认 10 GiB 缩略图缓存配额，
默认开启的 `automatic_discovery_enabled`、默认 2 且限制 1～4 的
`background_concurrency`、默认 8 且限制 1～16 的 `content_read_concurrency`，以及默认
跟随浏览器的中英语言偏好。migration 16 的 `resource_profile` 仅为历史迁移兼容列，不再是
运行时设置事实来源。
`revision` 支持强 ETag/If-Match，提交后才唤醒对应 scheduler/watcher。秘密值不得以明文
日志输出；设置不能成为任意键值存储。

缓存 ready 使用量达到配额 90% 后按 `(last_accessed_at_ms, asset_id)` 清理到 80%，每次
新发布还必须在写入后保留 512 MiB 文件系统可用空间。该策略只作用于可重建文件和
thumbnail 状态，不以清理数据库、设置或原媒体换取空间。

## 索引与查询

至少建立：

- 目录树：`(library_id, parent_id, name, id)`。
- 目录自然排序：`(library_id, parent_id, natural_name_key, id)`。
- 稳定路径浏览：`(library_id, relative_path, id)`。
- 目录媒体列表：`(library_id, directory_id, name, id)`。
- 日期排序：`(library_id, mtime_ns DESC, id DESC)`。
- 扫描清理：`(library_id, last_seen_generation)`。
- 任务领取：`(status, available_at, id)`。

媒体列表统一使用 keyset cursor，不使用 `OFFSET` 承担大型列表分页。游标编码当前排序字段和稳定 ID，并视为不透明、可校验的 API 值。

文件名和路径搜索使用 FTS5 派生索引，并与 `assets` 变更保持同一事务语义。S4 搜索
profile v1 已固定并由 migration 10 实现：`assets.search_name_key` 与
`assets.search_path_key` 分别保存可重建的 Unicode NFKC + full
case-fold 搜索键，按 Unicode 空白拆出的全部词执行字面子串 AND 匹配，保留变音符号并支持
一至二字符词；不得把用户输入直接解释为 FTS 查询语言。外部内容 FTS5
`asset_search` 使用 case-sensitive trigram tokenizer 为至少三字符的安全词生成候选集，
每个词最终仍由规范键上的 `instr` 精确确认；短词或含双引号、不能安全成为 FTS anchor 的
查询直接走相同精确谓词，因此 tokenizer 不是行为权威。

索引支持当前目录（直接或递归）、当前媒体库和全部媒体库三种作用域，以及媒体类型和 filesystem
mtime 的半开区间筛选。migration 10 的 singleton `catalog_search_state.revision` 在媒体库创建/移除或
可靠 full-scan generation 发布时，在同一提交语义中推进 revision，供跨库 cursor 绑定；
它不是扫描中间批次的快照版本。库内 cursor 继续绑定对应媒体库的可靠 generation。
asset insert/update/delete trigger 维护 FTS 行，scanner 在资产 upsert 的同一事务写规范搜索键；
启动升级以有界批次回填旧资产，并由 migration 5→10 重开测试验证 FTS 可查询。每次启动在
回填后执行 external-content integrity check；派生 FTS 不一致时以序列化短事务 rebuild
并再次校验。取消会终止恢复，rebuild 或复核失败则应用启动失败关闭，但不得删除 `assets`
权威索引。

### FTR-UIF-001 已实现的数据决定

`UIF-S1` 决定并由 `UIF-S2` 实现只追加 `00013` migration；migration 1～12 未修改：

- `users` 追加 `revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0)`。资料更新只推进
  account revision；改密在同一事务推进 account revision 与既有 `auth_version`，并把当前
  session 提升到新 auth version、撤销其他 session。
- `directories` 追加 `search_name_key TEXT NOT NULL DEFAULT ''`。catalog owner 按 search
  profile v1 派生和有界回填；scanner upsert 同时写入。查询仍先使用
  `directories_browse_children` 限定 parent，再用 `instr` 精确匹配，不新增 directory FTS。
- 追加单行 `cache_cleanup_state`，保存 revision、`idle/queued/running/succeeded/failed`、
  时间、初始/剩余/释放字节、删除项数和安全 error code。它只表示当前/最近一次清理，不是
  通用任务表或历史。
- 复用既有 `idempotency_records` 保存 cleanup operation 的固定摘要与响应；不保存明文 key、
  请求秘密、路径或文件列表。

`UIF-104` 的 10k 直接子目录末尾命中实测 P95 为 1.981167ms，查询计划使用 parent-scoped
browse index；详见 [UIF-001 spike](spikes/uif-001-directory-filter.md)。Fresh、12→13
upgrade、失败回滚、backfill 取消/重启、`integrity_check`、100k ready cache cleanup 和
四核/4 GiB 并发复验均已由
[UIF-S2 Backend Evidence Ready](gates/MVP-2026-07-23/uif-s2-backend-evidence-ready.md)
关闭；最新 10k/100k 扫描期并发结果见
[`UIF-405`](evidence/uif-405/README.md)。

## 扫描一致性

完整扫描开始时分配新 generation；分批 upsert 时更新 `last_seen_generation`。只有完整扫描成功后，才能删除更旧 generation 的目录和媒体记录。根目录离线、权限失败、任一子树无法可靠遍历、进程中断或用户取消都会使本次扫描失去清理资格。

增量扫描只更新明确检查过的路径，不进行媒体库级清理。定期完整扫描负责最终校准。详情见 [ADR-0003](adr/0003-scan-consistency.md)。

## Post-MVP 自动发现持久化

只向前 migration 12 扩展以下状态：

- `libraries` 增加 `automatic_discovery_status`、可空
  `automatic_discovery_error_code`、可空 `last_automatic_discovery_at_ms` 与正整数
  `content_revision`。状态/错误组合由 CHECK 限制；已有库以 `disabled`、无错误、revision 1
  升级，运行时依据设置、平台和 watch 建立结果转为 active/unsupported/degraded。
- `catalog_search_state` 增加独立正整数 `content_revision`。媒体库创建/移除、可靠
  full generation 发布和成功定向提交在同一写事务推进它；原有 `revision` 继续只绑定跨库
  搜索 cursor。
- `settings` 增加值域为 0/1、默认 1 的 `automatic_discovery_enabled`。
- 新增 `catalog_reconcile_jobs`。身份为
  `(library_id, relative_dir_path)`，路径只允许规范的媒体库相对目录，空字符串表示库根；
  行保存 `queued|running|failed`、`requested_revision`、可空 `claimed_revision`、
  debounce/available/lease 时间、有限 attempt、稳定错误码和创建/更新时间，不保存宿主机
  或容器绝对路径。

同路径事件使用 upsert 递增 `requested_revision`。claim 在短写事务中把当时水位复制到
`claimed_revision`；成功提交后，只有 `requested_revision=claimed_revision` 才删除任务，
否则清 lease 并重新排队。每库完整扫描与定向任务 claim 互斥；首版完整扫描不删除 queued
任务，完成后允许它们再次安全枚举。最多尝试 5 次，退避为 1/2/4/8/16 秒；耗尽、overflow、
资源不足或无法可靠枚举时保留旧索引、把库标记 degraded，并合并请求一次完整扫描。

每次定向事务只提交已可靠枚举范围的目录/资产、直接和递归计数、派生任务、
library/global content revision 与任务水位。文件系统 I/O、媒体探测和 watch 注册均在事务
之外。只有可靠完整 generation 可以执行媒体库级 stale cleanup；定向删除仅限已确认缺失的
直接子项或其子树。

## 删除与离线语义

- 删除媒体库：删除其配置、索引、任务和派生缓存，不修改原文件。
- 删除媒体或目录：可靠定向校准和成功完整 generation 都必须先在同一事务把 ready 派生路径
  写入 `cache_deletions`，再删除索引；提交后异步、幂等删除物理缓存。失败、取消、离线或
  部分不可读扫描不获得该资格。
- 媒体库离线：保留全部索引并记录状态；恢复后重新扫描。
- 媒体文件打开失败：返回稳定错误并触发后续校准，不在读取请求中直接大范围修改索引。
- 缓存缺失或损坏：将对应派生状态恢复为待处理，原媒体记录保持不变。

## 迁移与备份

## Post-MVP/4 收藏与标签持久化

2026-10-22 CR-2026-031 T1：不改已有表。`GET /tags` 可按库及媒体种类联结 `asset_tags` 与 `assets`，
以 distinct asset ID 统计，过滤时不返回零关联标签；无过滤保持全局词表兼容。图片包含 image/animated，
不含 video；游标绑定库、规范化种类、搜索与 curation revision。后续 T3 只追加以下 migration41/42，
旧人工关联（含人工接受的建议）不得重新解释为模型结果。

migration 21 追加 `curation_state`、`asset_favorites`、`tags` 与 `asset_tags`。收藏和资产标签
使用 `(library_id, asset_id)` 复合外键指向 catalog asset，并在可靠资产删除或媒体库移除时
级联清理；offline、失败、取消和部分扫描不删除可靠 asset，因此也不删除整理数据。

`tags.normalized_name` 使用 `internal/curation` 生成的 NFC、Unicode case-fold 唯一键；显示名
保留规范化大小写。每资产最多 20 个关联，由 capability 与单事务替换共同约束。
`curation_state.revision` 是全局单调失效令牌，触发器覆盖正常写入与 FK cascade；它不是事件
数量。收藏按 `(created_at_ms, asset_id)`、标签名按 `(normalized_name, id)`、标签资产按稳定
媒体 tuple 使用 keyset，不使用 OFFSET。

数据库迁移只向前自动执行，已发布迁移不得修改。备份 SQLite WAL 数据库时必须使用 SQLite 认可的在线备份或先完成安全 checkpoint/停机流程，不能只复制主数据库文件而忽略相关状态。缩略图缓存可重建，可以与不可丢的配置、管理员凭据和应用设置采用不同备份策略。

### 自动标签与人工保护（CR-2026-031 / ADR-0021）

权威定义为 migrations 41/42、`internal/curation`、`internal/autotag` 与
[`api/openapi.yaml`](../api/openapi.yaml)；设计理由见 [ADR-0021](adr/0021-bounded-inference-and-automatic-tags.md)。

- migration41 扩展唯一可见关联 `asset_tags`：`source=manual|ai`，AI 必须带模型身份、源 fingerprint、
  finite score；manual 的三个 AI 字段必须为空。旧行默认 manual，旧建议数据不自动转正。
- `asset_tag_suppressions` 以 `(library_id,asset_id,normalized_name)` 为键，不依赖 tag ID 或模型代次。
  人工移除写抑制；显式添加清同名抑制并升为 manual；替换列表按差量保留未变来源。全局删除只保护
  当前关联图片；全局改名（含同规范名的字形修改）保护旧名并将既有 AI 升为 manual。实际资产/库删除
  级联清除，离线不清除；同一资产源 fingerprint/kind 改变只清 AI 关联，人工与抑制保留。
- 合并只替换当前图 AI，人工优先，总计最多 20；相同规范名去重。成功零标签可清旧 AI，失败不能伪装
  成零标签。全局 `curation-rN` CAS 兼容保留；已是 manual 的单个添加可幂等返回，发生来源或关联变更
  必须当前 revision。单图及整理列表在同一读取快照内读取 revision 和关联，避免混合提交。
- migration42 增加 `autotag_library_settings`（enabled/revision/coverage_revision/blocked_model_identity）、
  `autotag_jobs`（operation、model、mode、checkpoint、准入上界、claim、attempt、lease、next_attempt_ms）、
  `autotag_job_requests`（哈希幂等键）及 `autotag_asset_runs`（每图最近源/模型/outcome）。
  每库最多一个活动任务、全局最多 32；missing 只选没有同源同模型运行记录的图片，all 可重试已有记录。
- 任务级失败或取消持久记录被阻止的模型身份；同模型自动准入不建新任务，显式 missing 返回冲突。
  显式 all 或重新启用清此限制；新模型身份不匹配旧限制，可重新处理。逐图失败仍由 all 重试。
  停机不冒充用户取消，保留 running 供租约恢复；最多三次 claim，耗尽终态失败并阻止自动重入。
- 推理暂忙不属于上述失败：单图 3 次尝试仍忙后，原 job/operation 进入 queued，`next_attempt_ms`
  持久延后 30 秒；不变更标签、run、checkpoint、计数或 blocked identity。原子增加 claim fencing、
  退还本次领取的 attempt（保留历史崩溃计数），到期领取才恢复；关闭/取消仍立即拦截，重启不跳过等待。
- SQLite 一个短写事务校验开关、running 状态、operation、claim、checkpoint、上界、模型与租约，
  复核当前资产 fingerprint/kind 后合并 curation，再写 run/checkpoint/operation 计数。实际媒体打开
  与推理在事务外，worker 在推理前后经原有媒体边界重开验证；错误不得留下半次合并。
- coverage 按当前部署模型身份及源 fingerprint 统计；旧模型 ready 不能算新模型完成。模型变更不会
  把旧 AI 来源直接改成新身份；后续成功单图合并替换旧 AI，人工与删除抑制始终保留。

这些来源与抑制属于需要备份的应用状态，不是可随意清掉的推理缓存。最终镜像/重启证据由
[CR-2026-031](changes/CR-2026-031-automatic-photo-tags.md) 汇总，schema 存在不等于发布通过。

## POST-MVP-5 revision 1：模型与图片语义数据合同

S1 冻结 schema 语义；S2 已以只追加 migration 22/23/24
建立模型、generation、setting、embedding、progress、backfill/clear job 基础表与 durable semantic request/attempt 状态，
禁止修改此前已提交 migration。

| 表 | 身份与关键约束 | 数据分类与删除 |
| --- | --- | --- |
| `ai_models` | opaque ID；精确 `purpose=semantic_image_text|face_detection_embedding`；package/version/arch/content hash 唯一；state、storage mode、availability revision | 模型配置/运行状态；备份；删除不能触碰 direct source；purpose 不可混淆 |
| `ai_model_operations` | opaque operation ID；kind、state、progress、stable error、lease、revision；不保存路径/URL；Get/create/transition 返回时均由 operation owner 校验 identity/owner/revision 绑定及 queued/active/terminal phase、finished time、error-code 一致性 | 可清理运维状态；保留最近终态由配额策略控制；不一致或串单的持久化结果失败关闭 |

不经过 `OperationService` 的安装/激活 admission 结果必须在 `ManagementService` 出口执行相同 operation
校验，并绑定正确 kind、激活 model ID、Created/Replayed 语义和新建 revision-1 queued 状态。
具体 install/activation admission 还必须在发出 worker wake signal 前验证 queue 返回的
idempotency/request、candidate/model owner、availability revision 和 operation；失败不得唤醒 worker。
| `ai_library_settings` | `library_id` PK/FK；enabled、revision、active generation ID、coverage revision | 应用设置；库删除 cascade；禁用默认不删除 embedding |
| `semantic_generations` | opaque ID；model ID、transform/output/index version、state、created/activated time；最多一个 active | 可重建派生元数据；旧代延迟清理 |
| `semantic_embeddings` | `(generation_id, library_id, asset_id)`；固定 dimension/blob length；source fingerprint | 可重建派生；asset/library/generation 删除 cascade |
| `semantic_library_progress` | `(generation_id, library_id)`；eligible/completed/failed/stale counts、checkpoint、revision | 可重建聚合状态；不得用失败数伪装完整覆盖 |
| `semantic_jobs` | opaque batch ID；library、generation、mode、state、lease/checkpoint、attempt、stable error、requested/claimed revision | restart-safe 派生任务；最多 claim 3 次，不保存 query 文本或逐资产错误原文 |
| `semantic_job_requests` | SHA-256 idempotency-key digest、固定 request digest、job ID、创建时间 | 幂等回放/主动同模式合并；不保存明文 key、请求体、路径或 query 文本 |
| `semantic_clear_jobs` | opaque clear ID；library、operation、state、lease、attempt、requested/claimed revision | restart-safe 有界清除 tombstone；不依赖 active model/generation，不删除原媒体或用户策展数据 |
| `semantic_clear_requests` | SHA-256 idempotency-key digest、request digest、expected settings revision、clear job ID | 强 ETag 清除意图的幂等回放；不保存明文 key 或确认请求体 |

复合 FK 必须延续 catalog 的 `(library_id, asset_id)` 边界，避免跨库引用。active generation 使用数据库
唯一约束加 compare-and-swap revision 保证单例；模型记录不能通过 cascade 意外删除仍被 active 或
recovery generation 引用的包。embedding blob 只能由 semantic repository 读写，API 和诊断查询不得
选择该列。revision 1 的 embedding blob 是固定维度、IEEE-754 binary16 little-endian；写入前先以
float64 累加平方和并 L2 normalize，读取 binary16 后必须再次 L2 normalize 再参与评分。NaN、Inf、
零向量、维度或 blob 长度不匹配均拒绝，不得把损坏行当作零分结果继续查询。

backfill 页提交必须在同一短事务写入 embedding、推进 `semantic_library_progress` 计数/revision/checkpoint、
推进 `semantic_jobs.checkpoint_id` 和公开 operation progress。提交同时匹配 claimed revision、progress
revision 与旧 checkpoint；任一不匹配、FK 或计数约束失败均整体回滚。解码和推理不在事务内执行。

回填入队从 `assets` 计算 image/animated eligible 数；`missing` 用 generation/library/asset 左连接同时
筛选缺失 embedding 和 source fingerprint 不一致，worker 用 `asset_id ASC` keyset 而非 OFFSET。每库只
运行一个 semantic job；lease 过期在 3 次上限内重排并更新 claim revision，达到上限才以稳定
`operation_interrupted` 失败。通用 model-operation 启动恢复不得抢先终结这些 restart-safe operation。
每批 progress 提交同时推进 enabled `ai_library_settings.coverage_revision`；未收敛保持 `building`，全部
eligible 已计入且无 failed/stale 才为 `ready`，否则终态为 `degraded`。禁用只改变设置状态与 revision，
不删除 embedding。

清除与 backfill 在同一媒体库上互斥。接纳 clear 时以 settings revision 做 compare-and-swap，立即把
`enabled=0,state=clearing` 与 durable clear job/operation 同事务写入；因此部分删除永远不会被搜索当成
完整索引。worker 每个短事务最多删除 1,000 行 embedding，成功终态再同事务删除该库全部 generation
progress、切换为 `disabled` 并完成 operation。失败或取消保留尚存派生行但状态为 `degraded`，允许显式
重试；模型、原媒体、收藏和人工标签均不在清除事务范围内。

搜索 cursor 绑定 query fingerprint、scope/filter fingerprint、catalog revision、model generation、
semantic progress/index revision 与最后 `(score, asset_id)`；query 原文不进入 token。任一绑定值变化
返回 stale conflict，不跨代拼页。

备份必须包含 SQLite 中的模型登记、媒体库 AI 设置、generation/progress/job 状态；managed model bytes
与可重建 embedding/index 可选择不备份，但恢复文档必须明确恢复后能力为 unavailable/rebuild required，
不能把缺失字节解释成完整。`/models` direct source永不属于 FolioPath 备份。

该 migration 的完整验证矩阵为：fresh install、21→22 upgrade、重复启动幂等、每个 DDL/回填故障点
事务回滚、外键检查、`integrity_check`、库/资产 cascade、active generation 唯一性，以及升级后与升级前
数据库备份配对恢复。当前只完成 fresh schema/约束与 repository 定向测试；升级、故障注入和恢复证据
仍属于 S2，不能因 migration 已存在而视为通过。

## POST-MVP-5 revision 2：C+D+E 数据与事务合同

本节源自 `INT-117` 的 schema 语义；[INT-S1R2 Contract Ready](gates/POST-MVP-5/int-s1r2-contract-ready.md)
转为 Go 后，C/D 已由只追加 migration 25～32 实现，migration 21～24 未修改。E 仍受独立隐私 Gate
失败关闭，不存在 production face migration。状态机与 owner 以
[C+D+E capability contract](architecture/intelligent-media-s1r2-contract.md)为准。

### C：受控标签建议

| 计划表 | 身份与约束 | 分类/删除 |
| --- | --- | --- |
| `ai_tag_vocabulary_snapshots` | opaque snapshot ID、单调 revision、state；同一时刻最多一个 active | migration 25 已实现；管理配置；备份；只有管理员显式发布新 snapshot |
| `ai_tag_vocabulary_entries` | `(snapshot_id, tag_id)`；tag FK 指向现有 `tags`；删除/改名触发新 snapshot，不原地改历史 | migration 25 已实现；配置快照；备份；不保存自由文本副本或 embedding |
| `ai_tag_suggestions` | opaque ID；唯一 `(generation_id, library_id, asset_id, vocabulary_snapshot_id, tag_id)`；finite `[0,1]` confidence、source fingerprint、pending/invalidated | migration 25 已实现；可重建派生；generation/asset/library cascade；不得 cascade review decision |
| `ai_tag_reviews` | 唯一 `(library_id, asset_id, tag_id)`；accepted/dismissed、revision、reviewed_at、可空 source suggestion ID、accepted curation revision | migration 25 已实现；用户应用状态；备份；普通 AI disable/clear/rebuild 保留；独立二次确认才删除 |
| `ai_tag_review_state`、`ai_tag_review_clear_jobs/requests` | 每库单调 review revision；强 ETag clear、hashed idempotency、lease/attempt、删除计数 | migration 31 已实现；只删除 review audit，不删除 `asset_tags`、收藏、媒体或模型 |
| `semantic_tag_asset_progress`、`semantic_tag_library_progress` | generation/library/vocabulary/source 绑定；显式记录 ready（含零建议）、degraded/failed/stale 与 checkpoint/revision | migration 32 已实现；可重建覆盖率，不能以 suggestion 行数冒充处理数 |
| `semantic_tag_jobs/requests` | missing/all、同库单 active、lease/retry/cancel、hashed idempotency、operation FK | migration 32 已实现；建议、覆盖率、job checkpoint 与 operation progress 在同一短事务提交 |

词表发布是短事务：创建完整 immutable snapshot、复核 entry 上限/唯一 tag、CAS 切 active revision；文本
embedding 在事务外生成并按 snapshot 重建。suggestion batch 最多 100，先在事务外验证排序/finite/
generation/source，再短事务 upsert pending；不能覆盖 accepted/dismissed review。

接受单条 suggestion 的事务 owner 必须锁定/复核 pending suggestion 和 `ai_tag_reviews` revision，再通过
curation capability 的同一 database composition 写 `asset_tags`。若跨 capability 无法共享一个短事务，
必须先保持 suggestion pending，curation 成功后以其返回 revision CAS 写 accepted；任何冲突返回可重试，
绝不记录 accepted 却没有人工 tag。批量 API 是最多 100 个独立有界 use case 的结果集合，不持有跨资产
大事务。

### D：视频代表帧 embedding

| 计划表 | 身份与约束 | 分类/删除 |
| --- | --- | --- |
| `semantic_video_frames` | 唯一 `(generation_id, library_id, asset_id, storyboard_fingerprint, plan_size, ordinal)`；plan size 仅 4/10；ordinal 范围受检；timestamp 非负；固定 binary16 dimension；source/storyboard fingerprint | 可重建敏感度低的派生向量；asset/library/generation cascade；不保存帧路径或 sprite bytes |
| `semantic_video_progress` | `(generation_id, library_id)`；eligible/ready/degraded/failed/stale、checkpoint、coverage revision | 可重建聚合；不得把部分 plan 计 ready |
| `semantic_video_jobs/requests` | 复用 semantic queue 语义但使用独立 kind、lease/checkpoint/attempt 和 hashed idempotency | restart-safe 派生任务；不保存 FFmpeg stderr/path/query |

同一视频只有完整 10-frame plan 或完整 4-frame fallback 能进入查询。每个 frame batch 与 job checkpoint/
progress 在一个短事务提交；解码/推理不持事务。单帧失败可保存已完成 frame 进度但视频计 degraded，查询
按 `GROUP BY asset` 的受界候选逐视频选择 `score DESC, ordinal ASC`，最终 keyset 为
`video_score DESC, asset_id ASC`。改变 max/best-frame 规则必须新 transform version，不原地复用行。

storyboard cache eviction 不直接删除 frame embedding；下一次来源复核发现 storyboard fingerprint 不可用
时将结果 stale。可靠 asset 删除和库删除 cascade；offline 保留。备份可排除 frame embedding/jobs，但恢复
后必须显示 rebuild required。

### E：face 派生与人物应用状态

| 计划表 | 身份与约束 | 分类/删除 |
| --- | --- | --- |
| `face_library_settings` | library PK/FK、enabled/state/revision、active face/cluster generation、coverage revision | 应用设置；备份；默认关闭 |
| `face_generations` | opaque ID、组合 `model_id/package_id`、detector/embedder hash、512 维/transform version、threshold profile ID/hash、state | 可重建代次元数据；最多一个 active face generation；不更新 semantic active pointer |
| `face_observations` | opaque face ID；`generation_id + library_id + asset_id + source_fingerprint`；受检 normalized bbox/quality；固定 binary16 embedding | 高敏感可重建派生；asset/library/generation cascade；不保存 crop/path |
| `face_clusters` / `face_cluster_members` | generation-bound anonymous cluster；member role 仅 core/edge；同 observation 每代最多一个 cluster | 高敏感可重建派生；generation cascade；不引用 person name |
| `people` | instance-level opaque person ID；NFC display name 1～100 code points、revision、created/updated、可空 tombstone；名称不唯一 | 不可重建应用状态；必须备份；删除只由 person owner |
| `person_face_anchors` | opaque anchor ID；person、library/asset、exact source fingerprint、受检 quantized bbox anchor、revision、state；不保存 embedding/crop | 高敏感应用状态；必须备份；可靠 asset/library 删除 cascade，普通 derived clear 保留 |
| `face_exclusions` | anchor/observation lineage、library/asset、source fingerprint、revision | 用户应用状态；优先于 cluster；必须备份 |
| `face_cannot_links` | 两个不同 anchor ID 的规范有序 pair 唯一；revision、created_at | 用户应用状态；必须备份；任一 anchor 删除 cascade |
| `face_audit_events` | opaque event ID、actor=administrator、closed action code、目标 opaque ID、前后 revision、时间；不存名称、bbox、路径、embedding | 有界应用审计；备份；按 retention 清理但不充当当前状态 |
| `face_jobs/requests/clear_jobs` | detection/cluster/reconcile/derived-clear/manual-clear 各自 durable kind；hashed idempotency、lease/checkpoint/attempt | 派生任务可恢复；manual clear 是危险操作且不得和 derived clear 混用 |

`person_face_anchors` 是人工关系跨派生重建所需的最小敏感应用状态：只保留 exact source fingerprint 与量化
bbox anchor，不保留 embedding/crop。相同 source 下新 observation 只有在 capability owner 的唯一匹配
规则通过时才可重新绑定；模型升级出现零个或多个候选时标记 `needs_review`，不得按 bbox 顺序或最近向量
静默猜测。人物搜索只使用 bound/confirmed anchor，不使用匿名 cluster 推断。

人物 create/assign/move/exclude/cannot-link/merge/split/undo 都是短事务并匹配相关 person/anchor/cluster
revision。person merge 先在事务外生成 bounded plan，事务内重新验证全部 revisions/cannot-link，迁移成员、
写审计并 tombstone source person；任一冲突整体回滚。undo 只应用于没有后续 revision 的最近事件。

derived clear 删除 observation、cluster、crop cache 和可重建进度，设置进入 disabled/degraded，但保留
people、anchors、exclusion、cannot-link 和 audit。manual clear 必须先写带影响计数/范围 digest 的 durable
intent，二次确认后按 bounded transaction 删除应用状态；两种 clear 永不删除 `assets` 或原媒体。

### Retention、cascade 与备份

- offline、失败、取消或部分不可读不触发任何 C/D/E stale purge；只有可靠 catalog 删除/成功 generation
  reconciliation 获得 asset cascade 资格。
- 删除媒体库会删除该库 suggestion/review、video/face 派生及 anchors/constraints；instance-level person
  可以保留为空人物，只有用户显式删除 person 才移除。不得因库删除影响其他库的同一 person。
- SQLite 备份必须包含 vocabulary/review、people/anchors/constraints/audit 和全部 settings/operation intents。
  可排除可重建 suggestion/frame/observation/cluster/embedding，但恢复时必须进入 rebuild required，不能
  显示完整 coverage。
- face crop 若未来缓存，只能位于 `/app/data` 的独立配额目录，key 绑定 observation identity/transform；
  不进入备份、API 或通用 thumbnail route，clear/eviction 原子且可重建。

### Migration 验收矩阵

未来只追加 migration 必须覆盖 fresh、24→新版本 upgrade、重复启动、每个 DDL/回填故障点回滚、FK/
`integrity_check`、active snapshot/generation 唯一、可靠 asset/library cascade、offline 保留、derived/manual
clear 分离、person 跨库隔离、cannot-link merge rollback，以及升级前数据库与对应应用版本的配对恢复。
## 独立 AI 代次空间绑定增量（2026-09-14）

CR-2026-030 / ADR-0019 / POST-MVP-5：migration 38 增加
`semantic_generation_spaces(generation_id, fingerprint)`，外键指向 semantic_generations，
删除代次级联删除；旧代次不自动填充。绑定记录不可更新，绑定后模型、维度、变换与索引版本
不能原地修改。SQLite 适配仅接受 building、同维度且无图片/视频/标签向量的初次绑定；
相同绑定可在 building 状态幂等重试。未激活新模型，未迁移旧 embedding。

CR-2026-053不新增SQLite表。自动准备进度是AI进程内瞬时状态，重启根据已校验模型文件重建；完整文件是安装检查点，临时文件不表示完成。模型和用户data独立，旧结果不重标。
