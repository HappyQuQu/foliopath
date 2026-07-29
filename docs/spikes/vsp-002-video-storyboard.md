# VSP-002：视频故事板抽帧与 sprite 有界 spike

## 结论

**状态：Passed for Architecture Ready；不替代 Backend Evidence Ready。**

- 验证日期：2026-07-29
- 目标版本：`POST-MVP-1`
- Feature：[FTR-VID-001](../features/video-storyboard-preview.md)
- 需求：`FR-MED-009～011`、`FR-UI-008`
- 风险：R-006、R-009、R-013、R-018
- 环境：macOS 26.5.2 / Darwin arm64、Go 1.26.5、FFmpeg/ffprobe 8.1.2、cwebp 1.6.0
- 可重复程序：[`spikes/video-storyboard`](../../spikes/video-storyboard/)

在完全合成、临时且低分辨率的视频上，输入侧目标时间 fast seek 可以按冻结采样合同生成
4～10 帧，并组合为单个静态 WebP sprite。10 分钟与 2 小时档中，10 次 fast seek 加拼接
分别为约 332ms 和 250ms；顺序 full-decode `fps + tile` 基线分别为约 458ms 和 857ms。
结果支持生产方案使用“每个目标时间一次有界 input seek，最后一次拼接”，不采用从头顺序
解码长视频的方案。

这些结果只证明开发机上的架构可行性。fixture 分辨率、码率和编码复杂度低于真实媒体；
macOS 本机 FFmpeg 缺少 `libwebp` encoder，因此本轮用 FFmpeg 输出 PNG sprite 后通过
`cwebp` 编码。发布 Dockerfile 已明确构建 `--enable-encoder=libwebp`，但正式方案仍须在
目标 linux/amd64、linux/arm64 运行时验证直接 FFmpeg WebP 编码、峰值 RSS、超时、取消和
敌意输入。

## 目标问题

1. 4～10 帧规则和中间 90% 分段中点公式能否产生严格递增、远离首尾的时间点；
2. fast seek 是否能避免成本随完整视频时长线性增长；
3. 4 帧与 10 帧能否组合成带确定 layout 的单张 WebP；
4. 低成本开发证据下应选择什么生产命令结构和资源边界；
5. 当前运行时能力还有哪些必须留给 Contract/Backend Gate 的缺口。

## Fixture 与方法

程序只使用 FFmpeg `lavfi testsrc2` 在 `os.MkdirTemp` 目录中生成视频，结束时删除整个临时
目录。它不读取 `/library` 或开发者媒体。

| Fixture | 时长 | 合成帧率 | 尺寸 | GOP | 目标帧 |
| --- | ---: | ---: | ---: | ---: | ---: |
| two-seconds | 2s | 10fps | 320×180 | 20 | 4 |
| four-seconds | 4s | 10fps | 320×180 | 40 | 4 |
| ten-seconds | 10s | 10fps | 320×180 | 60 | 10 |
| ten-minutes | 600s | 15fps | 320×180 | 150 | 10 |
| two-hours | 7200s | 5fps | 160×90 | 50 | 10 |

fast-seek 方案对每个目标时间执行：

```text
ffmpeg -ss <timestamp> -i <source> -frames:v 1
       -vf scale=<最长边不超过 320> -vcodec png <bounded-temp-frame>
```

取得全部帧后，用一个 `tile=<columns>x<rows>` 命令拼接。当前开发机因缺少 FFmpeg
`libwebp` encoder，先输出 PNG sprite 再使用 cwebp quality 75；目标运行时计划直接输出
WebP。顺序基线使用 `fps=N/duration,scale,tile`，会解码到接近最后一个采样点。

## 实测结果

