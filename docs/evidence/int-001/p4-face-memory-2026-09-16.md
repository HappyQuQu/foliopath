# P4 大图人脸内存复现与修复

范围：[修复记录](../../changes/FIX-2026-09-16-face-large-jpeg-memory.md)，FR-INT-009 / NFR-INT-002、003，Owner imagevips/app/测试。原媒体不修改；原始路径不进入报告。

## 失败复现与排除

只读取已有远端匿名副本，复制到本地 `/tmp/foliopath-p4-small.p3v449`；没有更换样本。独立 `facedecode` 诊断经 internal/files 打开只读 `/library`，按匿名编号逐张调用正式 DecodeFaceImage。未调用识别身份或质量评分。

- 原代码 / 768 MiB：在 sample-062.jpg 前后 OOMKilled=true、137；原代码 / 1 GiB 仍 OOM。
- sample-062.jpg 是 8742×11656，单独在 768 MiB 能结束，但实测 cgroup peak 顶到 805306368 bytes。单文件成功不代表批量安全。
- 单独 81 MP 普通 JPEG（含 EXIF 1/6/8）、PNG、渐进 JPEG 都能在 768 MiB 通过；同进程组合测试则能触发 OOM。不能把一个简单 fixture 当完整回归。
- 更改全部 JPEG shrink 倍率、关闭 operation cache、Go 128 MiB soft limit、malloc arena/trim、线程缓存清理、仅提前 shrink 的单因素候选都未让原序列在 768 MiB 完成。全部保留为失败，不认领修复。
- 内存分类显示，libvips tracked memory/refs 归零时仍有 Go/原生常驻内存及大量文件页缓存；不能将整个 cgroup 占用都称为泄漏。原生开图的瞬时峰值和进程余量需要一起验证。

## 最终候选与本地证据

JPEG 用标准库仅读头部，在第一次 native open 前设置既有 >100 MP shrink 参数，移除完整打开后再重开的路径；关闭跨请求 operation cache。没有保留曾试验的 input-v2 身份或改变倍率规则，输入上限/输出处理语义不变。新增 >100 MP 合成回归验证原有 1500×1125 输出。

测试配置 `p4-large`：app 1 CPU/1024 MiB、AI 3 CPU/2304 MiB，禁用 swap；总预算仍 4 CPU/3328 MiB。不是把两边内存上限都提高。旧 `p4-native` 配置保留供复现。

固定候选本地镜像 `foliopath-app:p4-oom-fix`，由原 `p4-search` runtime 加当前 Go 二进制构建，前端/原生库基线不变，不是发行镜像或线上更新。

| 验证 | 结果 |
| --- | --- |
| 同一 90 图片独立连续解码，1 GiB | 退出 0，未 OOM；不再在第 62 个样本退出 |
| 完整应用图片语义 | 54 完成 / 36 超限失败，保留原限制 |
| 完整应用人脸处理 | 90/90，failed/degraded=0，44277 ms；期间 45 次浏览成功 |
| 完整应用视频 | 10/10，10068 ms；期间 11 次浏览成功 |
| 标签 | eligible/completed=54，未把无图片向量的 36 项写成已完成 |
| 中英文搜索 | 8 请求成功；单次 34～51 ms，不是 P95 |
| 副本哈希 | 前后不变 |
| 结束状态 | app/AI 均 running、OOMKilled=false；快照 app 287.4 MiB / AI 2.059 GiB |
| 整体探针 | **退出 1**，因图片语义覆盖未完整；内存修复不能覆盖此失败 |
| 图像处理包 | 最终本地 race 测试通过（包含新增合成大图及已有格式/预算测试） |

原始报告 `test-results/p4-oom/remote-samples-local-fixed.json`，诊断/对照日志同目录；真实样本不提交仓库。更严格的结果判定拒绝 settled job 冒充完整覆盖。

## 原生 amd64 与性能复验

server222 使用已固定 runtime 的 `/opt` 库和独立编译工具，当前源码与锁定依赖 vendor 离线编译。首次代理下载 EOF、随后缺开发头文件均为构建失败，不是测试通过；补齐依赖后构建成功。原生候选 `sha256:3eb413689db5a0b5737571d8e5f7ccf58d07ea2f9fb70bc84c4c8ce1571ec158`（linux/amd64），不是现有生产镜像。

原生图像处理全包 race 二进制，在 1 CPU / 1 GiB / network none 下退出 0；81 MP PNG、JPEG 方向、渐进 JPEG、>100 MP shrink 合同四项均实际执行通过，无跳过。日志 `test-results/p4-oom/native-imagevips.log`。构建源 face_libvips.go 与工作区 hash 相同；runtime.go 仅缺随后补充的说明注释，常量完全一致。

原生完整应用复验（`/tmp/foliopath-p4-native.j0BD3j`，同一远端匿名 90 图/10 视频，p4-large）：

- 图片语义：54 完成 / 36 失败，15150 ms，覆盖未完整。
- 人脸：90/90、失败/降级 0，83633 ms，期间 84 次浏览成功。
- 视频：10/10、失败/降级 0，15119 ms，期间 16 次浏览成功。
- 标签：eligible/completed=54；中英文图片/视频 8 次搜索均成功，52～58 ms（非 P95）。
- app/AI 结束时均 running/healthy、OOMKilled=false；内存快照 236.2 MiB / 1.884 GiB（不是峰值）；副本 hash 前后不变。
- **整体退出 1**，来自严格覆盖检查，不掩盖 36 项失败。原始报告 `test-results/p4-oom/remote-fixed.json`。
- 维护脚本 `services_restored=1`，四个原服务健康、镜像 ID 前后完全相同。只更新独立测试镜像，没有线上部署。

结论：本地与原生 amd64 均在候选代码＋新内存分配下完成此前 OOM 的同一人脸序列；不保证旧 768 MiB 配置、不代表长期/任意输入稳定性或完整质量通过。

本地固定候选 100k 媒体 / 10k 目录基础容量复跑退出 0：扫描 109000 ms（预算 240000）、浏览 P95 62.829 ms、搜索 P95 136.703 ms、全局 61.279 ms、宽搜索 113.164 ms、宽全局 126.784 ms，内存峰值 308592640 bytes。原始机器报告 `test-results/p4-oom/capacity-fixed.json`。只证明这次基础目录回归在预算内，不替代联合 AI 压力。

静态检查：`make arch-check`、小样本结果门禁 Node 测试 3 项、维护脚本 bash 语法、p3-environment Node 语法及 `git diff --check` 均成功。本轮未运行全部七项仓库门禁，不能据此发布。

## 门禁边界

原 36 张图片语义上限、独立质量标注、长期稳定性和完整双架构数值/排名仍未关闭；P4 0/8、总清单 18/34 不因这项修复自动变化。未提交、推送、部署或修改原媒体。
