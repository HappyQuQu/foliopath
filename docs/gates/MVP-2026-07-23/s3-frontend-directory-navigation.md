# S3-101 前端目录导航完成记录

## 结论

**Done — S3-101 桌面/移动目录导航已连接真实浏览 API。**

本记录只完成媒体库切换、桌面固定侧栏、移动抽屉、延迟加载目录树、面包屑和
可复制/可刷新直达 URL。它不把 Stage 3 标记为 Integrated Done，也不宣称递归模式、
媒体网格、虚拟化、缩略图状态或非模态预览已经完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 3 / `S3-101`
- 需求：`FR-BRW-001`、`FR-BRW-003`、`FR-BRW-005`、`FR-BRW-008～009`、
  `FR-UI-001`、`NFR-ACC-001`、`NFR-PERF-001～002`
- 后端 Gate：[S3-007 Backend Ready](s3-browse-thumbnail-backend-ready.md)
- HTTP 权威：`api/openapi.yaml`
- 前端 owner：`web/src/features/browse`
- HTTP adapter：`web/src/lib/api/catalog.ts`
- 共享壳层：`web/src/components/patterns/AppShell`

## 已交付行为

- `/libraries/:libraryId/browse/:directoryId?` 是唯一目录 URL codec；根目录省略
  `directoryId`，深层目录刷新后通过真实 directory detail 恢复。
- 媒体库选择器读取规范 library query owner，切换时进入目标库根 URL。
- 目录树使用嵌套导航语义；展开与选择分开，根和展开节点只读取直接子级，每页
  50 项并保留服务端 `nextCursor` 的“载入更多”入口。
- directory detail 的完整 breadcrumb 同时驱动主区面包屑与祖先展开链；每一级均为
  可聚焦直达链接。
- 桌面复用固定 AppShell 侧栏；窄屏复用同一侧栏抽屉、背景遮罩、Escape 关闭和
  触发按钮焦点恢复，不另造 feature-local sheet。
- 复制按钮只复制当前浏览器地址并用规范 Toast 报告结果；失败时明确提示从地址栏复制。
- offline 库显示可靠索引保留提示；完整 loading/error/empty/thumbnail 状态矩阵仍归
  `S3-104`。

## 自动证据

- `web/src/lib/api/catalog.test.ts`：生成 client 路径/query、direct-child 有界分页和
  root-to-current breadcrumb 映射。
- `web/src/features/browse/pages/BrowsePage.test.tsx`：深层 URL 恢复、面包屑链接、
  祖先自动展开、展开/收起分离和子目录直达链接。
- `web/src/components/patterns/AppShell/AppShell.test.tsx`：移动抽屉 Escape 与焦点恢复。
- `web/tests/e2e/auth.spec.ts`：真实后端 setup/建库/扫描后进入根目录、打开真实子目录、
  刷新保持直达 URL、390px 抽屉、1024px 固定侧栏、无横向溢出和 axe
  serious/critical 检查。
- Product Design 对照：已确认静态原型 `5/15 主浏览界面`；生产桌面页面在
  `127.0.0.1:5173` 用真实认证会话检查目录侧栏、库选择器、面包屑、空目录和主题入口。

## 验证

完成记录创建时成功执行：

```sh
npm --prefix web run check:types
npm --prefix web run test
npm --prefix web run build
npm --prefix web run check:architecture
make test-web-e2e
```

仓库完整门禁在提交前另行执行并以实际结果为准。

## 保留边界

- `S3-102`：当前目录/递归模式及其 URL 状态。
- `S3-103`：真实 asset page、缩略图、自适应网格/瀑布流与统一虚拟化。
- `S3-104`：完整 skeleton、empty、error、offline、pending/failed 状态。
- `S3-105～106`：共享非模态图片/视频预览。
- `S3-107～108`：容量预算、核心 E2E 和 Stage 3 Integrated Done。

- 评审日期：2026-07-28
