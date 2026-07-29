# FIX-2026-07-29 浏览媒体类型三态筛选

- 类型：已批准 MVP slice 内的例行 UI 修复
- 关联范围：`FR-BRW-004～005`、`FR-UI-001～007`
- 目标版本与阶段：MVP / Stage 3 Integrated Done 后的界面修复
- 关联 Gate：S3 Browse Contract Ready、S3 Browse Integrated Done
- 受影响不变量：筛选状态保留在 URL；大型集合继续使用稳定 cursor 与虚拟化
- Owner：`web/src/features/browse` 的 URL codec、query 与工具栏组合；
  `web/src/lib/api/catalog.ts` 继续作为资产 API adapter
- 合同：既有 `GET /api/v1/libraries/{libraryId}/assets` 的 `kind` 参数；
  不改变 OpenAPI、数据库、部署或媒体处理合同

## 修复与证据

浏览页工具栏提供“全部、图片、视频”三个互斥状态。“图片”规范映射到后端
`image,animated`，使 GIF 动图与静态图片一起显示；“视频”映射到 `video`；“全部”
省略 `kind` 参数。非默认筛选写入 URL，切换时保留当前媒体库、目录、direct/recursive
范围与排序，并通过 TanStack Query key 重置 cursor。切回已经访问的状态可复用有界页面
缓存。

回归证据由 `web/src/features/browse/urlState.test.ts`、
`web/src/lib/api/catalog.test.ts` 和
`web/src/features/browse/pages/BrowsePage.test.tsx` 提供，覆盖 URL 规范化、GIF 分组、
API 参数、三态语义与缓存恢复。
