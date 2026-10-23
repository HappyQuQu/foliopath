# 语义推理 session 资源计数

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：已批准 S2A 后端切片内的例行完整性修复
- Requirement：`NFR-INT-002`、`NFR-INT-004`、`INT-202`
- 目标版本与阶段：`POST-MVP-5` revision 1 / S2A
- Owner：`internal/app`（generation-bound inference session 生命周期）
- 既有 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)（仍为 No-Go）

## 修复

`semanticSessionOwner` 已有单常驻 session、串行 Run、切代前关闭、故障丢弃和 idle unload 约束，
但没有可并发读取的资源计数。运行中的 native Run 因持有 owner mutex，不能通过读取 session 字段形成
有效诊断证据。

本修复由同一 lifecycle owner 记录当前常驻 session、active Run，以及累计 load、Run、unload 数。
计数不包含 generation/model ID、模型路径、查询、向量或 runtime 原始错误，不改变 HTTP、SQLite、
任务状态机或并发上限。

## 回归证据与边界

- `go test -race ./internal/app -run 'TestSemanticSessionOwner' -count=1`
- 测试在阻塞 Run 期间并发读取 `resident=1`、`active=1`，完成后 active 回到 0，Close 后 resident
  回到 0；切代的两次 load 对应两次 unload。
- 审核模型、真实 RSS、最终 package、原生 Linux/amd64 和完整容量仍由 `INT-202/210` Gate 持有；
  本修复不把 `INT-202` 或 S2A 改为完成/Go。
