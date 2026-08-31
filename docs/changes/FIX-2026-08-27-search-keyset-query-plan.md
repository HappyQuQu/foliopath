# 10 万媒体搜索 keyset 查询计划维护提案

- 日期：2026-08-27
- 状态：**Proposal — 未授权生产修改**
- 类型：已冻结 MVP 搜索能力的性能回归维护
- Requirement：`FR-SRH-002～004`、`NFR-PER-001`、`NFR-REL-001`
- 目标版本与阶段：MVP maintenance，须经 `S4-003` 补充 Gate 批准
- Owner：`internal/catalog`（查询/计数语义）、`internal/store/sqlite`（唯一 SQL owner）、performance/release
- 既有 Gate：[S4-002 搜索 keyset](../gates/MVP-2026-07-23/s4-search-keyset.md)、
  [S4-003 搜索 Backend Ready](../gates/MVP-2026-07-23/s4-search-backend-ready.md)
- 证据：[arm64 100k query-plan 与 order-first 对照](../evidence/int-001/catalog-search-query-plan-linux-arm64-2026-08-27.md)、
  [scope/filter/sort/cursor matrix](../evidence/int-001/catalog-search-order-first-matrix-linux-arm64-2026-08-27.md)

## 问题

当前 10,000 目录/100,000 媒体的广匹配、名称正序搜索未继续满足既有两页 keyset P95 ≤ 250 ms。
生产查询先由 FTS 候选流按 rowid 回表，再为派生 folder/name tuple 建临时 B-tree；第二页 keyset
不会改变执行形态。该问题在无 AI runtime 的基线中已复现，不能归因给 AI，也不能通过放宽预算掩盖。

本记录不改变搜索 profile、API、cursor、默认排序或数据库权威字段，也不授权修改生产 SQL。它只把
已有能力的性能债务从 `POST-MVP-5` AI Gate 中拆回其规范 owner。

## 保留候选

benchmark-only order-first 查询强制复用既有 `assets_browse_folder_name_v2`，按规范 tuple 扫描资产，
对每项执行 FTS membership 与精确 `instr` 判断。原生 Linux/arm64、4 CPU/4 GiB、100k 档得到：

- 广匹配前/次页各 101 个 ID 与生产结果逐项一致，P95 33.990/32.451 ms；
- 完整 asset/thumbnail/storyboard/favorite 首页面装配 P95 33.668 ms；
- 稀疏 `asset-099` 首页 101 个 ID 与生产结果一致，P95 110.061 ms；
- 四个候选执行计划均不再出现 `USE TEMP B-TREE FOR ORDER BY`。

这些结果只允许保留候选。仅测了库级、名称正序和两个合成查询；不能据此修改生产实现。

后续矩阵将正确性扩展到 26 个首页面和 22 个可分页的 scope/filter/sort/order 组合，候选单页最坏
约 203 ms；三次 production repository broad + modified-window 首/次页约
9.44～19.00/9.55～19.00 s。`EXPLAIN` 证明 `assets_modified` 范围扫描驱动、逐行 FTS 探测和临时排序；
80k image/10k video/10k animated 及 image+video 组合过滤已覆盖，但其他组合 kind、跨库
mixed-media/date/sparse、真实选择性、重复 P95、矩阵完整 hydration 和
native amd64 仍缺，不能用矩阵正确性跳过下一 Gate。

当前最慢 kind、递归名称排序和 broad modified-window 三个代表场景又分别对首页/第二页采样 20 次；
候选 P95 最坏为 174.591/135.115 ms，仍低于 250 ms。其余矩阵仍是单次观测，因此不能把这项结果
外推成完整查询分布或生产修复验收。

独立两库 follow-up 又以两个不重叠库各 5k/50k 验证 global library-ID tie-break；加入不同
image/video/animated 比例、image+video、选择性日期窗口和约 2% 稀疏词后，11 个首页/11 个第二页
全部保持顺序，候选最坏约 83 ms。合成跨库 mixed-media/date/sparse 正确性子证明已关闭，但真实
分布/选择性、重复 P95、full hydration 和 amd64 仍缺。

2026-08-31 在当前 worktree 的 darwin/arm64、`GOMAXPROCS=4` 又完整执行同一两库 10k/100k
follow-up：11 个首页和 11 个第二页继续逐 ID 等价；候选单页最慢为 49,099 µs（稀疏查询第二页），
其余覆盖 name/modifiedAt/size 双向、video/animated/image+video、选择性日期窗口。该运行关闭“当前
候选在本机是否仍成立”的复核，但不是 native Linux 双架构、完整 hydration 或生产修改授权。

