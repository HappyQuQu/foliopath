# S5-007E 内建健康检查运行时切片

## 结论

**Completed — 已移除生产 curl 闭包，但不解除 S5-007 的 No-Go。**

## 范围

- 目标版本：`MVP-2026-07-23`
- task：`S5-007E`
- 需求/质量：`NFR-COMP-001`、`NFR-SEC-001～002`
- owner：后端负责人拥有探针语义；发布负责人拥有镜像与 smoke；安全负责人拥有复扫
- 合同：`GET /health/ready`、`foliopath healthcheck`、根 `Dockerfile`
- 风险：R-014、R-017
- 架构影响：没有改变部署单元、HTTP 契约、信任边界或依赖方向，不新增 ADR

## 实现与边界

- 新增无参数 `foliopath healthcheck`，只访问固定
  `http://127.0.0.1:8080/health/ready`，总超时 2 秒且不跟随 redirect。
- 只有 readiness 返回 HTTP 200 时退出 0；连接失败、超时、redirect 和非 200 均返回
  稳定失败码，不输出底层网络或路径细节。
- 生产 Docker `HEALTHCHECK` 直接执行 `/app/foliopath healthcheck`，运行层不再安装
  `curl`。需要 cookie、POST 或响应体断言的 release smoke 使用隔离的测试专用 HTTP
  sidecar；该 sidecar 不进入生产镜像、生产 SBOM 或发布产物。

## 证据

- `make test-release-image` 在本机原生 linux/arm64 通过完整 SPA、MVP 媒体矩阵、
  Compose/代理、优雅停止、备份/恢复、强杀、满盘与损坏数据库 smoke。
- 最终本机镜像 `foliopath:s5-no-curl-runtime-final`：
  - platform：linux/arm64
  - size：50,342,686 bytes
  - image ID：`sha256:061f76fef6b2bd126ca3cdfeb2a723eae11d8ee951ae5f9de27281f78fec2711`
  - image SPDX：114 packages，且不含 `curl` 或 `libcurl4t64`
- 固定 Trivy 0.70.0 数据库复扫：
  - 35 个包级发现
  - 25 个唯一漏洞
  - 5 Critical / 30 High
  - `fixedAvailable=0`
- Critical 仍为 `perl-base` 4 条和 GLib 1 条。相对 S5-007D，Critical 未下降，
  High 减少 14 条；移除项来自 `curl`、`libcurl4t64` 及其依赖闭包。
- 证据目录：`build/supply-chain-no-curl-runtime-final/`，其中保留 source/npm/image
  SPDX、完整漏洞 JSON、摘要和 SHA-256。

## 剩余阻断

`S5-007B` 仍需在最终 linux/amd64 与 linux/arm64 digest 上重跑并审阅，处置或逐项、
限时地正式接受 5 Critical / 30 High，完成 notices、LGPL 分发审阅与 provenance。
在这些条件满足前，S5-007、S5-009 与 Release Candidate 继续为 No-Go。
