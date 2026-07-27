# S2-105 扫描故障与重启恢复实现记录

## 结论

**Go — S2-105 实现完成；进入 S2-106。**

本记录确认取消、离线、部分不可读、挂载边界、根替换和进程重启不会把不完整扫描发布为
可靠代次，也不会删除最后可靠索引；启动重扫和过期 lease 恢复已进入正式应用组合。
它不把扫描后端标记为 Backend Ready，也不提前证明满盘、数据库 I/O、目标容量或正式发布
镜像的强杀恢复；这些仍由 `S2-106～107` 和 Release Gate 完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-SCN-001`、`FR-SCN-005～008`、`NFR-REL-001`、`NFR-SEC-001`
- Contract Ready：[可靠扫描](s2-scan-contract-ready.md)
- generation、失败语义与 startup admission owner：`internal/scanner`
- durable claim、heartbeat、周期 lease recovery owner：`internal/jobs`
- kernel-anchored 文件故障分类：`internal/files`
- keyset 枚举、lease 和 terminal persistence adapter：`internal/store/sqlite`
- 正式生命周期组合：`internal/app`

## 已实现行为

- 应用启动 worker 后，由 `AdmissionService` 按 library ID、每页 64 条枚举可用媒体库，
  admission 或 coalesce `startup` full scan；处于 queued/running removal 的媒体库跳过。
  全局 active 上限达到 256 时等待 worker 释放容量后重试，不创建无界内存列表或第二队列。
- worker 在初始恢复后仍按固定 idle interval 检查过期 lease。因此进程启动时尚未到期、
  但稍后到期的 running scan 也能重新排队；第三次 lease 到期仍按既有状态机终止为
  `interrupted`。
- 根离线记录 `library_root_unavailable`；根 symlink、nested mount、root identity 变化、
  部分目录不可读和普通扫描 I/O 分别记录契约内稳定码。SQLite adapter 对契约外 code
  统一收敛为 `internal_error`，不持久化路径、errno 或自由文本。
- 后代 mount crossing 不再作为可跳过条目；扫描失败关闭。目录 symlink 和既定
  system-derived/recycle 规则仍按已接受策略跳过。
- cancelled、offline、failed 和 interrupted run 都不执行 stale cleanup。失败过程中已经
  安全提交的新行可以保留，但 `current_generation` 与最后可靠旧行不变；后续完整成功扫描
  才收敛并清理陈旧行。
- 正常停机继续协作取消正在运行的 scanner；重启后的 startup admission 会安排完整校准。

## 自动约束与证据

- `internal/scanner/admission_test.go` 验证 startup admission、coalesce、removal skip、
  active-capacity retry 和仅对新 durable work 发 wake。
- `internal/jobs/worker_test.go` 验证恢复先于领取、启动后周期恢复、全局并发上限、
  heartbeat 和 durable cancellation。
- `internal/files/scanner_test.go` 验证 root/child 的 offline、symlink、permission、
  mount-boundary 和一般 I/O 到稳定 scanner error 的唯一映射。
- `internal/store/sqlite/scan_worker_test.go` 验证 startup library keyset 与 active removal
  排除，以及 lease requeue/第三次 interrupted。
- `tests/integration/full_scan_test.go` 验证部分不可读、取消、offline、根替换均保留最后可靠
  generation 和旧索引，之后成功完整扫描才收敛。
- `internal/app/runtime_integration_test.go` 以真实文件、SQLite、正式 composition 和
  worker 验证无 active scan 的启动重扫，以及进程启动后才过期的旧 lease 自动恢复。
- `tests/architecture/dependencies_test.go` 固定周期恢复、startup admission、SQLite keyset
  和 composition 的唯一所有权。

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
```

## 保留限制与交接

- `S2-106` 继续覆盖 256 active 边界、跨库公平、深目录、损坏拓扑、满盘/SQLite I/O 与
  约 10 万媒体/1 万目录容量回归。
- `S2-107` 才能汇总扫描 handler、故障、容量和双架构证据并判断扫描 Backend Ready。
- 正式容器强杀、代表性 NAS 断连、在线备份/恢复和发布镜像升级仍由 Release Gate 验证。
- 不自动重试已经明确 terminal 的 failed/offline/cancelled run；下一次启动、管理员手动
  请求或 schedule 通过同一 durable admission 重新扫描。
- 禁止声明：扫描 Backend Ready、Stage 3 已授权、前端可依赖全部扫描流程或 MVP 可发布。
- 评审日期：2026-07-27
