# Stage 2 可靠扫描 Contract Ready

## 结论

**Go — S2-101 Contract Ready。**

完整扫描的 durable admission、状态与阶段、历史/详情、有限 issues、协作取消、默认 24 小时
计划、全局资源上限、租约恢复和媒体库生命周期交接已经冻结。允许进入 `S2-004` 媒体库
生命周期以及 `S2-102` 扫描服务实现；两者必须消费同一份 `scan_runs` admission，不能建立
临时队列或第二套状态机。

本记录不表示扫描后端已经可用，不授权前端接入、Stage 3～5 或发布。

## 范围与权威来源

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- Roadmap：Stage 2 / `S2-SCN`
- 需求：`FR-SCN-001～009`、`FR-LIB-001`、`FR-LIB-006～008`、
  `NFR-SAFE-001`、`NFR-SEC-001～002`、`NFR-PRIV-001`、
  `NFR-REL-001`、`NFR-PERF-001～002`、`NFR-OPS-001`
- Architecture Ready：
  [Stage 2 媒体库与可靠扫描](stage-2-architecture-ready.md)
- HTTP：`api/openapi.yaml`
- 数据：`migrations/00001_initial.sql`、`00003_library_contract.sql`、
  `00004_scan_contract.sql`
- Owner：`internal/scanner`；durable admission/公平领取由 `internal/jobs` 协议负责
- Adapter：`internal/files`、`internal/store/sqlite`
- Composition：`internal/app`
- ADR：0001、0002、0003、0006、0008、0009
- 风险：R-002、R-003、R-004、R-005、R-011、R-012、R-013、R-016

## HTTP 契约

### 创建与合并

- `POST /api/v1/libraries/{libraryId}/scans` 只创建 `manual` 完整扫描请求，不等待遍历。
- 没有 active scan 时，以一个短写事务创建 `queued` `scan_runs` 后返回 `202`、Location、
  强 ETag；提交后才发送进程内 wake signal。
- 同库已有 `queued`/`running` 时原子返回该 run 和 `200`，不创建第二条记录。
- offline 库允许 admission；worker 重新检查根，所以该端点也是唯一离线重试入口。再次
  offline 只产生 terminal offline run，保留可靠索引。
- active removal 阻止新 scan，稳定返回 `409 idempotency_conflict`。不存在的库返回
  `404 library_not_found`。
- 全局 active scan 达到 256 时，普通手动 admission 返回 `429 rate_limited`。创建媒体库
  的首次 scan 与 library 必须同事务成功或整体回滚，不能留下无首次 scan 的库。

### 历史与详情

- `GET /api/v1/libraries/{libraryId}/scans` 使用 `created_at_ms DESC, id DESC` keyset，
  默认 50、最大 200；cursor 带查询指纹且不可跨库复用。
- terminal、offline、cancelled、interrupted 历史不会被后续成功记录隐藏。
- `GET /api/v1/scans/{scanId}` 返回阶段、单调计数、最多 50 个聚合 issue、时间和取消能力；
  支持强 ETag 与 `If-None-Match`/`304`。
- 公开表示任一字段变化都推进 `scan_runs.revision`。ETag 形式由服务端生成且视为不透明，
  客户端不能从 ID 或 revision 自行构造。
- 通常无法预知完整目录树大小，因此 `progressRatio` 默认为 null。只有来自同一次扫描的
  可靠有界分母才能返回 0～1；不得用目录猜测或历史扫描数据伪造百分比。

### 协作取消

- `POST /api/v1/scans/{scanId}/cancel` 对 queued run 在短事务中直接写为 cancelled；对
  running run 只记录首次 `cancel_requested_at_ms` 并立即返回 `202`。
- 重复取消 pending run 幂等返回当前表示。terminal run 返回
  `409 scan_already_finished`，不存在返回 `404 scan_not_found`。
- worker 至少在每个目录、每个最多 256 项数据库批次、finalize 之前检查取消，并把 context
  传播到 walker 与未来媒体子任务。取消绝不进入 stale-generation cleanup。
- 取消允许保留同 generation 已安全提交的新增记录；只有后续完整成功扫描负责收敛。

## Durable 状态与事务

### `scan_runs`

- `queued`：`phase=queued`，尚无 start/heartbeat/lease。
- `running`：phase 依次为 `checking_root → walking/indexing → finalizing`；具备 heartbeat
  和 lease。
- terminal：`succeeded`、`failed`、`cancelled`、`offline` 或 `interrupted`，
  `phase=completed` 且具有 finish 时间。
- `revision` 是公开表示 validator；计数、phase、status、issue 摘要、取消和时间变化均在
  同一短事务中推进。
- 同库 active partial unique index 是 admission 的最终并发防线；内存 mutex/channel
  不能承担正确性。
