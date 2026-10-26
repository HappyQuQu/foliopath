# FolioPath 分层工作台：全功能交互原型

状态：历史设计原型；2026-10-24 用户接受搜索/人物/收藏/标签侧栏，此范围已归档，正式实现与稳定 Storybook 为准。
2026-10-25 用户进一步授权对齐整体视觉；本目录完整归档为接受证据，不再维护重复生产页面。
全功能导览/状态选择器及模拟动作仍仅用于审稿，不导入生产，不代表新能力或发行就绪。
Owner：产品与前端设计。目标：合并后 POST-MVP-5 的界面层级探索。
视觉来源：`evidence/source-visual.png`（本聊天最后一张分层工作台稿）。
用户要求：当前布局基本合适，增强层次感与精致度；全部功能都需要可查看。

## 范围

22 个功能页 + 全功能导览，导航均可点击。

| 原型入口 | 现有能力/合同所有者 |
| --- | --- |
| browse / search / viewer / video | FR-BRW-001–015、既有搜索/查看器合同；BrowsePage、SearchPage、MediaViewerPage、MediaCollection |
| people / favorites / tags | 人物工作流、整理与自动标签；AIReviewPage、CurationPage、AssetCurationControls，CR-2026-031/033（本地化标签） |
| libraries / scans / results | 多库配置、扫描代际与安全失败诊断；LibrariesPage、NewLibraryPage、ScanStatusPage |
| ai / jobs / models | FR-INT-009/016、POST-MVP-5；AISettingsPage 与服务端生成客户端，既有三页签信息架构 |
| general / schedule / cache | GeneralSettingsPage、StorageSettingsPage；NFR-COMP-001 与既有资源/缓存设置 |
| account / logs / about | AccountSettingsPage、LogsPage、AboutPage |
| login / setup / unavailable | AuthPage、SystemUnavailablePage |

原型依据 `web/src/routes/paths.ts` 与 `AppRouter.tsx` 盘点。页面状态通过原型工具条切换，
不是新的产品设置。全功能导览本身也是审稿辅助入口，不属于生产导航提案。

## 交互与限制

- 布局切换、缩略图密度、目录筛选、选择预览、收藏、标签增删、人物建立/详情、AI 开关、
  任务示例状态、清理确认、日志筛选、明暗主题、查看器与视频控件可演示。
- 所有计数、日期、模型状态、任务、账号和标签均为合成示例；操作只改变组件内的临时状态。
- 复用原项目已有九张合成人像资源，代替视觉稿的京都素材，以检查相同视觉体系在真实纵横比下的表现。
- 视频为仓库已有 1 秒 320×180 合成媒体；主图列表的视频封面仅是演示图片，文件信息属于合成 DTO，非元数据核验。
- 登录/首次设置仅演示页面跳转，不创建真实账户、不读取或保存凭据；请勿填写真实密码。
- 保存、重试、清理、移动/合并等维护动作使用演示确认，不调用真实服务。模型元数据仅只读展示。
- 语言、排序和详细参数用于视觉评审；未实现完整英文翻译、真实搜索/推理、分页、认证或服务器任务。
- 图片查看器提供适应/缩放/滚动浏览和浏览器全屏入口；1:1 是视觉状态示范，不证明原图像素映射。
- 不读取真实媒体，不连接 6010、不接 SQLite。不替换生产路由、主题或 API 合同。

## 验证

运行 `npm run build`、`npm run test:sites`，以及实际 in-app browser 检查。
证据和已知差异见 `design-qa.md`；没有对生产 Go/React 做改动，未重复运行生产测试套件。

本地预览仅监听 loopback，见 `package.json` 的 dev 脚本；本次服务端口 4187。

## 2026-10-24 功能侧栏迭代

搜索、人物、收藏和标签的下半侧栏按功能切换；示例筛选分别保存于各自状态，范围和主内容联动。
标签云从主区域移入侧栏，主区域显示所选标签的图片；人物明确全局与本库组作用域。
本轮提案与证据见 `docs/changes/PROTOTYPE-2026-10-24-contextual-sidebar.md`（仓库根目录）。
