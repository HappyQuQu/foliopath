# FolioPath 项目目录与依赖约束

## 状态

本文描述首个可用版本的目标仓库结构。当前已有部分 Go spike 包，以及 `web/` 下的 OpenAPI
生成类型、唯一 client、strict TypeScript 和目录级约束；应用、生产 API 与 React 产品脚手架
尚未完整创建。目录应在出现首个真实文件时建立，不为“以后可能会用”预建空包。

具体代码必须同时遵守根目录的 [`AGENTS.md`](../AGENTS.md)。改变模块边界、部署单元或依赖方向前，应先更新架构并在需要时新增 ADR。

## 目标目录树

```text
.
├── cmd/
│   └── foliopath/             Go 进程入口
├── internal/
│   ├── app/                   依赖组装、配置、生命周期、优雅退出
│   ├── api/                   HTTP 路由、处理器、DTO、中间件
│   ├── auth/                  管理员初始化、会话与 CSRF 边界
│   ├── library/               媒体库配置、状态与重叠规则
│   ├── catalog/               目录、媒体、浏览与搜索
│   ├── scanner/               遍历、扫描代次与校准
│   ├── thumbnail/             缩略图任务与缓存语义
│   ├── media/                 媒体探测与原文件响应
│   ├── jobs/                  有界、可恢复的后台任务
│   ├── resourcecontrol/       实例级资源数值限制与共享并发许可
│   ├── pathpolicy/            不接触 I/O 的相对路径词法策略
│   ├── files/                 `/library` 路径安全边界
│   ├── store/sqlite/          SQLite 适配器和生成查询代码
│   └── webassets/             Vite 产物的 Go 嵌入包装
├── web/
│   ├── src/
│   │   ├── app/               应用壳、路由、Provider 与启动
│   │   ├── routes/            路由级页面组装
│   │   ├── features/          auth、libraries、browse、search、viewer、settings
│   │   ├── components/
│   │   │   ├── ui/            唯一共享 UI 原语
│   │   │   └── patterns/      唯一跨 feature 交互模式
│   │   ├── lib/               API 客户端和小型基础设施适配
│   │   └── styles/            token、全局样式与主题
│   └── tests/                 前端共享测试支持
├── api/                       OpenAPI 源文件
├── migrations/                只追加的 SQLite 迁移
├── tests/
│   ├── integration/           跨 Go 组件与真实 SQLite/文件系统测试
│   ├── e2e/                   浏览器端关键流程
│   └── fixtures/              小型、合成、许可清晰的媒体样本
├── deploy/                    Dockerfile、Compose 与部署辅助文件
├── docs/                      产品、交互、工程文档和 ADR
├── AGENTS.md                  仓库级实施约束
└── README.md                  用户入口与项目概览
```

## 后端职责与依赖方向

能力包拥有自己的业务类型、服务和所需接口。适配器实现这些接口，最后只在 `internal/app` 组装。

```text
cmd/foliopath
      │
      ▼
 internal/app ─────► internal/api
      │                    │
      ├────────────► capability packages
      │               auth / library / catalog / scanner / thumbnail / media / jobs
      │                    ▲
      └────────────► adapters
                      files / store/sqlite / media-tool implementations
```

允许的方向：

- `cmd/foliopath` 依赖 `internal/app`，不包含业务逻辑。
- `internal/api` 依赖能力服务与公开业务类型，不依赖 SQLite、真实路径或 FFmpeg 调用。
- `internal/auth` 拥有管理员初始化与会话规则；API 中间件只调用其服务，不直接读取用户或会话表。
- `internal/store/sqlite` 和其他适配器可以依赖能力包定义的接口与类型。
- `internal/pathpolicy` 是无 I/O 的内层策略；library、scanner 和 files 可以复用它。只有
  `internal/files` 可以把策略通过的路径解析为真实文件并验证身份。
- 能力包之间只通过明确的服务或窄接口协作；出现循环依赖时重新划分职责，不用共享包掩盖。
- 只有 `internal/app` 知道完整的具体实现图。

禁止的捷径：

