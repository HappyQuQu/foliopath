# S2-103 目录索引与计数实现记录

## 结论

**Go — S2-103 实现完成；进入 S2-104。**

本记录确认完整扫描会索引全部可读目录（包括空目录），并只在成功 finalize 时发布直接与
递归媒体计数。它不把扫描后端标记为 Backend Ready，也不提前证明 fingerprint、媒体探测、
完整故障/重启矩阵或目标容量；这些仍由 `S2-104～S2-107` 完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-SCN-002`、`FR-SCN-004`、`FR-SCN-009`、`FR-BRW-008`、
  `NFR-REL-001`、`NFR-PERF-001`
- Contract Ready：[可靠扫描](s2-scan-contract-ready.md)
- 遍历与 generation owner：`internal/scanner`
- 只读文件系统 adapter：`internal/files`
- 目录拓扑与计数 adapter：`internal/store/sqlite`
- 正式执行链：`internal/jobs`、`internal/app`

## 已实现行为

- scanner 在每次 generation 中显式写入空路径根目录，并按 walker 的串行 pre-order 输出
  写入每个可读子目录；目录不因自身或后代没有受支持媒体而被隐藏。
- 隐藏目录照常扫描。只有维护清单中的系统派生/回收目录以及文件边界报告的 symlink、
  mount crossing、特殊节点或非法路径会被跳过；目录与文件跳过数分别写入
  `skipped_directories`、`skipped_files`，legacy total 保持二者之和。
- catalog 固定以 256 项批次流式提交，repository 硬上限仍为 1000；parent 必须已在同一
  generation 中出现，不会在 Go 内存中保存完整目录树。
- 成功 finalize 在同一 SQLite 事务中验证根、同库 parent/child 与 generation 关系，删除
  stale row，再计算并发布所有目录的 `direct_asset_count` 与
  `recursive_asset_count`。
- 计数使用 SQLite O(D) 临时拓扑表，以最多 500 个叶目录为一批向父级传播；不会展开
  asset×ancestor 行。每个目录必须恰好更新一次，循环、孤立根、跨库关系或当前行指向陈旧
  parent 会使整个 finalize 回滚。
- failed、cancelled、offline 或 interrupted run 不执行计数发布或 stale cleanup，已有可靠
  generation 与计数保持可用。

## 自动约束与证据

- `internal/store/sqlite/scanner_test.go` 覆盖空根、128 层目录链、逐级直接/递归计数、
  stale cleanup、循环、跨库关系、current-to-stale parent 和 finalize 回滚。
- `tests/integration/full_scan_test.go` 通过真实 `internal/files` walker 验证根、嵌套目录、
  隐藏目录、空目录、系统目录跳过，以及失败/取消/offline/根替换后保留可靠计数并在后续
  成功扫描收敛。
- `internal/app/runtime_integration_test.go` 使用真实认证 HTTP、production composition、
  jobs worker 和文件 SQLite，验证 creation scan 保留根与两层空目录，发布 5 个目录、
  3 个媒体及逐级 `1/3`、`1/2`、`1/1`、`0/0` 计数。
- `tests/architecture/dependencies_test.go` 强制目录发射策略归 scanner、计数 SQL 只存在于
  SQLite scanner adapter，禁止另一 capability 复制直接/递归计数实现。

本切片要求的完整验证入口为：

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

- S2-103 的“媒体计数”沿用冻结 MVP 格式候选分类；source fingerprint、媒体记录增量语义和
  stale asset/目录完整验收属于 `S2-104`。
- 部分不可读目录、权限变化、取消时机、启动 admission 与强杀恢复矩阵属于 `S2-105`。
- 1 万目录/10 万媒体、跨库公平、深目录与队列压力主档属于 `S2-106`；本记录不把既有
  spike 数字转写为生产容量结论。
- 目录树 HTTP 查询与前端展示属于 Stage 3；本切片只建立其可靠派生数据。
- 禁止声明：扫描 Backend Ready、Stage 3 已授权、正式前端可依赖完整扫描流程或 MVP
  可发布。
- 评审日期：2026-07-27
