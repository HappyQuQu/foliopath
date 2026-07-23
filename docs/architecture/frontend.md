# FolioPath 前端子系统目标架构

## 状态

本文定义 FolioPath MVP 前端的**目标架构与实施门槛**，用于约束后续
`web/` 脚手架和业务实现。当前仓库尚无 React 应用、Node 工具链、
`api/openapi.yaml`、Storybook、前端测试或前端 CI；下文出现的目录、工具和
门禁均不得描述为已经落地。

本文不改变已确认的产品范围和 UI 行为。信息架构、响应式、可访问性、状态与
动效以 [`docs/ui-design.md`](../ui-design.md) 为准；运行拓扑和技术栈以
[`docs/architecture.md`](../architecture.md) 及已接受 ADR 为准；公开接口在
`api/openapi.yaml` 已是唯一 HTTP 结构化契约；前端实现不得从原型或手写类型另造行为。

本文落实[ADR-0007：单一共享前端设计系统](../adr/0007-shared-frontend-system.md)，
业务前端的开始顺序受[ADR-0006：契约驱动、切片内后端优先](../adr/0006-contract-driven-backend-first.md)
和[交付治理](delivery-governance.md)约束。

## 架构目标

- 让页面组装、业务能力、共享 UI、网络边界和视觉 token 各有唯一所有者。
- 相同语义和行为只实现一次，避免多个按钮、对话框、列表、错误映射或分页
  hook 随页面分别演化。
- 通过依赖方向和静态检查防止业务逻辑进入路由、共享组件或生成代码。
- 让加载、空、错误、离线、权限、取消和恢复状态与正常状态同等可测试。
- 对 10 万媒体目标档使用有界请求、游标分页和虚拟化，不能用增加客户端内存
  消耗换取实现简单。
- 将设计与工程约束纳入仓库现有 ADR、OpenAPI、UI 规范和测试门禁治理。

## 逻辑层次与允许依赖

目标源码仍使用既定顶层目录：

```text
web/src/
├── app/          应用启动、Provider、全局错误边界和路由注册
├── routes/       URL 解析、路由加载和页面级组合
├── features/     按用户能力组织的业务 UI、query、hook、类型和测试
├── components/   无单一业务归属的共享 UI 原语和通用交互模式
├── lib/          API、i18n、持久偏好等小型基础设施适配
└── styles/       token、主题、reset 和极少量全局样式
```

允许的生产依赖方向为：

```text
app ───────► routes ───────► feature public API
 │               │                    │
 │               └────────► components│
 │                                    ├────────► components
 ├────────► lib providers             └────────► lib/api
 └────────► styles

components ─────► styles / 明确允许的纯基础设施
lib/api ────────► lib/api/generated
```

具体规则：

- `app/` 只负责启动、Provider、全局边界和路由表，不拥有媒体库、搜索或查看器
  业务规则。
- `routes/` 读取和写入 URL、调用 feature 暴露的页面/query 接口并组合页面；
  不直接调用 `fetch`、生成客户端或编写缓存更新逻辑。
- 每个 `features/<name>/` 拥有该能力的组件、query key、query/mutation、hook、
  业务视图模型和测试，并通过一个明确的公开入口暴露最小接口。
- feature 不得导入另一个 feature 的内部文件。确有跨 feature 复用时，要么形成
  一个单向、记录清楚的公开依赖，要么将无业务归属的部分提升到共享层；不得用
  复制代码或笼统 `common` 目录回避所有权决定。
- `components/` 不得依赖 `features/`、`routes/`、`app/`、TanStack Query 或
  `lib/api`，也不得认识 Library、Asset、Scan 等服务端 DTO。
- `lib/api/generated/` 只包含生成文件；除 `lib/api` 的手写 adapter 外，生产
  代码不得直接导入它。
- `styles/` 是叶子层，不导入业务代码。测试支持代码可以依赖被测对象，不能
  反向成为生产依赖。
- 禁止循环依赖。任何 lint 例外必须有原因、owner、清理条件和期限，不能永久
  使用目录级全局豁免。

## 设计系统与唯一所有权

### 三种组件所有权

1. **UI 原语**由 `components/` 唯一拥有，例如 Button、IconButton、Field、
   Dialog、Sheet、Menu、Tooltip、Banner、Toast、Spinner、Skeleton 和焦点管理。
