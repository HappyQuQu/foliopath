# 媒体处理韧性与诊断纠偏

## 归属

- 日期：2026-08-12
- 类型：已批准媒体处理与视频故事板切片内的例行可靠性修复
- 需求：`FR-MED-001`、`FR-MED-004～005`、`FR-MED-009～011`、`FR-OPS-008`
- 既有证据边界：`S3-004`、`S3-006`、`S3-007`、`VSP-301`
- 风险：`R-006`、`R-018`
- Owner：`internal/media` 的格式核验、资源预算与工具错误；`internal/thumbnail` 的故事板降级、
  派生任务终态与显式重排
- 不变量：原媒体只读；只写 `/app/data` 中的可重建派生资源；浏览器继续独立决定原视频能否播放

## 问题

媒体派生链存在四类会把可处理媒体误报为损坏或异常的边界：超大 JPEG 与其他图片共用
100 MP 解码上限；工具成功并交付有效产物时，超过 64 KiB 的 stderr 仍可能反向覆盖成功；
视频的计划时间点没有帧时未尝试附近的有界时间点；4K 故事板降级与 AV1 解码器闭包不足。
与此同时，0 字节文件和像素超限需要在既有处理结果中给出可区分、可操作的诊断。

## 决定

### 图片核验、预算与诊断

- 图片先由处理器核验真实格式，不以 `.jpg` / `.jpeg` 扩展名授予 JPEG 特例。伪装格式、未知
  格式和真实损坏数据仍按实际探测结果失败。
- 非 JPEG 图片继续执行 100 MP 总源像素上限。只有真实 JPEG 可以进入独立的 180 MP
  源像素上限，并使用 libvips shrink-on-load 在完整像素展开前缩小；真实 JPEG 不超过
  100 MP 时继续使用原有变换路径，避免改变既有输出。
- 256 MiB 编码输入、32,768 px 单边、native concurrency、缓存和最终输出上限保持不变。
- 0 字节输入复用既有永久诊断
  `invalid_media / source_read / invalid_media_data / filesystem`；像素超过适用的 100 MP 或
  180 MP 上限复用
  `media_processing_failed / probe / source_too_large / libvips`。截断解码与未知工具故障继续
  分别使用稳定的 `decode_failed` 和 `tool_failed`，不把四类结果折叠成同一原因。

### 工具产物与视频恢复

- 工具进程以退出状态、stdout 产物及产物验证共同决定成功。进程退出 0 且交付的有界产物
  有效时，即使 stderr 超过 64 KiB 也接受结果；stderr 仍只保留最多 64 KiB 的脱敏诊断，
  不持久化原始输出。无效或超限产物不会因为退出 0 而被接受。
- 故事板目标时间点没有视频帧时，只在该点前后最多 `±1s` 内尝试有界邻帧，并以相邻采样点
  的中线收窄窗口，确保短视频恢复帧仍严格按时间排序；10 帧计划无法
  完成后沿用同一 job 内的 10→4 帧降级。4 帧计划耗尽邻帧时记录永久
  `media_processing_failed`，耗尽处理预算时记录永久 `media_processing_timeout`；两者都不再
  把相同输入投入自动重试循环，poster 和原视频访问保持可用。
- 1080p/10 帧基线仍为 45 秒。4K 的 4 帧降级尝试获得 120 秒，主计划与降级计划合计仍不得
  超过 5 分钟；取消、单线程、进程组和输出上限不变。
- 发布镜像的 FFmpeg 使用外部 `libdav1d` 完成 AV1 poster/storyboard 等服务端派生处理。
  这不是转码或直放承诺；原视频播放仍由当前浏览器及平台原生能力决定。

## 兼容性与恢复

本修复复用现有 API 错误码和 `media_job_attempts` 的 stage/reason/tool 字段，不改变
`api/openapi.yaml`、SQLite schema、已发布 migration、缓存键或 transform version；既有 ready
派生资源不会仅因升级而重建，也不需要 ADR。

部署新镜像并确认 ready 后，管理员应在受影响媒体库执行一次“补齐缺失”，把缺失、缓存丢失和
既有失败的派生项重新排入有界队列；不需要因此执行“全部重建”或重新扫描文件树。真正的 0 字节、
截断图片、缺少 moov、不可解码数据及其他真实损坏文件仍会以相应稳定原因失败；显式重排不会
修改原媒体，也不会把永久损坏变成成功。

## 回归证据

- 真实格式与伪装扩展名 fixture 固定 JPEG 特例只由探测格式触发；非 JPEG 100 MP、JPEG
  100 MP 原路径、100～180 MP shrink-on-load 和超过 180 MP 拒绝分别覆盖。
- 0 字节、像素超限、截断图片和未知 libvips 故障固定为四组既有诊断字段与永久/暂时策略。
- 超过 64 KiB stderr 的退出 0 有效 FFprobe JSON/FFmpeg 图片产物成功，非零退出、无效 stdout
  和 stdout 超限仍失败，且持久层与日志不出现原始 stderr。
- 故事板覆盖计划点命中、`±1s` 邻帧恢复、10→4 降级、四帧无帧耗尽永久
  `media_processing_failed`、预算耗尽永久 `media_processing_timeout`、1080p 45 秒、
  4K fallback 120 秒和总计不超过 5 分钟。
- 最终 distroless 生产 FFmpeg 闭包与合成 AV1 fixture 验证 external libdav1d 派生链；浏览器播放矩阵继续按
  原生成功/失败事件判定，不从服务端解码结果推断。
- 升级后“补齐缺失”只重排需要处理的派生项；已有 ready 资源、schema、transform version 和
  原媒体 hash/mtime 保持不变，真实损坏 fixture 仍终态失败。
