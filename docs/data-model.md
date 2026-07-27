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
- `last_seen_generation`。

唯一约束为 `(library_id, relative_path)`。重命名在首版表现为新路径新增，并在成功完整扫描后清理旧路径；不承诺依赖 inode 自动识别跨路径移动。
S3 浏览的直接名称序使用 `(natural_name_key, name, relative_path, id)`，修改时间序使用
`(mtime_ns, id)`；migration 7 已建立与两个 tuple 和目录 scope 相符的索引。
`OFFSET` 不是容量档下的可接受实现。

### `thumbnails`

- `asset_id`、`variant`。
- `source_fingerprint`、`transform_version`。
- `cache_rel_path`：相对于 `/app/data/cache`。
- `status`、`width`、`height`、`byte_size`。
- `created_at`、`last_accessed_at`。

唯一约束为 `(asset_id, variant)`。缓存键必须包含源指纹和变换版本。每库派生文件固定放在
`libraries/lib_<library-id>/` 子树内，使 removal worker 无需接触 `/library` 即可幂等清理。
文件先原子落盘，再提交可用状态；数据库不得把不存在或未完成的缓存文件标记为可用。

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
以及默认跟随浏览器的中英语言偏好。`revision` 支持强 ETag/If-Match，提交后才唤醒
scheduler。秘密值不得以明文日志输出；设置不能成为任意键值存储。

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

文件名和路径搜索使用 FTS5 派生索引，并与 `assets` 变更保持同一事务语义。索引必须支持当前目录（可递归）、当前媒体库和全部媒体库三种作用域。短查询、多语言大小写和自然排序需要通过 spike 明确定义行为，不能假设 SQLite 默认排序能覆盖所有 Unicode 语义。

## 扫描一致性

完整扫描开始时分配新 generation；分批 upsert 时更新 `last_seen_generation`。只有完整扫描成功后，才能删除更旧 generation 的目录和媒体记录。根目录离线、权限失败、任一子树无法可靠遍历、进程中断或用户取消都会使本次扫描失去清理资格。

增量扫描只更新明确检查过的路径，不进行媒体库级清理。定期完整扫描负责最终校准。详情见 [ADR-0003](adr/0003-scan-consistency.md)。

## 删除与离线语义

- 删除媒体库：删除其配置、索引、任务和派生缓存，不修改原文件。
- 媒体库离线：保留全部索引并记录状态；恢复后重新扫描。
- 媒体文件打开失败：返回稳定错误并触发后续校准，不在读取请求中直接大范围修改索引。
- 缓存缺失或损坏：将对应派生状态恢复为待处理，原媒体记录保持不变。

## 迁移与备份

数据库迁移只向前自动执行，已发布迁移不得修改。备份 SQLite WAL 数据库时必须使用 SQLite 认可的在线备份或先完成安全 checkpoint/停机流程，不能只复制主数据库文件而忽略相关状态。缩略图缓存可重建，可以与不可丢的配置、管理员凭据和应用设置采用不同备份策略。
