# scan-worker 自动退出修复

状态：已复现维护故障路径的本地修复/验收通过，6010 已更新；不宣称所有未知 worker 故障或长期发行验收通过。
用户授权继续修复并在本地 Docker 验证后更新进度。

- Scope：既有 FR-SCN-004/006/007、NFR-OPS-001，MVP 扫描运行可靠性维护；不新增能力或部署单元。
- Owner：jobs（worker 退出阶段）、scanner（准入/调度）、SQLite（事务/错误）、app（生命周期与安全日志）。
- 现象：6010 的 ce6911fb 候选在空闲/浏览期间 scan-worker 退出，主应用自动重启；数据库与扫描行未提供底层原因。
- 先补固定、脱敏的来源/阶段/存储错误码，保留原 fail-closed 行为；不得打印异常原文、SQL、路径或凭据。
- 在合成数据与临时 Docker 下复现根因，再修复并覆盖并发、取消、租约、重启及数据保留；不以重建容器清零计数当通过。
- 不操作 server222；测试不读取或修改真实媒体。无 API/界面/迁移变更时原型无需另造页面。

## 根因与证据范围

诊断候选 `sha256:79523609b355fd4a5c550675664b3546576c4ab0a63704652d19ea6162aac5ae` 保持原 fail-closed 行为，
只新增错误来源、worker 阶段、固定分类和 SQLite 数值码，不返回异常原文。

1. 合成项目 `fp-scan-worker-5e5d31c0`：120 秒，519 次扫描、24,559 次读取、4,721 次客户端取消，
   无组件退出/重启；Go 临时数据库取消快照/长 SQL 与扫描 claim/recovery 并发也通过。
2. 独立合成项目 `fp-scan-worker-ddb6c95d`：先正常完成 44 次扫描、2,030 次读取、402 次取消，restart=0。
   然后只对该项目私有临时目录用 macOS sqlite3 执行 quick_check，同时从容器发起合成库扫描。
   2026-10-23 03:19:44 +08:00，复现 `scan-worker` / source=worker / stage=claim /
   storage_code=522 / failure_code=storage_io，随后 restart=1；宿主机亦出现 FTS 构造 I/O 错误。
   522 是 SQLITE_IOERR_SHORT_READ。该危险访问仅用于一次隔离复现，不进入正常验收流程或真实库。
3. 原 6010 在停止宿主机直接 SQLite 检查后，诊断候选未再观察到退出。此前助手曾对运行中的 bind-mounted
   数据从 macOS 执行 sqlite3 检查，这条检查路径存在已复现风险，应纠正，不能把它当作可靠健康证据。
   旧镜像没有保存数值码，不能声称每一次历史退出都被逐次归因；这里关闭的是已复现的故障路径。

## 修复与合同

遵循 [ADR-0023](../adr/0023-same-kernel-sqlite-maintenance.md)。不把真实 I/O 故障改成自动无限重试，
不修改 scan/job 状态机、租约、事务/索引清理、公共 API、迁移或界面。

- `internal/jobs.WorkerFailure` 保存固定 recovery/claim/heartbeat 阶段并保留 errors.Is/As 链。
- app 日志仅投影固定来源/原因及数值存储码；启动、运行、停止失败均不输出异常原文。未知来源/阶段不透传。
- 增加 `docker exec <app> /app/foliopath database-check`：Linux-only，固定数据位置，mode=ro、只读快照，
  15 秒内执行 quick_check＋foreign_key_check，固定成功/失败文案。不会组合/启动应用组件、迁移、模型或扫描。
- AGENTS 与运维说明禁止从 macOS/Windows 直接打开正在运行的 Linux SQLite WAL 文件。
  离线备份检查必须先停主应用并完整复制数据；原媒体始终只读。

## 最终候选与已执行检查

最终主镜像：`sha256:0a0244c0f3d5d9d637cf4ed48746d8d273e723e0a1b136c3ce07bf9325b0c917`。
AI 未变：`sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8`。

- PASS：`make fmt`、`make arch-check`、`make generate-check`、`make contract-check`、
  `go vet ./internal/... ./cmd/...`、`go test ./internal/... ./cmd/...`。
- PASS：`go test -race ./internal/jobs ./internal/app ./internal/store/sqlite -run 'Test(WorkerPool|RuntimeFailure|DatabaseCheck|IntegrityCheck|ScanPolling)' -count=1`。
  CLI 先失败后成功输出顺序、参数拒绝、macOS 平台拒绝、只读 SQL、活跃 WAL、缺失/损坏/外键/取消和日志保密均有回归。
- PASS：扩展盘 `TMPDIR=/Volumes/extended/foliopath/scan-worker-go-tests-4lOSgj make test-integration`。
- PASS：`make test-e2e`，这是 runtime/application Docker smoke，不是完整产品浏览器矩阵。
- 未通过：无限定 `make lint` / `make test` 的 native facedecode spike 缺本机 pkg-config/libvips；默认内部盘临时目录
  的模型下载集成还触发 insufficient_space 安全余量拒绝。未安装系统依赖、放宽保护或把专项结果冒称全量通过。
