# CR-2026-010：AVI 支持与文件大小排序

## 状态

- 状态：Confirmed
- 变更等级：C2
- 目标版本：`POST-MVP-1` / `Post-MVP/1`
- Scope revision / 范围状态：[POST-MVP-1 revision 2](../releases/POST-MVP-1-scope-r2.md)
- Change Record ID / 基线事件：CR-2026-010 / 2026-07-30 产品用户确认“优化”
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/media`（格式）、`internal/catalog`（排序）
- UI Owner：`web/src/features/browse`、`web/src/features/search`

## 用户问题与价值

- 既有 AVI 视频无法进入索引，历史家庭视频集合不完整。
- 已索引媒体包含文件字节数，但用户无法按大小寻找大文件或小文件。

## 范围与验收

- `FR-MED-012`：将 `.avi` / `video/x-msvideo` 纳入视频索引、FFprobe 元数据、
  FFmpeg poster/storyboard 与原文件 Range 契约；不承诺浏览器原生播放其 codec。
- `FR-BRW-011`：浏览和三种搜索范围支持 `sort=size&order=asc|desc`；默认排序不变。
- `OPT-AC-001`：AVI 扩展名不区分大小写，损坏、超限和不兼容 codec 使用既有失败语义。
- `OPT-AC-002`：大小排序使用 `(size_bytes, id)` 完整 keyset，跨页不重复、不漏项。
- `OPT-AC-003`：浏览与搜索提供中英文“从大到小／从小到大”选项，URL 可恢复。

不包含 AVI 转码、新 codec、修改原文件、按磁盘占用排序或改变默认排序。复用既有
FFmpeg、SQLite、cursor、认证与 `/library` 只读边界；使用只追加 migration 放宽格式
CHECK 并增加大小索引，无 ADR、新表或新部署单元。

## 影响与证据

- OpenAPI 增加 `sort=size` 与 `video/x-msvideo`；生成客户端随源契约再生成。
- SQLite migration 14 保留既有 catalog/FTS/派生外键，向格式 CHECK 追加 AVI，并为
  `assets.size_bytes` 增加有界 scope 索引；cursor 绑定排序值、资源 ID 和查询指纹。
- 验证覆盖格式注册、AVI 真实 FFmpeg fixture、资源上限、SQLite 双页 keyset、
  API 契约、URL codec、浏览/搜索 UI 和中英文文案。
- 风险沿用 R-006、R-007、R-012；fallback 可撤下 AVI admission 或大小排序 UI，
  不影响原媒体和既有索引。
