# FIX-2026-07-29 应用导航精简

- 类型：已批准 MVP slice 内的例行 UI 修复
- 关联范围：`FR-BRW-001～009`、`FR-UI-001～007`
- 目标版本与阶段：MVP / Stage 3–4 Integrated Done 后的界面修复
- 关联 Gate：S3 Browse Integrated Done、S4 Search/Media Integrated Done
- 受影响不变量：媒体库与目录状态保留在 URL；移动端目录抽屉保持键盘与焦点可用
- Owner：共享 `web/src/components/patterns/AppShell` 与 `web/src/routes`
- 合同：`docs/user-flows.md`、`docs/ui-design.md`；不改变 OpenAPI 或持久化合同

## 修复与证据

浏览页侧栏只保留媒体库切换器与目录树，不再重复呈现“浏览”“搜索”“设置”应用菜单。
现有搜索工具继续位于浏览页顶部，设置改为右上角语义链接。搜索页提供返回浏览与设置，
设置页提供返回浏览与搜索，并保持来源媒体库上下文，不显示指向当前页面的自链接。没有
目录树的设置类页面不再渲染空侧栏；移动端仍通过明确按钮打开同一目录抽屉。
浏览页和其他页面的搜索入口统一使用放大镜图标；漏斗只用于搜索页内部的筛选区域。
用户入口固定在主题切换右侧，并通过共享 AppShell 账户菜单直接提供安全退出；菜单支持
点击外部和 `Escape` 关闭，退出仍复用现有会话、CSRF 与登录路由合同。

认证后访问根路由时，应用优先进入 ready 媒体库，其次进入仍有可靠索引的媒体库，再其次
进入首个已配置媒体库；仅在没有媒体库时进入媒体库欢迎与创建流程。

回归证据由 `AppShell.test.tsx`、`AppRouter.test.tsx`、`LibrariesPage.test.tsx` 和
`SearchPage.test.tsx` 提供，覆盖重复菜单移除、右上角设置入口、账户入口顺序与直接退出、
目录抽屉焦点恢复、默认浏览路由，以及媒体库卡片仍可进入浏览页。
