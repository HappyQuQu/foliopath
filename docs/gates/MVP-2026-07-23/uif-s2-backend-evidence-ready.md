# Gate MVP-2026-07-23 / UIF-S2 / Backend Evidence Ready

- 日期：2026-07-30
- 目标版本：`MVP-2026-07-23` revision 4
- Feature：[FTR-UIF-001](../../features/frontend-prototype-fidelity.md)
- 前置：[UIF-S1 Contract Ready](uif-s1-contract-ready.md)
- 任务：`UIF-201～215`
- 结论：**Go — 授权 UIF-307～311 使用生成 client 接入真实后端**

## 已实现纵向合同

| 切片 | 生产 owner | 已实现语义 |
| --- | --- | --- |
| Account | `internal/auth` | profile NFC/trim、no-op revision、强 ETag；改密验证当前密码，单事务推进 account/auth revision、保留当前 session、撤销其他 session |
| Directory q | `internal/catalog` | NFKC/full fold、空白 term AND、直接子目录 literal substring、自然 keyset、query/generation cursor binding |
| Cache | `internal/thumbnail` | aggregate usage/quota/waterline/free-space/pressure；单例 durable cleanup、active coalesce、SHA-256 idempotency、重启恢复、有界 LRU 批次 |
| Adapters | SQLite/cachefs/API/app | migration 13、确定性回填、文件先删/ready state 后删、session/CSRF、no-store、ETag、限流和生产 composition |

`00013_frontend_fidelity.sql` 只追加/重建应用数据结构，不触碰原媒体。既有
`idempotency_records` 扩展 `cache_cleanup` operation，并只保存 32 字节 key/request 摘要。
`cache_cleanup_state` 不保存路径、文件列表、明文 key 或错误详情。

## 失败、安全与恢复证据

- account service 覆盖 Unicode display name、错误当前密码、hash 失败和 session race；
- SQLite account 测试覆盖 no-op、stale revision、当前 session 升版和其他 session 同事务撤销；
- directory 覆盖空 query、Unicode、cursor mismatch，并共用 canonical search key；
- cache 覆盖 active/replay、不同 key coalesce、权限失败安全码、重启恢复和 completed replay；
- HTTP 写请求复用 session-bound CSRF；账户/cache 响应 `no-store` 且有显式限流；
- cleanup 只能消费 ready derived cache relative path；cachefs 把删除锚定在应用缓存根。

## 容量与恢复证据

- 10k 直接子目录 P95 100ms 护栏；spike 实测 P95 1.981167ms；
- 100k ready 派生项清理测试保证每批不超过 64 项；
- 既有 100k 媒体/10k 目录、RSS、SQLite busy、浏览并发及四核/4 GiB 证据继续适用；
- migration 12→13 保留幂等记录并执行 `PRAGMA integrity_check`；
- running cleanup reopen 后保留首次 startedAt 并继续。

## 已执行验证

`make fmt`、`make arch-check`、`make generate-check`、`make lint`、`make test`、
`make test-integration`、`make test-e2e`、`make openapi-lint`、`make release-docs-check` 和
`git diff --check` 均成功。OpenAPI lint 仅保留既有 health 4xx 两条 warning。

## 授权与阻断

允许 `UIF-301～319` 共享视觉基础、真实 generated-client 页面接入、可访问性和视觉矩阵。
仍阻断缺失缓存补齐、全部重建、通用任务中心、备份/恢复/诊断、AI/OCR/人脸识别、mock
业务行为，以及在 `UIF-S3/S4` 前宣称 feature 或 MVP 发布完成。

## 结论

账户、目录过滤与最小缓存维护均已到达可消费的生产后端边界。`UIF-S2` 为 Go。
