# FolioPath 前端开发清单

## 当前状态

- 当前阶段：Stage 4
- 静态设计原型：已于 2026-07-24 完成 15 个界面/状态的浅色、深色和响应式验收；
  位于 [`prototypes/foliopath-static-ui`](../prototypes/foliopath-static-ui/README.md)，不放入
  生产 `web/src` import graph
- 当前生产任务：Stage 1 的 `S1-201～207` 已完成并记录
  [认证 Integrated Done Gate](gates/MVP-2026-07-23/s1-auth-integrated-done.md)；
  Stage 2 的真实媒体库管理、扫描状态、通用扫描/缓存设置和完整验收矩阵
  `S2-201～208` 已通过
  [Integrated Done Gate](gates/MVP-2026-07-23/s2-library-scan-integrated-done.md)；
  Stage 3 的目录浏览、缩略图集合、非模态预览和容量矩阵 `S3-101～108` 已通过
  [Integrated Done Gate](gates/MVP-2026-07-23/s3-browse-integrated-done.md)；
  Stage 4 的统一搜索、共享非模态预览、完整查看器及媒体交互矩阵 `S4-004～008`
  已连接真实 API 并完成 Chromium 桌面/移动验收；下一步完成纵向 E2E 与
  `Integrated Done` Gate
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
    下一页错误保留已载入媒体并局部重试。仅存在 pending 缩略图且已载入不超过 4 页时
    按 2.5 秒刷新资产页，超过预算或全为 terminal 状态即停止；failed/unavailable
    不伪造后端未提供的重新生成操作；分页错误存在时停止自动预取，只允许显式重试。证据见
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
- [x] `S3-107` 验证十万媒体规模下 DOM、请求数量、滚动、播放资源释放与焦点恢复预算。
  - 10 万项合成集合在默认测试视口挂载不超过 64 项；真实 1280×720 Chromium 首屏
    挂载 42 项，跳至第 100,000 项后挂载 40 项并恢复其按钮焦点。
  - 资产保持 50 项 cursor 页、接近末端才预取且一个在途请求时禁止重复触发；pending
    缩略图轮询最多覆盖 4 个已载入页，避免大集合周期性重取全部页。
  - 焦点恢复使用虚拟滚动与最多 12 帧有界重试，卸载时取消；视频按资产 key 切换并验证
    旧节点释放、DOM 只保留一个视频。证据见
    [S3-107 前端容量预算](gates/MVP-2026-07-23/s3-frontend-capacity.md)。
- [x] `S3-108` 完成核心浏览/预览 E2E 与 `Integrated Done` Gate。
  - 一次性真实后端链覆盖建库扫描后的目录树、direct/recursive、真实 ready WebP、
    原内容预览、布局记忆、URL 前进后退、移动抽屉、固定/双击/Escape/焦点恢复。
  - 受控契约状态覆盖 skeleton、pending→failed、分页错误保留与显式重试、首屏错误、
    empty 和 offline；浏览页稳定浅/深主题、390/1024/1280px、无横向溢出及 axe
    serious/critical 均通过。证据见
    [Stage 3 浏览与预览 Integrated Done](gates/MVP-2026-07-23/s3-browse-integrated-done.md)。

真实目录/媒体数据集成依赖后端 `S3-007`。

## Stage 4：搜索与媒体查看器界面

- [x] `S4-004` 实现统一 SearchInput、filter、结果列表、URL 状态和无结果恢复。
  - 共享 `SearchInput` 与既有 `MediaCollection` 组合真实游标结果；搜索 feature 的唯一
    URL codec 保存 query、当前库/当前目录（可递归）/全部库范围、类型、日期和排序，
    浏览器刷新与前进/后退可恢复。
  - query owner 根据范围选择库内或全局搜索 operation，固定每页 50 项并沿用缩略图
    pending 的有界刷新；结果显示可访问文件名、库名和 library-relative 来源，不暴露
    宿主路径。
  - loading、无结果、离线和请求失败保持不同恢复动作；组件、URL、adapter、浅/深主题、
    长名称、1024px 响应式、无横向溢出和 axe serious/critical 已通过测试。证据见
    [S4-004 前端搜索界面](gates/MVP-2026-07-23/s4-frontend-search.md)。