| Fixture | 源大小 | Sprite layout | WebP 大小 | fast seek + sprite | full decode baseline |
| --- | ---: | ---: | ---: | ---: | ---: |
| 2s | 44,852 B | 4×1 / 1280×180 | 16,748 B | 129ms | 未测 |
| 4s | 94,431 B | 4×1 / 1280×180 | 17,550 B | 135ms | 未测 |
| 10s | 241,662 B | 5×2 / 1600×360 | 44,572 B | 300ms | 未测 |
| 10min | 20,942,364 B | 5×2 / 1600×360 | 45,602 B | 332ms | 458ms |
| 2h | 55,279,535 B | 5×2 / 800×180 | 24,536 B | 250ms | 857ms |

完整命令运行：

```text
7.33s real / 8.37s user / 1.21s sys
74,465,280 bytes maximum resident set size
```

完整时间包含生成全部合成源、两种策略、probe、损坏输入和 Go 启动，不是单个生产任务预算。
损坏 MP4 负例被 FFmpeg 非零退出拒绝，没有生成 ready sprite。

采样点验证：

- 2s：`0.325, 0.775, 1.225, 1.675`
- 4s：`0.650, 1.550, 2.450, 3.350`
- 10s：`0.950 ... 9.050`，间隔 0.9s
- 10min：`57 ... 543` 秒
- 2h：`684 ... 6516` 秒

全部时间点严格递增并位于中间 90% 的分段中点。

## Architecture Ready 决定输入

### 选择

- 每个 storyboard 是一个 durable 逻辑任务；
- 内部最多执行 10 次 input-side fast seek 和 1 次 sprite 拼接；
- 所有目标帧都必须成功，首版不发布部分 storyboard；
- 单帧保持方向和宽高比，最长边不超过 320px；
- 最多 5 列，实际行列/cell 尺寸进入 ready 元数据；
- 目标运行时使用现有最小 FFmpeg 的 `libwebp` encoder；不把 cwebp 加入生产依赖；
- poster/grid 成功后才 admission，storyboard 使用较低优先级；
- 任意失败只降级 poster，不改变资产 probe、播放或索引状态。

### Contract Ready 必须固定

- 总 storyboard job deadline：候选上限 45 秒，须由真实长 GOP/高码率 fixture 验证；
- 每个临时帧最大 1 MiB、全部临时帧最大 10 MiB；
- 最终 sprite 最大 8 MiB，并校验最大约 1.024M pixels；
- 命令必须使用继承 FD、参数数组、单 decoder/filter thread、进程组取消；
- 临时文件必须位于 `/app/data`、随机命名、失败/取消清理，最终文件原子发布；
- source fingerprint 在处理前后 CAS；旧任务不得发布；
- backfill 分批 admission、跨库公平和 grid 优先级的精确 claim tuple。

上述值是 Contract Ready 输入，不是已通过的生产预算。

## 未关闭边界

- 真实 H.264/HEVC/AV1、VFR、旋转 metadata、长 GOP、高码率、4 GiB 输入；
- 继承 FD 配合多次 input seek 的 Linux 行为；
- FFmpeg 直接 `libwebp` sprite 编码及目标最小运行时所需 filters/demux/decoder；
- 单任务 deadline、取消延迟、进程组清理、临时空间和 ENOSPC；
- 两 worker 下 grid 优先级、跨库公平、10 万资产 backfill 和浏览 P95；
- linux/amd64、linux/arm64 峰值 RSS 与相同输出合同；
- Chromium/Firefox/WebKit 对 sprite decode 和 cell 定位的一致性。

这些项目分别由 `VSP-S1 Contract Ready`、`VSP-S2 Backend Evidence Ready` 和
`VSP-S3 Consumer/UI Ready` 阻断。不得用本地微型结果宣称 feature 已实现或达到发布容量。

## 已执行命令

```sh
cd spikes/video-storyboard
gofmt -w main.go
go test ./...
go run .
/usr/bin/time -l go run .
```

结果均成功；第一次执行还按预期发现本机 FFmpeg 缺少 `libwebp` encoder，随后 spike 明确
加入“目标 FFmpeg libwebp / 开发 cwebp fallback”的能力报告。fallback 仅服务 spike，
不改变生产依赖。
