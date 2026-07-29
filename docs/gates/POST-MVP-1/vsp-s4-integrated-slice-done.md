# VSP-S4 Integrated Slice Done

## 当前结论

**No-Go / Pending — VSP-302 与 VSP-303 尚未完成，本 Gate 未签署。**

本文预先聚合 `VSP-AC-001～008`，避免最终签署时重新解释验收语义。它不改变
[`POST-MVP-1` scope revision 1](../../releases/POST-MVP-1-scope.md)，也不授权把
FTR-VID-001 写入稳定发布。只有每项证据为通过、前置 Gate 均完成且无未处置严重风险，
结论才能改为 Go。

## 前置 Gate

| Gate | 状态 | 证据 |
| --- | --- | --- |
| VSP-S0 Architecture Ready | Go | [记录](vsp-s0-architecture-ready.md) |
| VSP-S1 Contract Ready | Go | [记录](vsp-s1-contract-ready.md) |
| VSP-S2 Backend Evidence Ready | Go | [记录](vsp-s2-backend-evidence-ready.md) |
| VSP-S3 Consumer/UI Ready | Go | [记录](vsp-s3-consumer-ui-ready.md) |
| VSP-301 Product Vertical | Done | [记录](vsp-301-product-vertical.md) |
| VSP-302 Target Platform | **Pending** | [原生双架构复验](vsp-302-target-platform.md) |
| VSP-303 Documentation Convergence | **Prepared / Pending** | [收敛审计](vsp-303-documentation-convergence.md) |

## 验收聚合

| 验收 | 当前证据 | 状态 |
| --- | --- | --- |
| `VSP-AC-001` 4/10 帧采样、WebP sprite、原件不变 | VSP-S2 的采样/FFmpeg/原件 hash+mtime；VSP-301 生产链复核 | 已有自动证据 |
| `VSP-AC-002` poster 优先且 backfill 不阻塞核心能力 | VSP-S2 的 priority/fairness/100k 档；VSP-301 浏览、搜索和原视频预览 | 已有自动证据 |
| `VSP-AC-003` source/cancel/restart/cache/ENOSPC 安全收敛 | VSP-S2 故障矩阵与 VSP-301 cache 202→200 repair | 已有自动证据 |
| `VSP-AC-004` 认证 API 与 OpenAPI 一致、错误脱敏、请求线程不生成 | VSP-S1 合同、VSP-S2 API/contract/integration、VSP-301 200/202/304 | 已有自动证据 |
| `VSP-AC-005` 300ms hover、500ms/帧、生命周期停止、单活动动画 | VSP-S3 组件/浏览器/容量；VSP-301 真实产品 hover | 已有自动证据 |
| `VSP-AC-006` touch/键盘/reduced-motion/a11y/焦点无回归 | VSP-S3 输入/axe/键盘矩阵；VSP-301 预览关闭焦点恢复 | 已有自动证据 |
| `VSP-AC-007` 浏览/搜索共享实现且资源有界 | VSP-S3 架构 fitness、100-video 和 100k 三引擎容量；VSP-301 双入口 | 已有自动证据 |
| `VSP-AC-008` 目标双架构、浏览器、容量、视觉与缓存清理 | 浏览器/容量/缓存证据已有；原生 amd64+arm64 同提交成对候选仍缺失 | **Blocked by VSP-302** |

“已有自动证据”表示对应子结果已通过，不等于本 Gate 通过。`VSP-AC-008` 是完整验收的一部分，
不得用本机 arm64 预检、QEMU/跨架构模拟或未分配 runner 的 workflow 替代。

## 最终签署清单

- [ ] VSP-302 为 Go，记录同一 commit/run、两个实际架构、image digest、FFmpeg、
  fixture/layout/pixel hash、cache repair、资源限制和成对校验结果。
- [ ] VSP-303 为完成，发布说明已填真实版本/commit/digest/证据且不再标记为草案。
- [ ] 当前提交的适用仓库完整验证表面实际成功；未执行项有明确边界且不被写成通过。
- [ ] `VSP-AC-001～008` 每项都有最终不可变链接，不只引用计划。
- [ ] R-018 没有未处置严重风险；fallback 仍可禁用 storyboard 而保留 poster。
- [ ] 产品、架构、capability、安全/数据、可访问性和发布负责人完成 Go 签署。

## 当前外部阻断

提交 `48fb468ce9037580a9508e0193d8dc943596f4e3` 的 GitHub Actions run
`30440591928` 已创建两个 storyboard candidate jobs，但均在 runner 分配前以
`steps=[]` 失败；annotation 明确为账户付款失败或 spending limit。该结果不是测试失败，
也不是任何架构的运行证据。
