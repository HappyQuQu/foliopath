# 限流时钟回拨恢复

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：共享 HTTP transport 的例行安全与可用性修复
- Requirement：`FR-AUTH-002`、`FR-INT-002`、`NFR-SEC-003`、`NFR-INT-008`
- Owner：`internal/api` request rate limiter
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 发现与修复

固定窗口 bucket 使用去除 monotonic 信息的 UTC wall clock。系统时钟回拨后，旧 `start` 会位于未来：
已用满 bucket 的 `Retry-After` 可能超过一分钟；当 4,096 bucket 已满时，未来 bucket 也无法过期，导致
新客户端被长期拒绝。

limiter 现在将未来的 bucket 起点重基准到当前时间，同时保留 `used`：

- 同一客户端不会因回拨恢复额度，登录和 semantic search 继续失败关闭；
- `Retry-After` 保持在当前一分钟窗口内；
- 满载 bucket 表在重基准后的一个窗口正常淘汰并接受新客户端。

该策略不信任客户端时间，也不改变 endpoint、客户端地址或 operation-key 所有权。

## 验证

- `go test ./internal/api -run 'TestRequestRateLimiter' -count=50`：通过；
- `go test -race ./internal/api -run 'TestRequestRateLimiter|TestSemanticSearchHasIndependentRatePolicy'`：通过；
- `make fmt`：通过；
- `make arch-check`：通过；
- `make lint`：通过；
- `make test`：通过；
- `git diff --check`：通过。

本修复维护共享 transport，也覆盖当前隔离 semantic route 的 30/min policy；它不注册 production 搜索，
不解除模型、质量集、原生 amd64、容量或供应链阻塞，S2A 仍为 **No-Go**。
