# FolioPath 前端开发清单

## 当前状态

- 当前阶段：Stage 3
- 静态设计原型：已于 2026-07-24 完成 15 个界面/状态的浅色、深色和响应式验收；
  位于 [`prototypes/foliopath-static-ui`](../prototypes/foliopath-static-ui/README.md)，不放入
  生产 `web/src` import graph
- 当前生产任务：Stage 1 的 `S1-201～207` 已完成并记录
  [认证 Integrated Done Gate](gates/MVP-2026-07-23/s1-auth-integrated-done.md)；
  Stage 2 的真实媒体库管理、扫描状态、通用扫描/缓存设置和完整验收矩阵
  `S2-201～208` 已通过
  [Integrated Done Gate](gates/MVP-2026-07-23/s2-library-scan-integrated-done.md)；
  下一步进入 Stage 3 的目录浏览与缩略图界面
- 代码所有权：`web/`、前端组件/浏览器测试；不得修改后端业务实现

前端与后端可以由不同任务并行开发。应用壳、token、共享原语和契约 fixture 可独立推进；
连接真实业务数据、实现业务提交行为前，对应后端必须达到 `Backend Ready`。

## 固定交接规则

| 交接点 | 后端提供 | 前端可以开始 |
| --- | --- | --- |
| Contract Ready | 已评审 OpenAPI、错误码、状态和 fixture | API adapter、静态状态和组件行为 |
| Backend Ready | 真实 handler、集成/故障测试和可启动服务 | 连接真实 API、提交行为和业务页面 |
| Integrated Done | 稳定测试环境与可观察状态 | 联调、浏览器 E2E 和最终交互验收 |

前端不得从 mock 反向发明 API。mock/Story 只能使用已评审契约 fixture；生产代码只通过统一
`web/src/lib/api` adapter 访问服务。

## 已确认的界面与交互基线

生产实现必须逐项对照[原型界面清单](../prototypes/foliopath-static-ui/screen-inventory.md)，而不是
只实现主浏览页：

| 组 | 必须交付的界面 / 状态 | 生产阶段 |
| --- | --- | --- |
| 启动与认证 | 首次管理员设置、登录、会话失效、启动/数据目录不可用 | Stage 1 / Stage 5 |
| 媒体库 | 无媒体库欢迎、新建三步向导、媒体库列表、重命名、移除确认 | Stage 2 |
| 扫描 | 进行中/取消、离线保留索引、失败/部分不可读与重试 | Stage 2 |
| 浏览 | 当前目录、递归模式、目录树/移动抽屉、扫描横幅、空/加载/故障 | Stage 3 |
| 搜索 | 三种范围、类型/日期、结果、无结果、离线与请求失败 | Stage 4 |
| 媒体 | 桌面非模态预览、窄屏入流预览、固定/双击切换、完整查看器与降级状态 | Stage 3 / Stage 4 |
| 设置 | 主题、语言、扫描周期、缓存配额、退出登录 | Stage 1 / Stage 2 / Stage 5 |

视觉与行为规则以[界面设计规范](ui-design.md)和
[CR-2026-001](changes/CR-2026-001-non-modal-media-preview.md)为准：

- 浅色、深色和跟随系统使用同一 token 来源，不得创建第二套主题机制。
- 桌面固定导航，移动目录抽屉；所有页面在 390px 宽度不得产生页面级横向滚动。
- 默认媒体点击打开不遮挡父列表的非模态预览；未固定单击切换，固定后单击只选择、双击切换。
- 完整查看器是显式操作与可直达路由，不替代父页面中的快速预览。
- 原型中的数据和操作只用于设计验收；生产状态、错误码和提交行为必须来自已评审契约。

## 共享所有权先行

以下能力必须先在 `shared` 建立唯一 owner，并在组件工作台固定 API、状态和可访问性，再由 feature
组合；不得从原型复制成多份局部实现：

- `Button`、`IconButton`、`Field`、`Select`、`Dialog`、`Drawer/Sheet`、`Menu`、`Banner`、
  `Toast`、`Progress`、`Empty/Error/OfflineState`。