2. **通用交互模式**由 `components/` 中明确命名的模式模块唯一拥有，例如确认
   流程、异步内容状态、分页集合壳、虚拟列表壳和响应式面板。它们不包含业务
   DTO 或业务判断。
3. **业务组件**由对应 feature 唯一拥有，例如目录树、媒体卡片、媒体库状态、
   扫描进度和查看器。其他页面需要相同行为时复用其公开接口，或经过评审后
   提升无业务部分；不得重新实现一个近似版本。

不为了可能的未来复用预建业务抽象。Button、Field、Dialog、Sheet、状态壳和集合
基础设施等已声明的基础语义，从第一次使用起就进入共享系统；“第二个消费者再提升”
只适用于业务专用组合。业务组合第一次实现先归属明确 feature，一旦第二个消费者
需要相同语义，必须在复制前决定规范所有者。共享并不等于允许无限配置；组件 API
应表达稳定语义，而不是把整套样式和内部状态全部开放为参数。

### 禁止重复实现的基线

- 产品操作统一使用规范 Button/IconButton。原生 `<button>` 只允许出现在原语
  实现内部，或经说明和测试证明无法由规范控件表达的特殊复合控件中。
- Modal、Dialog、Sheet 和确认流程必须建立在同一套可访问性基础上；不得在
  feature 中各自实现 focus trap、Escape、背景 `inert`、滚动锁定和焦点恢复。
- Field、Label、帮助文本、校验错误和必填语义必须使用统一表单模式；不得让
  页面分别拼装不可一致读屏的表单结构。
- 加载、空、请求错误、离线、部分失败和无权限状态使用规范状态组件。状态文案
  与可执行恢复动作由 feature 提供，布局、公告和焦点行为不重复实现。
- Toast 只表示短暂反馈；持续离线、扫描和部分失败统一使用 Banner/InlineStatus，
  不允许同一状态在不同页面任意选择表现。
- 分页、重试、取消过期请求、去重、滚动锚点和虚拟焦点恢复使用规范集合模式；
  不允许每个列表各写一套 `IntersectionObserver`、cursor 合并或重试状态机。
- 日期、文件大小、时长、相对路径、错误码和国际化消息使用唯一 formatter 或
  adapter；不得在组件中散落格式化实现。
- 项目选择图标来源后只使用一个规范图标集合；无文字 IconButton 必须由组件
  API 强制要求可访问名称。

在选择原生控件或第三方无样式原语库时，只能确定一套基础方案。引入第二套
Dialog、Menu、Popover 或 Toast 基础库属于架构变化，必须先评审其体积、
可访问性、样式隔离和迁移成本。

## Token、主题与 CSS 策略

目标实现采用 **CSS custom properties 作为运行时 token 单一来源，CSS Modules
作为组件样式边界**：

- `styles/` 定义语义颜色、字体、间距、尺寸、圆角、阴影、层级、断点和动效
  token；浅色、深色、高对比和 reduced-motion 只覆盖语义 token 或能力开关。
- 组件引用 `--color-surface`、`--space-3`、`--motion-fast` 等语义 token，不能
  依赖某主题的具体颜色名称，也不能复制 `docs/ui-design.md` 中的原始数值。
- 除 token 定义文件、媒体本身的固有颜色或经评审的测试 fixture 外，组件样式
  禁止新增十六进制、RGB/HSL 颜色字面量。
- 间距、圆角、阴影、字号、z-index 和动效时长必须来自相应阶梯。仅与内容几何
  直接相关的动态值，例如虚拟列表位移、媒体宽高比和缩放矩阵，可以使用内联
  style；内联 style 不得承载主题或产品视觉常量。
- 全局 CSS 仅包含 reset、字体基线、token、主题和必要的可访问性基础规则。
  业务页面不得用全局选择器覆盖其他 feature，也不得用 `!important` 解决所有权
  或层叠问题。
- 不使用 CSS multi-column 实现瀑布流；不改写浏览器滚动物理；普通页面不全局
  使用 `backdrop-filter`。动效必须响应 reduced-motion/reduced-transparency。
- 主题切换通过应用根元素属性和 token 覆盖完成，组件内部不得判断主题后选择
  两套 JSX。

