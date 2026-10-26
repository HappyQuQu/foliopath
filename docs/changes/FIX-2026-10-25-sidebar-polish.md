# FIX：库菜单浮层与紧凑标签目录

- 状态：2026-10-25已修复并同步6010，真实库只读UI核对通过。
- 范围/Owner：POST-MVP-5 / UI-2026-10-25-layered-workspace-alignment、FR-AUTOTAG-003、FR-UI-001～003/006、NFR-UIF-001；web/browse/curation/shared patterns。
- 用户反馈：库下拉展开挤开快速访问；标签下部控件/大块标签堆叠，层次粗糙。
- Gate/合同：既有已接受侧栏与分层工作台的维护修复；GET libraries、GET tags的库内图片计数、cursor/tagQ/kind/sort及标签显示语言owner不变；无API、schema、模型或部署边界变化。

## 两步核对与修复

1. 库切换：复现展开后快速访问下移；菜单改为picker内绝对定位浮层、有界高度和自身滚动，四工作区共用同一DirectoryNavigation。保留选中、键盘上下/Home/End、外部点击关闭与原库范围。Escape由picker先处理，手机不会同时关闭外层抽屉，焦点回到触发器。状态：已修复。
2. 标签：当前库的scope缩短，搜索优先，语言用同一TagLanguageControl的紧凑变体；标签改为对齐的目录行与数量徽标，五档12/13/14/15/16px字级保留数量关系，颜色集中表达当前选中，移除面向用户的技术说明。列表独立滚动，分页与下方媒体类型/排序固定；DirectoryNavigation的fill插槽与ContextSidebar紧凑密度仍由各自owner控制。手机抽屉标题采用68px，不再沿用双行搜索header的112px。状态：已修复。

不新增标签分组/排序方式，不改变标签ID、名称、来源、翻译/人工优先或计数。双语/长名可换行、完整名称可读；本页数量明确标为“本页”，不伪称全库总数。原型重复页面保持归档；生产CurationPage的DenseTagDirectory与ContextSidebar CompactFilters提供明确合成fixture。

## 验证

- 类型检查、前端架构/视觉reference manifest、Storybook及冻结源码Docker生产构建通过。
- 6文件49项定向单测通过：LibraryTagCloud、TagLanguageControl两个密度、CurationPage、BrowsePage、ContextSidebar、AppShell。
- Linux context-sidebar 18项通过，32张基线更新后不更新模式复验通过；390/768/1440/1920，zh-CN/light与en/dark配对。新增100个长中英标签的有界列表，验证菜单不推动导航、菜单初始/方向键焦点、Escape不关抽屉、标签滚动不移动搜索和底部筛选、窄屏列表保留至少100px可用高度、无横向溢出、axe WCAG2 A/AA零问题。
- `make arch-check`、`git diff --check`通过。本次无Go业务修改，没有重跑全量Go/集成、真实模型、容量或发行矩阵。
- 真实6010：4k标签当前页74个，点击“海滩”显示已载入50项，保持API分页；菜单覆盖导航而不挤开，Escape恢复触发器。纯只读UI，不编辑收藏、标签、人物或媒体。
- 截图与对照：`.artifacts/sidebar-polish-20261025/01～07`、`comparison-tags.png`、`comparison-menu-state.png`及browser-final-check；用户提供的两张截图为反馈来源，当前运行截图为修复证据。截图本身不代替行为/无障碍自动化。

## 6010部署与保留

当前融合搜索后端dac00067先用旧前端重建，程序逐字节一致，SHA256为
`3a65f65a4f8be620a67d00925c15ad5e321038e006ecc3df90413e4511c17e7e`。
再只替换前端，基于冻结的已部署web源码，未引入工作区尚在开发的人物整理等改动；融合搜索不回退。

新主镜像`sha256:1a0324a53d64e45f9a96d5004113d05702ea364b79de59c7e7145b4680aad6af`；主StartedAt `2026-10-25T06:02:16.444546795Z`。
AI仍52f4a40c，同一容器及2026-10-24 21:48:25 +08启动时间未变，未重启AI；两个png-v4 profile、153类策略、semantic640及人物身份保持。

完整停写备份：`/Volumes/extended/foliopath/backup-6010-sidebar-polish-20261025-lsjHkh`（700，私有配置600），37376文件/1095937762字节逐文件核对一致。
manifest `c9062ef7768294d7926e597c2c3c9bcbe3bacfc61fff0429b7cc55ae14ab4077`；DB `7d4049c299a5c435acbd42597810696fa0319d9ec0456f8f517101d42cd37f41`。
复制出的Linux验证目录用旧/新镜像database-check均通过，未打开在线宿主SQLite或修改备份原件。
仅ai.env的主image改变，三Compose、挂载、端口、环境变量值及不限内存覆盖保持；原件始终只读。

更新前后JSON与27组保护摘要完全一致：45条AI operation、8700标签run、48917标签关联、153标签、22365语义向量及人物/人工整理等保留。
8690 ready、10既有源failed、0 pending不变；AI/扫描/派生/reconcile/清理活动队列为空。未取消、重提或清除任务。
原库人工标签/收藏/人物起始为空，不冒充非空保护证明。
最终原主容器database-check通过，新SPA五个文件与六个深链逐字节匹配候选；双服务healthy。
本次没有schema或后端变化，单独界面回退可正常停主并仅恢复dac00067，保留data/AI；完整恢复仍须配对备份并先保存更新后的用户整理。
