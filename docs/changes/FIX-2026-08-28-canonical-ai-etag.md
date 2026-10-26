# AI 强 ETag 规范匹配

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：已批准 S2A HTTP adapter 的例行并发正确性修复
- Requirement：`FR-INT-001`、`FR-INT-005`、`NFR-INT-008`
- Owner：`internal/api` AI/semantic revision ETag parser
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 发现与修复

模型激活、operation 取消以及 semantic settings/clear 共用的 `If-Match` parser 原先只把 revision
片段交给 `strconv.ParseInt`。因此服务端从未签发的 `r01`、`r+1` 等文本会被解释成 `r1`，以不同
opaque tag 命中同一数据库 revision，违背强 ETag 必须匹配当前 representation validator 的语义。

parser 现在解析 revision 后再使用 canonical base-10 形式回写比较；只有服务端实际生成格式
`"{resourceId}-r{positiveRevision}"` 才能进入 service。缺失 validator 仍返回 `428`，错误资源、
弱 validator、别名、溢出和 stale revision 仍统一失败关闭为 `412`。

## 验证

- `go test ./internal/api -run 'TestParseAIRevisionETagRejectsNonCanonicalAliases|TestAIModelActivationContract|TestAIModelInstallAndOperationCancellationContract|TestSemanticHTTPSettingsAndBackfillWireContract' -count=100`：通过；
- `go test -race ./internal/api -run 'TestParseAIRevisionETagRejectsNonCanonicalAliases|TestAIModelActivationContract|TestSemanticHTTPSettingsAndBackfillWireContract'`：通过；
- `make fmt`：通过；
- `make arch-check`：通过；
- `make generate-check`：通过；
- `make lint`：通过；
- `make test`：通过；
- `git diff --check`：通过。

本修复不改变 OpenAPI ETag 格式，不注册 production semantic search route，也不解除模型、质量集、
原生 amd64、容量或供应链阻塞；S2A 仍为 **No-Go**。
