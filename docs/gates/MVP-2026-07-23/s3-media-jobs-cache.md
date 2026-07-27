# S3-005 媒体任务与缓存保护实现记录

## 结论

**Go — S3-005 实现完成；进入 S3-006。**

本记录确认媒体派生已进入 restart-safe、有界后台生命周期，source fingerprint 变化会原子
失效旧状态并登记新任务，缩略图缓存具备默认 10 GiB 配额、LRU 水位和安全磁盘余量。
它不把浏览/缩略图后端标记为 Backend Ready，也不授权资产/缩略图 HTTP、敌意媒体发布
验收、前端集成或发布。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-MED-001～003`、`FR-MED-008`、`NFR-PERF-001`、
  `NFR-REL-001`、`NFR-SEC-001`
- 通用 lease/heartbeat worker owner：`internal/jobs`
- 媒体任务 payload、handler 结果和缓存策略 owner：`internal/thumbnail`
- durable queue、LRU 与失效记录 adapter：`internal/store/sqlite`
- 缓存空间/文件删除 adapter：`internal/thumbnail/cachefs`
- worker、cache manager 与生命周期组合：`internal/app`

## 已实现行为

- migration 9 建立 `media_jobs`、跨库公平游标和 `cache_deletions`。任务只保存 asset ID、
  variant、transform version、source fingerprint、稳定错误、租约和时间，不保存媒体路径
  或命令。
- 扫描批次首次发现资产时原子登记一个 grid 任务；fingerprint 变化时保留 asset ID，
  清空旧 metadata/probe 状态、删除旧 thumbnail 状态、登记可恢复文件清理并把同一任务
  重置为 queued。fingerprint 不变不会重新打开已终止任务。
- 任务以 SQLite 为事实来源，进程内 signal 只降低唤醒延迟。全局最多 2 个媒体 worker；
  每次只领取一个任务，使用 15 秒 heartbeat、120 秒 lease，并按媒体库最近领取顺序公平
  选择候选。
- transient source/cache/tool 错误最多尝试 3 次，使用 5/10 秒退避；invalid/unsupported
  直接终止。进程退出不伪装完成，租约到期后重排；第三次崩溃或 transient 失败收敛为稳定
  failed 状态。
- 旧任务完成时必须同时匹配 job ID、fingerprint、running 状态和 attempt；扫描已登记新
  fingerprint 时，旧 worker 无法覆盖新任务或发布旧 ready 状态。
- worker 启动和定期恢复时每批最多协调 256 个旧 transform version；旧缓存进入幂等删除
  队列，任务重置为当前变换版本，不要求部署者手工清库。
- 缓存配额默认 10 GiB。ready 使用量超过 90% 水位时按
  `(last_accessed_at_ms, asset_id)` LRU 清理到 80%；每次发布还要求底层文件系统在写入后
  至少保留 512 MiB。
- cache reservation 在“容量检查 → 文件原子发布 → SQLite ready 提交”期间串行，避免两个
  worker 同时越过配额。只删除可重建缓存；`/library` 不在删除 adapter 的能力范围。
- 设置更新分别唤醒 scan scheduler 或 cache manager；缩小缓存配额无需重启即可触发回收。

## 自动约束与证据

- `internal/store/sqlite/media_jobs_test.go`：租约、有限重试、跨库公平、过期恢复、
  fingerprint 失效、旧缓存清理登记和同 asset 重新排队。
- `internal/thumbnail/capacity_test.go`：90%→80% LRU、512 MiB 余量、失效文件优先清理和
  无可重建空间时失败关闭。
- `internal/thumbnail/cachefs/cache_test.go`：磁盘余量、限定相对路径删除和 traversal 拒绝。
- `internal/app/media_processing_integration_test.go`：真实 SQLite、scanner、安全媒体根、
  generic worker、cache manager 与 lifecycle；任务最终 succeeded 且原媒体逐字节不变。
- `tests/architecture/dependencies_test.go`：migration、唯一任务/缓存策略 owner、扫描原子
  admission 和唯一 composition fitness checks。
- 原生 libvips CI job额外编译带 `libvips` tag 的正式 app composition；普通开发构建继续
  使用失败关闭的非原生 stub，不构成发布镜像证据。

本切片要求的完整验证入口为：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
```

## 保留限制与交接

- S3-006 才完成像素炸弹、更多敌意输入、原生调用不可中断区间、磁盘满、并发资源峰值、
  transient 退避压力和最终双架构媒体矩阵。
- 当前 LRU 已拥有正确顺序和水位；`last_accessed_at_ms` 的 authenticated thumbnail HTTP
  命中刷新由 S3-007 接入，不能由前端或 cache adapter自行维护第二套热度。
- S3-007 才实现/冻结资产与 thumbnail HTTP、pending/ready/failed 响应，并汇总 Stage 3
  Backend Ready。
- 禁止声明：浏览/缩略图 Backend Ready、Stage 3 Integrated Done 或 MVP 可发布。

- 评审日期：2026-07-27
