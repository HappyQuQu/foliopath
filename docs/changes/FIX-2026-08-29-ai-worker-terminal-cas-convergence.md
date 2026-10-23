# FIX-2026-08-29：AI worker 终态 CAS 竞态收敛

- 目标版本：`POST-MVP-5` revision 1 / S2A
- 关联 Gate：`INT-S2A Backend Evidence Ready`（保持 **No-Go**）
- 影响任务：`INT-202`、`INT-209`
- 影响不变量：operation 状态机只有一个 capability owner；取消与 worker 终态竞争时必须确定性收敛

## 问题

安装与激活 worker 在读取 `running` 后提交失败或成功终态。若取消请求恰好在该读取和 CAS transition
之间推进 revision，原 transition 返回 precondition failure。部分失败出口忽略这个错误，使 operation
可能停在 `cancelling`，只能等进程重启后的 interrupted-operation recovery 才收敛。

已有安装 finalization 路径具备一次重新读取，但安装错误、激活 load/new-ID/generation/commit 错误没有
复用同一策略，形成多个不一致的终态所有者行为。

## 修复

- 在 `aimodel` operation owner 内增加单一 worker-failure 收敛函数。
- 每次读取最新 operation：`cancelling` 优先完成为 `cancelled`；`running` 才提交原 worker failure。
- CAS precondition failure 后有限重试一次，覆盖唯一预期竞争者——并发取消；不做无限重试。
- 安装错误/finalization 与激活 load、generation ID、generation validation、activation commit 的失败出口
  全部复用该 owner。
- app shutdown 的 parent context 已取消时仍保留既有行为，由启动 recovery 收敛，不在关停期继续写状态。

## 回归证据

故障注入让第一次 worker failure transition 同时把 repository 状态推进为 `cancelling` 并增加 revision。
收敛 owner 重新读取后提交 `cancelled/completed`、错误码 `cancelled` 和 finished timestamp，未把用户取消
覆盖成模型失败，也未留下 active operation。

本修复只维护已批准的 operation/worker 合同；不提供最终模型、native amd64、合法质量集、完整 app
强杀链或供应链签署，任务勾选和 S2A **No-Go** 不变。