Token 文档值当前仍是原型起点。首轮可点击原型验证后，代码 token 和
`docs/ui-design.md` 必须在同一变更中同步；不能让实现成为唯一规格。

## API 生成层与手写 Adapter

API 调用分为三层：

```text
api/openapi.yaml
        │ deterministic generation
        ▼
lib/api/generated/       生成的 DTO 与请求函数，禁止手改
        │
        ▼
lib/api/                 transport、认证/CSRF、错误与 DTO adapter
        │
        ▼
features/*/queries       query key、query/mutation 和缓存策略
```

- `api/openapi.yaml` 是实现期唯一结构化契约；生成必须确定性，并由
  `make generate-check` 检测漂移。
- generated 层不包含产品文案、React、Query 或手写业务逻辑。修改行为必须先改
  OpenAPI 或生成配置，再重新生成。
- `lib/api` 是唯一网络出口，统一同源 base URL、安全 Cookie、CSRF、请求 ID、
  `AbortSignal`、超时策略、JSON 解码和响应错误处理。组件、route 和 feature
  不得直接调用 `fetch` 或再引入第二套 HTTP client。
- `lib/api` adapter 只拥有 transport、wire normalization、统一错误和生成类型隔离，
  不拥有页面视图模型。每个 feature 拥有自己的业务/展示模型；真正出现第二个消费者
  的领域映射必须在复制前指定一个公开 owner。视图不能依赖生成器特有类型或把服务端
  `message` 当作分支条件。
- Storybook、单元测试和开发 mock 使用与 OpenAPI 相同的类型或契约 fixture，
  不维护一套独立、可能漂移的“前端接口”。

## Query、URL、偏好与错误所有权

### TanStack Query

- 服务端状态只由 TanStack Query 管理。每个资源的 query key factory 只有一个
  feature owner；组件不得手写重复 key 数组。
- query/mutation 定义负责调用 API adapter、取消过期请求、分页去重、重试边界、
  乐观更新和成功后的失效策略。展示组件只消费明确的状态和动作。
- 同一资源不得由多个 feature 使用不同缓存形状重复请求。需要不同展示形状时，
  由 adapter 或 `select` 派生，不创建第二份事实来源。
- 默认不对永久错误、校验错误或认证错误自动重试；暂时错误的退避规则在一个
  Query client 策略入口配置，feature 只声明必要例外。

### URL 与本地状态

- 媒体库、目录、搜索范围、递归、筛选、排序、查看器来源上下文以及可恢复的
  列表锚点属于 URL 状态，由 route 的类型化 codec 负责解析、默认值、规范化和
  序列化。组件不得各自操作 `URLSearchParams`。
- 改变查询语义时必须丢弃旧 cursor；浏览器前进/后退和复制 URL 要能恢复同一
  可导航状态。
- 短暂展开、悬停、输入草稿等 UI 状态留在最近组件。不得把服务端状态复制到
  Context，也不得未经 ADR 引入 Redux 或其他全局状态库。
- 需要浏览器持久化的展示偏好通过一个带版本的 `lib/storage` adapter 管理；
  feature 不直接读写 `localStorage`。实例级设置仍以服务端为事实来源。

### 错误

- `lib/api` 将统一 API error envelope 映射为稳定 `AppError`，只保留错误码、
  本地化 message 与 request ID；MVP 不接受任意 details，不保留 SQL、绝对路径或原始 stderr。
- feature 根据稳定错误码决定本地化文案和恢复动作；不得根据可变的服务端
  `message` 字符串分支。
- app 拥有未捕获异常边界，route 拥有页面加载失败边界，feature 拥有可恢复业务
  错误，单个媒体卡片只拥有局部派生资源错误。不得把所有错误统一降级成 toast。

## 大列表与集合模式

媒体网格、瀑布流、搜索结果和大目录树共享以下不可变规则：

- API 使用稳定、不透明 cursor；前端使用 TanStack Query 的有界 infinite query，
  禁止 `OFFSET` 假设、禁止一次请求全部数据。
- TanStack Virtual 只渲染可视窗口和有限 overscan。不得为每个未显示条目保留
  observer、图片解码或动画任务。
- 自适应网格与瀑布流共享同一查询、item identity、选择状态和焦点模型，只替换
  布局算法，不能维护两份列表实现。
