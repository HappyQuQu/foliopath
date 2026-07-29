# FIX-2026-07-29 分页重试恢复过期游标

- 类型：已批准 MVP slice 内的例行修复
- 关联范围：`FR-BRW-001～009`、`FR-SRH-001～004`
- 关联 Gate：S3 Browse Integrated Done、S4 Search Media Integrated Done
- 受影响不变量：媒体分页保持有界；分页失败保留已显示项目；服务端
  `invalid_cursor` 不静默回退第一页
- Owner：前端共享 infinite-query 重试策略；browse/search query consumer

## 修复与证据

普通传输失败继续重试当前下一页。可靠 catalog generation/revision 推进导致
`invalid_cursor` 时，客户端显式刷新已载入的有界 cursor 链，再自动请求下一页；服务端
仍保持 fail-closed。共享集合在重试期间显示 loading 并阻止重复点击。

回归证据由 `retryInfiniteNextPage.test.ts`、`MediaCollection.test.tsx` 以及 browse/search
页面测试提供。