同一测试随后加入 cardinality Gate 并重跑：0、1、10、恰好 100、超过一页五种结果规模均与生产
repository 逐 ID 等价；超过一页还验证第二页 keyset。扩展后完整测试以退出码 0 结束，最慢候选页为
49,073 µs。中文/组合字符等规范化已有 capability/SQLite 回归，但候选与生产 SQL 的统一生产 owner、
native amd64 和完整恢复/负载证据仍未关闭。

随后在相同 darwin/arm64、`GOMAXPROCS=4`、两库 10k/100k 档把 11 个首页和 11 个第二页全部改为
同时执行完整 asset/library/grid/storyboard/favorite hydration；22 页均与 production repository 逐 ID
一致。测试还真实执行 FTS5 `rebuild` + `integrity-check`，关闭并重新打开独立数据库连接，再以完整
hydration 复核代表性 global page，结果继续一致。该轮最慢 candidate ID page 为 47,924 µs，测试总计
81.06 秒并通过。由此本地“完整 hydration”和“FTS rebuild/连接重开”子项关闭；扫描并发、旧 cursor、
production 授权和 native linux/amd64+arm64 预算仍未关闭。

同一矩阵随后加入 6 个特殊文本首页：中文两字、NFKC 组合字符、sharp-s 大小写折叠、标点、多词 AND、
含双引号时禁用 FTS anchor 的 exact fallback。两库 10k/100k 重跑得到 17 个首页、11 个第二页与
28 个完整 hydrated 页面全部等价，rebuild/reopen 复核继续通过；最慢 candidate ID 页 67,374 µs，
总计 82.04 秒。该结果关闭当前 Gate 的 Unicode/短词/标点/多词合成语义子项，但不替代真实分布、
扫描并发、旧 cursor、production owner 接受或 native 双架构容量。

下一轮在相同档启动库 A 的真实完整重扫，同时连续执行 10 次 sparse global production/candidate 完整
hydration 对照，并在扫描成功发布后再复核一次；扫描中和发布后都逐 ID 等价。特殊文本/分页/rebuild
矩阵也继续通过，最慢 candidate ID 页 67,471 µs，总计 108.43 秒。由此本地扫描并发子项关闭；旧
cursor 的 service-level 语义虽有既有回归，但 production 策略尚未获接受，仍须在实现变更时复跑。

同日前置 `make spike-capacity` 仍以生产查询 `searchKeysetP95Us=296212` 超过 250,000 µs 失败，
因此本提案不能因候选快速而掩盖现状，也不能通过提高预算验收。

## 必须完成的补充 Gate

- 结果等价：直接目录、递归目录、整库、全局；无结果、少于/等于/超过一页；中文、组合字符、
  sharp-s、1～2 字符、标点和多词 AND；offline library 与 generation/revision cursor。
- 筛选/排序：kind、modified time；name/modified/size 的升序和降序；第一页、第二页及深分页。
- 选择策略：比较 FTS-first、order-first、bounded/materialized candidate；若使用已完成 count 作为
  planner hint，阈值必须由 broad/sparse 分布实测确定，不能硬编码单个 fixture 的经验值。
- 行为与恢复：完整 hydration、请求取消、扫描并发、FTS rebuild/integrity、数据库重开和旧 cursor。
- 容量：原生 linux/arm64 与 amd64，4 CPU/4 GiB，10k/100k，保持 `s4-search-v1` 250 ms；不得提高
  SQLite family、RSS 或扫描预算。
- 契约：优先保持 OpenAPI/schema/cursor 不变。若内部 repository 参数增加 count hint，其唯一 owner、
  一致性与测试必须在实现前写入 Gate；不得把 SQL planner 选择泄露给 API 或 UI。

## 接受与回退

只有完整矩阵结果相等、双架构预算通过、取消/恢复无回归且相关 repository/HTTP/browser 测试通过，
本提案才能改为 accepted maintenance fix。任一条件失败则保留现有生产查询，继续比较
bounded/materialized candidate；不得降低 250 ms 预算或让 AI Gate 暗中承担修复。回退是撤销内部
查询策略并保留既有 index/contract，不涉及 migration down，也不触碰原媒体。
