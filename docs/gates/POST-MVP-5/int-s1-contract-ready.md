# INT-S1：A+B Contract Ready

## 当前判断

**Go（2026-08-27，仅 POST-MVP-5 revision 1 的 A+B）**。产品、capability、数据、OpenAPI、模型包、
安全、部署、测试和事务合同已经冻结，授权进入 S2A 模型管理与图片语义搜索后端实现。

本 Gate 不授权 AI 标签、视频语义、人脸或人物，也不授权在线下载/国内镜像。S2 必须后端优先；
生产 UI 继续等待 `INT-S2 Backend Evidence Ready`。

## 接受证据

- [Frozen scope](../../releases/POST-MVP-5-scope.md)
- [产品需求](../../product-requirements.md)、[用户流程](../../user-flows.md)、[UI 设计](../../ui-design.md)
- [A+B capability/事务合同](../../architecture/intelligent-media-s1-contract.md)
- [离线模型包合同](../../architecture/ai-model-package-contract.md)
- [技术架构](../../architecture/intelligent-media-discovery.md)、[ADR-0013](../../adr/0013-local-ai-runtime-and-derived-vector-index.md)
- [数据模型](../../data-model.md)、[安全](../../security.md)、[部署](../../deployment.md)、[测试策略](../../testing-strategy.md)
- 权威 `api/openapi.yaml`、`api/openapi.sha256` 与生成 TypeScript client

## Contract Ready 条件

- [x] revision 1 需求只包含模型基础和图片语义搜索；历史 `FR-INT-004` 视频需求被明确排除。
- [x] `internal/aimodel`、`internal/semantic`、capability-owned ports、jobs、catalog、files、API、SQLite
  与 app composition 各有唯一 owner 和向内依赖方向。
- [x] 模型/package/operation、library setting、generation、embedding、coverage 与 semantic job 的状态、
  FK/cascade、revision、backup、retention 和删除分类已冻结。
- [x] 只追加 migration 计划与 fresh/upgrade/idempotent/failure/restore 验收矩阵已冻结；S1 未提前创建
  migration，实际 `00022`（或顺延编号）及其证据归 S2。
- [x] OpenAPI 只暴露 A+B endpoint，使用 opaque ID、ETag、幂等、CSRF、稳定错误与有界 operation；
  query、向量、路径、文件名、URL 和 raw runtime error 不可见。
- [x] revision 1 没有 download endpoint、source/mirror/proxy 配置或占位承诺。
- [x] `/models:ro` 的 directory package v1、内建 catalog 比对、managed/direct、扫描与包大小上限、
  8 GiB managed quota、磁盘安全余量和 activation rollover 已冻结。
- [x] partial failure、取消、并发 activation、direct unavailable、clear 与 generation rollover 的事务
  owner 和恢复结果已列成 decision table。
- [x] 合法质量 fixture、Top-20、100k/4 GiB、双架构数值/资源、供应链和原媒体不变判定已冻结；
  它们仍是 S2/Release 的真实证据要求，不因本 Gate 为 Go 而视为通过。
- [x] OpenAPI 离线解析/结构检查、权威 operation set、摘要锁和 TypeScript client 生成确定性通过。

## S2 获准范围

按 `INT-201～213` 实现：

1. 建立 `internal/aimodel`、`internal/semantic` 及 capability-owned ports；
2. 创建只追加 migration 与 SQLite adapters；
3. 实现固定 `/models` 扫描、managed/direct、operation、activation 与不可用恢复；
4. 实现图片 embedding、SQLite float16 exact、coverage、semantic search 与 stable cursor；
5. 接入 API/auth/CSRF/limits，并完成合法真实质量、native 双架构、容量和供应链证据。

若实现发现必须改 endpoint、数据分类、部署单元、网络边界、model package 格式或 transaction owner，
必须先回到 S1 更新合同/ADR，不能由实现静默决定。

## 未解除的后端/发布阻塞

- 合法代表性图片评测集尚未到位，不能通过 Semantic Backend Evidence Gate。
- 原生 linux/amd64 完整 runtime/数值/容量证据尚未到位。
- 最终 runtime/model SBOM、再分发、notices、漏洞/VEX 与 provenance 尚未签署。
- 当前权重包真实 release catalog 值尚未通过 Release Gate，示例 manifest 不能作为发布包。
