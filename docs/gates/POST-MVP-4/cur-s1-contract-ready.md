# Gate POST-MVP-4 / CUR-S1 / Contract Ready

- 日期：2026-08-10
- 结论：**Go — 授权后端实现；前端 No-Go**
- 前序：[CUR-S0](cur-s0-architecture-ready.md)
- Feature：[FTR-CUR-001](../../features/favorites-and-tags.md)

## 冻结合同

1. 权威 HTTP operation、请求/响应、错误与分页由 `api/openapi.yaml` 提供。
2. migration 21 只追加 `curation_state`、`asset_favorites`、`tags`、`asset_tags` 及必要索引。
3. `internal/curation` 拥有规范化、状态转换、revision、query fingerprint 和 cursor payload。
4. SQLite adapter 在短事务内写关系并递增 revision；不访问文件系统或媒体工具。
5. 写请求沿用认证、CSRF、同源、统一错误和资源 ID 编解码。
6. 列表使用 `(favorite_created_at_ms DESC, asset_id DESC)` 或现有稳定资产 tuple；默认 50、最大 200。
7. Asset curation 状态返回 revision，并以 ETag 支持单资产标签替换的 `If-Match` 防丢失更新。

## Backend Ready 出口

- capability unit：Unicode、上限、幂等、revision/cursor/query binding。
- SQLite：全新 migration、历史升级、FK/级联、事务回滚、索引查询计划与并发写。
- HTTP：认证/CSRF、DTO、ETag、not-found/conflict/invalid-cursor、错误脱敏。
- OpenAPI lint、生成漂移、架构检查、Go unit/contract/integration 全部实际通过。
- 记录 `CUR-S2 Backend Evidence Ready` 后，才允许生成客户端 adapter 与生产 UI 接入。
