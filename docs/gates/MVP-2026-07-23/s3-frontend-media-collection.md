# S3-103 前端媒体集合完成记录

## 结论

**Done — S3-103 已用真实缩略图 API 交付默认自适应网格、可记忆瀑布流和统一虚拟化集合。**

本记录不把 Stage 3 标记为 Integrated Done，也不宣称 S3-104 完整状态矩阵、
S3-105～106 非模态预览或 S3-107 十万项容量 Gate 已完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 3 / `S3-103`
- 需求：`FR-BRW-004～005`、`FR-BRW-009`、`FR-MED-001`、`FR-UI-001～003`、
  `NFR-ACC-001`、`NFR-PERF-001～002`
- 后端 Gate：[S3-007 Backend Ready](s3-browse-thumbnail-backend-ready.md)
- HTTP 权威：`api/openapi.yaml`
- 查询 owner：`web/src/features/browse/queries.ts`
- 集合 owner：`web/src/components/patterns/MediaCollection`
- 偏好 owner：`web/src/lib/storage/preferences.ts`

## 已交付行为

- BrowsePage 继续使用每页 50 项的 TanStack infinite query；layout 不复制 query、
  cursor、选择或 item identity。
- 共享 `MediaCollection`/`MediaCard` 使用 `@tanstack/react-virtual` lanes。DOM 顺序
  等于查询顺序、key 使用资产 ID、overscan 随列数有界；200 项合成测试证明未挂载全部 DOM。
- grid 默认按可用宽度生成 1～6 列并使用 4:3 裁切；masonry 使用索引宽高比，
  限制异常比例并为未知尺寸回退 4:3，不使用 CSS multi-column。
- grid/masonry 控件为可命名、可聚焦的 `aria-pressed` IconButton group。选择写入现有
  `foliopath.preferences.v1`，刷新后恢复且不改变 browse URL。
- ready 引用直接显示同源 WebP；图片延迟加载、异步解码，媒体卡片拥有文件名/类型可访问
  名称。pending/failed/unavailable 先提供稳定尺寸的安全占位，完整恢复动作归 S3-104。
- 目录无子项时不再用大面积空卡阻断媒体首屏；同视口原型对照后，媒体密度与工具条节奏
  已恢复到确认方向。

## 自动与视觉证据

- `MediaCollection.test.tsx`：200 项有界 DOM、稳定 posinset/key、ready URL、masonry
  原始比例和失败占位。
- `preferences.test.ts`：默认 grid、非法值回退、masonry 记忆且不覆盖主题/语言。
- `BrowsePage.test.tsx`：共享布局控件、`aria-pressed` 和统一 preference namespace。
- `auth.spec.ts` + `web_auth.sh`：真实只读 JPEG → libvips ready WebP、direct/recursive、
  grid→masonry→reload 记忆、390px/1024px 无溢出与 axe serious/critical。
- 组件工作台：`Patterns/MediaCollection` 的 80 项 grid/masonry 合成集合。
- Product Design：静态原型主浏览屏与真实后端生产页面在同一桌面视口对照；证据在
  `web/qa/s3-103-comparison-grid-light-v2.png`、grid/masonry/light/dark 单独捕获及
  `web/design-qa.md`。

## 验证

完成记录创建后成功执行：

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

- `S3-104`：完整 skeleton、empty、error、offline、thumbnail pending/failed 恢复。
- `S3-105～106`：共享非模态图片/视频预览、固定与双击规则。
- `S3-107～108`：十万媒体预算、核心 E2E 和 Stage 3 Integrated Done。

- 评审日期：2026-07-28
