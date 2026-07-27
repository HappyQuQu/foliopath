# S3-001 目录与媒体浏览 Contract Ready

## 结论

**Go — S3-001 Contract Ready。**

目录树、根节点、面包屑、当前目录与递归媒体列表的输入、排序、游标、离线和失败语义已经冻结。
允许进入 `S3-002` catalog 查询/cursor 实现与 `S3-003` 目录树实现。

本记录不表示浏览后端已经可用，不授权缩略图、媒体探测、搜索、前端真实集成或 Stage 4～5。
OpenAPI 中已经存在的搜索参数和全局搜索 operation 仍是目标契约；生成 client 中出现它们不等于
已经通过相应 Backend Ready Gate。

## 范围与权威来源

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- Roadmap：Stage 3 / `S3-BRW`
- 需求：`FR-BRW-001～005`、`FR-BRW-007～009`、`FR-LIB-003`、`FR-LIB-006`、
  `NFR-SAFE-001`、`NFR-SEC-001`、`NFR-PRIV-001`、
  `NFR-REL-001`、`NFR-PERF-001～002`
- HTTP 权威：`api/openapi.yaml`
- 数据：`directories`、`assets`、`libraries.current_generation`
- 语义 owner：`internal/catalog`
- Adapter：`internal/store/sqlite`
- 传输：`internal/api`
- 组合：`internal/app`
- ADR：0001、0002、0003、0008、0009
- 风险：R-003、R-004、R-005、R-012、R-013、R-016

## 根节点与目录树

- SQLite 中 `relative_path=''` 的目录行是唯一 indexed root。公开 `Directory` 将它表示为：
  当前媒体库显示名、空 `relativePath`、null `parentId`；数据库空名称不能泄漏到 DTO。
- 根目录 ID 是合法 `Directory` ID。目录/资产查询省略目录参数表示根；显式传入同库 root ID
  等价，并在 cursor query fingerprint 中规范为同一 scope。
- 跨库目录 ID 与不存在的 ID 都返回 `404 directory_not_found`，不能通过状态码、message 或
  timing 有意区分资源是否存在于另一媒体库。
- 目录列表只返回选中 parent 的直接 indexed children；parent 自身不重复出现在 `items`。
  所有可读目录都保留，包括空目录和没有支持媒体的目录。
- 目录默认按 `(natural_name_key ASC, name ASC, id ASC)` keyset；默认 50、最大 200。
- API 请求只读 SQLite 派生索引，不在请求内遍历、stat 或打开 `/library`。

## 面包屑

- `GET /api/v1/directories/{directoryId}` 返回完整 root-to-current breadcrumb，包含根和当前项。
- 根目录 detail 的 breadcrumb 恰好一个元素；每项仅有 ID、显示名和 library-relative path。
- 最大 2049 项（根加最多 2048 个一字符 component）。该上限覆盖 4096 字符相对路径并与 1000 层
  容量 fixture 相容；实现必须迭代或使用有界 recursive CTE，不能递归占用 Go 调用栈。
- 检测到环、断 parent、跨库 parent 或超过 schema 上限时 fail closed 为内部 catalog
  corruption；不得返回部分 breadcrumb 冒充有效导航。

## 当前目录与递归媒体

- `directoryId` 省略或传同库 root ID表示媒体库根。
- `recursive=false` 只返回 `asset.directory_id` 等于选中目录的媒体。
- `recursive=true` 返回选中目录自身及所有 indexed descendants；根递归即整库。
- 每个结果保留实际 `directoryId` 与 library-relative `relativePath`，同名媒体可区分并可导航
  回来源目录。
- 无搜索、非递归浏览默认
  `(natural_name_key, name, relative_path, id) ASC`；递归默认 `(mtime_ns, id) DESC`。
  显式 `sort`/`order` 作用于整个 tuple。
- OpenAPI 已规划的搜索/过滤字段属于 cursor fingerprint；S3 实现不能忽略已提供字段或返回与
  参数无关的数据。Stage 4 Backend Ready 前，产品 UI 不得调用尚未交付的搜索 profile。

## Cursor 与索引变化

- cursor 使用 `internal/cursor` 的唯一 authenticated opaque-token codec；`internal/catalog`
  独占 payload、query normalization、排序版本和有效性规则。
- fingerprint 至少绑定 library、规范目录、recursive、query、kind、effective sort/order、
  排序版本和 `current_generation`。跨 scope、跨 generation、篡改、过长或版本未知均返回
  `400 invalid_cursor`，绝不回退第一页。
- successful finalize 推进 reliable generation 后旧 cursor 失效。扫描过程中安全提交的新增项
  可以被严格 keyset 的后续页观察；不提供跨 generation 的长事务快照。
- tie-breaker 必须是稳定唯一 ID；禁止 `OFFSET`，禁止把完整结果加载内存后分页。

## 离线、扫描中与空结果

- offline 库继续以 `200` 返回最后保留的目录、计数和资产，并在 Asset 中表达
  `sourceAvailability=offline`；browse 请求不触碰文件系统。
- pending/scanning/offline/error 状态由 Library 资源拥有。空 `items` 只表示当前索引查询没有
  条目，不能单独证明源目录为空。
- 扫描中目录计数可以保持最后 reliable generation 的值；UI 结合 Library/Scan 状态表达正在更新。
- missing library 返回 `library_not_found`；wrong-scope/missing directory 返回
  `directory_not_found`；非法 query 返回 `invalid_request`；cursor 问题返回 `invalid_cursor`。
- 任何公开错误都只含稳定 code、本地化安全 message 和 request ID，不泄露 SQL、主机路径、
  container root、errno 或 stack。

## 实现交接

`S3-002` 必须：

1. 在 `internal/catalog` 建立唯一 query model、root mapper、query normalizer 与 cursor payload。
2. 追加 migration 建立 directory/asset `natural_name_key` 及两个浏览 tuple 所需索引；不得修改
   已发布 migration。
3. 在 SQLite adapter 实现严格 keyset 与 context cancellation，并覆盖 generation/cross-scope。
4. 在 API 只做 strict query/DTO/error 翻译；handler 不查询 SQLite、不解析目录路径。

`S3-003` 必须：

1. 实现 direct-child page、root detail 和完整 breadcrumb；
2. 覆盖空目录、root、1000 层链、断 parent、环、跨库 parent、offline 与扫描中旧计数；
3. 用真实认证 HTTP/SQLite/composition 证明所有 browse 请求不访问 `/library`。

## 自动证据

- OpenAPI 离线解析、外部 lint、兼容性比较、生成 TypeScript 与摘要锁。
- contract test 固定三个 operation 的 error code、root mapping、recursive 范围、排序 tuple、
  generation cursor、offline preserved index 与无请求时文件系统遍历。
- schema test 固定 root 可序列化以及 breadcrumb `minItems=1`、`maxItems=2049`。
- architecture fitness 必须阻止 API 直连 SQLite/files、第二 cursor codec 和 `OFFSET` 浏览查询。

## Gate 判断

- Requirement / target / owner / contracts / evidence：完整。
- 用户可见新能力：无；只冻结已确认 MVP 浏览能力。
- 架构或信任边界变化：无；沿用 modular monolith、SQLite 派生索引与只读 `/library`。
- S3-002 / S3-003：**Go**。
- 浏览 Backend Ready、缩略图、搜索、前端集成、发布：**No-Go，等待各自 Gate**。

- 评审日期：2026-07-27