- `AppShell`、`ThemeProvider`、`LocaleProvider`、焦点恢复与 reduced-motion。
- `MediaCard`、虚拟集合控制器、`MediaPreview`、媒体播放资源生命周期和 `MediaViewer` chrome。
- Query-key factory、URL codec、API error mapper、认证失效处理、invalidations 和生成 client adapter。

每个共享 owner 的第一位消费者之前完成行为/axe/键盘测试；第二个 feature 消费前完成聚焦视觉回归。

## Stage 1：前端基础与认证界面

- [x] `S1-201` 建立 React/Vite 应用壳、路由、Provider 和全局错误边界。
  - 可在认证 Backend Ready 前完成壳层，但不得连接虚构认证接口或实现假登录成功。
  - 路由至少预留 setup、login、browse、search、media、libraries、general settings 与系统错误页。
  - 已完成应用启动、Query/主题/Toast Provider、全局安全错误边界和 setup、login、
    general settings、system unavailable 产品路由；后续业务 URL 已由唯一 paths owner
    预留。
- [x] `S1-202` 建立唯一 token、主题、排版、焦点和 reduced-motion 基础。
  - 覆盖浅色、深色、跟随系统、持久化覆盖和首屏无闪烁。
- [x] `S1-203` 建立唯一 Button、Input、Dialog、Toast、FormField 等共享原语及组件工作台。
- [x] `S1-204` 依据已通过的 `S1-106` 实现首次管理员设置、登录、会话失效、退出和安全返回 UI，
  只通过领域 API adapter。
  - 已使用真实后端验证“首次创建管理员 → 退出 → 再次登录 → 通用设置”；错误映射、
    CSRF 退出、路由守卫和会话过期安全提示均通过统一 auth adapter/query owner。
- [x] `S1-205` 实现启动/数据目录不可用页和全局安全错误边界；错误不得显示宿主路径、堆栈或原始响应。
  - 应用启动先消费公开 `/health/ready` 契约；四种安全 reason code 与网络故障进入
    阻断恢复页，页面不显示宿主路径、`/app/data`、SQLite、堆栈或原始响应。
- [x] `S1-206` 加入组件测试、axe、键盘、主题、语言和 390/768/1024/1440px 响应式验证。
  - 组件工作台与测试覆盖新增状态、LocaleProvider 和语言选择器；真实 Chromium
    E2E 覆盖简体中文/英文、浅/深主题、axe serious/critical、键盘语义和四档宽度，
    所有断点无页面级横向溢出。
- [x] `S1-207` 完成认证纵向切片 E2E，并记录 `Integrated Done` Gate。
  - `make test-web-e2e` 使用一次性真实后端、空合成媒体根和 Playwright Chromium
    验证 setup → settings → logout → expired return → login，并验证安全 readiness
    fixture；CI 有独立 Authentication browser E2E job。

## Stage 2：媒体库与扫描界面

- [x] `S2-201` 实现设置中的媒体库列表、状态、空态和错误态。
  - 通过分页 Query owner 消费真实媒体库列表，覆盖 pending、scanning、ready、offline、
    error，并保留安全重试状态；列表不会把请求故障表现成空库。
- [x] `S2-202` 实现无媒体库欢迎页及只读媒体、`/library` 挂载的次级帮助。
  - 登录后真实空实例进入已确认欢迎页；部署帮助只显示容器挂载示例，不显示宿主实际路径。
- [x] `S2-203` 实现“命名 → 选择服务端批准目录 → 确认 → 创建并自动扫描”三步流程。
  - 禁用并解释重叠、不可读、符号链接/越界目录；提交时重新验证。
  - 路径选择器只消费 `/api/v1/library-paths`，显示服务端阻断原因；创建携带 CSRF
    和稳定幂等键。真实后端已验证选择空合成 `/library` 根、创建、首次扫描入队和返回列表。
