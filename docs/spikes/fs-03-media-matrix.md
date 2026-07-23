# FS-03：固定媒体格式矩阵 spike 报告

## 结论

**状态：Conditional（局部证据通过，完整门槛未通过）**

**验证日期：2026-07-23**

**目标范围：`MVP-2026-07-23` / Stage 0 / `FR-MED-001`、`FR-MED-004～006`、`NFR-PERF-001`、`NFR-COMP-001`**

**验证环境：macOS 26.5.2（Darwin 25.5.0/arm64）、Go 1.26.4、FFmpeg/ffprobe 8.1.2、
cwebp 1.6.0；另在 arm64 Docker Desktop 上以 QEMU 运行 Debian linux/amd64 的
FFmpeg/webp 包重跑同一 fixture；Debian bookworm arm64 容器中使用 libvips 8.14.1 与
govips 2.18.0 运行隔离图片链路**

固定扩展名契约已经由 scanner 单元测试完整锁定：JPEG、PNG、WebP、GIF 与
MP4、MOV、MKV 进入 MVP 索引候选；SVG、HEIC/HEIF、AVIF、DNG、CR3 和无扩展名
文件不进入该候选集。RAW 负例覆盖 DNG、CR3、NEF、ARW、RAF 与 RW2。Darwin/arm64
本机的 FFmpeg 8.1.2 可以探测三个视频容器，
并能为 H.264/yuv420p 的 MP4、MOV、MKV 和 Matroska/FFV1 生成 PNG 封面；截断 MP4
会被 `ffprobe` 拒绝。

同一脚本也在 Debian linux/amd64 的 QEMU 容器中通过，证明 fixture 和发行版 FFmpeg/webp
组合在该模拟环境可运行；这不是原生 amd64、最终镜像或跨架构性能证据。仓库已经定义原生
linux/amd64 与 linux/arm64 CI runner 矩阵；PR #1 的同一合成脚本已在两个原生 runner 通过。

隔离的 `spikes/fs03-vips` module 已在 Debian arm64 验证 JPEG/PNG/WebP 元数据与缩略图、
PNG/WebP alpha、orientation、四帧 GIF/首帧策略和截断 PNG 拒绝。该 module 不进入生产
Go module；项目尚无媒体 adapter、任务队列、HTTP Range 或浏览器 E2E，因而也没有验证
单资产 native call 取消、生产并发上限、像素/帧炸弹隔离、ICC fixture 或目标浏览器直放。
当前证据只允许继续完成媒体后端边界，不足以关闭 R-006～R-008，也不替代最终容器矩阵。

## 目标与证据边界

本 spike 对以下问题分别取证，避免把“扩展名可索引”“工具能解码”和“浏览器能
播放”合并成一个模糊的支持结论：

1. scanner 是否只接受冻结的 MVP 扩展名和 MIME 映射；
2. 小型、完全合成的图片和视频 bitstream 是否可被本机工具探测；
3. FFmpeg 是否能从三种视频容器抽取封面；
4. 动画 GIF 是否保留多帧特征；
5. 一个候选的“可索引、可抽封面、不可假定浏览器直放”编码能否进入验证矩阵；
6. 截断输入是否由直接工具调用安全返回失败；
7. 当前环境中哪些依赖和系统边界仍不存在。

本轮没有读取任何用户媒体。所有 bitstream 都由
[`tests/fixtures/media-matrix/verify.sh`](../../tests/fixtures/media-matrix/verify.sh)
在 `mktemp` 目录内使用 FFmpeg `lavfi` 测试图案生成，并在退出时删除；仓库不保存
生成的媒体二进制。生成方法和 manifest 见
[`tests/fixtures/media-matrix/README.md`](../../tests/fixtures/media-matrix/README.md)。

## 工具与构建能力

