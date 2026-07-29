# FIX-2026-07-29 redesign 应用壳恢复

- 类型：已批准 MVP slice 内的例行 UI 回归修复
- 关联范围：`FR-BRW-001～009`、`FR-UI-001～007`、`FR-SCN-010～014`
- 目标版本与阶段：MVP / Stage 3–4 Integrated Done 后的界面修复；自动发现 S3 本地 UI
- 关联 Gate：S3 Browse Integrated Done、S4 Search/Media Integrated Done、WCH-S3
- 受影响不变量：URL 保存库/目录/筛选/排序；刷新仅重取派生索引；原始媒体只读
- Owner：`web/src/components/patterns/AppShell`、`web/src/features/browse`
- 合同：`docs/ui-design.md`、`prototypes/apple-redesign`；不改变 OpenAPI 或持久化合同

## 决定

生产界面重新以 `prototypes/apple-redesign` 为视觉验收基线。桌面应用壳恢复固定侧栏、
纯文字品牌、目录树下方的搜索/设置导航、面包屑顶栏与精简浏览工具栏。浏览页不显示指向
自身的“返回浏览”入口；没有目录树的页面仍使用同一应用级导航。

媒体类型筛选保留真实 URL/query 行为，但收进顶栏漏斗菜单。手动目录刷新保留已批准的
WCH-S3 行为，但使用工具栏末端的紧凑图标。账户菜单保留安全退出，因此作为相对原型的
明确产品约束差异继续位于顶栏。

本记录取代 `FIX-2026-07-29-navigation-simplification.md` 的当前界面决定；旧记录仅作为
历史保留。

## 证据

- `AppShell.test.tsx`：固定侧栏、底部导航、移动抽屉焦点和退出菜单。
- `BrowsePage.test.tsx`：漏斗媒体筛选、刷新、目录/媒体查询与 URL 状态。
- `web/qa/redesign-regression-2026-07-29/13-final-comparison.png`：1265 × 712 同视口
  原型/生产对照。
- 根 `design-qa.md`：比较历史、允许差异和最终结论。
