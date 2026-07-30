# FolioPath Gate 记录

Gate 记录保存 Architecture、Contract、Backend、Frontend、阶段和 Release 判断的可审计证据。
规范见[交付与架构治理](../architecture/delivery-governance.md)。

路径固定为：

```text
docs/gates/<target-version>/<stage-or-slice>-<gate>.md
```

每个实施 PR 必须链接其当前 Gate。`Go` 要求适用检查实际执行并通过；检查缺失或未执行时只能记录
`Conditional Go` 或 `No-Go`，并明确获准的有限下一步。

后续切片 [FTR-VID-001](../features/video-storyboard-preview.md)使用 `POST-MVP-1` 目录记录
`VSP-S0 Architecture Ready`、`VSP-S1 Contract Ready`、`VSP-S2 Backend Evidence Ready`、
`VSP-S3 Consumer/UI Ready` 和 `VSP-S4 Integrated Slice Done`。当前
[VSP-S0 Architecture Ready](POST-MVP-1/vsp-s0-architecture-ready.md)与
[VSP-S1 Contract Ready](POST-MVP-1/vsp-s1-contract-ready.md)、
[VSP-S2 Backend Evidence Ready](POST-MVP-1/vsp-s2-backend-evidence-ready.md)和
[VSP-S3 Consumer/UI Ready](POST-MVP-1/vsp-s3-consumer-ui-ready.md)已 Go；
[VSP-301](POST-MVP-1/vsp-301-product-vertical.md)已完成真实产品纵切。
[VSP-302](POST-MVP-1/vsp-302-target-platform.md)的原生证据入口已建立但仍为 Pending，
因此 S4 尚未签署。

后续切片 [FTR-SCN-001](../features/automatic-library-discovery.md)使用 `POST-MVP-2` 目录。
[WCH-S0 Architecture Ready](POST-MVP-2/wch-s0-architecture-ready.md)已 Go；WCH-001
Linux/arm64 spike 通过、ADR-0011 已接受；[WCH-S1 Contract Ready](POST-MVP-2/wch-s1-contract-ready.md)
已 Go，WCH-S2 生产后端与证据已实现。
WCH-S2 已完成首条 Linux/arm64 真实 watcher 到 catalog 的生产组合链，但仍缺原生 amd64、
其余受控 ENOSPC、nested mount/unmount、强杀、目标容量和 HTTP 故障矩阵均已完成。
[WCH-S2 Backend Evidence Ready](POST-MVP-2/wch-s2-backend-evidence-ready.md)当前仍为
发布 No-Go。2026-07-29 产品负责人明确授权 revision 2 的有界消费者 UI 开发与本地
Linux/arm64 验证；该有限授权不解除原生 amd64 发布阻塞，也不允许宣称跨平台完成。
[WCH-S3 Consumer/UI](POST-MVP-2/wch-s3-consumer-ui.md)已完成刷新按钮、目录导航重取与
cursor 首页面裁剪的本地实现和前端证据，当前为 Conditional Go。

当前记录：

