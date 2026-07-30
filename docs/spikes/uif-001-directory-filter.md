# UIF-001：直接子目录关键字查询 Spike

## 状态

- 日期：2026-07-30
- Feature：[FTR-UIF-001](../features/frontend-prototype-fidelity.md)
- 任务：`UIF-103～104`
- 结论：Go
- 可重复入口：
  `go test -count=1 -run '^TestDirectoryFilterContractSpike$' -v ./tests/performance`

## 问题

`FR-BRW-010` 要求当前目录关键字对可靠索引中的全部直接子目录生效，不能只过滤浏览器已经
加载的 cursor 页。查询还必须保持 Unicode 语义、自然顺序、稳定 keyset 和 10k 目录档的有界
响应。

现有 `directories` 具备
`(library_id, parent_id, natural_name_key, name, id)` 浏览索引，但没有与资产搜索一致的
Unicode 规范搜索键。SQLite 内建 `lower()` 不能成为 Unicode full case folding 的权威。

## 候选

1. 浏览器加载全部 cursor 页后过滤：拒绝；网络、内存和 DOM 无界，cursor 语义错误。
2. 请求时遍历文件系统：拒绝；破坏可靠索引、离线浏览和路径安全边界。
3. 为目录建立 FTS5 trigram：当前不采用；10k 直接子目录档没有证明需要额外虚表、trigger
   和 rebuild 复杂度。
4. 追加 capability-derived `search_name_key`，先按 parent 索引限定最多 10k 直接子目录，
   再用 `instr(search_name_key, term)` 做精确 AND 谓词：接受。

## Fixture 与查询

- SQLite in-memory；
- 单媒体库、单父目录、10,000 个直接子目录；
- `search_name_key` 由 catalog capability 预先派生，测试包含中文、ASCII、数字和末尾命中；
- 查询使用 parent-scoped browse index，执行 literal substring 谓词，再按
  `(natural_name_key, name, id)` 返回最多 51 项；
- 5 次预热、50 次测量；末尾唯一命中强制扫描该 parent 的完整 10k 范围。

2026-07-30 本机结果：

```text
plan="SEARCH directories USING INDEX directories_browse_children
      (library_id=? AND parent_id=?)"
p50=1.703250ms
p95=1.981167ms
budget=100ms
```

自动测试只把 100ms 作为跨环境回归护栏；上述本机数字不是所有 NAS 的性能承诺。S2 仍需在
四核/4 GiB、并发扫描和真实 100k/10k fixture 上复验。

## 决定

- `directories` 在后续只追加 migration 中增加非空 `search_name_key TEXT`；
- catalog owner 复用资产搜索 profile v1：trim、Unicode NFKC、full case folding、Unicode
  whitespace terms、literal substring AND、保留变音符号、支持一至二字符；
- scanner/backfill 在应用层派生该键；SQLite 不自行实现 Unicode 规范化；
- 查询先由 `(library_id, parent_id, ...)` 限定直接子目录，再执行 `instr`；
- cursor fingerprint 加入规范 terms 和 search profile；改变/清除 query 或 generation
  变化返回 `invalid_cursor`；
- 当前不增加 directory FTS。若目标设备 S2 证据超过预算，再以新 spike/Change Record
  评估，不能静默改变匹配语义。

## 边界

本 spike 只接受合同、查询形状与 migration 方向，不实现生产 repository、handler 或
migration。生产实现等待 `UIF-S1 Contract Ready`。
