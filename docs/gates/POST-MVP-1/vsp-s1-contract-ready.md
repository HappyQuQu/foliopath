# VSP-S1 视频故事板 Contract Ready

## 结论

**Go — VSP-S1 Contract Ready。**

`FTR-VID-001` 的 capability、HTTP、持久化、调度、资源和失败合同已经冻结，可以按
`VSP-106～113` 实现和验证生产后端。本 Gate 不授权产品前端；前端必须等待
`VSP-S2 Backend Evidence Ready`。

## 基线与所有权

- 目标版本：[POST-MVP-1 revision 1](../../releases/POST-MVP-1-scope.md)
- Change Record：[CR-2026-004](../../changes/CR-2026-004-video-storyboard-preview.md)
- Feature：[FTR-VID-001](../../features/video-storyboard-preview.md)
- 前序 Gate：[VSP-S0 Architecture Ready](vsp-s0-architecture-ready.md)
- 可复现实验：[VSP-002](../../spikes/vsp-002-video-storyboard.md)
- 权威 HTTP 合同：[`api/openapi.yaml`](../../../api/openapi.yaml)
- Capability Owner：`internal/thumbnail`
- 媒体 adapter：`internal/media/videoffmpeg`
- 调度 owner：`internal/jobs`
- 持久化 adapter：`internal/store/sqlite`

不新增 ADR：本切片不改变部署单元、核心技术、信任/持久化边界、事务所有权或模块依赖方向。
如果实现需要第二 worker、另一种持久化或新的共享调度状态机，必须暂停并先记录 ADR。

## Capability 合同

### Eligibility 与采样

- 仅 `kind=video`、probe 已成功、时长至少 `2,000 ms` 且 `grid` poster 已 ready 的资产
  eligible。
- `2,000 ≤ duration < 5,000 ms` 生成 4 帧；`duration ≥ 5,000 ms` 生成 10 帧。
- 第 `i` 帧的整数时间戳为
  `floor(duration_ms * (i + 1) / (frame_count + 1))`，`i` 从 0 开始。
- 时间戳必须严格递增，全部满足 `0 < timestamp < duration_ms`；不采首尾端点。
- 首版为 all-or-nothing：任一计划帧抽取、解码、缩放或合成失败，整项失败；不发布部分
  sprite。失败只影响 storyboard，现有 poster 与原视频保持可用。
- 布局每行最多 5 格：`columns=min(frame_count, 5)`，
  `rows=ceil(frame_count/columns)`；单格宽高都在 `1..320 px`。

### 派生身份与状态

- variant 固定为 `storyboard`，独立 transform version；派生键绑定
  `asset identity + source fingerprint + variant + transform version`。
- 只有最终 WebP、实际布局、长度和当前源指纹全部验证成功后才能原子发布并提交 ready。
- 处理期间源指纹变化时丢弃结果；旧任务不得提交，当前指纹重新 admission。
- 取消、进程重启、超时、磁盘不足和 adapter 错误不得留下 ready 行或可见半成品。
- cache missing/长度不符只使该 storyboard 回到 pending 并重新排队，不影响 grid。
- 库 offline 保留可靠索引和派生记录，不解释为空，不清除原有 ready 状态。

## HTTP 与 DTO 合同

权威 OpenAPI 已追加：

- `ThumbnailVariantParameter` 允许 `grid | storyboard`；
- `Asset.storyboard` 为必需的 `StoryboardReference`；
- 状态为 `pending | ready | failed | unavailable | not_applicable`；
- ready 时 URL 与全部 layout 字段必须有值；非 ready 时不得暴露伪造 layout；
- 非视频或短于 2 秒为 `not_applicable`；
- `GET /api/v1/assets/{assetId}/thumbnail?variant=storyboard` 复用现有认证、ETag、
  `private, max-age=31536000, immutable`、`nosniff`、Range 之外的派生读取语义。

响应保持现有错误族：

| 响应 | Storyboard 语义 |
| --- | --- |
| `200 image/webp` | ready 且当前派生文件验证成功 |
| `202 application/json` | eligible，但 queued/running/backfill 尚未 admission |
| `304` | 强 ETag 条件命中 |
| `404` | asset 不存在，或请求的 variant 对该 asset 不适用 |
| `409` | library/source 当前 offline |
| `422` | 已知不支持、媒体损坏、处理失败或处理超时 |
| `429` | 读取限流，带有界 `Retry-After` |
| `500` | 脱敏内部错误 |

列表/详情中的 URL 不触发自动下载。后续前端只能通过生成 client 和 hand-written domain
adapter 消费，不能复制 wire type 或直接 `fetch`。

## 只向前 migration 设计

实现必须新增 migration 11（或当时下一个未占用序号），不得修改已可能发布的 migration 8/9。
SQLite 不能原地修改 `CHECK`，因此 migration 在一个短事务内以
`create replacement → copy → validate → drop old → rename → recreate indexes` 重建两张表。

### `thumbnails`

- 保留主键 `(asset_id, variant)`、外键、状态、源指纹、cache identity 和 LRU 字段。
- `variant CHECK` 扩展为 `grid | storyboard`。
- 增加 nullable：
  `frame_count`、`sprite_columns`、`sprite_rows`、`cell_width`、`cell_height`。
- grid 的五个 layout 字段必须全为 NULL。
- storyboard ready 的五个字段必须全非 NULL并满足：
  `frame_count IN (4,10)`、`1≤columns≤5`、`1≤rows≤2`、单格 `1..320`，
  且格子容量覆盖 frame count。
