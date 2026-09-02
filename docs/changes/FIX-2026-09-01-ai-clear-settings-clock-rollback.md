# FIX-2026-09-01：AI 清除设置时钟回拨收敛

- 目标版本：`POST-MVP-5` revision 2 / S2A、S2C
- 关联 Gate：`INT-S2A Backend Evidence Ready`、`INT-S2C Privacy Ready`（均保持 **No-Go**）
- 影响任务：`INT-209`、`INT-249`、`INT-250`
- 影响不变量：清除任务必须原子、失败关闭，持久化时间不得违反 `updated_at >= created_at`

## 问题

语义或人脸库还没有对应设置行时，清除 admission 会用 library 的创建时间初始化 `created_at_ms`，同时把
调用时钟写入 `updated_at_ms`。系统 wall clock 回拨到 library 创建时间之前时，这会违反数据库约束，
导致本应安全可重放的清除请求以存储错误失败。继续审计还发现 claim、lease refresh、批次进度、取消和终态
也直接写入 wall clock；任务创建后再次回拨会在后续状态迁移触发相同约束，或让 lease 早于完整租期。

## 修复

- 首次创建语义或人脸库设置时，把 `updated_at_ms` 钳制为
  `max(library.created_at_ms, request.created_at_ms)`。
- 已存在的设置行使用 SQLite 行级 `MAX(created_at_ms, request_time)`，不假设调用方 wall clock 单调。
- clear job、operation、人工 anchor 与设置的后续更新时间逐行钳制到各自创建时间；terminal
  `finished_at_ms` 使用相同规则。
- claim/refresh 的 lease deadline 至少为 `row.created_at_ms + lease`，回拨不会产生已经过期或缩水的 lease。
- job、operation、幂等键和 revision 语义不变；修复不移动、修改或删除原媒体。
- 两条回归明确读取真实 library 创建时间，再把服务时钟回拨一分钟，验证语义与人脸清除均成功入队且
  设置时间满足约束。
- 另两条完整生命周期回归在每个阶段继续回拨时钟，覆盖 claim、refresh、空批次、success finish 与第二个
  queued job cancel，并核对 settings/job/operation 的 created、updated、finished 和 lease 不变量。

## 证据边界

四条定向回归各连续运行 20 次通过；`make spike-ai` 与完整 `make test-race` 也通过。该修复不替代最终
审核模型、合法质量集、原生 amd64、联合容量、供应链或 owner 签署，因此清单分子和两个 Gate 判断不变。
