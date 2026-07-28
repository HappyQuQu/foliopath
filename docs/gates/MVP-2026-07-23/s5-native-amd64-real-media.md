# S5-002A/S5-005C 原生 amd64 与真实媒体验证

## 结论

**Go — 原生 linux/amd64 运行矩阵和一轮真实 ZFS 媒体验证通过。**

2026-07-28 在一台原生 `x86_64` Linux、4 CPU、4 GiB 服务器上运行候选镜像。完整
镜像 smoke（启动、health、MVP 媒体、Compose/代理、恢复与失败关闭）通过；随后把
真实媒体树以只读方式挂载到 `/library`，完成扫描、认证浏览及视频封面交付。测试过程中
发现最小 FFprobe 不接受 `-nostdin`，修复后重新构建候选，并从头重复完整 smoke 和真实
媒体验证。

这份记录形成时不替代同提交、同 run、不可变 digest 的 amd64/arm64 CI artifact，
也不等同 100k/10k 主容量档或浏览器性能签署。随后操作者明确指定本轮以该原生 amd64
服务器与本机原生 arm64 结果作为运行矩阵证据，不等待计费阻断的 amd64 CI；最终不可变
digest、provenance 与发布签署仍是独立条件。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-002A`、`S5-005C`
- 需求/质量：`FR-MED-001～002`、`FR-DEP-001～004`、`NFR-SAFE-001`、
  `NFR-PERF-001～002`、`NFR-COMP-001`
- owner：`internal/media/videoffmpeg` 拥有 FFprobe/FFmpeg 调用合同；
  `tests/release` 拥有候选镜像 smoke
- 风险：R-002、R-005、R-006、R-008、R-009
- 架构影响：没有改变部署单元、媒体信任边界、持久化边界或模块方向；属于已批准
  Stage 5 发布验证中的缺陷修复

## 环境与输入

- Ubuntu 22.04.5 LTS LXC，kernel `7.0.0-3-pve`，原生 `linux/amd64`
- Docker 27.5.1，4 CPU、4 GiB，无 swap
- 媒体位于本地 ZFS dataset；测试前确认目标树以下没有嵌套 mount
- 真实树：1,709 个目录、34,829 个文件，其中 31,899 个文件匹配 MVP 扩展名
- `/library` 为只读 bind；`/app/data` 使用独立临时目录；容器保持只读根、
  `cap-drop ALL`、`no-new-privileges` 和 4 CPU/4 GiB 上限
- Docker Hub 认证请求超时，因此镜像在受控开发机交叉构建为 `linux/amd64` 后以
  `docker save/load` 传输；运行和验收是原生 amd64，但本轮不是原生构建或供应链证明

测试没有移动、重命名、修改或删除媒体。测试前后抽样原文件 SHA-256 相同；退出时删除
测试容器和独立临时 `/app/data`，未触碰同机其他服务。

## 结果

修复后的候选镜像为临时标签
`foliopath:s5-distroless-amd64-server-fix1`，本地 image ID
`sha256:fc458310...`，大小 27,991,999 B。临时标签和截断 ID 只用于复现实验，不是
发布标识。

| 验证项 | 结果 |
| --- | --- |
| 原生 amd64 完整 `image_smoke.sh` | 通过 |
| 媒体 / Compose 安全 / 代理 / recovery | 全部通过 |
| 真实完整扫描 | 37.369 s |
| 扫描目录 / 资产 | 1,709 / 31,899 |
| 30 次递归浏览 P95 | 24.680 ms |
| 运行时抽样资源 | 142.7 MiB / 4 GiB；瞬时 148.45% CPU |
| `/library` mount | `rw=false` |
| 原文件抽样 SHA-256 | 扫描前后相同 |
| 后台派生抽样 | image ready 1,474；animated ready 10；video ready 228；0 failed |
| 认证视频缩略图 | HTTP 200，`RIFF....WEBP` magic |

上述时间来自单轮真实树诊断，不是稳定 SLA；媒体派生统计是队列运行中的抽样，不代表
31,899 个资产已全部派生完成。

## 发现、修复与回归

首轮真实扫描本身成功，但已经处理的视频全部以 `invalid_media` 失败：最小 FFprobe
把 `-nostdin` 解释为未知选项，而不是媒体损坏。`internal/media/videoffmpeg` 的 probe
调用现已移除该参数；FFmpeg poster 命令仍保留其支持的 `-nostdin`。

回归 fake probe 明确拒绝 `-nostdin`，防止通用 FFmpeg 参数再次被错误复制到最小
FFprobe。修复后：

```text
go test ./internal/media/videoffmpeg ./internal/media ./internal/app
```

通过；新候选的完整原生 amd64 smoke 再次通过，真实树中视频派生开始持续进入 `ready`，
认证 API 返回有效 WebP。

## 剩余阻断

- `S5-002`：后续原生 amd64 构建/完整 smoke 已与本机原生 arm64 配对并关闭；`S5-001`
  仍需最终不可变 digest 与供应链签署。
- `S5-005`：真实 ZFS 树关闭了一项实际存储兼容和媒体混合风险；后续 100k/10k、
  无变化复扫、完整媒体吞吐、DB/cache 增长、取消及目标浏览器 FPS/RSS 已由汇总容量
  Gate 验证并关闭。
- `S5-007`：修复后的最终双架构 digest 仍须重新扫描并完成漏洞、许可证和来源签署。

因此本证据可勾选 `S5-002A/S5-005C`；后续矩阵与容量证据已关闭 `S5-002/S5-005`，
但不能勾选 `S5-009` 或 Release Candidate。
