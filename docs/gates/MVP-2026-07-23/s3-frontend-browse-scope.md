# S3-102 前端浏览范围完成记录

## 结论

**Done — S3-102 当前目录/递归范围与稳定 URL 已连接真实资产 API。**

本记录完成 direct/recursive 范围、各自默认排序、显式排序 URL、真实 keyset 资产摘要、
递归来源返回和浏览器历史恢复。它不把 Stage 3 标记为 Integrated Done，也不宣称
缩略图网格、瀑布流、虚拟化、完整状态矩阵或非模态预览已经完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 3 / `S3-102`
- 需求：`FR-BRW-002～003`、`FR-BRW-005`、`FR-BRW-007`、`FR-BRW-009`、
  `FR-UI-001`、`NFR-ACC-001`、`NFR-PERF-001～002`
- 后端 Gate：[S3-007 Backend Ready](s3-browse-thumbnail-backend-ready.md)
- HTTP 权威：`api/openapi.yaml`
- 前端 URL owner：`web/src/features/browse/urlState.ts`
- 前端 query/UI owner：`web/src/features/browse`
- HTTP adapter：`web/src/lib/api/catalog.ts`

## 已交付行为

- direct 模式规范默认 `recursive=false`、`sort=name`、`order=asc`，query 为空。
- recursive 模式规范默认 `recursive=1`、`sort=modifiedAt`、`order=desc`；默认
  sort/order 不重复写入 URL。
- 非默认排序以 `sort=name|modifiedAt`、`order=asc|desc` 明确进入 URL；非法值
  fail-safe 规范为当前模式默认。cursor 只存在 TanStack Query 分页状态，不进入 URL。
- 模式切换保留 library/directory、采用目标模式默认排序并产生可返回历史；刷新、复制、
  前进/后退恢复相同查询指纹。
- 资产请求固定每页 50 项，通过生成 client 发送 directory、recursive、sort、order 和
  opaque cursor；不调用搜索字段，不加载无界集合。
- 递归摘要显示每项 library-relative 来源；来源链接使用 `Asset.directoryId` 返回所属
  目录并关闭递归，不能从显示路径反推或提交路径。
- direct 空结果若 directory detail 证明后代存在媒体，显示真实差额并提供开启递归动作；
  pending/scanning/offline 的完整集合状态继续归 `S3-104`。

## 自动证据

- `web/src/features/browse/urlState.test.ts`：direct/recursive 默认、非法值规范化、
  非默认排序和深层可复制 URL。
- `web/src/lib/api/catalog.test.ts`：真实 operation 路径及 directory/recursive/sort/order/
  limit/cursor query 绑定。
- `web/src/features/browse/pages/BrowsePage.test.tsx`：递归恢复、默认排序、模式切换、
  来源链接关闭递归和真实资产 query 参数。
- `web/tests/e2e/auth.spec.ts` + `tests/e2e/web_auth.sh`：两层合成 JPEG 索引下 direct
  隐藏后代、recursive 显示后代、显式排序 URL、浏览器两次返回、来源目录跳转、刷新恢复、
  390px/1024px 响应式、无溢出和 axe serious/critical 检查。
- Product Design 对照：已确认静态原型 `6/15 递归浏览状态`；生产页面在相同桌面视口对照
  递归开关、默认排序、侧栏、面包屑与内容层级，真实后端页面另验证 direct/recursive/
  sort/返回交互。

## 验证

完成记录创建时成功执行：

```sh
npm --prefix web run check:types
npm --prefix web run test
make arch-check
make test-web-e2e
```

仓库完整门禁在提交前另行执行并以实际结果为准。

## 保留边界

- `S3-103`：真实缩略图、自适应网格、可记忆瀑布流和统一虚拟化集合。
- `S3-104`：完整 skeleton、empty、error、offline、pending/failed 状态。
- `S3-105～106`：共享非模态图片/视频预览。
- `S3-107～108`：容量预算、核心 E2E 和 Stage 3 Integrated Done。

- 评审日期：2026-07-28
