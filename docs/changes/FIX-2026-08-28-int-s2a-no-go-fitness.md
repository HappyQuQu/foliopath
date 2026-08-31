# INT-S2A No-Go 生产组合 fitness check

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：已批准 S2A 切片的 Gate 防漂移维护
- Requirement：`FR-INT-001～003`、`FR-INT-005`、`NFR-INT-001～010`
- Owner：architecture governance / `internal/app` composition
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 修复

新增 `tests/architecture/intelligent_media_gate_test.go`。当 ADR-0014 仍标为“提议”且 INT-S2A Gate
仍为 **No-Go** 时，AST 级检查强制生产 `internal/app/run.go`：

- 只构造一个 `aimodel.NewCatalog(nil)`，审核 catalog 保持为空；
- 继续组合已批准的 `Semantic` 设置、回填和清理边界；
- 不向 `api.RouteDependencies` 注入 `SemanticSearch`，因此生产搜索 route 保持不注册。

ADR 或 Gate 状态发生变化时检查会先失败，要求同一变更显式更新架构检查和生产组合；不能只改文档或
只接 route 静默越过 Gate。该检查不阻止隔离 adapter、HTTP 单测或已批准的后端证据维护。

## 验证

- `make fmt`：通过；
- `make arch-check`：通过；
- `git diff --check`：通过。

本修复不完成新的 `INT-xxx` 主任务，也不解除最终模型、许可/供应链、合法质量集、原生 amd64、容量或
完整纵向阻塞；S2A 仍为 **No-Go**。
