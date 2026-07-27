# S3-006 敌意媒体与资源安全实现记录

## 结论

**Go — S3-006 实现与验证完成；进入 S3-007 Backend Ready 汇总。**

本记录确认不可信媒体在进入 govips/FFmpeg 前具有统一的编码大小、解码维度和像素上限，
native 图片运行时与 FFmpeg 线程/进程树受全局预算约束，取消、工具输出超限和真实缓存
文件系统写满均失败关闭。它不授权资产/缩略图 HTTP、前端集成或发布；这些分别由
S3-007、前端 Gate 和 Release Gate 决定。

## 范围、需求与风险

- 需求：`FR-MED-001～008`、`NFR-SEC-001`、`NFR-PERF-001`
- 风险：`R-006`、`R-008`、`R-009`、`R-013`、`R-016`
- 后端 owner：`internal/media`、`internal/thumbnail`
- adapter：`internal/media/imagevips`、`internal/media/videoffmpeg`、
  `internal/thumbnail/cachefs`
- composition/lifecycle：`internal/app`
- 证据环境：Go 单元/race/集成；Linux Debian + libvips；8 MiB tmpfs 真正 `ENOSPC`

## 固定资源策略

| 边界 | MVP 固定值 | 失败语义 |
| --- | ---: | --- |
| 图片编码输入 | 最大 256 MiB | `invalid_media`，不进入 native processor |
| 视频输入 | 最大 4 GiB | `invalid_media`，不启动 ffprobe/FFmpeg |
| 单边维度 | 最大 32,768 px | `invalid_media`，不执行 thumbnail/poster decode |
| 解码像素 | 最大 100,000,000 | `invalid_media`，不执行 thumbnail/poster decode |
| 工具标准输出 | 最大 8 MiB | `media_processing_failed`，缓冲内存不继续增长 |
| 工具标准错误 | 最大 64 KiB | `media_processing_failed`，不返回原始 stderr |
| FFmpeg 单任务 | 15 秒；decoder/filter 各 1 thread | 超时杀整个进程组并按 transient 退避 |
| govips runtime | 1 native concurrency；64 MiB/32 entry cache；0 cached files | 启动失败则应用失败关闭 |
| 媒体任务 | 全局 2 worker；最多 3 attempt；5/10 秒退避 | 单资产隔离，不热循环、不停止全局队列 |

`internal/media` 是输入大小、维度和像素策略的唯一 owner。thumbnail service 在打开并
核对 source fingerprint 后先执行同一策略；govips 与 FFmpeg adapter 在信任边界内再次
调用它，避免未来绕过 service 时失去保护。

## 取消、native 与子进程

- FFmpeg/ffprobe 不经过 shell，只接收 inherited read-only descriptor；Unix 上进入独立
  process group，context 超时/取消向整个组发送 `SIGKILL`，测试证明孙进程不会残留。
- decoder 与 filter 显式使用单线程；两个媒体 worker 是全局上限，不能按库或请求再创建
  第二组 limiter。
- govips 由应用 `image-runtime` lifecycle 显式 Startup/Shutdown；不再依赖自动启动的
  默认并发和缓存。
- libvips 当前仍是进程内 native 调用，context 不能在任意 C 调用中间抢占。缓解方式是
  在求值前拒绝超大输入/像素、将 native concurrency 固定为 1，并在调用返回后的第一个
  安全点重新检查 cancellation；取消结果不得发布缓存或提交 ready。若未来要求进程级崩溃
  隔离，必须先接受改变媒体执行边界的 ADR。

## 磁盘满与恢复

- cache reservation 继续串行覆盖“空间检查 → 临时文件 → 原子 rename → DB ready”。
- 真实 Linux tmpfs fixture 把 8 MiB 文件系统写到 `ENOSPC`，确认发布失败、不留下
  `.thumbnail-*.tmp`，释放空间后同一 publisher 可重新成功发布。
- cache 文件删除失败时不先删除 SQLite ready 状态；发布写失败时不提交 ready/failed，
  durable job 按 cache transient 进行 5/10 秒退避并在第三次后稳定失败。
- 本切片不宣称已覆盖 SQLite WAL 与 cache 同卷在长期生产负载下的全部竞争；最终卷大小、
  强杀、WAL/temp 压力和代表性存储仍由 Release Gate 阻断。

## 自动证据

- `internal/media/processing_test.go`：大小、单边维度和 100 MP 上限。
- `internal/media/imagevips/processor_libvips_test.go`：真实 libvips 正常/截断输入与
  20,000 × 20,000 JPEG 像素炸弹在 thumbnail 求值前拒绝。
- `internal/media/videoffmpeg/*_test.go`：真实格式矩阵、继承 FD、单线程参数、敌意 metadata、
  8 MiB 输出上限、超时进程组回收、oversized sparse input 和稳定错误。
- `internal/thumbnail/service_test.go`：service 前置大小拒绝、native 返回后取消不发布、
  cache 写失败不提交 ready。
- `internal/thumbnail/capacity_test.go`：reservation 串行/取消、真实删除失败保留 DB、LRU
  水位和无可重建空间失败关闭。
- `internal/thumbnail/cachefs/disk_full_linux_test.go`：真实 `ENOSPC` 清理与恢复。
- `internal/store/sqlite/media_jobs_test.go`：5/10 秒指数退避、第三次 attempt 与 terminal。
- `tests/architecture/dependencies_test.go`：资源 owner、native lifecycle、线程/进程取消和
  Linux 满盘 CI fixture 不得移除。

执行入口：

```sh
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
```

另在 Linux/libvips 容器执行 `libvips` 与 `mediafull` tagged fixture；CI 在 amd64/arm64
govips job 中挂载 8 MiB tmpfs 重复真实满盘测试。

## 保留限制与交接

- S3-007 冻结资产/缩略图 HTTP 的认证、pending/ready/failed、缓存命中热度刷新与错误映射，
  然后汇总 S3-001～006 判断浏览/缩略图后端是否 Backend Ready。
- 原内容、HEAD/Range 与条件请求仍属于 Stage 4 媒体内容后端，不在 S3-007 顺带实现。
- 浏览器直放、最终镜像 digest、代表性存储、长期 WAL/temp 压力与进程级 native crash
  隔离不属于本结论。