- storyboard 非 ready 的 layout 字段必须全为 NULL。

### `media_jobs`

- 保留 `(asset_id, variant)` 唯一约束、lease、attempt、退避、错误和重启恢复语义。
- `variant CHECK` 扩展为 `grid | storyboard`。
- 新增非负、由调度 owner 写入的 `priority`：`grid=0`，`storyboard=100`。
- claim 顺序固定为
  `(minimum eligible priority, least recently claimed library in that priority,
  minimum eligible job id in that library)`。
- 因而任何已 eligible 的 grid 都先于 storyboard；同一 priority 内继续跨库公平。不得在
  thumbnail adapter 复制另一套公平或重试状态机。
- 索引必须覆盖 eligible state、`priority`、`next_attempt_at`、library 和稳定 job id。

### 升级与 backfill

- migration 覆盖 fresh install，以及 migration 10 数据库中的 ready/failed/pending grid 和
  running lease；复制后运行 FK 与 `integrity_check`，失败则整次 migration 回滚/失败关闭。
- 旧 grid 行的 layout 为 NULL、priority 为 0，状态和 lease 不丢失。
- backfill reconciler 每个短事务最多选择并插入 128 个 eligible 历史视频；不得一次把全库
  asset/job 载入内存或在一个事务中排 10 万项。
- 新 grid 成功时，可在同一短事务幂等 admission 对应 storyboard；唯一约束处理重复请求。
- backfill eligibility 为：video、probe ready、duration≥2 秒、grid ready、当前不存在同
  asset storyboard job。反复批处理最终收敛，不需要第二个持久化游标所有者。

## 媒体处理与资源合同

- 每个 durable storyboard job 最多执行 10 次 input-side fast seek 抽帧和 1 次 compose；
  不允许完整顺序解码长视频。
- 总 wall deadline 45 秒，包括抽帧、合成、验证和原子发布；context 取消必须终止整个进程组。
- 每个临时帧上限 1 MiB、所有临时帧合计 10 MiB、最终 sprite 8 MiB、解码后 sprite
  最大约 1.024M 像素。
- 临时根目录由 app 注入并位于 `/app/data/tmp/storyboard`；文件以安全创建方式生成，
  成功后仍通过现有 publisher 原子 rename。失败、取消和启动恢复会幂等清理。
- FFmpeg 参数使用 argv，不经 shell；仅继承明确需要的文件描述符；decoder/filter thread
  各限制为 1；stderr 只进有界、脱敏诊断。
- 生产镜像必须使用现有启用 libwebp 的 FFmpeg，不新增 `cwebp` 运行时依赖。开发机 fallback
  只属于 spike 证据。
- 文件系统打开继续经 `internal/files` 的 Linux `openat2` 边界；处理期间不持有 SQLite
  transaction，原媒体永远只读。
- storyboard worker 使用现有全局有界队列/lease；首版有效并发上限为 1，并低于 grid。
  S2 容量证据可以进一步降低 admission，但不得提高合同上限。

45 秒是安全上限，不是吞吐承诺；VSP-S2 必须在 Linux amd64/arm64 目标镜像验证。若 grid
等待、API P95、RSS 或磁盘余量不满足门禁，按降低 admission/尺寸/质量、改为闲时 admission、
禁用 storyboard 的顺序降级，poster 始终保留。

## 威胁与故障合同

- 视频、probe 输出和图片字节均视为不可信；验证尺寸、数量、格式和总字节后才发布。
- API 不接受绝对路径，不返回主机路径、FFmpeg 输出、SQL、stack 或临时文件名。
- storyboard 继续要求 session；不增加公开 token、匿名 LAN 或新的网络出站。
- symlink、nested mount、源替换与 fingerprint CAS 沿用现有唯一 owner，不做 pathname
  预检查替代 kernel-anchored open。
- 失败重试沿用 durable job attempt/backoff 上限；稳定失败进入 failed，HTTP/DTO 只暴露
  稳定 error code。

## 后端验证计划

`VSP-S2` 前至少必须提供：

- 采样、eligibility、布局、variant/version、派生键与 invalid result 的 table tests；
- fresh/upgrade/rollback migration、CHECK/FK、running lease、grid preservation 和
  `integrity_check`；
- claim priority、同 priority 跨库公平、lease/retry/cancel/restart、128 项 backfill 边界；
- synthetic 2s/4s/10s/10min/2h、portrait、VFR、长 GOP、黑片头、损坏和不可 seek fixtures；
- timeout、进程组取消、每帧/总 temp/final/pixel 上限、ENOSPC 与 temp cleanup；
- 指纹竞争、offline、cache missing、原子发布、LRU 独立 variant 与原文件 hash/mtime 不变；
- authenticated 200/202/304/404/409/422/429/500、ETag/cache/nosniff 和错误脱敏；
- 四核/4 GiB、约 10 万资产/1 万目录代表负载下的 grid 等待、跨库公平、RSS、队列、
  缓存增长和浏览/搜索 P95。

## 授权边界

本 Gate 只授权 `VSP-106～113` 后端代码、migration、集成/容量证据和
`VSP-S2 Backend Evidence Ready`。在 S2 Go 前：

- 不实现生产 hover UI；
- 不使用 mock 发明 storyboard 状态；
- 不把功能描述为可用；
- 不跳过生成 client 边界。
