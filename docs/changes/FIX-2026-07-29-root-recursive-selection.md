# FIX-2026-07-29 根目录递归保持目录当前项

- 类型：已批准 MVP slice 内的例行修复
- 关联范围：`FR-BRW-001～009`
- 关联 Gate：S3 Browse Integrated Done
- 受影响不变量：浏览 URL 状态可恢复；目录侧栏当前项与用户选择的位置一致
- Owner：`web/src/features/browse` 浏览 URL codec 与目录导航

## 修复与证据

媒体库根目录开启“包含子目录”只改变当前目录的浏览范围，不再把侧栏当前项切换到
“全部媒体”。只有用户显式点击“全部媒体”时才选中该入口，并通过 `view=all` 与普通
根目录递归 URL 区分；进入具体目录时 codec 会移除该根级导航标记。

回归证据由 `urlState.test.ts` 和 `BrowsePage.test.tsx` 提供，覆盖 URL 规范化、深层目录
去除根级标记、显式全部媒体选中，以及根目录切换递归后保持当前项。
