# Cursor 规范编码与语义 snapshot 失败关闭

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：共享 cursor 与已批准 S2A 切片的例行安全/正确性修复
- Requirement：`FR-BRW-003`、`FR-INT-002`、`FR-INT-005`、`NFR-INT-004`、`NFR-INT-008`
- Owner：`internal/cursor` codec；`internal/semantic` search snapshot
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 发现与修复

检修中的 race 运行复现了语义 cursor “改动末字符后偶尔仍通过”的不稳定测试。除测试可能把末字符改成
原值外，根因还包括 Go Raw URL Base64 decoder 接受末尾未使用 bit 的非规范别名：不同 token 字符串
可以解码为同一组已通过 AEAD 认证的 bytes。

- 共享 `internal/cursor.Codec.Decode` 解码后必须重新编码并与原 token 完全一致；非 canonical alias
  在 AEAD/JSON 处理前返回 `cursor.ErrInvalid`。所有使用共享 codec 的资源因此只有一个规范 token 表示。
- 语义 cursor 篡改测试改为确定性修改中间字符，不再随机保留原字符。
- semantic snapshot 对 generation ID 长度、coverage outcome 总量、member 排序以及 member/excluded
  互斥失败关闭。repository 返回的畸形状态使用专用 `ErrInvalidSemanticSnapshot`，HTTP 映射为脱敏
  `500 internal_error`，不再错误归因于客户端 `400 invalid_request`。
- 多库 coverage 的 eligible/completed/failed/stale/revision 聚合改为受检 `int64` 加法；任一字段将溢出
  时按畸形 snapshot 失败关闭，不向 API 返回回绕后的负计数或错误 complete 状态。

## 验证

- `go test ./internal/cursor ./internal/semantic ./internal/api -count=20`：通过；
- `go test -race ./internal/cursor ./internal/semantic ./internal/api`：通过；
- `make test-race`：通过；
- `go test -shuffle=on -count=10 ./internal/aimodel ./internal/semantic ./internal/api ./internal/app`：通过；
- `make fmt`：通过；
- `make arch-check`：通过；
- `make lint`：通过；
- `make test`：通过；
- `git diff --check`：通过。

本修复不改变 cursor payload、公开 API schema 或排序合同，不注册 production semantic search route，
也不解除模型、质量集、原生 amd64、容量或供应链阻塞；S2A 仍为 **No-Go**。