- 不创建笼统的 `utils`、`common`、`helpers`、`base` 包。
- 没有仓库外部真实消费者时不创建 `pkg/`。
- HTTP handler 不执行 SQL、不拼接媒体绝对路径、不直接启动子进程。
- 业务服务不依赖 HTTP DTO、React 生成类型或具体 SQLite 查询实现。
- 不为了测试导出原本应为内部实现的 API；优先从行为边界测试。

## 前端组织规则

完整依赖图、组件所有权、API/Query/URL 状态和自动门禁见[前端子系统架构](architecture/frontend.md)。

- `routes/` 负责读取 URL、加载路由级数据和组合页面，不堆放所有交互逻辑。
- `features/` 按用户能力组织。每个 feature 可包含组件、query、hook、类型和测试，但不能绕过统一 API 客户端。
- `components/` 只放确实跨 feature 使用的基础组件；业务专用组件留在所属 feature。
- `components/ui` 中每种基础语义只有一个规范实现，包括 Dialog/Sheet 的焦点、Escape、inert 与滚动锁；
  `components/patterns` 统一 ConfirmDialog、异步状态、分页/虚拟集合等跨 feature 流程。已有所有者时新增
  variant，不复制近似组件。
- TanStack Query 保存服务端状态；媒体库、目录、搜索、排序和递归开关等可导航状态保存在 URL。
- 短暂 UI 状态优先保持在最近组件。新增全局状态库需要 ADR。
- token 与主题集中在 `styles/`；组件不得各自发明相近但不一致的颜色、间距或动效常量。
- 业务代码不直接导入生成客户端或调用 `fetch`；只通过 `lib/api` adapter。Query key、URL codec、
  错误映射与失效策略各自只有一个 owner。

建议的 feature 起点如下，只有实施对应范围时才创建：

```text
web/src/features/
├── auth/
├── libraries/
├── browse/
├── search/
├── viewer/
└── settings/
```

## API、数据库与生成代码

- REST 契约开始编码后以 `api/openapi.yaml` 为唯一结构化来源；设计阶段参阅 [API 设计](api-design.md)。
- SQL 生成配置固定为 `internal/store/sqlite/sqlc.yaml`，权威查询放在同目录的 `queries/`，
  提交的生成输出只放在 `dbgen/`；adapter 调用生成包并负责领域类型/错误映射，不复制同组 SQL。
- 迁移按连续版本追加到 `migrations/`。已经进入发布版本的迁移不能覆盖或重排。
- Vite 生产输出写入 `internal/webassets/dist`，供同目录 Go 文件嵌入；该构建产物不提交。
- 需要参与离线构建的生成 Go/TypeScript 源码应提交，并通过生成一致性检查防止漂移。

## 测试文件放置

- 单一包或组件的测试与源码同目录。
- 需要真实临时目录、SQLite 或多个能力包协作的测试放入 `tests/integration`。
- 浏览器关键路径放入 `tests/e2e`，不在单元测试中模拟完整产品流程。
- 媒体样本放入 `tests/fixtures`，保持小型、合成且许可证清晰；测试不得访问开发者真实 `/library`。
- 仅供测试共享的代码放在对应测试目录，不把测试便利函数提升为生产通用包。

## 配置与运行时文件

- 环境变量只处理进程启动前必须知道的部署设置，例如监听地址、数据路径和时区；用户可调整的媒体库与应用选项写入 Web 设置。
- `/library` 仅为只读输入，`/app/data` 是唯一持久可写边界。
- 数据库、日志、缓存、临时文件和 Vite 构建输出不得混入源码目录或提交到 Git。
- 顶层新增一个部署单元、外部服务或持久化系统属于架构变化，不是普通目录整理。

## 新文件放置检查

新增文件前依次判断：

1. 它属于哪个用户能力或运行时适配器？
2. 它是否能放在现有能力目录，而不是创建通用目录？
3. 它的依赖是否指向业务边界内部？
4. 它是源文件、生成源、构建产物还是运行时数据？
5. 是否需要同步更新 API、数据模型、安全、测试或部署文档？

如果前两项没有明确答案，先完善职责定义，再创建目录。
