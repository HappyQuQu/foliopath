# AI JSON 请求体硬上限

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：已批准 S2A HTTP adapter 的例行资源安全修复
- Requirement：`FR-INT-001`、`FR-INT-005`、`NFR-INT-008`
- Owner：`internal/api` AI/semantic JSON decoder
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 发现与修复

共享 AI JSON decoder 原先使用 `io.LimitReader(body, 4097)`，但没有检查是否实际触及限制。合法小对象后
附加超过 4 KiB 的空白时，JSON decoder 会把限制边界当作 EOF，导致超限请求仍进入 install、semantic
settings/backfill/clear 服务。

decoder 现使用显式 `io.LimitedReader` 保存剩余额度，并在完成唯一 JSON value 与 trailing-content 校验后
要求第 4,097 个字节未被读取。达到该字节即统一返回 `400 invalid_request`；unknown field、第二个 JSON
value、非 JSON Content-Type 和畸形 JSON 的既有失败关闭语义不变。与认证和媒体库写接口一致，
`Content-Type` 现按标准媒体类型解析，因此接受合法的 `application/json` 参数（例如
`application/json; charset=utf-8`），仍拒绝其他媒体类型和畸形参数。

## 验证

- `go test ./internal/api -run 'TestAIModelInstallAndOperationCancellationContract|TestSemanticHTTPSettingsAndBackfillWireContract|TestAIModelInstallRejectsOversized|TestSemanticJobRejectsOversized|TestAIModelRoutesRejectInvalid|TestSemanticHTTP' -count=50`：通过；
- `go test -race ./internal/api -run 'TestAIModelInstallAndOperationCancellationContract|TestSemanticHTTPSettingsAndBackfillWireContract|TestAIModelInstallRejectsOversized|TestSemanticJobRejectsOversized'`：通过；
- `make fmt`：通过；
- `make arch-check`：通过；
- `make lint`：通过；
- `make test`：通过；
- `git diff --check`：通过。

本修复不改变 OpenAPI body schema，不注册 production semantic search route，也不解除模型、质量集、
原生 amd64、容量或供应链阻塞；S2A 仍为 **No-Go**。
