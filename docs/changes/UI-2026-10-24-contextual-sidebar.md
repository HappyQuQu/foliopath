# 按功能切换侧栏：生产实施

- 状态：用户已接受侧栏原型，2026-10-24 授权落实正式界面；验证结果见文末。
- ID：UI-2026-10-24-contextual-sidebar；Owner：Codex / web。
- 目标：POST-MVP-5，INT-S3 既有前端消费者体验完善；MVP 搜索/整理既有合同内布局修正。
- 需求：FR-BRW-001/004/007、FR-SRH-001～004、FR-CUR-001～008、FR-INT-009～014、FR-UI-001/002/003/006/010。
- 接受证据：[原型记录](PROTOTYPE-2026-10-24-contextual-sidebar.md)及其截图。

## 控件、状态和合同映射

| 可见项 | 唯一实现/状态所有者 | 既有合同及范围 |
| --- | --- | --- |
| 库选择、快速访问、当前项 | DirectoryNavigation；页面提供 scope 与 destination | libraries adapter，库 ID；人物已命名视图用只读“全部媒体库”替代库选择，按库未命名组仍可切库 |
| 搜索范围、包含子目录、文件名/画面语义、类型、日期、排序、重置 | SearchPage / search/urlState；只在侧栏编辑，顶部只读摘要；来源目录 ID 保留以便范围切回，但只有目录范围带入请求 | `/libraries/{id}/assets`、`/assets`、`/semantic/assets`、`/semantic/videos`；语义仍只支持图片或视频及相似度排序，不发明日期或混合排序 |
| 人物视图、查找、人物列表、当前人物、本库组类型 | AIReviewPage / usePeopleQuery / useFaceClustersQuery；sidebar 与 main 共享 query key | `/people` 全局50项有界分页，`/libraries/{id}/ai/face-clusters` 按库；人物列表 URL `peopleCursor` 与详情媒体 `cursor` 分开，旧列表 cursor 无损替换为 peopleCursor |
| 收藏全部/图片/视频计数及排序 | CurationPage / useFavoritesQuery | `/favorites?libraryId=...`；计数用服务端 counts，不用已加载分页长度。图片含动图；最近收藏/修改时间/名称/大小沿用现有排序 |
| 标签云、查找、中英文别名、显示语言、清除选中 | LibraryTagCloud / useTagDisplay / 原有 tagQ/tagCursor | `/tags` 库内图片统计；`/tags/{id}/assets` 结果。语言只影响展示，标签 ID/来源和已选集合不变 |
| 加载、空、错误、失效游标、离线与重试 | 既有共享 AsyncState + 各 feature 错误映射 | 使用真实请求状态，不根据空数组推断库离线或模型就绪；主区保留结果与编辑任务 |
| 手机工具入口、关闭、Escape、目的地切换、焦点恢复 | AppShell | 既有 sidebar slot 的命名扩展。手机关闭抽屉不参与键盘/无障碍导航；查找/侧栏分页不触发目的地关闭 |
| 收藏/标签移出结果后的固定预览 | useMediaPreviewController 的显式参数 | CurationPage 关闭保留范围外固定项，避免取消收藏后仍展示旧项。搜索/浏览原有固定预览跨筛选行为保留 |

## 共享所有权及架构

复用既有 `sidebarContent` / `quickAccessDetail`；`ContextSidebar` 仅拥有分区/字段/列表布局，
不导入 feature、API、Query 或路由。控件复用 Button/Input/Select，颜色、间距和主题来自中央 tokens。
AppShell 唯一拥有抽屉；DirectoryNavigation 唯一拥有库和快速访问。无新部署、持久化、API 或依赖方向，
属于 ADR-0007 内的共享布局模式扩展，不需要新架构决策。

## 与原型的有意差异

- 主区保留已有创建/改名/合并人物及审核操作，侧栏只负责选择和筛选。
- Person 合同没有头像字段，侧栏显示姓名、ID 尾部与服务端媒体数量，不从某张样本推断身份封面。
- 标签仍保留既有“全部/图片/视频”和排序合同，移到侧栏的标签云下面；选云标签默认图片。
  不把已支持的视频标签深链接降级为图片，也不把演示的“管理标签”变成不存在的批量改名功能。
- 原型导览、模拟日期结果和合成数量不进入生产。已接受的侧栏副本标为历史归档，稳定验收以生产
  SearchPage / AIReviewPage / CurationPage Storybook（显式合成合同数据）和回归测试为准。

## 验证

2026-10-25 完成代码与 UI 验证（延续 2026-10-24 的原型接受）：

- `npm ci --prefix web`：通过，Node 24.22.1 / npm 11.19.1；无依赖变更。
- `npm run test --prefix web`：69 文件 / 432 项通过；包含作用域、URL、类型/计数、抽屉与移出结果的预览回归。
- `npm run build --prefix web`（含两个 TypeScript 工程检查）、`build:storybook`：通过；保留现有大 chunk 提示。
- `check:architecture`、`check:visual-references`、`audit:high`：通过。
- `context-sidebar.spec.ts --project context-visual-chromium`：Linux 10 项通过，32 张侧栏基线在不更新模式下复验通过。
  四功能 ×390/768/1440/1920，zh-CN/light 与 en/dark 配对；全部页面 axe WCAG2 A/AA 零问题、无横向溢出。
  另验证媒体筛选不提前关闭手机抽屉、目录范围切回、人物双游标和标签语言不改变 ID/结果。
- `search-guidance.spec.ts --project chromium`：10 项通过（另含320px）；原有搜索模式文案/请求/键盘行为保持。
- 原有 `visual-regression.spec.ts` 浏览顶部/底部/预览与搜索结果4张 Linux 基线随新导航更新，2项通过。
- Linux 浏览器使用本机既有 `foliopath-p3-browser:local`，实际 Node24.22.1 / Playwright1.61.1，
  image ID `sha256:13131a07430ca9e761d91d8967ae23bd2b86b02b3174148d46b33f1d594ad7dd`。
  应用是当前源码构建的生产 SPA，通过127.0.0.1预览；仅为容器访问显式允许host.docker.internal。
- `make fmt-check arch-check generate-check lint`：通过。无 Go 源码变更，使用只读 fmt-check 避免格式化其他在途工作。
- `make test-e2e`：通过；独立临时只读合成媒体、真实 Linux 容器启动/重启、媒体不变校验。
- `make test`、独立 `make test-integration`：未通过既有模型下载集成测试
  `TestAIModelDownloadPersistsProgressAndRejectsCorruption/{success,cancel}`，错误 `insufficient_space`。
  本机数据卷460GiB/可用约34GiB，而 managed model store 要求总容量10%（约46GiB）保留空间；
  测试在下载前被现有安全边界拒绝。未修改磁盘保护规则或清理其他数据；其余 Go 包在该轮通过。

截图与执行产物：`.artifacts/context-sidebar/`；提交基线位于 `web/tests/e2e/context-sidebar.spec.ts-snapshots/`。
正式组件可在本机Storybook的 Search/Mode guidance、Intelligence/Review/People Sidebar、
Curation/Photo tags/Favorites Sidebar 与 Overview 查看，均标明合成数据；服务只监听127.0.0.1:6007。
原型/正式差异已列明；没有部署到既有实例。未重跑真实模型、D04真实接口全矩阵、容量或双架构发行门禁，
这些不是本次前端合成验收通过的推论。
