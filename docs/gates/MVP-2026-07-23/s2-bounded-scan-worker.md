# S2-102 有界扫描 worker 实现记录

## 结论

**Go — S2-102 实现完成；进入 S2-103。**

本记录只确认 durable scan queue 已接入正式应用，并由全局有界 worker 执行已领取的
generation 扫描。它不把扫描后端标记为 Backend Ready，也不提前证明全部目录/计数、
fingerprint、完整取消/离线/重启故障矩阵或目标容量；这些仍由 `S2-103～S2-107` 完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-SCN-001～002`、`FR-SCN-005～006`、`NFR-REL-001`、
  `NFR-PERF-001`
- Contract Ready：[可靠扫描](s2-scan-contract-ready.md)
- 通用任务协议 owner：`internal/jobs`
- generation 扫描业务 owner：`internal/scanner`
- 文件系统 adapter：`internal/files`
- SQLite adapter：`internal/store/sqlite`
- 生命周期与依赖组装：`internal/app`

## 已实现行为

- `scan_runs` 是唯一 durable queue。容量为 1 的进程内 signal 只负责唤醒；worker 即使漏掉
  signal，也会通过有界 idle poll 再次查询数据库。
- `internal/jobs` 唯一拥有泛型 lease queue、worker 生命周期、heartbeat、协作取消传播和
  全局并发上限。默认固定 2 workers、15 秒 heartbeat、120 秒 lease；非法并发或时序配置
  失败关闭。
- SQLite 在 serialized 写事务中按
  `available_at_ms ASC, created_at_ms ASC, id ASC` 原子领取，领取增加 attempt 并把库标记为
  scanning。heartbeat 不推进公开 revision。
- 应用启动先恢复过期 running lease：第一、二次到期重新排队，第三次写
  `interrupted/scan_interrupted`。恢复完成后 worker 才开始领取。
- `internal/scanner` 只执行已领取的 generation run，不分配第二个 run 或建立私有队列；
  checking-root、walking、finalizing、completed 阶段由唯一状态机推进。
- `internal/app` 组合数据库、allowed-root anchored walker、scanner processor 与 jobs
  worker，并把 worker 放在媒体根和数据库之前关闭，避免关闭期间继续领取或访问资源。
- 正式 composition root 集成测试证明媒体库创建提交的 creation scan 会被真实 worker
  领取、扫描 synthetic 子目录并进入 ready。

## 自动约束与证据

- `internal/jobs/worker_test.go` 证明恢复先于领取、三个任务最多同时运行两个、释放 capacity
  后继续处理，以及 heartbeat 能把 durable cancel 传播给 handler。
- `internal/store/sqlite/scan_worker_test.go` 证明公平顺序、attempt/heartbeat/lease、
  heartbeat revision 稳定，以及第二/第三次过期的 requeue/interrupted 分支。
- `internal/app/runtime_integration_test.go` 使用临时媒体根、真实认证 HTTP、真实 SQLite 和
  production composition 验证 creation scan 自动完成。
- `tests/architecture/dependencies_test.go` 强制 job 协议归 `internal/jobs`、scanner 不领取
  队列、不接触 concrete store/files，并固定单一 durable queue、2-worker 和容量 1 signal。

本切片实际执行并通过：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
```

## 保留限制与交接

- 启动时为全部现有媒体库分批 admission/coalesce startup scan、进程强杀和更完整的
  cancel/offline/权限故障恢复矩阵仍属于 `S2-105`。
- 空目录与完整直接/递归计数验收属于 `S2-103`；媒体 fingerprint、增量 upsert 与成功后
  stale cleanup 验收属于 `S2-104`。
- 跨库压力、公平性、深目录、队列上限和 10 万媒体/1 万目录主档仍属于 `S2-106`。
- 禁止声明：扫描 Backend Ready、依赖扫描的正式前端流程已获准、Stage 3 已开始或 MVP
  可发布。
- 评审日期：2026-07-27
