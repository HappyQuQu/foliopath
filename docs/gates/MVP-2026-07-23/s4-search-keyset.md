# S4-002 搜索与 keyset 实现完成

## 结论

**Implemented — 允许进入 S4-003，不是 Backend Ready。**

搜索 profile v1、三种范围、类型/修改时间筛选、稳定排序、库内与跨库 cursor、离线结果和
认证 HTTP 已接入真实 SQLite/composition。S4-003 仍须完成主容量档、扫描并发、重建/取消扩展
矩阵与最终审计；在该 Gate 前，产品前端不得把搜索 operation 当作已交付能力。

## 实现范围

- 目标版本：`MVP-2026-07-23`
- Requirement：`FR-SRH-001～004`、`NFR-PERF-001～002`、`NFR-PRIV-001`
- Capability owner：`internal/catalog`
- SQLite adapter：`internal/store/sqlite`
- HTTP adapter：`internal/api`
- Composition：`internal/app`
- Migration：`00010_catalog_search.sql`
- Contract：[S4-001 Contract Ready](s4-search-contract-ready.md)与 `api/openapi.yaml`
- 风险：R-005、R-012、R-016

## 唯一所有权与数据流

1. `internal/catalog` 唯一拥有 search profile v1 的 NFKC/full case-fold、Unicode 空白分词、
   去重/AND、scope、默认排序、mtime 边界、fingerprint 与 cursor payload。
2. scanner 在资产 upsert 的同一短事务写 `search_name_key`、`search_path_key`；SQLite trigger
   在 asset insert/update/delete 时同步外部内容 FTS5 表。
3. 至少三 Unicode 字符且不含双引号的最长词可作为 trigram FTS 候选 anchor；所有词仍用
   `instr` 在规范键上精确验证。1～2 字符和不能安全成为 anchor 的词不拼接 FTS 语法，
   直接走相同精确谓词。
4. API 只解析严格 query、UTC RFC 3339 和 DTO；不拥有 Unicode 规则、不执行 SQL、不触碰
   `/library`。全局与库内 operation 都经过既有会话中间件。

## 范围、排序与一致性

- `q` 且无 `directoryId`：整个当前媒体库；此时显式 `recursive` 被拒绝。
- `q` 加 `directoryId`：当前目录；false/省略为直接资产，true 包含 indexed descendants。
- `/api/v1/assets?q=...`：全部媒体库，q 必填，不接受目录/递归参数。
- kind 集合与 `[modifiedFrom, modifiedBefore)` filesystem mtime 在 SQL 中组合过滤。
- 库内 name tuple 为 `(natural_name_key,name,relative_path,id)`；跨库 name tuple 在 name
  后加入 `library_id`；mtime tuple 为 `(mtime_ns,id)`，全部使用严格 keyset，无 `OFFSET`。
- 库内 cursor 绑定 reliable generation；跨库 cursor 绑定 singleton catalog revision。
  library create/delete 与 reliable generation publish 通过数据库 trigger 在同一事务推进 revision。
- offline 库继续返回保留索引并逐资产标记 `sourceAvailability=offline`；请求不访问文件系统。

## 自动证据

- Catalog 单元测试：Unicode NFKC/full fold、去重、空白/空/NUL/超长拒绝、whole-library 与
  explicit-root scope 区分、mtime 反向区间拒绝。
- SQLite 测试：中文、大小写、`Straße→strasse`、组合字符、变音符号保留、短词、`%/_`、
  跨字段 AND、直接/递归/整库/跨库、kind/mtime、name keyset、query/revision cursor、
  offline preserved result 与 context cancellation。
- 升级测试：version 5 数据库升级到 migration 10 后，搜索键有界回填且 FTS MATCH 可读取旧资产。
- HTTP 测试：全局与目录搜索路由、UTC/零偏移时间、未知/重复/越权 scope 参数、稳定错误。
- Composition 测试：真实扫描后的 `GET /api/v1/assets?q=...` 经认证返回 indexed 视频；
  未认证业务请求仍被默认拒绝。
- Architecture fitness：规范化函数单一 owner、HTTP 不拥有 FTS/Unicode/文件访问、
  scanner 复用 catalog key、SQLite/migration 独占 FTS 与 revision。

## 尚未关闭

- 约 100,000 媒体/10,000 目录主档的 FTS、短词 fallback、keyset 延迟与内存预算。
- 扫描写入/可靠发布与并发搜索的长期压力、取消延迟和 FTS rebuild/integrity 故障矩阵。
- S4-003 搜索 Backend Ready 汇总与远端双架构证据。

## Gate 判断

- S4-002 implementation：**完成**。
- S4-003 correctness/capacity audit：**Go**。
- 搜索 Backend Ready、搜索前端集成、发布：**No-Go，等待 S4-003 及后续 Gate**。

- 评审日期：2026-07-27
