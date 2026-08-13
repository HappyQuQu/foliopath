# 高负载视频故事板任务隔离

- 日期：2026-08-13
- 类型：已批准 `FTR-VID-001` 切片内的例行修复
- 需求/质量：`FR-MED-009～011`、`NFR-PERF-001`
- 目标版本：`POST-MVP-1`
- Gate：`VSP-301 Product Vertical` 后续回归
- Owner：`internal/thumbnail` 采样计划；`internal/media` 处理预算

## 问题

10-bit HEVC 等高解码成本视频可能具有有效元数据、poster 和可读取的目标帧，但 10 个独立
fast-seek 在四核、4 GiB 目标环境中仍可能耗尽三分钟预算。直接把所有 4K 或大文件固定为
4 帧会明显降低悬停预览的信息量；允许两个长 storyboard 同时运行又会占满两个媒体 worker，
阻塞图片缩略图和视频 poster。

生产样本 `3840×2160/60fps`、约 13 GiB 的 HEVC 视频证明各等分点均可抽帧，失败仅发生
在完整 hover storyboard 的累计处理阶段，原视频和 poster 均可用。

## 决定

- 4K 和大文件与普通长视频一样，仍首选均匀 10 帧；不按分辨率或文件大小预先降低质量。
- 全局同时最多领取一个 `storyboard` job。另一个媒体 worker 可继续领取高优先级 `grid`
  job，保证图片缩略图和视频 poster 继续推进；storyboard 并发规则按 variant 判断，不依赖
  可变的数字优先级。
- 10 帧仍沿用像素工作量预算；普通 4K 单次最多三分钟，源文件不小于 8 GiB 时提高到
  四分钟，以覆盖高码率和远程 seek 成本。真实超时后在同一 job 内降级为均匀 4 帧，
  fallback 最多两分钟，总预算最多六分钟。
- 4 帧仍超时后记录永久 `media_processing_timeout`，不再原样自动重试三轮。
- 原视频保持只读；不转码、不改写、不提高 FFmpeg 并发、内存、输出或总时间上限。
- 4/10 帧均已属于 transform version 1 和公共 API 合同，因此不变更 schema、API、cache
  key 或已有 ready 资源。

## 证据

- policy/service 测试固定普通 4K 从 10 帧/三分钟开始，8 GiB 边界从 10 帧/四分钟开始，
  并在超时后使用最多两分钟的 4 帧 fallback。
- SQLite claim 测试固定两个 storyboard 只能领取一个，同时新到的 grid 仍可被另一 worker
  领取；判断绑定 `storyboard` variant。
- job 分类测试固定 fallback 再超时后终态为 `media_processing_timeout`。
