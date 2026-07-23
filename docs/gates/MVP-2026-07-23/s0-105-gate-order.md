# Gate allocation record S0-105：路径证据与生产切片顺序

- 日期：2026-07-23
- 目标版本：`MVP-2026-07-23`
- 类型：交付门禁澄清；不改变产品 scope、架构、安全不变量或发布要求
- 影响：FS-01、Stage 0 Gate、首个受保护文件 API Backend Gate、FS-05/Release Gate
- 结论：`Accepted`

## 问题

原 Stage 0 口径一方面禁止生产 feature handler，另一方面要求 FS-01 在完整关闭前提供生产
handler、认证错误 envelope 和发布容器证据。前置 Gate 要求一个只有通过该 Gate 后才允许创建的
产物，会形成循环依赖，也无法产生诚实的阶段结论。

## 决定

证据按最早能够真实产生它的 Gate 分配，不删除或降低任何要求：

1. **FS-01 / Stage 0 路径可行性范围**
   - 唯一词法策略和 `internal/files` 所有权；
   - Linux `openat2` 失败关闭；
   - traversal、编码、symlink、特殊节点、根替换与错误脱敏；
   - 原生 linux/amd64 与 linux/arm64 的同设备、跨设备和 self-bind mount 拒绝；
   - 测试 HTTP capability 对 ID、GET/HEAD、条件请求和单 Range 的组合方向。
2. **首个受保护文件 API / Backend Gate**
   - 生产 router/handler 对权威 OpenAPI 的一致性；
   - 认证、CSRF、错误 envelope、取消、日志脱敏与代理边界；
   - 复用 FS-01 攻击矩阵，不能以测试 harness 代替生产证据。
3. **FS-05 / Release Gate**
   - 非 root 最终镜像、`/library:ro`、默认 seccomp/LSM 下的 `openat2`；
   - 运行期 unmount、权限变化、`EIO`/`ESTALE`、健康状态和恢复行为；
   - 最终双架构镜像，而不是通用 CI runner。

## 判断

FS-01 的 Stage 0 路径可行性范围已通过。生产 handler 与发布容器要求保持开放，并分别阻断
对应 Backend Gate 和 Release Gate；它们不再反向阻断一个禁止创建这些产物的 Stage 0
可行性任务。

这项调整不代表路径能力已经发布，也不允许提前实现业务 UI。Stage 0 整体仍因 FS-03、FS-04
剩余范围和 FS-05 保持 `Conditional Go`。

## 证据

- [FS-01 报告](../../spikes/fs-01-path-boundary.md)
- [PR #1 原生双架构 CI](https://github.com/HappyQuQu/foliopath/actions/runs/29985018814)
- [ADR-0006：契约驱动、切片内后端优先](../../adr/0006-contract-driven-backend-first.md)
- [ADR-0009：Linux openat2 与单一媒体根](../../adr/0009-linux-openat2-single-media-root.md)
- [当前 Stage 0 Gate](stage-0-current.md)

## 后续强制规则

- 前置 Gate 不得要求只有通过该 Gate 后才允许创建的生产产物。
- 调整证据归属时，必须记录新的阻断 Gate、Owner 和复审触发；不得把“后移”解释为“可选”。
- 首个受保护文件 API 未通过上述 Backend Gate 前，不得标记为 Backend Ready。
- 最终镜像未通过 FS-05/Release Gate 前，不得把通用 CI runner 结果描述为发布证据。
