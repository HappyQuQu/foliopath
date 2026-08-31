# AI operation 持久化状态校验

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：已批准 S2A 后端切片的例行状态机完整性修复
- Requirement：`FR-INT-001`、`FR-INT-005`、`NFR-INT-004`
- Owner：`internal/aimodel` operation state machine
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 发现与修复

`OperationService.Get` 已校验基本 enum、进度、revision 和 terminal/finished 关系，但没有完整约束
state、phase 与 error code 的组合。数据库损坏或 adapter 缺陷产生的 `queued+completed`、
`running+queued`、成功但带错误码、失败/取消却无错误码等状态仍会被当作合法 operation，继而进入
worker 或 API response。此外，create/transition 的 repository 返回值原先直接透传，写后异常可以完全
绕过只在 `Get` 上执行的校验。

状态机 owner 现统一要求：queued 只能处于 queued phase；running/cancelling 只能处于 active phase；
所有非终态和 succeeded 不得带错误码；failed/cancelled 必须带 1～128 byte 稳定错误码；所有终态仍
必须处于 completed phase 并具有 finished time。`Get`、create 和每次 transition 的 repository 返回值
现在全部通过同一校验。任何不一致的持久化状态返回
`aimodel.ErrRepositoryState`，HTTP 只输出脱敏 `internal_error`。此外，Get 必须返回请求的 operation ID；
create 必须返回刚创建的 identity/kind/owner/初始状态；transition 必须保持 identity/kind/owner/created time，
精确返回请求的状态、phase、progress、total、error code 和单步递增 revision。即使另一条 operation 本身
状态合法，也不能通过 adapter 串单进入调用方。

模型安装和激活 admission 不经过 `OperationService`，因此 `ManagementService` 出口另行复用同一
operation 校验，并要求正确的 operation kind、激活模型绑定以及互斥且完备的 Created/Replayed 标志；
新创建结果还必须是 revision 1 的 queued/queued 状态。可替换 queue/admission adapter 不能通过返回
结构体绕过业务不变量或把错误 operation 写入 `Location`/`ETag`/JSON。

## 验证

- `go test ./internal/aimodel ./internal/store/sqlite ./internal/app ./internal/api -run 'Management|Admission|Operation|Install|Activation|Semantic' -count=20`：通过；
- `go test -race ./internal/aimodel -run 'TestManagementServiceRejectsInvalidAdmissionResults|TestOperationServiceValidatesRepositoryWriteResults'`：通过；
- `make fmt`：通过；
- `make arch-check`：通过；
- `make generate-check`：通过；
- `make lint`：通过；
- `make test`：通过；
- `git diff --check`：通过。

本修复不修改 migration、不注册 production semantic search route，也不解除最终模型、质量集、原生
amd64、容量或供应链阻塞；S2A 仍为 **No-Go**。