- [UIF-S0：生产前端原型一致性 Architecture Ready](MVP-2026-07-23/uif-s0-architecture-ready.md)
- [UIF-S1：生产前端原型一致性 Contract Ready](MVP-2026-07-23/uif-s1-contract-ready.md)
- [UIF-S2：生产前端原型一致性 Backend Evidence Ready](MVP-2026-07-23/uif-s2-backend-evidence-ready.md)
- [UIF-S3：生产前端原型一致性 Consumer/UI Ready](MVP-2026-07-23/uif-s3-consumer-ui-ready.md)
- [UIF-S4：生产前端原型一致性 Integrated Slice Done](MVP-2026-07-23/uif-s4-integrated-slice-done.md)
- [MVP-2026-07-23 / 阶段 0 当前判断](MVP-2026-07-23/stage-0-current.md)
- [S0-105：路径证据与生产切片顺序](MVP-2026-07-23/s0-105-gate-order.md)
- [S1-101：单管理员认证 Contract Ready](MVP-2026-07-23/s1-auth-contract-ready.md)
- [S1-106：单管理员认证 Backend Evidence Ready](MVP-2026-07-23/s1-auth-backend-ready.md)
- [Stage 2：媒体库与可靠扫描 Architecture Ready](MVP-2026-07-23/stage-2-architecture-ready.md)
- [S2-001：媒体库管理 Contract Ready](MVP-2026-07-23/s2-library-contract-ready.md)
- [S2-101：可靠扫描 Contract Ready](MVP-2026-07-23/s2-scan-contract-ready.md)
- [S2-102：有界扫描 worker 实现完成](MVP-2026-07-23/s2-bounded-scan-worker.md)
- [S2-103：目录索引与计数实现完成](MVP-2026-07-23/s2-directory-counts.md)
- [S2-104：媒体候选与增量收敛实现完成](MVP-2026-07-23/s2-media-convergence.md)
- [S2-105：扫描故障与重启恢复实现完成](MVP-2026-07-23/s2-scan-recovery.md)
- [S2-106：扫描容量与并发回归完成](MVP-2026-07-23/s2-scan-capacity.md)
- [S2-107：可靠扫描 Backend Ready](MVP-2026-07-23/s2-scan-backend-ready.md)
- [S2-004：媒体库生命周期实现完成](MVP-2026-07-23/s2-library-lifecycle-implemented.md)
- [S2-005：媒体库文件系统安全矩阵](MVP-2026-07-23/s2-library-safety-matrix.md)
- [S2-006：媒体库移除原媒体不变证明](MVP-2026-07-23/s2-library-removal-invariance.md)
- [S2-007：媒体库管理 Backend Ready](MVP-2026-07-23/s2-library-backend-ready.md)
- [Stage 2：媒体库与扫描前端 Integrated Done](MVP-2026-07-23/s2-library-scan-integrated-done.md)
- [S3-001：目录与媒体浏览 Contract Ready](MVP-2026-07-23/s3-browse-contract-ready.md)
- [S3-002：Catalog 排序与游标实现完成](MVP-2026-07-23/s3-catalog-keyset.md)
- [S3-003：目录树与详情实现完成](MVP-2026-07-23/s3-directory-tree.md)
- [S3-004：媒体处理基础实现完成](MVP-2026-07-23/s3-media-processing.md)
- [S3-005：媒体任务与缓存保护实现完成](MVP-2026-07-23/s3-media-jobs-cache.md)
- [S3-006：敌意媒体与资源安全实现完成](MVP-2026-07-23/s3-media-resource-safety.md)
- [S3-007：浏览与缩略图 Backend Ready](MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)
- [S4-001：搜索 Contract Ready](MVP-2026-07-23/s4-search-contract-ready.md)
- [S4-002：搜索与 keyset 实现完成](MVP-2026-07-23/s4-search-keyset.md)
- [S4-003：搜索 Backend Ready](MVP-2026-07-23/s4-search-backend-ready.md)
- [S4-004：前端搜索界面完成](MVP-2026-07-23/s4-frontend-search.md)
- [S4-005：搜索复用非模态预览](MVP-2026-07-23/s4-frontend-search-preview.md)
- [S4-006：完整媒体查看器完成](MVP-2026-07-23/s4-frontend-media-viewer.md)
- [S4-007：媒体播放与降级状态完成](MVP-2026-07-23/s4-frontend-media-strategy.md)
- [S4-008：媒体交互矩阵完成](MVP-2026-07-23/s4-frontend-media-matrix.md)
- [Stage 4：搜索与完整查看器 Integrated Done](MVP-2026-07-23/s4-search-media-integrated-done.md)
- [S5-001A：发布候选镜像基础](MVP-2026-07-23/s5-release-image-foundation.md)
- [S5-001B/002：Compose 与候选镜像矩阵](MVP-2026-07-23/s5-compose-candidate-matrix.md)
- [S5-002A/005C：原生 amd64 与真实媒体验证](MVP-2026-07-23/s5-native-amd64-real-media.md)
- [S5-003：可信代理与发布 HTTP 安全](MVP-2026-07-23/s5-trusted-proxy-security.md)
- [S5-004A：恢复与失败关闭候选演练](MVP-2026-07-23/s5-recovery-failure-smoke.md)
- [S5-005A：发布候选容量](MVP-2026-07-23/s5-release-capacity-candidate.md)
- [S5-005D：原生 amd64 目标容量](MVP-2026-07-23/s5-native-amd64-capacity.md)
- [S5-006A：浏览器质量候选](MVP-2026-07-23/s5-browser-quality-candidate.md)
- [S5-007A：候选镜像供应链](MVP-2026-07-23/s5-supply-chain-candidate.md)
- [S5-007C：最小媒体运行时切片](MVP-2026-07-23/s5-minimal-media-runtime.md)
- [S5-007D：最小 FFmpeg 运行时切片](MVP-2026-07-23/s5-minimal-ffmpeg-runtime.md)
- [S5-007E：内建健康检查运行时切片](MVP-2026-07-23/s5-built-in-healthcheck-runtime.md)
- [S5-007F：无 shell 最小运行时切片](MVP-2026-07-23/s5-distroless-runtime.md)
- [S5-007G：修复来源 GLib 运行时切片](MVP-2026-07-23/s5-patched-glib-runtime.md)
- [S5-008：发布文档](MVP-2026-07-23/s5-release-documentation.md)
- [S5-009A：Release Candidate 当前 No-Go 判断](MVP-2026-07-23/s5-release-candidate-current.md)
