# JPEG 有界容错与 MPEG-TS 派生兼容

## 归属

- 日期：2026-08-12
- 类型：已批准媒体处理切片内的例行兼容修复
- 需求：`FR-MED-001` 、`FR-MED-004～006`、`FR-MED-009～011`、`FR-OPS-008`
- 既有证据边界：`S3-004`、`S3-006`、`S3-007`、`VSP-301`
- 风险：`R-006`、`R-008`、`R-014`、`R-018`
- Owner：`internal/media` 的实际格式核验、解码兼容与处理诊断；
  `internal/thumbnail` 的派生发布、job 终态与 attempt 审计
- 不变量：原媒体只读；只写 `/app/data` 的可重建缓存；不为浏览器转码或 remux

## 问题

一些 JPEG 的标识、尺寸和大部分像素均可解码，但尾部截断或内部数据段提前
结束会让 libvips 严格路径失败。对于家庭媒体预览，若可以在不改写原文件的前提下
生成经验证的有界 WebP，完全拒绝派生会不必要地丢失可用预览。

另有少量已按现有 `.mp4/.mov/.mkv/.avi` 候选扩展名索引的文件，实际容器为
MPEG-TS。这类文件可能能被 FFmpeg 安全读取并生成 poster/storyboard，但它们并不
因此变成合规的 MP4，也不应扩大对 `.ts` 或浏览器直放的承诺。

## 决定

### JPEG 窄范围容错

- 第一次始终使用严格解码。只有已核验为真实 JPEG、原尺寸不超过
  `100,000,000` 像素，且严格缩略图/导出错误精确匹配 `premature end`、
  `incomplete scan` 或 `corrupt JPEG data` 时，才执行一次
  `FailOnError=false` 的 shrink-on-load 容错重试。
- 容错结果仍必须通过既有尺寸、输出字节、WebP 结构、缓存容量和原子发布校验。
  全部成功后 thumbnail 进入 `ready`、job 进入 `succeeded`；不把可用产物伪装成失败。
- 成功容错同时以 `output_validation / decode_recovered / libvips` 记录在成功 attempt
  的内部审计中。当前没有公开 API/schema 的 warning/degraded 字段，失败列表也只返回
  failed job；因此这不是用户可见 warning，不得在产品或运维文档中声称已有可见
  degraded 状态。
- 非 JPEG、超过 100 MP 的 JPEG、0 字节、伪装扩展名、无 JPEG 标识、probe
  失败、未列入 allowlist 的错误，以及容错后产物无效，均继续按既有稳定原因失败。
  100～180 MP 真实 JPEG 仍只使用严格 shrink-on-load，不进入本次容错。

### MPEG-TS 仅派生兼容

- scanner 候选格式不变。只有已因 `.mp4/.mov/.mkv/.avi` 扩展名进入现有视频链、
  且 FFprobe 将真实容器规范识别为 `mpegts` 的输入，才可用 MPEG-TS demuxer 生成
  grid poster 和 storyboard。
- 这是 derive-only 兼容：不新增 `.ts/.mts/.m2ts` 扩展名、`video/mp2t` MIME、
  格式枚举或公开支持承诺，也不 remux、转码、修补、重命名或改写原件。
- 探测和派生仍只通过已锚定的只读文件描述符读取。元数据及有效派生产物成功时
  probe/thumbnail 可 ready，但 `playback_status` 保持 `unknown`。原内容仍按已声明扩展名
  的 MIME 原样 Range 交付；只有当前浏览器的真实播放成功/失败事件能决定能否直放。
- 任何其他未允许的真实容器、无法探测的数据、缺失视频流或缺失解码器仍失败；
  MPEG-TS 容错不是对错误扩展名的通用自动修复。

## 兼容性、审计与部署

本修复不新增公开 API 字段，不改变 SQLite schema、已发布 migration、缓存键或
transform version。既有 ready 派生资源不会因升级而重建。`media_job_attempts` 仍只是每个
job 最近 10 条的有界内部审计，不是永久的源媒体完整性事实；UI 中的尝试计数
明确表达为“本轮”，不把之描述为 job 的全部终身历史。

发布镜像的最小 FFmpeg 包因 MPEG-TS demuxer 变更为 `foliopath-ffmpeg 7.1.5-4`。
它不新增 decoder、filter、encoder 或网络能力；最终双架构 SBOM、notice、漏洞复扫和
不可变 digest 仍由现有 Stage 5 供应链 Gate 阻断。

升级并确认 readiness 后，管理员对受影响媒体库执行一次“补齐缺失”，即可将旧失败
重新纳入有界队列。不需要重新扫描或“全部重建”；不在上述窄容错范围内的媒体
仍会失败。

## 必需回归证据

- 正常 JPEG 仍走严格路径且输出不变；两类受控截断 fixture 证明“严格失败→一次
  容错→有效 WebP→ready/succeeded + 内部 warning”。
- 非 JPEG、伪装 JPG、0 字节、超过 100 MP JPEG、未知 libvips 错误、容错产物无效
  均不进入 ready；原件 hash/mtime 始终不变。
- 真实 MPEG-TS 伪装的现有视频候选可用最终镜像的 mpegts demuxer 完成 probe/poster/
  storyboard，并保持 playback `unknown`；`.ts/.mts/.m2ts` 仍不进入索引合同。
- 未允许容器、无效 TS、解码器缺失与无视频流继续失败；不从服务端派生成功
  推断浏览器可播放。
- 当前轮次尝试计数与中英 `decode_failed` 文案回归；文档不声称内部 warning
  已通过公开 API 或 UI 可见。
