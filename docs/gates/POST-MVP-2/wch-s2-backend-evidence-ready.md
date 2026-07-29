# WCH-S2 媒体库自动发现 Backend Evidence Ready

## 当前结论

**No-Go — 等待原生 Linux/amd64 证据。**

生产后端、Linux/arm64 安全/恢复/容量矩阵及 HTTP 合同已经完成，但本机与 Docker daemon
均为 arm64。QEMU `linux/amd64` 因不能提供项目要求的 `openat2` 安全边界而按设计失败关闭，
不能替代原生 amd64。此前端仍未获授权。

## 已完成证据

- migration 12 fresh/upgrade、CHECK/FK/唯一键、library cascade、full scan/reconcile
  双向互斥与 content revision 独立性；
- durable upsert、requested/claimed watermark、lease、1/2/4/8/16 秒退避、跨 SQLite
  重开恢复，以及独立领取进程被操作系统强杀后的 attempt 2 恢复；
- Linux inotify create/close-write/move/rename/delete、空目录、后续新目录递归注册、
  slow copy、symlink skip、root invalidation、100k event burst 与 fallback scan；
- 受控真实内核 `ENOSPC` 映射和部分注册回滚；运行时 nested bind mount 失败关闭、旧索引
  保留以及真实 unmount 后 durable 重试收敛；
- catalog state 的真实认证 `401/200/304`、强 ETag/no-store，以及受控
  `429/500`、`Retry-After` 和错误脱敏；
- 四核、4 GiB Linux/arm64 的 10k 目录/100k 媒体主档及增量自动发现容量。

## Linux/arm64 容量结果

| 项目 | 结果 |
| --- | ---: |
| 完整扫描 | 55.3 s |
| 完整档峰值 RSS | 51.2 MiB |
| 10,001 directory watches 注册 | 1.35 s |
| watch 注册后的目录级额外 FD | 0（513-watch 回归允许测试噪声上限 4） |
| 100 个跨目录新增端到端发布 | 1.37 s |
| 单目录 reconcile P95 | 13.3 ms |
| 增量阶段进程写调用 | 55.6 MB；约 556 KB/变更 |
| 既有容量预算违规 | 0 |

`tests/performance/automatic_discovery_linux_test.go` 固定 watch 注册 30 秒、事件收集 5 秒、
100 项端到端 10 秒及 1 MiB/变更的证据守卫线。

## 实际执行

2026-07-29 成功执行：

```text
make fmt
make test
make arch-check
make generate-check
make lint
make contract-check
make fmt-check
make test-integration
make test-e2e
go test -race -count=1 ./internal/api ./internal/store/sqlite
Linux/arm64 files/app/integration
Linux/arm64 10k/100k full capacity
Linux/arm64 automatic-discovery 10k/100k capacity
Linux/arm64 privileged watch ENOSPC
Linux/arm64 privileged nested mount/unmount recovery
git diff --check
```

QEMU `linux/amd64` 的 API 测试通过；依赖媒体路径的 files/app/integration 因
`openat2` 不可用返回 `kernel path boundary unavailable`。这是预期的安全失败，不记录为
原生平台通过。

## 唯一阻塞与下一授权

必须在原生 Linux/amd64 runner 上重复：

```text
go test -count=1 ./internal/files ./internal/api ./internal/app ./tests/integration
go test -race -count=1 ./internal/api ./internal/store/sqlite ./internal/app
FOLIOPATH_CAPACITY=1 FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1 GOMAXPROCS=4 \
  go test -count=1 -run '^TestCapacityBaseline$' -v ./tests/performance
FOLIOPATH_AUTOMATIC_DISCOVERY_CAPACITY=1 GOMAXPROCS=4 \
  go test -count=1 -run '^TestAutomaticDiscoveryCapacity$' -v ./tests/performance
```

并在具有 `CAP_SYS_ADMIN` 的隔离 namespace 重跑 `fsboundary` 与受控 watch-resource
测试。全部通过后才可把本 Gate 改为发布 Go。2026-07-29 产品负责人明确授权 revision 2
的刷新按钮与目录导航重取进入 WCH-S3 本地实现和 Linux/arm64 验证；该有限授权不解除
原生 amd64 发布阻塞。Revision 2 UI 不宣称停留页面会自动变化。
