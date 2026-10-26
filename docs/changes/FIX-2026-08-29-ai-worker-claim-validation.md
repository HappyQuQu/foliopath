# FIX-2026-08-29：AI worker claim 返回值失败关闭

- 目标版本：`POST-MVP-5` revision 1 / S2A
- 关联 Gate：`INT-S2A Backend Evidence Ready`（保持 **No-Go**）
- 影响任务：`INT-202`、`INT-209`
- 影响不变量：持久任务由 capability owner 校验；损坏 derived state 不得进入模型文件或推理边界

## 问题

模型安装与激活 admission 已验证 repository 创建/重放结果，但 worker 在 durable queue claim 后直接把
返回值交给安装器或激活加载链。SQLite 正常路径会返回预期字段，错误 adapter 或损坏 derived state
仍可能伪造 request hash、candidate/model 绑定、operation phase/revision 或计数。该缺口会让不可信
claim 在被识别前进入文件复制、模型源校验或 native runtime。

进程内 wake signal 不是问题来源：它只提供合并提示，两个 worker 都有一秒 durable queue 轮询，正确性
不依赖通知不丢失。

## 修复

- 安装 worker 在调用 `Install` 前验证 idempotency 长度、canonical request hash、candidate/package/
  manifest/source 绑定，以及唯一合法的 claimed operation 形态 `running/verifying/r2`。
- 激活 worker 在读取模型或调用 source/runtime 前验证相同的请求绑定，以及唯一合法的
  `running/loading/r2` operation 形态。
- 任一异常返回 `ErrRepositoryState`，终止该 worker component，由既有 app lifecycle 失败关闭；不会
  调用安装器、打开模型文件或进入推理 runtime。下次进程启动时，既有 interrupted-operation recovery
  会把已经 claim 的非 restart-safe operation 收敛为 `operation_interrupted`，不会自动重放副作用。

## 回归证据

故障注入 queue 返回结构合法但 request hash 损坏的 claimed work：

- install worker 返回 repository-state error，installer 调用次数为 0；
- activation worker 返回 repository-state error，runtime 未调用且模型文件未打开。

本修复不提供最终审核模型、native amd64、合法质量集、完整 app 强杀或供应链签署，因此不改变任务勾选
与 S2A **No-Go**。
