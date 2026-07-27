# S4-003 搜索 Backend Ready

日期：2026-07-27
版本：`MVP-2026-07-23`
需求：`FR-SRH-001～004`、`NFR-PER-001`、`NFR-REL-001`、`NFR-SEC-001`
风险：R-005、R-012、R-016
负责人：后端负责人

## 判断

**Go — 搜索后端达到 Backend Ready；允许前端通过生成 client 接入冻结的搜索 operation。**

本 Gate 只交付搜索后端。资产原内容、Range、完整查看器、搜索 UI、Integrated Done 和发布
仍未完成。

## 权威契约与唯一 owner

- `api/openapi.yaml` 继续是 `/api/v1` wire contract 的唯一来源；本切片没有修改已冻结的
  S4-001 operation。
- `internal/catalog` 唯一拥有 search profile v1、scope、筛选、排序、query fingerprint、
  generation/revision cursor 和错误语义。
- `internal/store/sqlite` 只实现 capability-owned repository：migration 10 的规范搜索键、
  trigram FTS5 候选、精确 `instr` 语义、稳定 keyset 和派生索引恢复。
- `internal/api` 只做严格 query/DTO/error 映射；不复制 normalization、scope 或 cursor
  规则。

前端获准接入：

- `GET /api/v1/libraries/{libraryId}/assets`：带 `search` 时支持整库或显式目录范围；
- `GET /api/v1/assets`：跨媒体库搜索；
- 两者共同支持冻结的 `kind`、`modifiedFrom`、`modifiedBefore`、`sort`、`order`、
  `cursor` 和 `limit`。

## 正确性与故障语义

- NFKC + Unicode full case folding、字面子串 AND、中文、组合字符、保留变音符号、
  sharp-s、1～2 字符和 `%/_` 标点与 S4-001 profile 一致。
- 当前目录、递归目录、当前媒体库和全部媒体库均使用同一查询 owner；全局 name tuple
  包含 library/path/id 唯一 tie-breaker。
- 库内 cursor 绑定可靠 generation；跨库 cursor 绑定 singleton catalog revision。
  query、scope、筛选、排序或 revision 改变均返回 `400 invalid_cursor`，不回退第一页。
- offline 媒体库继续返回最后可靠索引并标记 `sourceAvailability=offline`；搜索不读取或
  stat 原媒体。
- 请求 cancellation 传播到 SQLite。扫描 visitor 的 repository 错误保持原始分类，不再
  被文件系统 adapter 误映射为 `scan_io_error`。
- 启动时执行 FTS external-content integrity check。派生索引不一致时在序列化短事务中
  rebuild 并复核；取消立即停止，rebuild 失败则启动失败关闭，`assets` 权威索引不删除。

## 10 万媒体容量证据

既有 `stage0-comparable-v1` 扫描预算保持不变；本 Gate 追加
`s4-search-v1` 搜索预算并由同一个显式重型档共同执行。搜索预算包括：

- 扫描期间搜索 p95 ≤ 500 ms；
- FTS、短词、全局与两页 keyset p95 各 ≤ 250 ms；
- cancellation 或已先完成请求的收敛延迟 ≤ 250 ms；
- FTS rebuild ≤ 120 s；
- 完整扫描 ≤ 120 s，峰值 RSS 和 SQLite family 各 ≤ 1 GiB。

| 环境 | 扫描 | 并发搜索 p95 | FTS / 短词 p95 | 全局 / keyset p95 | rebuild | 峰值 RSS | DB+WAL |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| macOS arm64，`GOMAXPROCS=4` | 49.06 s | 63.92 ms | 9.68 / 0.38 ms | 9.74 / 134.29 ms | 0.78 s | Go heap 43.97 MB | 131.22 MB |
| Linux arm64，2 CPU / 4 GiB | 42.50 s | 15.14 ms | 8.92 / 0.39 ms | 8.79 / 148.63 ms | 0.80 s | 48.58 MB | 133.30 MB |

两个主档均使用 10,000 个目录、100,000 个媒体、最大深度 32，并在 full scan 写入期间持续
执行真实 catalog service 搜索；另有 1,000 层 rollup。Linux 档使用与 CI 相同的只读源码
bind、2 GiB tmpfs、2 CPU 和 4 GiB memory limit。CI 的既有 amd64/arm64 capacity job
继续执行同一强制预算测试；当前远端运行仍受 GitHub 账户 billing/spending limit 阻断，
不能把未启动的 job 写成已通过。

## 执行证据

本切片实际执行并通过：

- 搜索/FTS repair 与 filesystem visitor 分类定向单元测试；
- 10k 缩小容量档；
- `make spike-capacity` 的本机 100k/10k 主档和 1,000 层 rollup；
- Linux arm64 2 CPU / 4 GiB 容器中的相同 100k/10k 强制预算档。

完整 repository verification 在本切片提交前仍须执行；结果写入 PR，不以本记录替代命令
输出。

## 未授权范围

- 资产原内容、HEAD、Range、条件请求和对应安全 Gate；
- 搜索页面、前端 URL/Query 状态、虚拟化、视觉与浏览器 E2E；
- relevance ranking、相似图、标签、人脸、OCR、地理搜索、watcher 正确性；
- 代表性 NAS 延迟、长期 WAL/checkpoint、在线备份和正式发布镜像。

下一后端任务是 `S4-005` 资产详情与原内容契约/实现。搜索前端可独立开始真实 API 接入，
但只有通过 Frontend Ready 与 Integrated Done 后才算产品搜索完成。
