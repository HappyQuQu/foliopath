# S3-002 Catalog 排序与游标实现记录

## 结论

**Go — S3-002 实现完成；进入 S3-003。**

本记录确认目录与资产浏览已有唯一 query model、自然排序键、严格 keyset、generation-bound
opaque cursor、跨库隔离和请求取消。它不把浏览后端标记为 Backend Ready，也不授权目录
详情/面包屑 HTTP、缩略图、搜索、前端集成或发布。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-BRW-001～005`、`FR-BRW-008～009`、`NFR-PERF-001`、
  `NFR-REL-001`、`NFR-PRIV-001`
- Contract Ready：[目录与媒体浏览](s3-browse-contract-ready.md)
- query/root/sort/fingerprint/cursor payload owner：`internal/catalog`
- opaque token 机制：`internal/cursor`
- 持久化与 keyset adapter：`internal/store/sqlite`
- schema：append-only migration 7

## 已实现行为

- omitted root 与同库 indexed root ID 统一规范为相同 query scope；缺失库和跨库/缺失目录
  保持不同的 capability 错误，不暴露路径。
- 目录固定使用 `(natural_name_key, name, id) ASC`；资产名称序使用
  `(natural_name_key, name, relative_path, id)`，修改时间序使用 `(mtime_ns, id)`，
  显式方向作用于完整 tuple。
- 默认页长 50、最大 200；repository 只取 `limit+1`，使用稳定 ID tie-breaker，生产 SQLite
  浏览查询不含 `OFFSET`，也不把完整结果加载到内存。
- 资产 query 统一规范化 recursive、kind、effective sort/order；kind 去重并固定顺序。
  已在 OpenAPI 规划但未进入 Stage 4 的搜索会明确返回 `ErrSearchUnavailable`，不会被忽略
  或退化为未搜索浏览。
- cursor 由认证加密机制封装，payload/指纹由 catalog 独占；指纹绑定 library、规范目录、
  recursive、query、kind、effective sort/order、排序版本和 `current_generation`。
  跨 scope、跨 generation、篡改、过长与未知版本均 fail closed。
- SQLite 查询全部使用传入 context；取消在 service 入口和 adapter 查询中传播为
  `context.Canceled`/deadline，不会转成第一页或部分成功。
- offline 库继续查询保留索引，并把资产 source availability 标为 offline；请求不访问
  `/library`。

## Schema 与写入

- migration 7 为 `directories`、`assets` 追加非空 BLOB `natural_name_key`。
- 启动升级以有界批次回填已有目录/资产；空名 indexed root 使用非空 sentinel key。
- scanner 每次 upsert 都调用 catalog 的唯一自然数字排序键 owner，名称变化会同时更新 key。
- 新索引覆盖 direct-child 目录、direct/root 名称资产、direct 修改时间资产；已有
  `assets_modified` 继续覆盖 library-wide 修改时间 tuple。
- migration 1～6 未修改；sqlc 生成仍保持确定性。

## 自动约束与证据

- `internal/catalog/catalog_test.go`：root 规范化、数字自然序、默认 query、kind canonicalization、
  cursor 等价 root、scope/generation 失效、搜索 fail closed、过长/篡改和入口取消。
- `internal/store/sqlite/catalog_test.go`：真实 migration/SQLite/scanner 数据上的两页 keyset、
  direct/recursive、kind、双方向、跨库 404 语义、offline preserved index、generation
  失效和 adapter 取消。
- `internal/store/sqlite/catalog_schema_test.go`：从 migration 6 升级、目录/资产回填及四个
  browse index。
- `tests/architecture/dependencies_test.go`：生产 SQLite 禁止 `OFFSET`；目录计数仍只允许
  scanner adapter 写入，catalog adapter 只能读取。

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

- S3-003 实现 direct-child HTTP page、root detail、breadcrumb、空/深/损坏拓扑及真实
  composition no-filesystem-access 证据。
- S3-004～006 才实现媒体探测、缩略图、缓存与敌意媒体/资源压力矩阵。
- Stage 4 才实现 FTS 搜索；本切片只有参数模型和明确拒绝，不宣称搜索可用。
- S3-007 汇总完整浏览/缩略图 Backend Ready；当前仍禁止 feature UI 调用未交付 operation。
- 禁止声明：浏览 Backend Ready、Stage 3 Integrated Done 或 MVP 可发布。

- 评审日期：2026-07-27