- [x] `S2-204` 实现改名、移除确认、离线重试和扫描状态/取消。
  - 移除确认必须明确“只移除配置/索引/任务/缓存，不删除原始媒体”。
  - 改名和移除先读取最新 ETag；移除使用稳定幂等键并轮询异步结果。确认对话框逐项列出
    配置、索引、任务和可重建缓存，同时明确原始目录与媒体保持不变。
- [x] `S2-205` 实现扫描进行中、取消中、离线保留旧索引、部分不可读/失败与重新扫描状态。
  - 独立状态路由展示阶段、进度、目录/媒体/问题计数、起止时间和脱敏问题摘要；运行中允许
    协作取消，终止、离线、失败和取消均明确保留最后可靠索引。
- [x] `S2-206` 实现扫描周期与缓存配额设置，并映射契约校验错误。
  - 通用设置通过 ETag 更新真实定时扫描开关/周期与缩略图缓存配额；覆盖客户端范围校验、
    `settings_invalid` 和并发修改错误，不建立第二套设置状态。
- [x] `S2-207` 验证键盘、移动端、长名称/路径、中英文、加载、重复提交和故障恢复。
  - 真实 Chromium 覆盖 128 字符库名、两层长路径、390/768/1024/1440px、中英文、
    加载/列表故障重试、扫描运行/取消/失败/离线、脱敏 issue 与焦点恢复；页面无横向溢出，
    axe serious/critical 为零。
  - 创建、改名和设置保存的快速双击均只产生一次请求；共享同步提交 guard 同时保护移除、
    手动扫描和取消操作，并有组件级回归测试。
- [x] `S2-208` 完成媒体库/扫描纵向切片 E2E 与 `Integrated Done` Gate。
  - 一次性真实后端完成 setup → 长路径建库 → 改名 → 扫描/设置 → logout/login →
    异步移除；契约状态 fixture 补足不可稳定制造的运行/取消/部分不可读/离线 UI 矩阵。
    证据见
    [Stage 2 Integrated Done](gates/MVP-2026-07-23/s2-library-scan-integrated-done.md)。

`S2-007` 已通过，允许前端通过生成 client 建立媒体库管理 adapter；依赖扫描实际执行、
扫描 `S2-107` Backend Ready 已通过，进度、取消和计划设置可连接生成 client；浏览与
缩略图流程仍须等待 Stage 3 对应 Backend Ready。纯展示组件和冻结契约状态可先在组件工作台
实现。

## Stage 3：文件夹浏览与缩略图界面

- [x] `S3-101` 实现桌面侧栏、移动抽屉、目录树、面包屑和可复制直达 URL。
  - 生产 `browse` feature 通过生成 client 的 `catalog` adapter 消费真实目录页与
    directory detail；根/深层目录 URL 可刷新恢复，树按展开路径延迟分页，移动端复用
    AppShell 抽屉的 Escape/焦点恢复。证据见
    [S3-101 前端目录导航](gates/MVP-2026-07-23/s3-frontend-directory-navigation.md)。
- [x] `S3-102` 实现当前目录/递归模式及对应稳定 URL 状态。
  - `recursive=1`、显式 `sort/order` 由单一 URL codec 规范化；直接/递归模式使用
    冻结契约的不同默认排序并重置 cursor。真实有界资产摘要显示来源路径，来源链接返回
    所属目录并关闭递归，浏览器返回/刷新可恢复。证据见
    [S3-102 前端浏览范围](gates/MVP-2026-07-23/s3-frontend-browse-scope.md)。
- [x] `S3-103` 实现默认自适应网格、可记忆瀑布流和统一虚拟化集合。
  - 共享 `MediaCollection`/`MediaCard` 使用 TanStack Virtual 的同一 lanes controller，
    以索引宽高预留 grid/masonry 空间并只挂载可视窗口；布局偏好写入统一
    `foliopath.preferences.v1`，不污染可复制浏览 URL。真实 libvips 浏览器链验证
    ready WebP 缩略图、切换/刷新记忆、主题、响应式与 axe。证据见
    [S3-103 前端媒体集合](gates/MVP-2026-07-23/s3-frontend-media-collection.md)。
