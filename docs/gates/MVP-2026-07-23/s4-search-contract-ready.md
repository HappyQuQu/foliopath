# S4-001 搜索 Contract Ready

## 结论

**Go — S4-001 Contract Ready。**

文件名/路径匹配、类型与修改时间筛选、当前目录/当前媒体库/全部媒体库三种范围、稳定排序、
cursor 失效、离线索引和错误语义已经冻结。允许进入 `S4-002` SQLite 搜索与 keyset 实现。

本记录不表示搜索后端已经可用，不授权搜索产品 UI、原内容/Range、查看器、Stage 5 或发布。
生成客户端出现搜索参数只说明 wire contract 可生成，不代表对应 handler 已通过 Backend Ready。

## 范围与权威来源

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- Roadmap：Stage 4 / `S4-SRH`
- 需求：`FR-SRH-001～004`、`FR-BRW-004～005`、`NFR-PERF-001～002`、
  `NFR-PRIV-001`
- HTTP 权威：`api/openapi.yaml`
- 数据计划：`assets`、FTS5 派生索引、`catalog_search_state.revision`
- 语义 owner：`internal/catalog`
- Adapter：`internal/store/sqlite`
- 传输：`internal/api`
- 组合：`internal/app`
- ADR：0001、0003、0006
- 风险：R-005、R-012、R-016

## Search profile v1

- 只搜索媒体文件名与媒体库相对路径，不在请求中遍历文件系统。
- 输入去掉首尾 Unicode 空白，执行 Unicode NFKC 与 full case folding，按 Unicode 空白
  分词并去重；全部词以 AND 组合，每个词必须是任一规范化字段的字面子串。
- 保留变音符号；引号、百分号、下划线、横线、点和路径分隔符都没有查询操作符含义。
  用户输入永远不解释为 FTS 语法。
- 一至二字符词是正式契约，不允许因 tokenizer 限制拒绝、忽略或扩大匹配。
- Search profile 通过 OpenAPI 顶层 `x-foliopath-search-profile.version=1` 版本化。改变上述
  规则属于 API 行为兼容性变更，必须先评审契约版本。

## 范围、筛选与排序

- `GET /api/v1/libraries/{libraryId}/assets?q=...` 且省略 `directoryId`：整个当前媒体库；
  此时必须省略 `recursive`。
- 同一端点同时提供 `q` 和 `directoryId`：当前目录；`recursive` 省略/false 只含直接资产，
  true 包含全部 indexed descendants。
- `GET /api/v1/assets?q=...`：全部媒体库；`q` 必填。默认搜索当前媒体库是 UI 路由选择，
  不是全局端点的隐式参数。
- `kind` 是 image/animated/video 的可选集合。时间筛选只使用 filesystem mtime：
  `modifiedFrom` inclusive、`modifiedBefore` exclusive，均为 UTC RFC 3339；同时提供时
  前者必须早于后者，否则 `400 invalid_request`。
- 所有搜索默认 `(modifiedAt, id) DESC`，MVP 不提供 relevance 排序。库内名称排序使用
  `(naturalNameKey, name, relativePath, id)`；跨库名称排序使用
  `(naturalNameKey, name, libraryId, relativePath, id)`；显式方向作用于 tuple 全部字段。

## Cursor、一致性与离线

- 库内 cursor 绑定 library、canonical scope、effective recursive、规范化 terms、kind、
  时间边界、effective sort/order、ordering/search-profile version 与 reliable generation。
- 跨库 cursor 绑定同样的 query 字段与一个持久化 global catalog revision。该 revision 在
  媒体库创建/移除或可靠 full-scan generation 发布时推进，避免 token 携带无界媒体库向量。
- 可靠 generation/revision 或任一绑定参数变化时返回 `400 invalid_cursor`，绝不回退第一页。
  扫描中安全提交的新增项可被后续 keyset 页看到；不承诺跨页数据库快照。
- offline 库继续贡献最后可靠索引，并由 `sourceAvailability` 标记。空结果不等于磁盘目录
  为空，请求也不 stat/open 原媒体。

## 实现交接

`S4-002` 必须：

1. 在 `internal/catalog` 建立唯一 search query normalizer、scope model、ordering version
   与 cursor payload；API handler 只做 strict parse/DTO/error 映射。
2. 追加 migration 建立可重建搜索键、FTS5/辅助索引和 singleton global catalog revision；
   资产变更与搜索派生状态保持同一事务语义，不修改已发布 migration。
3. 证明一至二字符与任意标点查询仍符合字面子串 profile；不能把 unsupported query
   静默当作无搜索浏览。
4. 覆盖三种 scope、kind、半开时间边界、稳定 name/mtime keyset、cursor 篡改/过期、
   offline preserved index、请求取消与索引重建。
5. 在约 10 万媒体主档验证搜索延迟、内存和扫描并发；不达预算时停止 S4-003 Gate，
   不能通过无界内存过滤规避。

## 自动证据

- OpenAPI 离线解析、结构/引用验证、外部 lint、生成 TypeScript 与摘要锁。
- contract test 固定 profile version、字段、规范化、字面 AND、三种 scope、mtime 半开区间、
  稳定 tuple、global revision、offline 和错误码。
- `make arch-check` 保持 API → capability → adapter 方向、唯一 cursor/查询 owner 与
  SQLite/文件边界。

## Gate 判断

- Requirement / target / owner / contracts / evidence：完整。
- 用户可见新能力：无；只冻结已确认 MVP 搜索行为。
- 架构或信任边界变化：无；沿用 modular monolith、SQLite 派生索引和只读 `/library`。
- S4-002：**Go**。
- 搜索 Backend Ready、搜索前端、内容/查看器、发布：**No-Go，等待各自 Gate**。

- 评审日期：2026-07-27
