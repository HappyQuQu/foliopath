# S4 搜索 keyset 容量回归复审（2026-08-31）

- 版本：MVP maintenance / POST-MVP-5 集成输入
- 需求：`FR-SRH-002～004`、`NFR-PER-001`、`NFR-REL-001`
- 风险：`R-005`、`R-012`、`R-016`
- Owner：`internal/catalog`、`internal/store/sqlite`、performance/release
- 当前判断：**No-Go for production query-plan change / Evidence work authorized**

## 结论

历史 [S4-003 Backend Ready](s4-search-backend-ready.md) 是当时证据的有效快照，但当前 10k/100k
强制基线已复现 `searchKeysetP95Us=296212`，超过冻结的 250,000 µs。发布和 POST-MVP-5 最终容量
Gate 必须按当前事实失败关闭；不得沿用历史 Go 隐藏回归。

benchmark-only order-first 候选继续值得推进，但还没有生产修改授权。当前证据只授权补充正确性、
恢复、完整 hydration、负载与 native 双架构矩阵。

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

实现与详细结果记录于
[10 万媒体搜索 keyset 查询计划维护提案](../../changes/FIX-2026-08-27-search-keyset-query-plan.md)。

## 转为 production Go 前仍需

1. 将候选提升为 repository 内部策略的唯一 owner 方案，并证明 OpenAPI/cursor/search profile 不变；
2. 旧 cursor 的变更后回归；取消、完整 hydration、扫描并发、FTS rebuild/integrity 与数据库连接重开
   已在 darwin/arm64 候选矩阵通过，仍须纳入 production 变更和 native 双架构 Gate；
3. native Linux/arm64 与 amd64 的 4 CPU/4 GiB、10k/100k 强制预算；
4. broad/sparse 的真实选择性与重复 P95；Unicode/短词/标点/多词 AND 合成等价矩阵已通过；
5. `make fmt/arch-check/generate-check/lint/test/test-integration/test-e2e` 全部通过；
6. backend/performance/release owner 接受本维护 Gate。

任一项缺失时保留现有生产 SQL，并让最终容量 Gate保持 No-Go。不得通过增加硬编码 fixture 阈值、
降低搜索语义或提高 250 ms 预算来获得 Go。
