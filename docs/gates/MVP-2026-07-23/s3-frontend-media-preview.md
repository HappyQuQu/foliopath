# S3-105 前端媒体预览完成记录

## 结论

**Done — S3-105 已交付共享 `MediaPreview` 的图片、原生视频、基本信息、前后项、关闭与宽度调整。**

本记录不把 Stage 3 标记为 Integrated Done，也不宣称 S3-106 的固定/双击/焦点恢复、
S3-107 十万项容量或 S3-108 核心浏览 Gate 已完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 3 / `S3-105`
- 需求：`FR-BRW-005～007`、`FR-MED-004～006`、`FR-UI-001～004`、
  `NFR-ACC-001`、`NFR-PERF-002`
- Change Record：[CR-2026-001](../../changes/CR-2026-001-non-modal-media-preview.md)
- HTTP 权威：`api/openapi.yaml` 的 `GET /api/v1/assets/{assetId}/content`
- 预览 owner：`web/src/components/patterns/MediaPreview`
- 集成 owner：`web/src/features/browse/pages/BrowsePage.tsx`
- 内容 URL owner：`web/src/lib/api/catalog.ts`

## 已交付行为

- `MediaCard` 提供覆盖卡片的语义按钮，未固定时单击打开或切换唯一活动预览；
  `aria-pressed` 标记当前项，递归来源链接仍保持独立可操作。
- 共享预览不导入 feature、生成 client 或 TanStack Query。feature 只传入经
  `encodeURIComponent` 构造的同源内容 URL、展示值和回调。
- image/animated 使用真实原内容与 contain；video 使用原生 controls、playsInline 和
  metadata preload，继续由后端 Range endpoint 交付，不新增转码或下载语义。
- 基本信息显示类型/MIME、library-relative 路径、修改时间、像素尺寸、格式化大小和
  可选时长；不展示 host path、EXIF 面板或后端诊断。
- 前后项按当前已载入查询顺序移动，在首尾禁用，不绕回也不伪造未载入 cursor 项。
- 桌面右侧面板默认 406px，可通过指针拖动或可聚焦 separator 的
  ArrowLeft/ArrowRight/Home/End 在 360～620px 内调整；≤1024px 进入内容流并隐藏
  不适用的垂直调宽控件。
- 图片/视频加载失败显示类型对应的安全降级状态。完整 offline/missing/unreadable/
  unsupported codec 恢复文案仍由 S4-007 接入。

## 证据

- `MediaPreview.test.tsx`：图片、信息、前后项、关闭、键盘调宽和上界。
- `MediaCollection.test.tsx`：卡片活动状态、单击激活与递归来源链接并存。
- `catalog.test.ts`：内容 URL 同源并编码 opaque asset ID。
- 组件工作台：`Patterns/MediaPreview` 的 image/video 状态，可切换浅色/深色。
- `auth.spec.ts`：真实认证和原内容 endpoint 的图片预览、首尾禁用、separator 键盘调宽、
  关闭；既有响应式与 axe 检查继续覆盖页面。
- Product Design 同视口证据：
  `web/qa/s3-105-source-preview-dark.jpg`、
  `web/qa/s3-105-implementation-preview-dark.jpg` 和
  `web/qa/s3-105-comparison-preview-dark.png`；完整记录在 `web/design-qa.md`。

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

- `S3-106`：固定后单击选择/双击切换、Escape、一个活动媒体、关闭焦点和虚拟锚点恢复。
- `S3-107`：十万媒体规模、播放资源释放和滚动预算。
- `S3-108`：核心浏览/预览 E2E 与 Stage 3 Integrated Done。
- `S4-006～009`：完整查看器、缩放/全屏、Range/codec/离线/删除状态和目标浏览器矩阵。

- 评审日期：2026-07-28
