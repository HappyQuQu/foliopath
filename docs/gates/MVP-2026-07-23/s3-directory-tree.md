# S3-003 目录树与详情实现记录

## 结论

**Go — S3-003 实现完成；进入 S3-004。**

本记录确认 direct-child 目录页、indexed root detail、完整 breadcrumb、可靠计数和正式
认证 HTTP/SQLite/composition 已接线。它不把整个浏览后端标记为 Backend Ready，也不授权
资产 HTTP、媒体探测、缩略图、搜索、前端浏览集成或发布。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-BRW-001`、`FR-BRW-003`、`FR-BRW-005`、`FR-BRW-008～009`、
  `NFR-PRIV-001`、`NFR-REL-001`
- Contract Ready：[目录与媒体浏览](s3-browse-contract-ready.md)
- 目录 detail、breadcrumb 与拓扑验证 owner：`internal/catalog`
- 持久化与迭代 ancestor 查询：`internal/store/sqlite`
- strict query/DTO/error 翻译：`internal/api`
- 唯一具体依赖组合：`internal/app`

## 已实现行为

- `GET /api/v1/libraries/{libraryId}/directories` 只返回选中 parent 的直接 indexed
  children；省略 `parentId` 与显式 indexed root 等价。默认/最大页长、自然 keyset 和
  generation cursor 继续由 S3-002 的 catalog owner 负责。
- 每项包含稳定 ID、同库 parent、library-relative path、直接/递归资产计数和
  `hasChildren`；空目录不会因媒体数为零被隐藏。
- `GET /api/v1/directories/{directoryId}` 返回 root-to-current 完整 breadcrumb。
  indexed root 的公开名称映射为当前媒体库名称、路径为空、parent 为 null，根 breadcrumb
  恰好一个元素。
- ancestor 查询是最多 2049 项的有界迭代，不使用 Go 递归调用栈。领域层逐项验证 ID、
  library、parent、规范路径、basename 与直接父路径关系。
- 环、断 parent、跨库 parent、重复节点和超限链均 fail closed 为内部错误；不会返回部分
  breadcrumb，也不会向公开响应泄露 SQL、主机路径或数据库损坏细节。
- offline 与扫描中目录继续返回最后可靠计数。请求不 stat、遍历或打开 `/library`，
  filesystem 是否可用不改变 indexed browse 的返回路径。

## 自动约束与证据

- `internal/catalog/catalog_test.go`：root 映射、完整 breadcrumb、路径/parent/library/重复
  节点损坏均由唯一领域 owner 拒绝。
- `internal/store/sqlite/catalog_test.go`：真实 scanner 索引上的 root/empty child/
  `hasChildren`/计数、1000 层链、环、断 parent、跨库 parent、scanning/offline 可靠计数。
- `internal/api/catalog_http_test.go`：严格 opaque ID/query、nullable cursor/parent、
  DTO 展平、稳定 400/404/500 与安全错误响应。
- `internal/app/runtime_integration_test.go`：真实管理员 setup、session middleware、
  production composition、SQLite 和 scanner；扫描完成后移走源目录，已认证目录页与 detail
  仍从保留索引成功返回，未认证请求被拒绝。
- `tests/architecture/dependencies_test.go`：API 不导入 SQLite/files，catalog adapter
  浏览查询不使用 `OFFSET`。

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

- S3-004 才实现 govips/FFmpeg 媒体探测、缩略图/视频封面和损坏媒体状态。
- S3-005～006 才完成有界媒体任务、缓存/磁盘保护和敌意输入/并发矩阵。
- S3-007 才汇总目录、资产、缩略图和容量证据并判断浏览后端是否 Backend Ready。
- 资产列表的 catalog query 已由 S3-002 实现，但公开 `Asset` 仍依赖 S3-004 的 metadata/
  thumbnail 状态，因此本切片没有提前接入不完整的资产 HTTP DTO。
- 禁止声明：浏览 Backend Ready、Stage 3 Integrated Done 或 MVP 可发布。

- 评审日期：2026-07-27
