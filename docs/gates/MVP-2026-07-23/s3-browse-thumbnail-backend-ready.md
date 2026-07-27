# S3-007 浏览与缩略图 Backend Ready

## 结论

**Go — Stage 3 浏览与缩略图后端已达到 Backend Ready；进入 S4-001。**

本 Gate 允许前端通过生成 client 接入媒体库资产分页、资产详情和 grid 缩略图状态/内容。
它不授权搜索、原内容/Range、查看器、非回环监听、匿名访问、Integrated Done 或发布。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-BRW-001～005`、`FR-BRW-007～009`、`FR-MED-001～002`、
  `FR-MED-008`、`NFR-SEC-001`、`NFR-PRIV-001`、`NFR-PERF-001～002`
- 后端 owner：`internal/catalog` 统一拥有资产 scope、排序、cursor、详情和派生状态读模型；
  `internal/thumbnail` 统一拥有缩略图交付状态、缓存命中、LRU touch 与缺失缓存修复
- adapter：`internal/store/sqlite`、`internal/thumbnail/cachefs`、`internal/api`
- composition：`internal/app`
- HTTP 权威：`api/openapi.yaml`；TypeScript 生成源：
  `web/src/lib/api/generated/schema.ts`

API handler 不查询 SQLite、不解析文件路径、不打开缓存或调用 govips/FFmpeg。缩略图 HTTP
只调用 `thumbnail.DeliveryService`；媒体生成始终由已有 durable worker 执行。

## 已冻结的可消费行为

| Operation | 成功/状态 | 稳定失败语义 |
| --- | --- | --- |
| `GET /api/v1/libraries/{libraryId}/assets` | 可靠 generation 上的 direct/recursive keyset page；offline 返回保留索引 | `invalid_request`、`invalid_cursor`、`library_not_found`、`directory_not_found` |
| `GET /api/v1/assets/{assetId}` | 单个索引快照与 probe/playback/thumbnail/source 状态 | `asset_not_found` |
| `GET /api/v1/assets/{assetId}/thumbnail` | ready 为 WebP；queued/running 为 `202` + `Retry-After`；ETag 命中为 `304` | `asset_not_found`、`source_offline`、`unsupported_media`、`invalid_media`、`thumbnail_failed`、`media_processing_timeout` |

- 三个 operation 都经过默认拒绝的 session middleware；未认证请求不返回资产或缓存字节。
- ready 响应固定 `private, max-age=31536000, immutable`、ETag、`nosniff` 和限制型 CSP。
- 每次真实 ready/304 命中刷新 `last_accessed_at_ms`，LRU 仍由 `internal/thumbnail` 策略拥有。
- 数据库标记 ready 但缓存缺失或长度不一致时，交付服务在短事务中删除陈旧 thumbnail
  状态、重置 probe、把同一 fingerprint job 归零重排并 wake worker，然后返回 `202`；
  请求线程不做媒体处理。
- library offline 优先返回 `source_offline`，不把离线解释为空、不清理可靠索引。
- API 只接受 opaque `lib_`、`dir_`、`ast_` ID；资产响应仅含媒体库相对路径，不暴露
  host/container 根、SQL、缓存路径或工具输出。

## 汇总证据

- `S3-001`：浏览 scope、root、breadcrumb、offline 与 cursor 契约。
- `S3-002`：自然排序、完整 tuple keyset、generation/query fingerprint、SQLite
  索引和 10 万媒体容量回归。
- `S3-003`：目录树、空目录、直接/递归计数、拓扑失败关闭和认证 HTTP。
- `S3-004～006`：真实 govips/FFmpeg、fingerprint 派生、原子 cache→DB、durable
  2-worker/3-attempt queue、LRU/磁盘水位以及敌意输入/超时/取消/满盘资源矩阵。
- `internal/catalog`、`internal/store/sqlite/catalog.go`：资产页、单资产详情和 preserved
  index 状态映射。
- `internal/thumbnail/delivery_test.go`、`internal/store/sqlite/thumbnail_test.go`：
  ready 打开、访问热度、pending/failed/offline 以及缺失缓存自愈。
- `internal/api/catalog_http_test.go`、`internal/api/thumbnail_http_test.go`：
  wire/query、200/202/304/409/422、安全错误和响应头。
- `internal/app/runtime_integration_test.go`：真实 composition、SQLite、session middleware
  下的资产页、详情、缩略图状态与未认证默认拒绝。
- `tests/architecture/dependencies_test.go`：handler 不绕过 capability/cache adapter，
  composition root 是唯一具体接线点。

适用检查入口：

```sh
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
```

## 交接与保留边界

- 前端现在可以消费 `listLibraryAssets`、`getAsset`、`getAssetThumbnail`；`q` 搜索参数和
  `searchAssets` 即使已存在于生成类型中也仍未 Backend Ready，不得接入。
- `GET/HEAD .../content`、单 Range、条件原内容、浏览器视频播放和查看器属于 Stage 4。
- Stage 3 这里只是后端交接完成；前端实现、浏览器 E2E 与 Integrated Done 仍是独立 Gate。
- 正式镜像、非回环/可信代理、备份恢复、代表性设备长期存储和发布验收仍由 Stage 5 阻断。
- 评审日期：2026-07-27
