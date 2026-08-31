# FolioPath 数据模型

## 原则

- 文件系统决定媒体是否存在以及目录如何组织。
- SQLite 保存媒体库配置、可重建索引、扫描状态和应用元数据。
- 所有目录、媒体、缩略图和扫描记录都必须归属于 `library_id`。
- 数据库不保存宿主机绝对路径，也不保存缩略图 BLOB。
- 媒体首版逻辑身份为 `library_id + normalized_relative_path`。

## 核心表

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

migration 21 追加 `curation_state`、`asset_favorites`、`tags` 与 `asset_tags`。收藏和资产标签
使用 `(library_id, asset_id)` 复合外键指向 catalog asset，并在可靠资产删除或媒体库移除时
级联清理；offline、失败、取消和部分扫描不删除可靠 asset，因此也不删除整理数据。

`tags.normalized_name` 使用 `internal/curation` 生成的 NFC、Unicode case-fold 唯一键；显示名
保留规范化大小写。每资产最多 20 个关联，由 capability 与单事务替换共同约束。
`curation_state.revision` 是全局单调失效令牌，触发器覆盖正常写入与 FK cascade；它不是事件
数量。收藏按 `(created_at_ms, asset_id)`、标签名按 `(normalized_name, id)`、标签资产按稳定
媒体 tuple 使用 keyset，不使用 OFFSET。

数据库迁移只向前自动执行，已发布迁移不得修改。备份 SQLite WAL 数据库时必须使用 SQLite 认可的在线备份或先完成安全 checkpoint/停机流程，不能只复制主数据库文件而忽略相关状态。缩略图缓存可重建，可以与不可丢的配置、管理员凭据和应用设置采用不同备份策略。

## POST-MVP-5 revision 1：模型与图片语义数据合同

S1 冻结 schema 语义；S2 已以只追加 migration 22/23/24
建立模型、generation、setting、embedding、progress、backfill/clear job 基础表与 durable semantic request/attempt 状态，
禁止修改此前已提交 migration。

| 表 | 身份与关键约束 | 数据分类与删除 |
| --- | --- | --- |
| `ai_models` | opaque ID；`purpose=semantic_image_text`；package/version/arch/content hash 唯一；state、storage mode、availability revision | 模型配置/运行状态；备份；删除不能触碰 direct source |
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
| `face_generations` | opaque ID、detector/embedder/cluster contract、model IDs、dimension、threshold profile、state | 可重建代次元数据；最多一个 active profile |
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
