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
- `name`：用户设置的显示名称。
- `root_rel_path`：相对于 `/library` 的规范化根路径。
- `status`：`pending`、`scanning`、`ready`、`offline` 或 `error`。
- `current_generation`：最近一次成功完整扫描的代次。
- `created_at`、`updated_at`。

`root_rel_path` 必须唯一。业务层还必须拒绝任意两个媒体库根路径的祖先/后代重叠。

### `directories`

- `id`、`library_id`、`parent_id`。
- `relative_path`：相对于媒体库根目录。
- `name`。
- `last_seen_generation`。
- 可选的目录修改时间和聚合统计。

唯一约束为 `(library_id, relative_path)`。`parent_id` 用于目录树和面包屑，根目录的相对路径使用统一的空路径表示。

### `assets`

- `id`、`library_id`、`directory_id`。
- `relative_path`、`name`。
- `kind`：图片、动画图片或视频。
- `mime_type`、`size_bytes`、`mtime_ns`。
- `width`、`height`、`duration_ms` 等可空媒体属性。
- `source_fingerprint`：至少包含大小和高精度修改时间，用于派生数据失效。
- `probe_status`、`probe_error_code`。
- `last_seen_generation`。

唯一约束为 `(library_id, relative_path)`。重命名在首版表现为新路径新增，并在成功完整扫描后清理旧路径；不承诺依赖 inode 自动识别跨路径移动。

### `thumbnails`

- `asset_id`、`variant`。
- `source_fingerprint`、`transform_version`。
- `cache_rel_path`：相对于 `/app/data/cache`。
- `status`、`width`、`height`、`byte_size`。
- `created_at`、`last_accessed_at`。

唯一约束为 `(asset_id, variant)`。缓存键必须包含源指纹和变换版本。文件先原子落盘，再提交可用状态；数据库不得把不存在或未完成的缓存文件标记为可用。

### `scan_runs`

- `id`、`library_id`、`generation`。
- `status`：`queued`、`running`、`succeeded`、`failed`、`cancelled` 或 `interrupted`。
- 开始、结束和心跳时间。
- 已发现目录数、媒体数、错误数和可安全展示的错误摘要。

同一媒体库最多有一个运行中的完整扫描。失败代次不能执行媒体库级陈旧记录清理。

### `media_jobs`

- `id`、`library_id`、`asset_id`、`kind`。
- `source_fingerprint`、`status`、`attempts`、`available_at`。
- `error_code`、时间戳。

任务必须幂等。进程重启时，超时的 `running` 任务可以安全返回队列；源指纹已变化的任务应被丢弃或替换。

### `settings`

只保存应用级设置和 schema 已知的配置。秘密值不得以明文日志输出。用户、会话和分享数据在对应功能进入范围后通过独立迁移增加。

## 索引与查询

至少建立：

- 目录树：`(library_id, parent_id, name, id)`。
- 稳定路径浏览：`(library_id, relative_path, id)`。
- 目录媒体列表：`(library_id, directory_id, name, id)`。
- 日期排序：`(library_id, mtime_ns DESC, id DESC)`。
- 扫描清理：`(library_id, last_seen_generation)`。
- 任务领取：`(status, available_at, id)`。

媒体列表统一使用 keyset cursor，不使用 `OFFSET` 承担大型列表分页。游标编码当前排序字段和稳定 ID，并视为不透明、可校验的 API 值。

文件名和路径搜索使用 FTS5 派生索引，并与 `assets` 变更保持同一事务语义。短查询、多语言大小写和自然排序需要单独定义行为，不能假设 SQLite 默认排序能覆盖所有 Unicode 语义。

## 扫描一致性

完整扫描开始时分配新 generation；分批 upsert 时更新 `last_seen_generation`。只有完整扫描成功后，才能删除更旧 generation 的目录和媒体记录。根目录离线、权限失败、任一子树无法可靠遍历、进程中断或用户取消都会使本次扫描失去清理资格。

增量扫描只更新明确检查过的路径，不进行媒体库级清理。定期完整扫描负责最终校准。详情见 [ADR-0003](adr/0003-scan-consistency.md)。

## 删除与离线语义

- 删除媒体库：删除其配置、索引、任务和派生缓存，不修改原文件。
- 媒体库离线：保留全部索引并记录状态；恢复后重新扫描。
- 媒体文件打开失败：返回稳定错误并触发后续校准，不在读取请求中直接大范围修改索引。
- 缓存缺失或损坏：将对应派生状态恢复为待处理，原媒体记录保持不变。

## 迁移与备份

数据库迁移只向前自动执行，已发布迁移不得修改。备份 SQLite WAL 数据库时必须使用 SQLite 认可的在线备份或先完成安全 checkpoint/停机流程，不能只复制主数据库文件而忽略相关状态。缩略图缓存可重建，可以与不可丢的配置和未来用户元数据采用不同备份策略。