- [x] `S4-005` 复用 Stage 3 的共享非模态预览，不得创建 search-only 预览。
  - 浏览与搜索现在共同使用 `useMediaPreviewController`、`MediaCollection` 和
    `MediaPreview`：未固定单击跟随、固定后单击只选择/双击切换、前后项、宽度调整、
    Escape/关闭及虚拟焦点恢复只有一个实现。
  - 搜索筛选或查询更新不会中断固定图片/视频；若媒体暂时离开结果集，预览保持并明确
    标示“固定预览不在当前结果中”，返回结果后仍可关闭并恢复卡片焦点。证据见
    [S4-005 搜索复用非模态预览](gates/MVP-2026-07-23/s4-frontend-search-preview.md)。
- [x] `S4-006` 实现可直达完整查看器：适应、缩放/平移、1:1、前后项、信息、全屏与关闭恢复。
  - 浏览与搜索的共享预览提供显式“进入完整查看器”，规范路由为
    `/libraries/:libraryId/media/:assetId`；`from` URL 只接受同源搜索或同库浏览/搜索路径，
    直接访问或不可信返回值安全回退到媒体库根浏览页。
  - 共享 `MediaViewer` chrome 支持图片适应、1:1、0.25～4 倍缩放、指针拖动平移、
    原生视频控件、基本信息开关、Fullscreen API、前后项按钮及方向键；Escape 退出，
    关闭后恢复来源 URL、虚拟滚动锚点和媒体卡片焦点。
  - 当前已载入结果的稳定 ID/媒体库 ID 序列仅作为路由瞬时状态，不复制完整资产或服务端
    事实；刷新后当前媒体仍可直达，无法重建的任意列表序列会明确禁用前后项。
  - `MediaViewer` 已进入组件工作台并通过组件、Query、URL 安全和页面测试；真实隔离后端
    在 1440×900 与原型同屏对照，并验证 390×844 底部信息面板。证据见
    [S4-006 完整媒体查看器](gates/MVP-2026-07-23/s4-frontend-media-viewer.md)。
- [x] `S4-007` 实现 GIF 策略、原生视频/Range、封面及不可播放、损坏、离线、已删除状态。
  - GIF 使用认证原内容 `<img>` 保留浏览器动画；视频使用原生 controls/playsInline/
    metadata preload，同源 ready thumbnail 作为 poster，Range 仍由浏览器与现有 content
    契约协商，不新增转码。
  - 唯一 `mediaAvailability` owner 按 source offline/missing/unreadable → probe
    failed/unsupported → video unsupported codec 的优先级映射；浏览、搜索预览和完整查看器
    复用同一状态卡，不复制错误判断。
  - 离线、缺失、不可读、损坏、不支持、codec、详情已删除和运行时读取失败均保留关闭、
    有界前后项与可靠索引信息；可恢复状态提供重新检查，不自动跳项、不暴露后端细节、不修改
    原文件。证据见
    [S4-007 媒体播放与降级状态](gates/MVP-2026-07-23/s4-frontend-media-strategy.md)。
- [x] `S4-008` 验证目标桌面/移动浏览器、键盘、触摸、Range、焦点与错误降级。
  - Desktop Chrome 与 Pixel 5 触摸仿真覆盖关闭按钮初始焦点、`I` 信息开关、方向键、
    Escape、触摸信息开关、离线重试及关闭/前后项可达性；页面无横向溢出且 axe
    serious/critical 为零。
  - 浏览器加载真实合成 MP4，并实际观察 `Range: bytes=…` 和 `206 Content-Range`；
    unsupported codec、offline、deleted 状态分别固定无播放器、有效重试与无效重试规则。
  - 工具条按钮获得焦点时仍允许查看器级快捷键；焦点位于原生视频、表单输入或可编辑内容时
    不抢占冲突按键。证据见
    [S4-008 媒体交互矩阵](gates/MVP-2026-07-23/s4-frontend-media-matrix.md)。
- [ ] `S4-009` 完成搜索/预览/查看器 E2E 与 `Integrated Done` Gate。

搜索真实数据依赖后端 `S4-003`；查看器真实内容依赖 `S4-005B`。

## 前端完成任务时

1. 更新组件工作台、UI 规范和必要的用户流程文档。
2. 运行 typecheck、组件、axe、键盘、响应式、浏览器 E2E 和适用视觉回归。
3. 复用唯一 token、共享组件、Query key、URL codec 和错误映射所有者。
4. 不修改生成类型，不直接 `fetch`，不在 feature 内复制按钮、弹窗、列表或状态模式。
5. 在 PR 中列出已对照的原型界面编号、主题/语言/断点矩阵、契约 Gate 和视觉回归证据。