- DOM 顺序始终等于查询排序；每项使用稳定媒体 ID，不能用数组下标作为 key。
- 规范集合 controller 负责页面合并、ID 去重、过期 cursor、加载更多语义节点、
  内联重试、滚动锚点和从查看器返回后的焦点恢复。
- 布局依赖索引宽高比预留空间；图片加载不能改变顺序。预取、相邻项加载、
  overscan 和解码并发必须有集中、可测的上限。
- 翻页失败保留已经加载的可靠项目；取消或新查询不得把旧请求结果写入新范围。

规范集合模式应先通过合成大列表验证，再供业务页面使用；不得在单个页面完成后
把偶然实现直接命名为通用组件。

## Storybook、测试与视觉回归

### Storybook

Storybook 在首个共享 UI 原语落地时建立，不为当前空仓库预建。以下目标实现后
必须有 story：

- 所有共享 UI 原语和通用交互模式；
- 被两个以上页面消费的业务组件公开状态；
- 加载、空、错误、离线、部分失败、禁用、长文本和焦点状态；
- 简体中文/英文、浅色/深色、窄屏/宽屏以及 reduced-motion 回退。

Story 使用契约 fixture，不访问真实 `/library` 或线上服务。Storybook 构建失败、
控制台错误或可访问性严重错误必须阻断合并。

### 自动测试

- 使用 TypeScript strict mode 和无输出 typecheck。
- 使用行为导向的单元/组件测试验证 query、URL codec、formatter、键盘、焦点恢复、
  状态公告、分页去重和取消；不以大量内部 DOM 快照代替行为断言。
- 共享组件和关键页面运行自动可访问性检查，并保留键盘人工检查清单。
- 浏览器 E2E 放在 `tests/e2e`，覆盖首次设置、登录、建库、扫描、浏览/搜索、
  查看器、离线恢复、删除确认、响应式布局和中英切换。
- E2E 使用合成 fixture、可观察状态和明确超时，不用固定长时间 sleep，也不接触
  开发者真实媒体。

### 视觉回归

视觉回归只覆盖稳定且高价值的 Storybook 场景和应用壳，不对随机媒体内容或所有
动态帧做脆弱全页截图：

- Button、Field、Dialog/Sheet、Banner、Toast、集合状态和媒体卡片；
- 浅/深主题，中文/英文，移动/桌面关键断点；
- 加载、空、错误、离线和焦点可见状态。

截图固定字体、浏览器、视口、时区、语言、数据、动画和网络响应。基线变更必须由
理解 UI 规范的 reviewer 审核，不能用批量重录掩盖非预期变化。像素阈值、目标
浏览器和基线存储方式在工具 spike 后写入测试策略；在此之前不能声称视觉回归已
建立。

## 静态检查与合并门禁

前端脚手架完成后，根级 Makefile/CI 至少统一提供以下能力；具体命令名可在首次
工具链评审时固定，但不能只存在于个人脚本：

- 格式检查；
- TypeScript strict typecheck；
- ESLint：React hooks、未处理 Promise、无障碍基础规则、禁止直接 `fetch`、
  禁止跨层/跨 feature 内部 import、禁止循环依赖和禁止直接导入 generated；
- Stylelint 或等价检查：禁止组件 raw color、非法全局选择器和未批准 token；
- 所有权位置检查：原生交互/无样式基础库只允许规范 `components/ui`，TanStack
  Virtual 和集合 observer 只允许规范 collection pattern，TanStack Query hook/key
  只允许 app provider 或 feature query owner，`fetch`/generated client 只允许
  `lib/api`，语义 token 声明只允许 `styles/`；
- 前端依赖 allowlist：新增 HTTP client、UI primitive、状态或样式基础库必须触发
  架构评审，不能悄悄形成第二套系统；
- OpenAPI 生成一致性检查；
- 单元/组件测试和自动可访问性检查；
- Storybook 静态构建；
- 受影响视觉回归和关键 E2E。

门禁分阶段启用：首个业务 feature 先强制 strict、依赖/token lint、被消费原语的
组件/axe 测试和 Storybook 构建；原语 API 稳定或出现第二个消费者前加入其聚焦视觉
基线；完整主题、语言和断点矩阵最迟在 Release Candidate 前阻断发布。

