# 媒体任务续租写入串行化

- 日期：2026-09-07
- 状态：**Implemented / locally verified**
- 类型：既有媒体处理与 durable job 切片的例行可靠性修复
- Requirement：`FR-MED-001～011`、`NFR-REL-001`、`NFR-PERF-001～002`
- Owner：`internal/store/sqlite` 媒体任务租约持久化
- 关联 Gate：[S3-005 媒体任务与缓存](../gates/MVP-2026-07-23/s3-media-jobs-cache.md)、
  [S5-005 候选容量](../gates/MVP-2026-07-23/s5-release-capacity-candidate.md)

## 问题

生产实例在完整扫描批量写入目录期间处理派生媒体任务。媒体任务每 15 秒刷新一次 lease，
但 `RefreshMediaJobLease` 直接通过连接池写 SQLite，绕过了实例内其他写路径共用的
`writeGate`。当刷新恰好与扫描事务竞争写锁时，它可能在 SQLite busy timeout 后失败；通用
worker 将 lease 刷新失败视为基础设施故障并停止，随后应用生命周期安全关闭，Compose 的
`restart: unless-stopped` 再次启动进程。每次启动按既定扫描合同为所有媒体库 admission
startup scan，因此一次单库手动重扫会表现为多个库反复扫描。

2026-09-07 的 `server222` 生产诊断记录到容器 18 次重启、`OOMKilled=false`，且单库
`POST` 后约一个 heartbeat 周期出现 `application.component_stopped`。扫描记录确认只有目标库
是 `manual`，其余库均为 `startup`。

## 修复

- 媒体任务 lease refresh 改由 SQLite adapter 的 `withWriteTx` 执行，与 scan、job、cache
  等现有写路径共用同一公平、可取消的进程内串行门。
- lease CAS 条件、持续时间、错误语义和 worker 并发数不变；不改变 API、数据库 schema、
  原媒体只读边界或启动扫描合同。
- 新增并发回归，在 busy timeout 短于既有写事务持有时间时，证明 lease refresh 等待
  `writeGate` 释放后成功，而不是旁路竞争并提前失败。

## 验证

- `go test ./internal/store/sqlite ./internal/app ./internal/jobs`：通过。
- `git diff --check`：通过。

生产镜像部署后的长时间扫描与派生媒体处理复验仍需执行；本记录不替代 S5 容量或发布 Gate。