- [x] `S3-104` 实现 skeleton、空、错误、离线、缩略图 pending/failed 状态。
  - 首屏使用与最终 grid 同比例/密度的共享 skeleton；空目录复用共享 EmptyState，
    离线且保留索引没有可显示媒体时使用专属 OfflineState，不把离线误报为空。首屏错误可重试，
    下一页错误保留已载入媒体并局部重试。仅存在 pending 缩略图时按 2.5 秒刷新资产页，
    terminal 状态停止轮询；failed/unavailable 不伪造后端未提供的重新生成操作。证据见
    [S3-104 前端浏览状态](gates/MVP-2026-07-23/s3-frontend-browse-states.md)。
- [x] `S3-105` 实现共享 `MediaPreview` 的图片/视频/基本信息/前后项/关闭和宽度调整。
  - 共享组件直接消费 feature 提供的同源 content URL，不导入 API client；图片使用
    contain 预览，视频使用原生 `controls`/`playsInline`/metadata preload。面板显示
    library-relative 位置、类型、修改时间、尺寸、大小和可选时长，并提供有界前后项。
    桌面宽度可指针拖动或用方向键/Home/End 调整，窄屏回到内容流；当前 S3-105 仅实现
    未固定时单击打开/切换，固定、双击、Escape 和精确焦点/虚拟锚点恢复仍归 S3-106。
    证据见 [S3-105 前端媒体预览](gates/MVP-2026-07-23/s3-frontend-media-preview.md)。
- [x] `S3-106` 实现 [CR-2026-001](changes/CR-2026-001-non-modal-media-preview.md)：
  - 桌面右侧停靠且父列表继续可用；窄屏进入内容流。
  - 未固定单击切换；固定后单击只选择、双击切换；任一时刻只允许一个活动媒体。
  - 关闭后恢复触发项、虚拟列表锚点与焦点。
  - `MediaCollection` 分开表达“已选择”和“正在预览”，固定状态下更新可访问操作文案；
    `MediaPreview` 提供固定/取消固定、状态说明和 Escape。切换视频以 keyed media
    节点卸载旧资源；关闭时虚拟控制器滚回当前预览项并恢复其语义按钮焦点。证据见
    [S3-106 前端固定预览交互](gates/MVP-2026-07-23/s3-frontend-pinned-preview.md)。
- [ ] `S3-107` 验证十万媒体规模下 DOM、请求数量、滚动、播放资源释放与焦点恢复预算。
- [ ] `S3-108` 完成核心浏览/预览 E2E 与 `Integrated Done` Gate。

真实目录/媒体数据集成依赖后端 `S3-007`。

## Stage 4：搜索与媒体查看器界面

- [ ] `S4-004` 实现统一 SearchInput、filter、结果列表、URL 状态和无结果恢复。
- [ ] `S4-005` 复用 Stage 3 的共享非模态预览，不得创建 search-only 预览。
- [ ] `S4-006` 实现可直达完整查看器：适应、缩放/平移、1:1、前后项、信息、全屏与关闭恢复。
- [ ] `S4-007` 实现 GIF 策略、原生视频/Range、封面及不可播放、损坏、离线、已删除状态。
- [ ] `S4-008` 验证目标桌面/移动浏览器、键盘、触摸、Range、焦点与错误降级。
- [ ] `S4-009` 完成搜索/预览/查看器 E2E 与 `Integrated Done` Gate。

搜索真实数据依赖后端 `S4-003`；查看器真实内容依赖 `S4-005B`。

## 前端完成任务时

1. 更新组件工作台、UI 规范和必要的用户流程文档。
2. 运行 typecheck、组件、axe、键盘、响应式、浏览器 E2E 和适用视觉回归。
3. 复用唯一 token、共享组件、Query key、URL codec 和错误映射所有者。
4. 不修改生成类型，不直接 `fetch`，不在 feature 内复制按钮、弹窗、列表或状态模式。
5. 在 PR 中列出已对照的原型界面编号、主题/语言/断点矩阵、契约 Gate 和视觉回归证据。
