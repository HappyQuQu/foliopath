# FIX-2026-09-01：人脸设置与任务时钟回拨收敛

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-241`、`INT-250`
- 影响不变量：设置和任务状态转换必须原子、失败关闭，持久化时间不得违反 `updated_at >= created_at`

## 问题

整仓测试在当天午夜后的真实建库时间与固定测试时钟组合下触发 `face_library_settings` 的
`updated_at_ms >= created_at_ms` 约束。生产路径同样可能在系统 wall clock 回拨时遇到：首次设置行继承
library 的创建时间，但使用较早的当前时间作为更新时间；停用 queued/running 任务时，operation/job 的
更新时间和 terminal finished time 也可能早于各自行创建时间。

## 修复

- 首次建立人脸设置行时把更新时间钳制到 `max(library.created_at, now)`。
- 更新既有设置时在 SQL 行级使用 `MAX(created_at_ms, now)`，不依赖调用方时钟单调。
- 停用分析时，queued/running operation 与 job 的更新时间均按各自行创建时间钳制；queued operation 的
  `finished_at_ms` 同样不会早于创建时间。状态、revision、取消语义和事务原子性不变。
- 分析任务 admission 更新既有 progress/settings 时不回退时间；claim/refresh 的 lease 至少覆盖从任务
  `created_at_ms` 起的完整租期，job/operation/progress 的进度、失败终态与取消时间逐行钳制。
- 回归不再依赖日历固定值：明确把时钟放到 library/job 创建之前，验证 enable、disable、queued cancel、
  operation finished time 和人工人物保留。
- 新增完整分析生命周期回归，在 admission 后继续逐分钟回拨，覆盖 claim、refresh、failed finish 与第二个
  queued job cancel，并核对 settings/job/operation 的 updated/finished/lease 不变量。

## 证据边界

三个回归各连续运行 20 次通过，整仓 `make test` 的原失败路径已修复。该修复只处理 S2C 设置/任务事务的
wall-clock 回拨，不替代最终 runtime、真实质量/偏差、原生双架构、联合容量或 owner 签署，因此 Gate
与任务分子不变。
