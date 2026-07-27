# S2-106 扫描容量与并发回归记录

## 结论

**Go — S2-106 实现完成；进入 S2-107 Backend Ready 汇总。**

本记录确认正式扫描路径在约 10 万媒体/1 万目录主档、深目录、跨媒体库竞争、256 个 active
full scan 上限与 SQLite 写满故障下保持有界和一致。它不把完整扫描后端标记为 Backend
Ready，也不证明媒体探测、缩略图、HTTP catalog、代表性 NAS 或正式发布镜像性能；
`S2-107` 仍须汇总扫描契约、handler、恢复与双架构 CI 证据。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-SCN-001～008`、`NFR-PERF-001`、`NFR-REL-001`
- Contract Ready：[可靠扫描](s2-scan-contract-ready.md)
- 批次和 active admission 上限 owner：`internal/scanner`
- durable 公平领取和全局 worker 上限 owner：`internal/jobs`
- SQLite generation、拓扑归并和事务 owner：`internal/store/sqlite`
- CI 容量档 owner：`.github/workflows/ci.yml`

## 已实现边界

- `scanner.DefaultBatchSize = 256` 是正式 catalog 写批次的唯一常量；
  `scanner.MaxActiveFullScans = 256` 是 creation、manual、startup 与 SQLite admission
  共同使用的唯一 active 上限。
- 256 个 active scan 时，第 257 个媒体库创建在同一事务中失败，不留下半创建 library；
  已存在请求仍可 coalesce。取消释放一个名额后，新请求可以进入，active 总数不越界。
- 两个独立 SQLite store 并发争抢最后一个 admission 名额时，只允许一个成功。
- 正式 `jobs.WorkerPool` 保持全局 2 worker；三个媒体库按 durable
  `available_at_ms, created_at_ms, id` 领取，前两个运行时第三个保持 queued，释放容量后继续。
- 128 层真实扫描、1,000 层 SQLite rollup、循环、跨库/缺失父关系和陈旧父关系继续在
  stale cleanup 前失败关闭。
- SQLite `max_page_count` 故障注入证明 catalog 写满记录稳定
  `database_unavailable`，不发布失败 generation、不删除最后可靠资产；解除限制后的完整
  扫描可以恢复并收敛。

## 容量证据

`tests/performance/TestCapacityBaseline` 贯通真实
`library → internal/files → internal/scanner → SQLite`，使用正式 256 批次并强制
`stage0-comparable-v1` 预算。2026-07-27 本地 Darwin/arm64、`GOMAXPROCS=4` 记录：

| 指标 | 结果 |
| --- | ---: |
| 目录 / 资产 | 10,000 / 100,000 |
| fixture 最大深度 / 分支 | 32 / 8 |
| fixture / 完整扫描 | 6,029 ms / 19,221 ms |
| 扫描期间读取次数 / P95 / max | 768 / 462 µs / 686 µs |
| 扫描后目录 P50 / P95 | 36 / 39 µs |
| 扫描后资产 P50 / P95 | 59 / 67 µs |
| Go heap 采样峰值 | 37,607,368 B |
| checkpoint 后 DB 族 | 28,618,752 B |
| 预算超限 | 0 |

1,000 层目录 rollup 同轮为 70 ms。以上是回归证据，不是用户可见 SLA。

CI 新增独立 `scan-capacity` job，在 Linux amd64/arm64 各以固定 Go 镜像、
`--cpus=4 --memory=4g`、2 GiB tmpfs、`GOMAXPROCS=4` 运行两个容量测试并强制预算。
普通单元任务不隐式承担这一重型档。

## 自动约束与证据

- `internal/store/sqlite/scan_worker_test.go`：256 上限、创建原子性、容量释放及跨连接竞态。
- `tests/integration/scan_worker_capacity_test.go`：真实 SQLite queue、scanner processor 与
  2-worker 跨库领取/阻塞/继续消费。
- `internal/store/sqlite/service_test.go`：真实 SQLite 满页、可靠 generation 保留及恢复。
- `internal/store/sqlite/scanner_test.go`：深目录、损坏拓扑和 finalize 原子回滚。
- `tests/performance/capacity_test.go`：10k/100k 主档、并发读、深链及预算。
- `tests/architecture/dependencies_test.go`：正式 batch/active 常量的唯一 owner，以及
  双架构 CI 资源约束。

本切片的完整验证入口为：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
make spike-capacity
```

## 保留限制与交接

- `S2-107` 汇总 S2-101～106、扫描 HTTP、应用组合及远端双架构结果，才可判断扫描
  Backend Ready。
- 媒体探测/缩略图、浏览查询、FTS、前端虚拟化与相应容量证据属于 Stage 3～4。
- 正式容器强杀、代表性本地存储/NAS、发布 RSS/WAL 与最终镜像属于 Release Gate。
- 禁止声明：扫描 Backend Ready、Stage 3 已授权、完整 FS-04 已通过或 MVP 可发布。
- 评审日期：2026-07-27
