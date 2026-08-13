# Gate POST-MVP-4 / CUR-S3 / Consumer UI Ready

- 日期：2026-08-10
- 结论：**Go — 本地消费者 UI 与生成 client 接入完成**
- 前序：[CUR-S2 Backend Evidence Ready](cur-s2-backend-evidence-ready.md)
- Feature：[FTR-CUR-001](../../features/favorites-and-tags.md)

## 证据

- `web/src/lib/api/curation.ts` 是生成 client 之上的唯一 curation HTTP adapter；feature 不直接
  使用 raw client。
- 目录侧栏/移动抽屉在真实目录树前提供“全部媒体、收藏、标签”快速访问，收藏与标签使用独立
  路由和非目录图标。
- 共享 `MediaCollection` 增加可访问的心形操作；Browse、Search、收藏页和标签资产页复用同一
  卡片、虚拟化与分页控制器。
- `MediaPreview` 和 `MediaViewer` 通过可组合区域复用 `AssetCurationControls`，支持收藏、创建并
  添加标签、移除标签；标签 replacement 使用 curation ETag。
- 收藏和标签资产页把媒体类型及排序写入 URL，并保持游标分页；标签词表也按 cursor 有界载入。
- 简体中文与英文文案已进入唯一 LocaleProvider；收藏按钮具备可访问名称和 `aria-pressed`，并有
  共享组件行为测试。

## 已执行验证

- `npm --prefix web run check:types`：通过。
- `npm --prefix web test`：42 个文件、158 项测试通过。
- `npm --prefix web run check:architecture`：通过。
- `npm --prefix web run check:visual-references`：12 页、4 viewport、2 locale、2 theme 清单通过。
- `npm --prefix web run build` 与 `npm --prefix web run build:storybook`：通过；只有既有 chunk-size
  warning。

完整 `npm run check` 被当前依赖审计中的既有 high advisories 阻断，详见 S4；不影响本 Gate 对
本地消费者结构、类型、行为与工作台构建的判断。
