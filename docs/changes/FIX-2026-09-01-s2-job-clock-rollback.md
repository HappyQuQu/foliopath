# FIX-2026-09-01：S2 durable job 时钟回拨收敛

- 目标版本：`POST-MVP-5` revision 2 / S2A、S2B、S2C
- 关联 Gate：`INT-S2A Backend Evidence Ready`、`INT-S2B Backend Evidence Ready`、
  `INT-S2C Privacy Ready`（均保持 **No-Go**）
- 影响任务：`INT-201`、`INT-202`、`INT-209`、`INT-214`、`INT-222`、`INT-223`、`INT-227`、`INT-242`、`INT-243`、`INT-244`、`INT-245`、`INT-247`、`INT-250`
- 影响不变量：durable job 必须 restart-safe、CAS-bound、lease 有界，数据库时间不得违反
  `updated_at >= created_at` 或 `finished_at >= created_at`

## 问题

模型可用性与语义库设置旁路，以及模型安装/激活 operation、语义 backfill、受控标签、视频语义与人脸
分析/清除任务，在更新、claim、lease refresh、进度、取消或终态时直接写入 wall clock。记录创建后系统
时钟回拨会触发 SQLite 时间约束，导致安全状态迁移以存储错误失败；旧的 lease 计算还可能让 deadline
早于任务创建时间，形成已经过期或缩水的租期。

## 修复

- claim/refresh 的 deadline 使用 `MAX(row.created_at_ms + lease, now + lease)`，保证完整租期。
- job 与 operation 的 updated/finished time 逐行钳制到各自 created time。
- 通用 AI operation transition/recovery 以及模型安装/激活 claim 使用同一规则；claim 返回值反映实际钳制后
  的持久化时间，不向上层报告一个数据库中不存在的较早时间。
- semantic/face progress 在重建或推进时不回退现有更新时间；人脸 settings 与人工 anchor 的关联转换使用
  相同规则。
- semantic embedding 的原子批次提交同步钳制 progress、job、operation 与 library settings 时间，确保
  checkpoint、coverage 和完成计数不会因回拨在同一事务中整体失败。
- tag review 幂等请求在逐项 outcome 与 completed 迁移时不再回退请求更新时间，保留 replay 的完整结果。
- stale/reviewed tag suggestion 的 invalidation revision 同样钳制到 suggestion 创建时间，避免回拨把
  generation/source replacement 或人工 review 的短事务整体回滚。
- 模型 availability CAS 与语义库 settings 的首次插入/后续 revision 更新也钳制到记录或 library 创建时间，
  避免不经过 job queue 的控制面旁路重新引入相同缺陷。
- 人脸 observation 原子替换和人物 rename/delete 的 revision 路径使用相同规则；tombstone 时间也不会早于
  人物创建时间，source-change 触发的人工 anchor 失效仍保持事务内更新。
- 人脸 staged cluster build 的激活时间保持 build 自洽，library settings 的 active-build 发布不会因回拨
  违反设置时间约束或留下未发布的 active 指针。
- 人脸 assignment/split/merge/undo/reconciliation 对既有人物、anchor、exclusion 的更新统一钳制；人工关系
  的 revision、audit、undo snapshot 和 generation reconciliation 在持续回拨下仍保持原子。
- revision、claimed revision、checkpoint、幂等、取消、retry/recovery 和事务边界不变；原媒体仍只读。

## 回归证据

新增十八条回归：六条完整生命周期分别覆盖 semantic backfill、tag、tag review clear、video、face analysis
和 semantic/face clear，三条覆盖通用 AI operation、模型安装 claim 与模型激活 claim，另外两条覆盖模型
availability CAS 与语义 settings 的首次插入及后续 revision 更新，一条覆盖 embedding 原子进度提交，两条
覆盖人脸 observation 与人物生命周期，一条覆盖 tag review 幂等请求，一条覆盖 cluster 激活发布，一条覆盖
人脸人工复核、撤销与换代 reconciliation，一条覆盖 stale/reviewed tag suggestion invalidation。
生命周期回归都在 admission 后继续把时钟逐步回拨，验证 claim、完整 lease、running cancel/refresh、finish；分析与
clear 还覆盖 failed finish、空批次和 queued cancel。每条连续运行 20 次通过。

该修复只关闭本地 durable state machine 的回拨缺陷，不替代最终模型、合法质量集、原生双架构、联合容量、
供应链或 owner 签署；三个 Gate 和清单分子保持不变。