- generation 在 admission 事务中按库单调分配。只有 succeeded run 的 finalize 事务可以
  发布可靠 generation、聚合计数并删除 stale rows。

### issues 与错误

- `scan_issues` 只保存稳定 code、正的聚合 count 和可空的媒体库相对示例路径；不保存
  errno、stack、stderr、宿主机/容器绝对路径或无界原始名称。
- 每个 run 最多 50 个 issue group，示例相对路径最多 4096 code point；超出时聚合计数
  继续累加，并把 `issues_truncated=true`。
- terminal `error_code` 只能取 OpenAPI 的 `ScanFailureCode`。message 在响应层按稳定 code
  本地化，数据库不持久化自由文本错误。

### 领取、公平与恢复

- `scan_runs` 自身是 durable queue；进程内 wake channel 容量固定为 1，只表示“可能有工作”。
- 全局同时最多 2 个 full-scan worker。同库 unique active 约束仍适用。
- 领取以 `available_at_ms ASC, created_at_ms ASC, id ASC`，在一个 serialized SQLite
  写事务中完成；连续工作仍回到该全局顺序，不为单个库保留私有 worker。
- heartbeat 每 15 秒更新，lease 为 120 秒。领取使 `attempt_count` 增加，最多 3 次。
- 到期 lease 在恢复事务中重新置 queued；第三次到期写为 interrupted。进程启动先完成
  stale lease 恢复，再按 library ID 分批 admission/coalesce `startup` scan。
- failed/offline/cancelled 不自动紧循环重试；管理员、下一次启动或 schedule 重新走同一
  admission。worker 不依赖 HTTP 请求存活。

## 批次、容量与可观察性

- walker 串行流式发出条目；不得把完整树读入内存，不得按 entry 启动 goroutine。
- catalog 写批次固定 256，repository 硬上限 1000；事务期间不做文件 I/O。
- active durable scan 上限 256、worker 2、issue group 50、HTTP page 200。
- 约 10 万媒体/1 万目录、四核/4 GiB 是 `S2-106` 的主要验收档，不是本 Gate 已证明结果。
- 至少观测 queued/running 数、最老 queued age、worker 数、library/scan ID、phase、
  counters、lease recovery、terminal safe code；日志不包含相对路径原文，必要时使用
  安全摘要。

## 计划设置

- `settings` 是 singleton typed row，默认 `scheduled_scan_interval_hours=24`。
- 允许 1～8760 小时；null 关闭 scheduled scan，但不关闭 startup/manual/creation。
- 设置 PATCH 使用强 If-Match；提交后才唤醒 scheduler。
- 每个库下一次 due 由最近一次 full-scan admission 的 `created_at_ms + interval` 推导。
  scheduler 以稳定 library ID 小批读取 due 库并走相同 admission；active scan 自动合并。
- watcher 不能替代 creation/startup/manual/scheduled reconciliation。

## 媒体库交接

- 创建 library、唯一 `library_created` queued scan 与创建幂等记录在一个短事务。
- 移除先阻止 admission，再请求 active scan 协作取消；只有 scan terminal 后才清理应用
  数据。removal 永远拿不到写入或删除 `/library` 的端口。
- 根检查和遍历在事务外且只经 `internal/files`。offline/身份改变/部分不可读都没有 cleanup
  资格，并保留 Library 上一次可靠计数。
- scheduler、手动重试、startup 和 creation 都调用同一个 scanner admission port。

## 自动证据

- OpenAPI 离线解析、外部 lint、生成 TypeScript 与摘要锁。
- operation 测试固定 scan 列表/admission/详情/cancel 的逐状态错误码、ETag、coalesce、
  offline retry、条件轮询和取消语义。
- version 3 → 4 真实 migration；schema 测试覆盖 phase/counters/cancel/lease/attempt、
  默认 24 小时 typed setting、50 issue 上限与非法设置/attempt 拒绝。
- migration/契约测试禁止 host path、absolute path、raw error 和 stderr 持久化。

## 明确未完成

- `S2-004`：媒体库创建事务、HTTP 生命周期、offline retry 和 removal worker。
- `S2-102`：生产 admission、领取、公平 worker 与扫描执行接线。
- `S2-103～106`：完整目录、计数、增量 upsert/finalize、取消/恢复和容量故障矩阵。
- `S2-107`：扫描 Backend Ready。

## Gate 判断

| 判断项 | 结论 |
| --- | --- |
| 范围、版本、Owner、依赖 | 通过 |
| HTTP、错误与轮询 | 通过 |
| durable admission、事务与恢复 | 通过 |
| 资源上限与取消 | 通过 |
| schedule 与媒体库交接 | 通过 |
| migration 与自动化证据 | 通过 |

评审日期：2026-07-27
