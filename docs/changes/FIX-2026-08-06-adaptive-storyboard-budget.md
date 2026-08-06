# 自适应故事板处理预算与四帧降级

- 日期：2026-08-06
- 类型：已批准 `FTR-VID-001` 切片内的例行修复
- 需求/质量：`FR-MED-009～011`、`NFR-PERF-001`
- 目标版本：`POST-MVP-1`
- Gate：`VSP-301 Product Vertical` 后续回归
- Owner：`internal/media` 处理预算；`internal/thumbnail` 采样降级与任务结果分类

## 问题

故事板处理此前对所有视频使用固定 45 秒总超时。正常的高分辨率视频可能在单线程、
有界 FFmpeg 解码下稳定超过该预算，随后相同输入和参数又被任务状态机原样重试三次。
这既不能提高成功率，也浪费后台媒体处理容量。

## 决定

- 保留不可信媒体处理的最终超时、取消、并发和输出上限，不允许 FFmpeg 无限运行。
- `internal/media` 按源像素数与计划帧数计算每次故事板预算：1080p/10 帧保持 45 秒，
  单次最多 3 分钟。
- 长视频的 10 帧计划耗尽预算后，`internal/thumbnail` 在同一 job 内只降级一次为均匀
  4 帧；两次预算合计最多 5 分钟。
- 降级结果仍是既有 transform version 1 允许的 4 帧 WebP sprite，不改变 API、schema、
  cache key、原视频或已有 ready 派生资源。
- 4 帧仍超时后以 `media_processing_timeout` 直接结束当前 job，不再原样自动重试三次；
  管理员显式重试仍然可用。普通缩略图以及来源/缓存暂时不可用仍保留原重试策略。

## 证据

- `internal/media` 单元测试固定 1080p、4K 与上限预算。
- FFmpeg adapter 测试固定 request 级 timeout 覆盖默认值并取消进程。
- storyboard service 测试固定 10→4 帧降级、总预算与双重超时失败。
- job 分类测试固定故事板预算耗尽为终态，同时保留一般 timeout 的可重试语义。

该修复不改变部署单元、依赖方向、事务所有权、信任边界或公共 API，因此不需要 ADR 或
migration。
