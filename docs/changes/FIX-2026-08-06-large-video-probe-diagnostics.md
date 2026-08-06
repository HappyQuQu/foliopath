# 大视频探测与失败诊断纠偏

## 归属

- 日期：2026-08-06
- 类型：已批准媒体处理切片内的例行可靠性修复
- 需求与 Gate：`FR-MED-001`、`FR-MED-004～005`，复用 S3-004、S3-006、S3-007
- 不变量：原媒体只读；FFprobe/FFmpeg 仍通过 `internal/files` 打开的描述符读取；不转码原视频

## 问题

视频源此前超过 4 GiB 就在 FFprobe 之前被拒绝，并被归类为 `invalid_media`。此外，FFprobe
和 poster 共用一个 15 秒 deadline，且任意非零工具退出都会永久记成媒体无效。这三项叠加
会把正常大视频、慢速 NAS 读取和暂时性工具故障误报为文件损坏。

## 决定

- 视频处理源上限提高到 1 TiB；4 GiB 以上视频继续交给 FFprobe。上限仍用于阻止明显异常的
  文件元数据，不代表应用会把文件读入内存。
- 图片仍保持 256 MiB 编码输入上限，因为当前 govips adapter 会有界读取完整编码输入。
- 超过对应上限返回 `ErrSourceTooLarge`，结构化原因使用 `source_too_large`，不再冒充损坏。
  该终态不原样重试；现有持久化顶层 code 保持 `media_processing_failed`，详情原因提供精确语义，
  因而无需改写 SQLite job 状态机或 migration。
- FFprobe 与 poster 各自获得独立的 60 秒 deadline；并发、单线程、进程组取消、8 MiB stdout
  和 64 KiB stderr 上限保持不变。
- 只有明确的无效数据、缺少 moov、解码损坏或无可提取帧才永久归为 `invalid_media`；缺少
  解码器归为 `unsupported_media`；未知工具非零退出归为可重试的 `media_processing_failed`。
- stdout 产物超过 8 MiB 才使用 `output_limit_exceeded`；stderr 超过 64 KiB 只表示诊断被
  截断，仍从已保留内容识别真实原因。AV1 的平台像素格式/`Function not implemented` 失败
  归为 `decoder_unavailable`，不再误报输出过大。
- FFprobe 成功但无法提供 duration 时仍保存视频尺寸、poster 和可播放性事实，duration 保持
  未知且不生成依赖时长的 storyboard。
- FFprobe 已成功但 poster 解码失败时也保留视频尺寸、时长与 `playback_status`；grid thumbnail
  单独失败，浏览器仍可尝试原视频。poster 能否生成不再决定视频是否被识别。
- `missing_moov_atom` 明确显示“MP4 索引缺失或文件未完整复制”。

## 保留的安全边界与已知限制

- 仍只按 `.mp4/.mov/.mkv/.avi` 扩展名进入视频索引；扩展名错误或其他容器不会自动探测。
- 0 字节文件、无视频流、单边超过 32,768 px 或总像素超过 100 MP 仍拒绝。
- 工具输出超过上限、单阶段超过 60 秒、源离线、缓存满或处理器缺少解码器仍会失败，但分别
  保留结构化原因；未知工具故障最多尝试 3 次。
- 故事板另有按分辨率/帧数计算的 45 秒～3 分钟单次预算、5 分钟总预算和 10→4 帧降级；
  它不影响 grid poster 已成功的视频识别。

## 回归证据

- 4 GiB 以上稀疏视频会实际进入 FFprobe/FFmpeg，并允许 duration 未知。
- 1 TiB 以上源在工具启动前返回 `source_too_large`。
- probe 与 poster 分别耗时但各自未越界时可成功完成。
- poster 解码失败时资产 probe 保持 ready，只有 grid thumbnail 记录失败。
- moov 缺失、未知工具退出、解码器缺失及损坏数据拥有不同分类。
- 超长 AV1 解码 stderr 被截断后仍保留 `decoder_unavailable`。
- OpenAPI/generated client 固定 `source_too_large` 结构化原因与双语文案。
