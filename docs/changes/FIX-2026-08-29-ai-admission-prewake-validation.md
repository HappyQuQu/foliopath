# AI admission 唤醒前校验

- 日期：2026-08-29
- 状态：**Implemented / verified**
- 类型：已批准 S2A 后端切片的例行队列边界完整性修复
- Requirement：`FR-INT-001`、`NFR-INT-004`、`NFR-INT-008`
- Owner：`internal/aimodel` install/activation admission
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 发现与修复

`ManagementService` 已能在 admission 返回后拒绝无效 operation，但具体 install/activation admission
原先先调用 `Wake()`，再由上层收到结果。queue adapter 若在 create 返回时篡改 candidate source、
operation identity/model binding 或状态，上层虽会失败关闭，worker 仍已被提前唤醒并可能 claim 损坏 work。

install admission 现在在唤醒前校验 idempotency key、request hash、candidate ID、storage mode、完整审核
candidate/package/manifest/source 与 operation kind/state；create 返回还必须与请求 candidate 和初始
operation 完全一致。activation admission 同样在唤醒前绑定 idempotency key、request hash、model ID、
availability revision 和初始 operation。replay 路径复用相同的持久化 work 校验。任何不一致返回
`aimodel.ErrRepositoryState`，不会调用 `Wake()`。

## 验证

- `go test ./internal/aimodel -run 'TestInstallAdmission|TestActivationAdmission|TestManagementServiceRejectsInvalidAdmissionResults' -count=100`：通过；
- `go test -race ./internal/aimodel -run 'TestInstallAdmissionValidatesQueueResultBeforeWake|TestActivationAdmissionValidatesQueueResultBeforeWake'`：通过；
- `make fmt`：通过；
- `make arch-check`：通过；
- `make generate-check`：通过；
- `make lint`：通过；
- `make test`：通过；
- `git diff --check`：通过。

本修复不改变 OpenAPI、不注册 production semantic search route，也不解除最终模型、质量集、原生 amd64、
容量或供应链阻塞；S2A 仍为 **No-Go**。
