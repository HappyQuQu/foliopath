# VSP-302 目标平台与资源复验

## 当前结论

**Pending native evidence — 自动化入口已完成，本 Gate 尚未签署。**

`VSP-302` 要求同一源码状态在原生 `linux/amd64` 与 `linux/arm64` runner 上完成生产
候选纵切，并由成对校验器确认输入、运行时和解码结果一致。只有远端原生矩阵实际成功后，
本记录才能改为 Go；本机 Docker Desktop 的跨架构模拟结果不能替代原生证据。

## 权威入口

- `.github/workflows/ci.yml` 的 `Storyboard candidate (amd64|arm64)` 使用
  `ubuntu-24.04` 与 `ubuntu-24.04-arm` 原生 runner；
- 每个 runner 执行 `make test-storyboard-vertical`，使用同一生产 `Dockerfile`、同一
  10 秒 fixture 合同和 4 CPU / 4 GiB 资源限制；
- runner 输出 `storyboard-evidence-<arch>.json`，记录 source commit、实际镜像架构和
  digest、FFmpeg 版本、源 SHA-256、5×2/10 帧/1600×360 布局、解码像素 SHA-256、
  cache 200→202→200 修复以及原媒体不变性；
- `Verify paired storyboard evidence` 下载同一 workflow run 的两个 artifact，并执行
  `make verify-storyboard-evidence`，拒绝架构、commit、run、FFmpeg、fixture、布局或
  解码像素不一致，也拒绝同一 run 的不同 run attempt 被误配；
- 校验成功后原子生成并上传
  `storyboard-paired-evidence-<commit>/storyboard-paired-evidence.json`，聚合两个候选
  image digest、实际架构、run/attempt、FFmpeg、fixture/layout/pixel hash、资源限制及
  所有通过项，作为 VSP-302 可归档签署输入。

目标浏览器与输入模式由同一 workflow 的 browser job 验证 Chromium、Firefox、WebKit，
本地完整矩阵另覆盖 Chrome Stable、forced-colors、触摸/粗指针和 reduced-motion。
浏览器证据与原生媒体候选证据是同一源码状态上的两个互补 Gate，不把模拟输入描述为实体设备。

## 本地预检

2026-07-29 在本机 Linux/arm64 Docker 运行时执行新增结构化证据路径成功：

- 实际候选架构：`linux/arm64`；
- FFmpeg：`7.1.5`；
- 布局：10 帧、5×2、1600×360；
- cache 删除后：200→202→200；
- 重建前后 decoded pixel SHA-256 相同；
- 原 MP4 SHA-256 与 mtime 不变。

这证明证据生成器、约束和 arm64 候选路径可运行，但不关闭原生双架构 Gate。

## 远端与模拟执行状态

提交 `0dddc5abf64544cd2ef444838da2a81895629b8e` 已触发 GitHub Actions run
`30439565333`。workflow 被正常解析并创建 24 个 jobs，包括
`Storyboard candidate (amd64)`、`Storyboard candidate (arm64)` 和成对校验 job；
但所有可运行 jobs 都在分配 runner 前失败，`runner_id=0`、steps 为空。GitHub check
annotation 明确说明账户付款失败或支出上限需要调整。因此这是基础设施/计费阻断，不是
测试失败，也不能计为任何平台运行证据。

同日额外用 Docker Desktop 在 arm64 主机交叉构建 `linux/amd64` 生产候选。镜像架构和
FFmpeg 入口可解析，但应用在 `media-root` 安全边界初始化时失败关闭，未生成 evidence。
这种 amd64 用户态＋arm64 LinuxKit 内核的组合不满足原生平台合同，不能为了让模拟环境
通过而放宽 `openat2`/mount 边界。

## 完成条件

- 两个原生 runner 来自同一 source commit 和 workflow run；
- 两个 artifact 均为 `result=passed`；
- 成对校验器成功，且 FFmpeg 版本、fixture SHA-256、布局与 decoded pixel SHA-256 一致；
- 同源码 browser job 成功；
- 相同提交的 Go/migration 与容量 jobs 未突破升级、恢复和冻结资源预算。

完成后才可勾选 `VSP-302` 并进入 `VSP-303` 文档收敛。
