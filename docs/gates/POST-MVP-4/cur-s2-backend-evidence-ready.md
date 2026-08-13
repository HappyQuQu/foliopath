# Gate POST-MVP-4 / CUR-S2 / Backend Evidence Ready

- 日期：2026-08-10
- 结论：**Go — 授权生成 client adapter 与生产前端接入**
- 前序：[CUR-S1 Contract Ready](cur-s1-contract-ready.md)
- Feature：[FTR-CUR-001](../../features/favorites-and-tags.md)
- 风险：R-005、R-012、R-016、R-023

## 已完成证据

- `api/openapi.yaml` 冻结九个 curation operation、统一错误、ETag、分页与生成类型；Redocly
  校验通过，只有两个既有 health 4xx 警告。
- migration 21 只追加四张表、复合外键、列表索引与 revision triggers；历史升级测试统一推进
  到 version 21。
- `internal/curation` 实现 NFC、Unicode case-fold、32 字符/20 标签上限、幂等收藏、原子标签
  替换、query binding 和认证 opaque cursor。
- SQLite 测试覆盖幂等时间、revision、旧 cursor、Unicode 等价标签、precondition、按标签列表、
  资产级联删除与 schema。
- HTTP 测试覆盖资源 ID、ETag、428/412、冲突、列表 scope、错误脱敏；真实 composition 测试覆盖
  setup、401、CSRF 写入、收藏/标签持久化、列表与原媒体 hash/mtime 不变。
- `make fmt-check`、`make arch-check`、`make generate-check`、`make lint` 均成功；相关 Go、contract、
  architecture 和 integration tests 成功。首次完整 `make test` 的既有 FFmpeg 独立超时测试一次
  超时，随后该测试及全部相关包单独无缓存复跑成功；这不是 curation 失败。

## 前端约束

前端只通过生成 client 和 `web/src/lib/api/curation.ts` adapter 消费。快速访问、收藏页、标签页、
卡片/预览/查看器状态必须复用现有 AppShell、MediaCollection、MediaPreview、MediaViewer、query-key、
URL codec 与虚拟化 owner，不创建第二套媒体卡片或列表。