| 工具/边界 | 本机结果 | 能支持的本轮结论 |
| --- | --- | --- |
| FFmpeg / ffprobe | 8.1.2，Homebrew arm64 构建；包含 `libx264`、FFV1、MJPEG、PNG、GIF 解码/编码能力 | 可合成与探测视频，抽取视频封面，验证直接 CLI 的损坏输入返回 |
| cwebp | 1.6.0 | 只用于把合成 PNG 转成合成 WebP fixture |
| libvips / pkg-config | Debian arm64：8.14.1 | JPEG/PNG/WebP/GIF loader、缩略图、alpha、orientation 与损坏 PNG 子范围通过 |
| govips | 隔离 module 固定 v2.18.0；不进入主 Go module | Go → govips → libvips 基础调用可行；不证明生产 adapter、取消或并发隔离 |
| FolioPath 媒体 adapter / jobs | 尚不存在 | 不能验证参数数组、超时、取消、进程组回收、输入/输出上限和全局有界并发 |
| HTTP / 浏览器 E2E | 尚不存在 | 不能验证 Range、条件请求、浏览器解码、播放失败状态或跨浏览器差异 |
| Linux amd64 QEMU fixture | Debian 容器中安装 `ffmpeg` 与 `webp` 后，同一脚本通过 | 只证明模拟环境的合成 CLI 矩阵可运行，不证明原生架构、最终镜像或性能 |
| Linux 原生双架构 CI fixture | PR #1 的 amd64/arm64 合成 FFmpeg/webp 脚本均通过 | 证明发行版 CLI 子矩阵可在两个原生 runner 运行；不证明 libvips、产品 adapter、最终镜像或性能 |
| Linux 双架构最终镜像 | 尚不存在 | 不能把 runner 结果泛化为最终 `linux/amd64` / `linux/arm64` 发布能力 |

FFmpeg 配置启用了 GPL 与 `libx264` 等组件。本报告只记录构建事实，不作许可证或
分发结论；SBOM、完整构建选项和 R-014 仍须在发布镜像上审查。

## 固定格式矩阵

“索引候选”表示 scanner 的扩展名/MIME 映射通过，不表示文件内容已经由魔数或
媒体解析器验证。图片的“工具探测”使用 `ffprobe` 只证明合成 bitstream 有效，
不替代指定的 libvips/govips 链路。

| 输入 | 索引候选 | 本机工具探测 | 缩略图/封面 | 浏览器直放 | 本轮判断 |
| --- | --- | --- | --- | --- | --- |
| JPEG 96×64 | 是，`image/jpeg` | `ffprobe`：MJPEG 96×64 | libvips 未运行 | 不适用 | Conditional |
| PNG 96×64 | 是，`image/png` | `ffprobe`：PNG 96×64 | libvips 未运行 | 不适用 | Conditional |
| WebP 96×64 | 是，`image/webp` | `ffprobe`：WebP 96×64 | libvips 未运行 | 不适用 | Conditional |
| GIF 96×64 | 是，`image/gif` | `ffprobe`：GIF，4 帧 | libvips 未运行；首帧/动画策略未定 | 不适用 | Conditional |
| MP4 + H.264/yuv420p | 是，`video/mp4` | 1 秒、96×64，探测通过 | 48×32 PNG 封面通过 | 未运行浏览器 | Conditional |
| MOV + H.264/yuv420p | 是，`video/quicktime` | 1 秒、96×64，探测通过 | 48×32 PNG 封面通过 | 未运行浏览器 | Conditional |
| MKV + H.264/yuv420p | 是，`video/x-matroska` | 1 秒、96×64，探测通过 | 48×32 PNG 封面通过 | 未运行浏览器 | Conditional |
| MKV + FFV1/yuv420p | 按 MKV 扩展名进入候选 | 1 秒、96×64，探测通过 | 48×32 PNG 封面通过 | 作为非直放候选；尚未实际验证 | Conditional |

MP4、MOV 与 MKV 的容器名称和 H.264 编码探测均通过，但不能据此声明三个文件可在
所有目标浏览器播放。浏览器能力取决于容器、编码、profile、音频、平台和响应协议；
生产 UI 必须以实际加载结果安全降级，不能因扩展名或 `ffprobe` 成功就假装可播放。

## 非契约与损坏输入

scanner 矩阵明确断言以下输入不属于 MVP：

| 输入 | 预期 |
| --- | --- |
| `.svg` | 不索引为受支持媒体 |
| `.heic` / `.heif` | 不索引为受支持媒体 |
| `.avif` | 不索引为受支持媒体 |
| `.dng` / `.cr3` / `.nef` / `.arw` / `.raf` / `.rw2` | 不索引为受支持媒体 |
| 无扩展名 | 不索引为受支持媒体 |

验证脚本还把有效 MP4 截断为 64 字节；`ffprobe -v error` 返回非零，脚本按预期通过。
这只证明直接工具调用会拒绝该样本。由于媒体 adapter 和任务状态机尚不存在，本轮
没有证明该失败能在期限内结束、不会泄露 stderr、不会阻塞队列、不会无限重试，
也没有覆盖畸形图片、像素炸弹、长动画、超大 metadata 或主动超时输入。