- PASS：最终人物 smoke `fp-face-diagnostic-a3e7aff8`，真实模型＋合成媒体，损坏图诊断、短暂扫描控制分支、
  只重试 1 图、重放、重启、隐私/库隔离和原件 hash 验证。不是对原库进行人脸解析，也不替代真实 scanner 长时序。
- PASS：最终主镜像首轮 `fp-scan-worker-45f51f9c`，180 秒、4 库/100 张合成图、760 次完整扫描、
  34,899 次读取、5,471 次客户端取消、318 次容器内 database-check、自动重启 0。
  随后显式重启、库恢复 ready、检查再成功、原件 hash 不变；缺失/损坏只读挂载均固定失败，不创建新库或泄露原文。
  测试末尾的两行 `database check failed` 是上述预期负例，不是主应用检查失败。

最终混合读取复验 PASS：`fp-scan-worker-ec43192b`，相同固定主镜像，180 秒、771 次完整扫描、
34,642 次读取请求、6,178 次客户端取消、321 次容器内 database-check、自动重启 0。
读取含目录资产、扫描历史、标签云和四类 AI 设置；请求计数含收到的限流响应，不冒称每次都做完整数据库读取。
随后的显式重启/恢复、再次检查、缺失与损坏负例、原件 hash 和无 OOM 同样通过。
合成数据保留于 `/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-scan-worker-ec43192b-h8WRnQ`，容器已移除。
初轮脚本因未配置 AI 路由请求 404 中止，不是 worker 故障；最终脚本未提供 AI 配置时只测核心路径，
传入部署描述时再扩展 AI 设置（上述最终复验已传入描述，没有把真实数据库/媒体用于负载测试）。

## 原 6010 更新与备份

- 原项目 `foliopath-ai-original-review` 已更新到上述最终主镜像，AI 未重建/重启，无 schema 变更（仍为 43）。
  未发起原库 AI 任务、清除或改动原件；正常启动扫描仍按既有合同执行。
- 可靠停写备份：`/Volumes/extended/foliopath/backup-6010-scan-worker-verified-20261023-FBogJQ`，完整 data、
  previous.env（ce6911fb）与 current.env（0a0244c0）；复制结束前主应用保持停止。
  仅对独立已停写副本进行宿主机 quick_check/foreign_key_check，结果 ok/无外键问题；随后 Linux 容器命令也通过。
- 更早的 `backup-6010-scan-worker-20261023-o6klZU` 保留但**不作为可靠备份**：当时复制尚在进行即尝试校验和重启，
  未宣称验证通过，随后重新停写、完整复制为上述 verified 目录；不删除或覆盖旧数据。
- 离线干净 WAL 库在完全只读的目录挂载下，SQLite 可能因不能建立所需 WAL/SHM sidecar 拒绝打开；首两次备份 CLI
  检查因此未通过。数据库 mode=ro 不等于 immutable；在 Linux 数据目录允许正常 sidecar 的条件下检查通过。
  不能为此给活跃库加 immutable=1 或忽略 WAL。正常 `docker exec` 使用主应用现有可写 `/app/data`，DB 连接仍只读。
- 更新后从 2026-10-23 03:29:04 +08:00 启动；03:33 后复查容器内 database-check=passed、ready=ready、
  双服务 healthy、restart=0/OOM=false，无 component_stopped/shutdown_failed 日志。诊断候选在停止宿主机直读后
  的先前约 15 分钟观察同样无重启；最终候选短时观察不是无限期稳定性保证。
- 浏览器只读核对原库：4 库、8,719 媒体且四库可用；coser 人物仍为 469/6202、失败 18，自动标签仍关闭。
  无用户整理丢失或人为清零，不把维护修复说成历史 AI 图片已修复。

诊断/故障复现及最终负载测试拥有的临时容器已清理，合成数据保留。既有原 6010、模型缓存与其他环境未被清理。
没有部署 server222。历史无数值码事件不作逐次因果断言，真实 I/O 故障仍会停止 worker；三分钟负载不能代表无限时稳定。

## 可复跑入口

```sh
FOLIOPATH_TEST_APP_IMAGE=sha256:0a0244c0f3d5d9d637cf4ed48746d8d273e723e0a1b136c3ce07bf9325b0c917 \
FOLIOPATH_TEST_SCAN_SECONDS=180 node tests/release/scan_worker_smoke.mjs
```

可选 `FOLIOPATH_TEST_AI_DEPLOYMENT` 使用独立核验的部署描述以增加 AI 设置读取，不要求 AI 推理服务在线；
脚本不下载/修改模型，也不自动读取原实例的环境。`FOLIOPATH_TEST_KEEP=1` 才保留临时容器；默认只清理自己标签和
ID 验证通过的测试容器/网络，所有合成数据保留。不要把清理后打印的随机端口当作长期服务入口。
