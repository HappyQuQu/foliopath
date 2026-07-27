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
