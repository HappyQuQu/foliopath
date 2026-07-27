# S4-004 前端搜索界面完成记录

## 结论

**Done — S4-004 统一搜索输入、筛选、真实结果与可恢复 URL 已连接 Backend Ready API。**

本记录完成搜索页面本身，不把 Stage 4 标记为 Integrated Done，也不宣称搜索结果已经接入
非模态预览或完整媒体查看器。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 4 / `S4-004`
- 需求：`FR-SRH-001～004`、`FR-BRW-004～005`、`FR-BRW-007`、`FR-BRW-009`、
  `FR-UI-001～004`、`NFR-ACC-001`、`NFR-PERF-001～002`
- 后端 Gate：[S4-003 Backend Ready](s4-search-backend-ready.md)
- HTTP 权威：`api/openapi.yaml`
- 前端 URL/query owner：`web/src/features/search`
- HTTP adapter：`web/src/lib/api/catalog.ts`
- 共享 UI owner：`web/src/components/ui/SearchInput`、`web/src/components/media/MediaCollection`

## 已交付行为

- `/libraries/:libraryId/search` 默认搜索当前库；`/search` 表达全部库搜索。范围可切换为
  当前库、当前目录或全部库，当前目录可独立开启递归。
- 单一 URL codec 保存 `q`、`scope`、`directoryId`、`recursive`、`kind`、`date`、
  `sort` 和 `order`；非法值安全规范化，改变查询指纹时重置服务端游标。
- query owner 根据范围调用库内或全局 search operation，固定 50 项 keyset 页面；
  虚拟集合只挂载可视窗口，接近末端才请求下一页。
- 类型筛选映射图片、动图和视频；日期筛选使用 filesystem modification time 的半开区间；
  名称/修改时间排序及升降序均由已评审契约执行。
- 结果复用共享媒体卡片与布局偏好，显示可访问文件名、来源媒体库和 library-relative
  目录链接；不从显示路径推导 API 参数，也不暴露宿主或容器绝对路径。
- loading、无结果、媒体库离线、首屏失败和下一页失败保持不同状态与恢复动作；没有筛选时
  的无结果动作回到搜索输入，而不是显示无意义的“清除筛选”。
- AppShell 搜索导航现在是可访问链接；库上下文页保留当前库，全部库入口可直接访问。

## 自动与设计证据

- `web/src/features/search/urlState.test.ts`：三种范围、递归、筛选、排序、非法值规范化和
  API 参数映射。
- `web/src/features/search/pages/SearchPage.test.tsx`：真实 query 参数、URL 恢复、范围/
  类型切换、无结果与离线状态。
- `web/src/lib/api/catalog.test.ts`：库内和全局 search operation、cursor 与过滤 query
  绑定。
- `web/tests/e2e/auth.spec.ts` + `tests/e2e/web_auth.sh`：真实索引数据下库内/全部库搜索、
  类型空结果、清除筛选、浏览器返回恢复、长库名、1024px 无横向溢出及 axe
  serious/critical。
- `SearchInput.stories.tsx` 固定默认、已有查询、错误和禁用状态。
- Product Design 对照：在同一 1280×720 视口逐项对照静态原型 `7/15 搜索结果` 与
  `8/15 搜索无结果`，覆盖浅色/深色 token、壳层、搜索命令、筛选密度、结果层级和空态。
  生产实现额外展示契约支持的显式排序；API 不提供总数，因此只显示已载入数量。

## 验证

完成记录创建时成功执行：

```sh
npm --prefix web run check:types
npm --prefix web run test
npm --prefix web run build:storybook
make test-web-e2e
```

仓库完整门禁在提交前另行执行并以实际结果为准。

## 保留边界

- `S4-005`：搜索结果复用 Stage 3 共享非模态预览，不创建 search-only 实现。
- `S4-006～007`：可直达完整查看器、媒体控制和不可用/损坏降级。
- `S4-008～009`：目标浏览器/输入方式矩阵和 Stage 4 Integrated Done。

- 评审日期：2026-07-28
