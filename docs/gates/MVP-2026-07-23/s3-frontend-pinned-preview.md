# S3-106 前端固定预览交互完成记录

## 结论

**Done — S3-106 已完成非模态预览的固定状态机、选择/预览分离、Escape 与关闭焦点恢复。**

本记录不把 Stage 3 标记为 Integrated Done，也不宣称 S3-107 十万项容量预算或
S3-108 核心浏览 Gate 已完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 3 / `S3-106`
- 需求：`FR-BRW-005～007`、`FR-MED-004～006`、`FR-UI-001～004`、
  `NFR-ACC-001`、`NFR-PERF-002`
- Change Record：[CR-2026-001](../../changes/CR-2026-001-non-modal-media-preview.md)
- 状态机 owner：`web/src/features/browse/pages/BrowsePage.tsx`
- 集合语义与虚拟焦点 owner：`web/src/components/patterns/MediaCollection`
- 预览固定、Escape 与活动媒体 owner：`web/src/components/patterns/MediaPreview`

## 已交付行为

- 未固定时单击媒体卡片同时更新选择与唯一活动预览；固定后单击只更新选择，双击才切换
  活动预览；取消固定时预览立即跟随当前选择。
- “已选择”由卡片描边与覆盖按钮 `aria-pressed` 表达；“正在预览”由独立眼睛标记、
  内描边和隐藏文字表达。固定状态下按钮可访问名称明确提示“双击切换固定预览”。
- `MediaPreview` 的固定按钮使用 pressed 语义，底部持续说明跟随或固定模式；Escape
  与关闭按钮走同一关闭路径。
- 关闭时保留当前预览资产 ID，通过共享 `MediaCollectionHandle` 的虚拟控制器滚回该项，
  等待虚拟项挂载后恢复其真实语义按钮焦点。
- 图片与视频分支互斥；视频元素以资产 ID 为 key，切换或关闭会卸载旧节点，因此 DOM
  内任一时刻只有一个活动媒体。十万项场景下的播放资源数量和性能阈值仍由 S3-107 固定。
- 桌面继续使用右侧停靠面板且父列表可滚动、选择；窄屏继续进入内容流，没有 modal、
  scrim 或 `inert`。

## 证据

- `MediaCollection.test.tsx`：选择与预览独立语义、单击/双击激活类型。
- `MediaPreview.test.tsx`：固定/取消固定、模式说明、Escape 与媒体分支。
- 组件工作台：`Patterns/MediaPreview/Pinned` 及集合状态，可切换浅色/深色。
- `auth.spec.ts`：真实认证页面验证未固定单击、固定后单击仅选择、双击切换、
  Escape 关闭及虚拟卡片按钮焦点恢复。
- Product Design 同状态、同主题证据：
  `web/qa/s3-106-source-pinned-dark.jpg`、
  `web/qa/s3-106-implementation-pinned-dark.jpg` 和
  `web/qa/s3-106-comparison-pinned-dark.png`；完整记录在 `web/design-qa.md`。

## 验证

完成记录创建后执行：

```sh
npm --prefix web run check
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make test-web-e2e
```

## 保留边界

- `S3-107`：十万媒体规模、DOM/请求/滚动/播放资源和焦点恢复预算。
- `S3-108`：核心浏览/预览 E2E 与 Stage 3 Integrated Done。
- `S4-006～009`：完整查看器、缩放/全屏、Range/codec/离线/删除状态和目标浏览器矩阵。

- 评审日期：2026-07-28