目标合并入口应被根级门禁覆盖：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

其中 `fmt`、`lint` 和 `test` 必须同时覆盖 Go 与前端，不得保留一个只检查 Go 却
被描述为全仓门禁的同名目标。CI、前端工具链或目标不存在时必须明确报告未执行，
不能用现有 Go 测试代替。

## Backend Ready：开始业务前端前的条件

仅限已通过 S0、进入 S1 的当前切片直接需要的最小应用壳、token 和共享原语可以
按明确时间盒在 Storybook 中先行验证；可丢弃原型不进入生产 import graph，也不为
未来 feature 预建组件。任何连接真实业务数据的 feature 开始前，其纵向切片必须
满足以下 Backend Ready 条件：

1. 对应产品需求、正常/空/错误/离线/权限状态和验收流程已经确认。
   Gate 记录同时指定 MediaCard 等业务组件、Asset/Library/Scan query key 和允许的
   单向 feature 依赖 owner，不能在开发过程中临时争夺所有权。
2. `api/openapi.yaml` 已包含该切片的请求、响应、统一错误码、认证/CSRF、分页和
   取消语义，并通过评审；不得从临时 handler 或前端 mock 反推契约。
3. 生成客户端可确定性生成，`generate-check` 通过，手写 adapter 的输入输出与
   error mapping 已测试。
4. 存在可启动的本地 Go HTTP 应用，按同源方式提供 API 和 SPA；初始化、会话、
   CSRF、健康状态及该切片端点具备契约测试。
5. 后端使用临时目录、真实测试 SQLite 和合成媒体提供可复现开发 fixture；前端
   开发不依赖个人照片库或手工修改数据库。
6. 对列表切片，端点已经固定稳定排序、cursor、默认/最大 limit 和取消行为；对媒体
   切片，端点已固定缩略图未就绪、离线、损坏和 Range/条件请求语义。不适用的契约不
   阻塞认证或其他独立切片，但必须在 Gate 中写明 `N/A` 理由。
7. 当前切片相关的安全/可行性 Gate 已通过；只有涉及路径/媒体读取的切片才要求
   FS-01 Linux/HTTP 端到端证据。仍为 Conditional 或被开发就绪评审阻断的相关能力
   不能被前端包装成可用功能。
8. 后端实现可以返回每个设计状态，且错误响应不泄露绝对路径、SQL、工具 stderr
   或堆栈；前端无需靠字符串匹配猜测状态。

当前仓库不满足上述 Backend Ready：OpenAPI 权威源、确定性 TypeScript 类型、唯一 API
client、strict typecheck、目录级 `AGENTS.md` 和 CI 工作流已经落地，但没有可启动应用、
生产 HTTP/auth、React 产品应用、Storybook、共享组件/token 或浏览器测试；CI 也尚未实际
运行。因此当前只可继续已批准的 spike、契约和原型工作，不能把契约基础描述为已接通业务能力。

## 架构治理与变更流程

- React/Vite/TanStack 技术栈、层次、允许依赖、API 边界、CSS 方案或运行时状态
  模型发生变化时，先新增或替代 ADR，再同步 `docs/architecture.md`、本文件和
  `docs/project-structure.md`。
- 页面、组件行为、响应式、可访问性、token 或动效变化同步
  `docs/ui-design.md` 和相关用户流程。
- API 路径、字段、错误码、分页或认证语义在实现期先修改 `api/openapi.yaml`，
  然后重新生成并更新契约测试。
- Storybook、视觉回归、支持浏览器或合并门槛变化同步
  `docs/testing-strategy.md` 与根级构建/CI 入口。
- `web/AGENTS.md` 已约束当前契约基础；创建 design system、feature 和视觉基线时还须指定
  代码审查 owner，owner 是职责而非个人目录。
- 新的共享组件、跨 feature 依赖、lint 例外和视觉基线更新必须在评审中说明唯一
  所有者及为什么现有实现不能满足；不得以时间紧为由复制第二套实现。
- 文档与实现冲突时，以已接受 ADR、已确认产品需求和 OpenAPI 的相应事实来源为
  先；不能静默选择更方便的一份。尚未决定的工具或参数应标为提案并通过 spike
  收敛，不能伪装成已落地能力。
