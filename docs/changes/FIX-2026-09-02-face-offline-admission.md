# FIX-2026-09-02：人脸任务 offline admission 收敛

- Slice：POST-MVP-5 revision 2 / S2C（`INT-250`）
- Gate：INT-S2C Implementation Authorized / Release No-Go
- 不变式：offline 不解释为空库、不清除最后可靠派生状态；禁用或取消后不得继续新分析

## 问题

人脸分析任务在领取后若媒体库转为 offline，旧实现仍会遍历候选并把每个媒体打开失败计入普通失败。
虽然旧 observation 不会被删除，但这会继续无效 admission，并把“来源不可用”错误表示成逐资产质量失败。

## 变更

- `face.JobCatalog` 增加 capability-owned library state port，SQLite adapter 只返回 `ready`、`offline` 或
  `not_ready`，不向 runtime 暴露媒体路径。
- worker 在每个候选前先续租并复核 cancellation，再复核 library state；running cancellation 始终优先
  收敛为 `cancelled`。
- offline/not-ready 分别以稳定 `library_offline` / `library_not_ready` 终止 operation，不再调用后续分析器、
  不重建空 cluster，也不改写已有 observation、asset result、人物或人工约束。
- 新任务 admission 也区分 offline、not-ready、disabled 与 model-unavailable；offline 请求不会创建 job/
  operation，HTTP/OpenAPI 使用独立 `library_offline` 409 错误，而不是误报模型不可用。
- enabled 不再被当作唯一可运行信号：只有 `building|ready|degraded` settings state 可接纳任务或激活
  cluster；`awaiting_model|disabled|clearing` 即使出现 enabled/state 竞争组合也失败关闭，且不会先写 job。
- job finish、queued cancel 与 lease recovery 只在 settings 仍由该 job 持有的 `building` 状态下写
  ready/degraded；并发推进到 `awaiting_model` 等较新状态时只收敛 job/operation，不再回写覆盖设置。
- 每候选控制复核现在同时绑定 requested generation、settings 与 generation state；运行中转为
  `awaiting_model`、active generation 变化或 generation 退出 active 会立即以 `model_unavailable` 终止，
  不继续调用 analyzer，也不把模型故障伪装成逐资产失败。
- settings/generation 行确实不存在或状态不兼容才映射 model unavailable；SQLite 查询故障继续作为内部
  repository error 失败关闭，不再被吞并成可恢复的模型缺失。
- observation 写入复核、cluster build/activation、analysis/clear job lease/progress/finish/cancel，以及人工审核
  undo 前置状态的查询错误现在也区分
  `sql.ErrNoRows` 与真实存储故障：缺失或 revision/state 竞争映射稳定 domain conflict/unavailable，连接关闭、
  I/O 或 SQLite 故障保留为 repository error，避免 worker 对基础设施故障执行错误的业务恢复分支。
- clear admission 的 optional settings 不再依赖故意触发 NULL scan error 再查询 library；单次 LEFT JOIN 使用
  `sql.NullInt64` 明确区分“库不存在”“库存在但尚无 settings”“真实查询故障”，避免后一次查询成功掩盖前一次
  storage error。
- 若来源在单个分析期间变为 offline，只允许该次已经完整产生并通过 source fingerprint CAS 的结果提交；
  下一候选前立即停止。
- 聚类的 staged build 在原子激活事务中再次绑定 library=ready、enabled、非 clearing、active generation 和
  generation=active，以及当前 job ID/claim revision 仍为 running；离线、禁用、取消、换代或旧 worker
  竞争会删除未激活 build 并保留旧 active build。长聚类返回后 worker 再次复核控制状态，禁用/取消竞争
  收敛为 cancelled，不会把 operation 误报 succeeded。

## 验证

- `go test ./internal/face ./internal/store/sqlite -count=1`
- offline-before-process：分析调用 `0`，既有 observation/result 保留，operation 为 `library_offline`。
- offline-between-candidates：只完成首个安全批次，后续分析调用为 `0`，进度不伪造失败项。
- running-cancel-before-process：分析调用 `0`，operation/job 均为 `cancelled`。
- offline admission：返回 `ErrFaceLibraryOffline`，job/operation 行数保持 `0`；HTTP wire 返回
  `409 library_offline`。
- cluster activation race：offline/disabled 均拒绝新 build，旧 active cluster 保持唯一，临时 build 无残留；
  clustering 期间 disable 后 job/operation 收敛为 `cancelled`，settings 保持 `disabled`。
- cancellation activation race：cluster activation 绑定 job claim；进入 cancelling 的 job 不发布 active build，
  worker/job/operation 收敛为 `cancelled`。
- stale-worker activation：错误 claimed revision 在任何 cluster build 行创建前返回 conflict，旧 worker
  不能覆盖新 worker 的 active build。
- non-runnable state admission：`enabled=1 + awaiting_model` 返回 `face_not_ready`，job/operation 行数保持 `0`。
- terminal settings race：running job 失败收敛后保留并发写入的 `awaiting_model`，不会覆盖为 degraded。
- model-unavailable-between-candidates：只提交首个已通过 source CAS 的批次，随后 analyzer 调用停止，
  job/operation 为 `model_unavailable` 且 settings 保持 `awaiting_model`。
- storage-error classification：关闭 SQLite 后的 generation contract、cluster rebuild 与 job cancel 均保留
  storage error，不会伪装成 `face_generation_unavailable`、`face_job_not_found` 或 clear conflict；人工审核
  precondition helper 也分别覆盖 storage error、missing row 和 state mismatch。无 settings 的已存在 library
  通过单次 LEFT JOIN 建立 revision 2 clearing state，不把 NULL 误判为查询故障。

以上只关闭 S2C offline/cancel failure-matrix 子项；最终审核 runtime、真实人脸质量/偏差、Linux 双架构、
联合容量与 privacy/compliance 签署仍未完成，Release Gate 保持 No-Go。
