# S4-007 媒体播放与降级状态完成记录

## 结论

**Done — GIF、原生视频封面与媒体不可用状态已由同一呈现策略接入预览和完整查看器。**

本记录完成 `S4-007`，不把 Stage 4 标记为 Integrated Done。Chromium 桌面/移动触摸、
真实 Range、键盘/焦点和错误组合矩阵由 `S4-008` 负责，完整纵向 E2E 由 `S4-009`
负责。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 4 / `S4-007`
- 需求：`FR-MED-002～008`、`FR-UI-001～004`、`NFR-ACC-001`
- 内容后端 Gate：[S4-005B Backend Ready](s4-media-content-backend-ready.md)
- 媒体可用性策略 owner：`web/src/lib/media/availability.ts`
- 稳定状态表面 owner：`web/src/components/patterns/MediaAvailabilityState`
- 消费方：共享 `MediaPreview`、共享 `MediaViewer` 与 `web/src/features/media`

## 已交付行为

- `animated`/GIF 沿用认证原内容 `<img>`，保留浏览器原生动画，不复制帧、不转码，也不把
  `probeStatus=pending` 误判为不可用。
- 视频继续使用唯一原生 `<video controls playsInline preload="metadata">`。同源 content
  URL 由 opaque asset ID 生成，浏览器自行发起 Range；`thumbnail.status=ready` 时使用同源
  缩略图作为 `poster`，没有可用缩略图时不伪造封面。
- 可用性优先级唯一化：`sourceAvailability` 的 offline/missing/unreadable 优先于派生探测
  状态；其次映射 probe failed/unsupported；视频再映射 `unsupported_codec`。浏览、搜索和
  查看器不得各自复制判断。
- 离线、缺失、不可读、损坏、不支持格式、不支持编码、详情已删除和运行时内容读取失败都
  保留原查看器 chrome、关闭与有界前后导航。状态使用图标、标题、原因和适用的重新检查，
  不显示原始后端错误，也不自动跳项或修改原文件。
- 已删除详情由规范 `asset_not_found` 映射为稳定状态；其他详情错误保留重新检查。离线资产
  继续显示可靠索引中的相对路径与基本信息。
- 运行时 `<img>/<video>` 失败可显式重新挂载当前媒体元素。重试不创建第二个播放器。
- 390px 初始进入时基本信息默认收起，避免底部信息面板覆盖媒体状态主操作；桌面仍默认显示
  基本信息，移动端可通过同一信息按钮打开。

## 自动与设计证据

- `availability.test.ts`：source 优先级、损坏/不支持/codec 映射、GIF ready 与封面策略。
- `MediaPreview.test.tsx` / `MediaViewer.test.tsx`：原生视频属性、poster、稳定状态、导航及
  单播放器替换。
- `MediaViewerPage.test.tsx`：真实 Query 状态映射 offline 与 `asset_not_found`，保留关闭
  和重新检查语义。
- 工作台增加 viewer/preview offline 与 unsupported-codec 稳定故事。
- Product Design：静态原型新增同状态查看器；原型和实现以 1440×900 深色同视口并排审查，
  390×844 另行验证状态主操作不被信息面板遮挡。证据见 `web/design-qa.md` 和
  `web/qa/s4-007-*`。

## 验证

完成记录创建时成功执行：

```sh
npm --prefix web run check
make test-web-e2e
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

## 保留边界

- `S4-008`：Chromium 桌面/移动触摸、键盘、真实视频 Range、焦点与错误组合验证。
- `S4-009`：搜索 → 预览 → 查看器的完整 E2E 与 Stage 4 Integrated Done。
- Firefox、Safari/WebKit 具体发布版本与真机性能矩阵归 Stage 5 发布 Gate。
- MVP 不转码，不承诺浏览器无法播放的 codec，也不新增显式下载。

- 评审日期：2026-07-28
