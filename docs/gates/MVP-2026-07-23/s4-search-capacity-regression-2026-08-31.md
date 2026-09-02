# S4 搜索 keyset 容量回归复审（2026-08-31）

- 版本：MVP maintenance / POST-MVP-5 集成输入
- 需求：`FR-SRH-002～004`、`NFR-PER-001`、`NFR-REL-001`
- 风险：`R-005`、`R-012`、`R-016`
- Owner：`internal/catalog`、`internal/store/sqlite`、performance/release
- 当前判断：**本地维护实现通过 / native 双架构与 release acceptance 待完成**

## 结论

历史 [S4-003 Backend Ready](s4-search-backend-ready.md) 是当时证据的有效快照，但当前 10k/100k
强制基线已复现 `searchKeysetP95Us=296212`，超过冻结的 250,000 µs。发布和 POST-MVP-5 最终容量
Gate 必须按当前事实失败关闭；不得沿用历史 Go 隐藏回归。

order-first 候选已在 2026-09-01 纳入 repository 内部实现并通过本地冻结预算；该事实不替代 native
双架构、最终模型联合负载或 release owner 验收。

## 当前证据

- darwin/arm64、`GOMAXPROCS=4`、两库合计 10k 目录/100k 资产；
- 11 个 scope/filter/sort 场景的首页与第二页，production 与候选逐 ID 等价；
- name/modifiedAt/size 双向、video/animated/image+video、稀疏词、选择性日期；
- 0、1、10、100、超过一页五种 cardinality，超过一页验证第二页 keyset；
- 候选最慢页面 49,073 µs；完整测试退出码 0；
- 后续 11×2 页面全部完成 asset/library/grid/storyboard/favorite hydration，并在 FTS5 rebuild、
  integrity-check 与数据库连接重开后保持逐 ID 等价；最慢 candidate ID 页 47,924 µs；
- 追加中文两字、组合字符、sharp-s、标点、多词 AND 与带引号 exact fallback 后，17 个首页、11 个
  第二页和 28 个完整 hydrated 页面全部等价；最慢 candidate ID 页 67,374 µs；
- 库 A 完整重扫期间 10 次 sparse global hydrated 对照及扫描成功发布后的再次对照均等价；最新最慢
  candidate ID 页 67,471 µs；
- production 强制容量测试仍失败，未提高预算。

本轮随后再次原样执行强制基线，darwin/arm64、`GOMAXPROCS=4` 的 production keyset P95 为
`319,616 µs`，继续超过 `250,000 µs`；order-first 候选首页、续页和完整 hydration P95 分别为
`9,822 µs`、`12,243 µs`、`9,913 µs`，26 个矩阵/22 个 cursor 场景仍逐 ID 等价。该复跑不提高预算，
也不把 benchmark-only SQL 提升为 production owner。

2026-09-01 将已验证方案提升为 `internal/store/sqlite` 的唯一查询实现：固定使用与 sort/scope 匹配的
既有索引进行有序外层扫描，以相关 FTS `EXISTS` 保持相同成员语义。两库 100k 完整矩阵重跑通过；随后
`make spike-capacity` 的 production `searchKeysetP95Us` 为 `133,637 µs`，production list 首页/续页 P95
为 `10,545/12,993 µs`，`budgetViolations=[]`。没有修改 API、cursor、排序、搜索语义、预算或 migration。

实现与详细结果记录于
[10 万媒体搜索 keyset 查询计划维护提案](../../changes/FIX-2026-08-27-search-keyset-query-plan.md)。

## 最终接受前仍需

1. native Linux/arm64 与 amd64 的 4 CPU/4 GiB、10k/100k 强制预算；
2. 最终模型联合负载下 broad/sparse 真实选择性、重复 P95 与深分页；
3. `make fmt/arch-check/generate-check/lint/test/test-integration/test-e2e` 全部通过；
4. backend/performance/release owner 接受本维护 Gate。

任一项缺失时保持最终容量 Gate No-Go；本地实现可继续接受回归验证，但不得通过增加硬编码 fixture
阈值、降低搜索语义或提高 250 ms 预算来获得发布 Go。
