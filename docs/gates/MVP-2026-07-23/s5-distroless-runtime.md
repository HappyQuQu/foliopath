# S5-007F 无 shell 最小运行时切片

## 结论

**Go — 无 shell 的固定 digest distroless 运行时已进入生产 Dockerfile；S5-007 总 Gate
仍为 No-Go。**

2026-07-28 本机原生 linux/arm64 已对同一根 `Dockerfile` 完成发布镜像、媒体矩阵、
Compose/代理、恢复失败语义和快速容量复验。生产镜像不包含 shell、curl、tar 或包管理器，
大小为 27,472,161 bytes。固定 Syft/Trivy 对精确镜像
`sha256:ca5c9c7d7c1fd16a8053dd962c7be80430ec61308451386893cce884f28f437e`
识别到 Debian 13.6、30 个 OS 包和 53 个镜像包；扫描结果为 15 条包级发现、14 个唯一
漏洞、1 Critical / 14 High，`fixedAvailable=0`。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-007F`
- 需求/质量：`NFR-COMP-001`、`NFR-SEC-001～002`
- owner：发布负责人拥有镜像闭包和运行验证；安全负责人拥有漏洞处置；合规负责人拥有
  notices、许可证和源码提供义务签署
- 合同：根 `Dockerfile`、`compose.yaml`、`tests/release/`、
  `scripts/generate-sbom.sh`、`scripts/scan-release-image.sh`
- 风险：R-014、R-017
- 架构影响：最终包装层从通用 Debian slim 改为固定 digest 的 Debian 13 distroless；
  Debian assembly stage 只复制已枚举的动态库、许可证和包状态元数据。单容器部署单元、
  持久化边界、信任边界、媒体工具和模块依赖方向均未改变，因此不新增 ADR

## 实现与证据

- 最终基础层固定为
  `gcr.io/distroless/cc-debian13@sha256:d97bc0a941b8d4be647dc0ee75b264ddbb772f1ac5ba690a4309c00723b23775`。
  Debian trixie-slim 只用于构建和 `runtime-assemble`，不会进入最终镜像。
- assembly stage 按包枚举并复制实际共享库，同时保留
  `/var/lib/dpkg/status.d/` 和许可证文件；SBOM/扫描因此仍能识别实际运行闭包，不以删除
  包数据库制造“零发现”。
- 生产健康检查继续使用 `/app/foliopath healthcheck`。需要 HTTP、归档或故障注入的发布
  smoke 使用隔离的测试 sidecar；生产容器不通过临时安装调试工具改变不可变运行时。
- `make test-release-image` 通过完整 SPA、非 root、只读根/媒体、MVP 图片/视频、
  Compose/可信代理、优雅停止、备份恢复、`SIGKILL`/WAL、64 KiB 数据盘满、损坏数据库
  与 linux/arm64 检查。
- 对上述精确镜像的 1,000 资产快速容量复验通过：scan 1,000 ms，
  browse p95 3.877 ms，search p95 7.469 ms，全局 p95 9.059 ms，
  峰值内存 176,779,264 bytes。
- 精确镜像 SBOM 含 source 48、npm 381、image 53 个包。扫描明确识别 Debian 13.6 和
  30 个 OS 包；当前发现来自 `libglib2.0-0t64`（7）、`libexpat1`（6）、
  `libblkid1`（1）和 `libmount1`（1）。

## 剩余阻断

唯一 Critical 是 `libglib2.0-0t64` 的 `CVE-2026-58016`。其余 14 条 High 与该
GLib、expat、blkid/mount 运行依赖相关；当前扫描数据库未提供修复版本。无 shell 运行时
显著缩小了暴露面，但“暂无修复”仍不能替代风险处置。

`S5-007B` 仍必须在最终 linux/amd64 与 linux/arm64 digest 上复扫，以升级、移除或对每个
具体 CVE 作有时限、可复审的正式接受，并完成 notices、LGPL 分发审阅、provenance 与
最终 digest 绑定。在此之前，`S5-007`、`S5-009` 和 Release Candidate 保持 No-Go。
