# S3-004 媒体探测与缩略图实现记录

## 结论

**Go — S3-004 实现完成；进入 S3-005。**

本记录确认图片/视频处理适配器、统一结果与错误分类、缩略图派生键、原子缓存发布以及
SQLite 派生状态已经形成可执行纵向基础。它不把整个浏览/缩略图后端标记为 Backend Ready，
也不授权同步请求中处理媒体、资产/缩略图 HTTP、搜索、前端集成或发布。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-MED-001～003`、`FR-MED-006～007`、`NFR-SEC-001`、
  `NFR-REL-001`、`NFR-COMP-001`
- 媒体结果、稳定错误码与 source fingerprint owner：`internal/media`
- 图片适配器：`internal/media/imagevips`
- 视频适配器：`internal/media/videoffmpeg`
- 派生键、处理编排与持久化 port：`internal/thumbnail`
- 安全媒体打开与 SQLite/cache adapter：`internal/app`、`internal/store/sqlite`、
  `internal/thumbnail/cachefs`

## 已实现行为

- 图片 production adapter 使用 govips/libvips 读取 JPEG、PNG、WebP 和 GIF；统一自动方向，
  GIF 只取首帧，输出不放大的 512×512 内接 WebP，质量 82，并剥离输出 metadata。
- 视频 adapter 使用 `ffprobe` 获取首个视频流的尺寸、codec 与时长，使用 `ffmpeg` 提取
  首帧 WebP poster。命令不经过 shell，输入由安全打开后的继承文件描述符提供，不把宿主机
  或媒体绝对路径交给子进程。
- MP4/MOV 中 H.264 标记为可直接播放；其他 codec 或 MKV 的浏览器播放能力保守标记为
  `unsupported_codec` 或 `unknown`，不承诺转码。
- `internal/media` 统一拥有成功结果验证以及 `invalid_media`、`unsupported_format`、
  `timeout`、`cancelled`、`processing_failed` 等稳定分类。数据库和公开模型只保存稳定
  状态/错误码，不保存 libvips、FFmpeg stderr 或绝对路径。
- migration 8 为资产追加尺寸、时长、probe/playback 状态，并建立按
  `library_id + asset_id` 隔离的 `thumbnails` 表。缓存 key 由 asset identity、source
  fingerprint、variant 和 transform version 唯一派生，不包含源路径。
- 缓存文件先写同目录临时文件、`fsync` 并原子 rename，随后才在短 SQLite 事务中提交
  ready 状态。提交前重新核对 source fingerprint；源变化时拒绝发布旧结果。
- 原媒体只通过 `internal/files` 的 anchored read-only boundary 打开。真实组合测试验证
  处理前后源字节不变、缓存文件存在且数据库状态与缓存一致。

## 自动约束与证据

- `internal/media/processing_test.go`：图片/视频结果语义、状态与稳定安全错误分类。
- `internal/media/imagevips/processor_libvips_test.go`：真实 govips 图片/GIF 首帧、
  尺寸限制和截断输入；由 `libvips` build tag 与原生依赖 CI job 执行。
- `internal/media/videoffmpeg/*_test.go`：参数/文件描述符/超时/脱敏单元测试，以及真实
  MP4/MOV/MKV、兼容与不兼容 codec、损坏视频的合成 integration fixture。
- `internal/thumbnail/*_test.go`：派生键、处理成功/失败/source changed、原子 cache publisher。
- `internal/store/sqlite/*media_processing*` 与 `thumbnail_test.go`：migration 8、
  ready/failed 状态、外键隔离和 stale fingerprint 拒绝。
- `internal/app/media_processing_integration_test.go`：真实 scanner、SQLite、安全 source、
  cache publisher 与 thumbnail service 组合。
- `tests/architecture/dependencies_test.go`：唯一结果/派生 key owner、govips、无 shell 的
  `CommandContext + ExtraFiles`、原子 rename 和 append-only migration fitness checks。

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

生产依赖补充验证在 Debian/libvips 环境执行 tagged govips 测试，并在具备 `libwebp`
encoder 的 FFmpeg 环境执行真实视频 fixture。普通本地 Go 构建不要求安装 libvips；
正式发布构建必须启用 `libvips` tag 并提供匹配的原生库。

## 保留限制与交接

- S3-005 才实现 restart-safe、有界媒体任务队列，将本切片 service 接入正式生命周期；
  同时负责 fingerprint 失效/重建、默认 10 GiB LRU、水位和安全磁盘余量。
- S3-006 才完成像素炸弹、更多敌意输入、原生调用不可中断区间、磁盘满、全局并发和
  失败退避矩阵。当前 512 MiB 输入与 8 MiB 输出上限不是发布级完整资源策略。
- S3-007 才接入并冻结资产/缩略图 HTTP 契约，汇总浏览、媒体任务和容量证据，并判断
  Stage 3 是否 Backend Ready。
- 生产代码不得在扫描或 HTTP request goroutine 中同步调用媒体处理器；必须等待 S3-005
  的 durable job owner。
- 禁止声明：浏览/缩略图 Backend Ready、Stage 3 Integrated Done 或 MVP 可发布。

- 评审日期：2026-07-27
