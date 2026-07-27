# FolioPath Gate 记录

Gate 记录保存 Architecture、Contract、Backend、Frontend、阶段和 Release 判断的可审计证据。
规范见[交付与架构治理](../architecture/delivery-governance.md)。

路径固定为：

```text
docs/gates/<target-version>/<stage-or-slice>-<gate>.md
```

每个实施 PR 必须链接其当前 Gate。`Go` 要求适用检查实际执行并通过；检查缺失或未执行时只能记录
`Conditional Go` 或 `No-Go`，并明确获准的有限下一步。

当前记录：

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
- [S3-001：目录与媒体浏览 Contract Ready](MVP-2026-07-23/s3-browse-contract-ready.md)
- [S3-002：Catalog 排序与游标实现完成](MVP-2026-07-23/s3-catalog-keyset.md)
- [S3-003：目录树与详情实现完成](MVP-2026-07-23/s3-directory-tree.md)
- [S3-004：媒体处理基础实现完成](MVP-2026-07-23/s3-media-processing.md)
- [S3-005：媒体任务与缓存保护实现完成](MVP-2026-07-23/s3-media-jobs-cache.md)
- [S3-006：敌意媒体与资源安全实现完成](MVP-2026-07-23/s3-media-resource-safety.md)
- [S3-007：浏览与缩略图 Backend Ready](MVP-2026-07-23/s3-browse-thumbnail-backend-ready.md)
- [S4-001：搜索 Contract Ready](MVP-2026-07-23/s4-search-contract-ready.md)
