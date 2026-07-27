# S4-006 完整媒体查看器完成记录

## 结论

**Done — 可直达完整图片/视频查看器已连接真实资产详情与原内容接口。**

本记录只完成 `S4-006` 的查看器结构与核心交互，不把 Stage 4 标记为 Integrated Done。
GIF/codec、不可播放、损坏、离线和已删除降级状态仍由 `S4-007` 负责；目标浏览器/输入矩阵
和综合 E2E 分别由 `S4-008～009` 负责。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 4 / `S4-006`
- 需求：`FR-MED-004～007`、`FR-UI-001～004`、`NFR-ACC-001`
- Change Record：[CR-2026-001](../../changes/CR-2026-001-non-modal-media-preview.md)
- 内容后端 Gate：[S4-005B Backend Ready](s4-media-content-backend-ready.md)
- 共享查看器 owner：`web/src/components/patterns/MediaViewer`
- 详情 query / 页面组合：`web/src/features/media`
- 安全返回与瞬时序列 codec：`web/src/lib/navigation/viewer.ts`
- 资产详情/content adapter：`web/src/lib/api/catalog.ts`

## 已交付行为

- 浏览和搜索的共享 `MediaPreview` 增加显式“进入完整查看器”；完整查看器使用规范路由
  `/libraries/:libraryId/media/:assetId`，不会取代父页面内不遮挡的快速预览。
- 查看器通过生成 client adapter 读取单项资产详情；图片、动图和视频使用同源 opaque
  asset ID content URL，不接受任意路径，也不暴露宿主路径。
- 图片支持适应窗口、1:1、0.25～4 倍按钮缩放和指针拖动平移；视频使用原生
  `controls`、`playsInline` 与 metadata preload。基本信息沿用共享详情格式化 owner。
- 共享 chrome 提供关闭、前后项、信息开关和 Fullscreen API；方向键在焦点不位于交互控件时
  切换，Escape 优先退出全屏，否则关闭查看器。
- 来源页面把当前已载入结果映射成最小 `{id, libraryId}` 序列。前后项只在这段有界序列中
  移动；刷新或直接进入时仍能查看当前媒体，但不会伪造无法重建的列表上下文。
- `from` 参数只接受 `/search` 或同一媒体库下的 browse/search 路径，协议相对 URL、其他库
  或无效输入都回退到当前媒体库根浏览页。
- 关闭后恢复来源 pathname/query，把当前 asset ID 交回共享虚拟集合并恢复真实媒体卡片焦点。
- 桌面保持近黑媒体舞台、边缘导航、右侧信息面板与底部状态栏；390px 下信息面板变为底部
  抽屉，查看器不创建第二套主题机制。

## 自动与设计证据

- `MediaViewer.test.tsx`：适应/1:1/缩放、信息开关、图片/视频分支、Fullscreen API、
  方向键/Escape 和控件焦点保护。
- `MediaViewerPage.test.tsx`：真实 Query owner、前后路由、来源恢复、焦点上下文和不可信
  `from` 回退。
- `viewer.test.ts`：最小序列映射、运行时状态校验和 same-origin/same-library 返回策略。
- `catalog.test.ts`：单资产详情必须通过生成 client operation。
- `MediaViewer.stories.tsx`：共享图片与视频稳定状态进入组件工作台。
- `auth.spec.ts`：真实后端图片从浏览预览进入查看器，验证适应、1:1、放大、信息关闭、
  返回来源和卡片焦点恢复。
- Product Design 对照：静态原型 screen 9 与真实隔离前端在同一 1440×900 视口做了同屏
  对照；媒体舞台、header/footer、边缘导航和信息面板没有可执行 P0/P1/P2 差异。
  390×844 另行验证移动 header、舞台、箭头和底部信息面板。记录见 `web/design-qa.md`。

## 验证

完成记录创建时成功执行：

```sh
npm --prefix web run check
make test-web-e2e
```

仓库完整 Go、集成与容器门禁在提交前另行执行并以实际结果为准。

## 保留边界

- `S4-007`：GIF 策略、codec/不可播放、损坏、离线、已删除和 source-change 降级状态。
- `S4-008`：目标桌面/移动浏览器、键盘、触摸、Range、焦点与错误矩阵。
- `S4-009`：搜索/预览/查看器完整 E2E 与 Stage 4 Integrated Done。
- Stage 5：发布镜像、可信代理/网络暴露、完整浏览器矩阵与 RC 视觉回归。

- 评审日期：2026-07-28