## 已执行命令与结果

从仓库根目录执行：

```sh
uname -a
uname -m
sw_vers
ffmpeg -hide_banner -version
ffprobe -hide_banner -version
cwebp -version
command -v vips
pkg-config --modversion vips
go list -m all
bash -n tests/fixtures/media-matrix/verify.sh
/usr/bin/time -l bash tests/fixtures/media-matrix/verify.sh
# 另在 --platform linux/amd64 的 Debian 容器中安装 ffmpeg/webp 后运行同一 verify.sh
go test -count=1 -v ./internal/scanner
go test -race -count=1 ./internal/scanner
# Debian arm64 + libvips-dev
make spike-vips
```

结果：

- shell 语法检查、合成媒体矩阵、scanner 单元测试与 scanner race detector 均通过。
- 同一合成媒体脚本在 Linux/amd64 QEMU 容器中通过；该结果不包含 Go `openat2` 路径测试，
  原生平台证据另由 PR #1 CI 提供。
- [PR #1 CI run 29985018814](https://github.com/HappyQuQu/foliopath/actions/runs/29985018814)
  的 Synthetic media amd64/arm64 jobs 均通过。
- Debian arm64 容器的 `make spike-vips` 在 libvips 8.14.1/govips 2.18.0 上通过；固定
  concurrency=1、cache memory=32 MiB，govips 报告测试 high-water mark 544.50 KiB。
- still-image 子测试覆盖 JPEG/PNG/WebP 96×64 load/export、PNG/WebP alpha 和最大 48×32
  thumbnail；orientation 6 归一化后尺寸为 64×96；GIF 报告 4 页并可显式只加载首帧；
  截断 PNG 返回错误。
- 合成矩阵生成 96×64 的 JPEG、PNG、WebP、4 帧 GIF，以及各 1 秒的
  MP4/H.264、MOV/H.264、MKV/H.264 和 MKV/FFV1。
- H.264 与 FFV1 视频均为 `yuv420p`；三个 MVP 容器和 FFV1 候选均成功产生
  48×32 PNG 封面。
- 一次记录运行中，完整 fixture 脚本为 `0.85 real / 0.61 user / 0.22 sys`，
  `/usr/bin/time -l` 报告 `26,935,296` 字节 maximum resident set size。
- 生成输入大小为：JPEG 2,765 B、PNG 1,873 B、WebP 1,442 B、GIF 7,196 B、
  MP4 15,064 B、MOV 15,011 B、H.264 MKV 14,872 B、FFV1 MKV 26,324 B。

这些时间和内存值只描述这台开发机上的一次微型 fixture 命令，不能作为产品吞吐、
并发上限、4 GiB 目标环境或跨架构性能结论。

## 剩余门槛

1. **libvips/govips 完整图片链路**：基础元数据、缩略图、方向、透明度和 GIF 首帧策略
   已有隔离 arm64 证据；仍需原生双架构 CI、ICC fixture、像素/帧限制与最终镜像版本锁。
2. **生产媒体边界**：实现 capability-owned 接口和受控 adapter 后，验证参数数组、
   deadline、取消、子进程回收、输出上限、像素/帧限制、并发 limiter、失败退避与
   不泄露原始 stderr；损坏输入必须隔离到单个资产。
3. **浏览器直放**：通过真实认证 HTTP、Range/HEAD/条件请求，在目标桌面和移动浏览器
   运行 H.264 与明确非兼容编码；记录“可索引、可抽封面、可直放”三种独立状态。
4. **双架构容器**：在 `linux/amd64` 和 `linux/arm64` 最终依赖构建上运行同一脚本和
   产品 adapter 测试，比较 codec、崩溃行为和资源边界。
5. **安全与合规**：补畸形图片、长动画、超大像素声明、探测超时和并发压力；
   对最终镜像生成 SBOM 并审查 libvips/FFmpeg 编译选项与许可证。

完成以上门槛前，FS-03 保持 Conditional，R-006、R-007、R-008 与 R-014 保持开放。
若目标镜像不能稳定提供已冻结矩阵，应按治理流程缩减格式/平台范围或新增 ADR，
不得悄悄用 FFmpeg 图片处理替代已接受的 govips/libvips 架构。
