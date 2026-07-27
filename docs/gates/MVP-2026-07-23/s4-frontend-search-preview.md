# S4-005 搜索复用非模态预览完成记录

## 结论

**Done — 搜索结果已复用 Stage 3 的共享非模态预览与唯一固定状态机。**

本记录完成搜索中的快速图片/视频预览，不把 Stage 4 标记为 Integrated Done，也不包含
完整媒体查看器、缩放/全屏、媒体降级矩阵或最终浏览器覆盖。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 4 / `S4-005`
- 需求：`FR-SRH-004`、`FR-MED-004～006`、`FR-UI-001～004`、
  `NFR-ACC-001`、`NFR-PERF-002`
- Change Record：[CR-2026-001](../../changes/CR-2026-001-non-modal-media-preview.md)
- 内容后端 Gate：[S4-005B Backend Ready](s4-media-content-backend-ready.md)
- 共享状态机 owner：`web/src/components/patterns/MediaPreview/useMediaPreviewController.ts`
- 集合/焦点 owner：`web/src/components/patterns/MediaCollection`
- 媒体渲染 owner：`web/src/components/patterns/MediaPreview`
- 业务组合：`web/src/features/browse/pages/BrowsePage.tsx` 与
  `web/src/features/search/pages/SearchPage.tsx`

## 已交付行为

- 浏览页原有预览状态迁入共享 controller；搜索与浏览不再分别拥有固定、选择、切换、
  宽度或关闭焦点恢复逻辑。
- 未固定时单击卡片更新选择和唯一活动预览；固定后单击只更新选择，双击才切换预览；
  取消固定时预览跟随当前仍存在的选择。
- 搜索命令和筛选保持可操作，桌面右侧预览不使用遮罩、dialog 或 `inert`；≤1024px
  复用既有入流布局。
- 固定预览保存当前媒体快照。查询、范围、类型、日期或排序改变导致媒体暂时离开结果集时，
  图片/视频节点继续存在并显示“固定预览不在当前结果中”；重新进入结果后恢复位置和导航。
- 图片与视频仍通过真实 opaque asset ID 内容 URL 加载；详情格式化成为共享纯映射，不导入
  生成 client 或 server-state query。
- 关闭和 Escape 使用同一路径；若卡片仍在结果中，共享虚拟控制器滚回并恢复真实按钮焦点。

## 自动与设计证据

- `SearchPage.test.tsx`：搜索内打开、固定、选择、双击切换、Escape/焦点恢复，以及筛选替换
  结果集时固定预览继续存在。
- `BrowsePage.test.tsx`、`MediaCollection.test.tsx` 与 `MediaPreview.test.tsx`：重构后
  Stage 3 的选择/预览分离、键盘、固定和媒体分支回归。
- `auth.spec.ts` + `tests/e2e/web_auth.sh`：真实图片索引下搜索打开并固定预览，视频筛选产生
  空结果时预览仍保持，清除筛选后关闭恢复卡片焦点；同时覆盖无横向溢出和 axe
  serious/critical。
- Product Design 同主题对照：已在 in-app Browser 中并排检查
  `web/qa/s3-106-source-pinned-dark.jpg` 与真实隔离搜索页的 1265×712 深色固定预览；
  搜索命令、结果和右侧面板同时可见，面板继续使用同一 header、stage、导航、详情和固定
  状态层级，无新增 P0/P1/P2 差异。完整记录见 `web/design-qa.md`。

## 验证

完成记录创建时成功执行：

```sh
npm --prefix web run check:types
npm --prefix web test -- src/features/search/pages/SearchPage.test.tsx src/features/browse/pages/BrowsePage.test.tsx src/components/patterns/MediaPreview/MediaPreview.test.tsx src/components/patterns/MediaCollection/MediaCollection.test.tsx
make test-web-e2e
```

仓库完整门禁在提交前另行执行并以实际结果为准。

## 保留边界

- `S4-006`：可直达完整查看器、适应/缩放/平移、1:1、前后项、信息、全屏与关闭恢复。
- `S4-007`：GIF、原生视频/Range、不可播放、损坏、离线和已删除状态。
- `S4-008～009`：目标浏览器/输入矩阵和 Stage 4 Integrated Done。

- 评审日期：2026-07-28
