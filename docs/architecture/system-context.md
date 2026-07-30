# FolioPath 系统上下文与运行视图

## 状态与读法

本文给出 FolioPath 的系统级导航视图：它把已有的产品约束、ADR、部署边界和关键运行流程放到同一上下文中，但不替代这些权威文档，也不新增 API、数据表或部署承诺。

图和表使用以下状态标记：

| 标记 | 含义 | 使用规则 |
| --- | --- | --- |
| **[C] Current** | 当前仓库中已经存在的代码或可执行检查 | 只能描述仓库可直接确认的事实，不能据此宣称完整产品已可运行 |
| **[T] Target** | 已接受决策所指向的首版目标架构 | 在对应实现和验证完成前仍是目标，不是交付证明 |
| **[S] Spike** | 有明确范围、环境和局限的可行性证据 | 只证明 spike 报告声明的范围，不能外推为生产能力或容量结论 |

当前快照是：`internal/pathpolicy`、`internal/files`、`internal/library`、`internal/scanner` 和
`internal/store/sqlite` 已有 [C]/[S] 代码与测试；FS-01 已取得 Darwin、Linux/arm64
`openat2` mount-boundary 与 HTTP test harness 子范围证据，SQLite/generation 通过当前
正确性范围，`api/openapi.yaml` 已成为 HTTP 结构权威。最小进程入口已经存在并把控制交给
`internal/app.Run`；应用组合根已拥有固定 `/library`、`/app/data` 与认证前回环监听配置，以及
根取消、顺序启动、失败回滚、运行故障传播、反向关闭和有界停机。正式 SQLite adapter 与嵌入
migration 已接入数据库 → HTTP → readiness 启动链路。
HTTP 运行边界已具备单 listener、服务端 request ID、统一安全错误、JSON 日志和
在途请求排空、liveness/readiness 和受保护系统状态路由；只有数据目录可用、数据库打开且
migration 成功后才进入 ready，失败时进程不提供业务服务。系统状态在认证实现前默认拒绝；
真实组合测试与测试专用应用容器已覆盖取消、重复启动、非 root、固定 volume 和内部 health；
认证 HTTP 与 `users`/`sessions` 数据契约已达到 S1-101 Contract Ready，但当前还没有业务
路由或认证 service。React 产品应用、
完整媒体工具链与正式发布容器仍未形成产品。生成 TypeScript
契约/client 与本地验证入口已建立；历史原生 amd64/arm64 结果保留，当前不运行自动 CI。准确状态以
[开发就绪评审](../development-readiness.md)和[可行性研究](../feasibility-study.md)为准。

发生冲突时，先按[交付与架构治理](delivery-governance.md)暂停受影响工作：已确认产品基线决定范围，
已接受 ADR 决定结构，当前 `api/openapi.yaml` 与已发布迁移决定可执行契约，`AGENTS.md`
约束实施，本文图示只作系统导航。改变单进程部署、路径模型、扫描一致性、认证边界或模块依赖方向前，
必须先新增或替代 ADR。

## 架构驱动因素

| 优先级 | 驱动因素与质量场景 | 目标策略 | 权威依据与验证方向 |
| --- | --- | --- | --- |
| P0 | 原始媒体即使在删除媒体库、扫描失败或缓存故障时也不能被修改 | `/library` 只读挂载；所有原媒体访问集中在文件边界；删除仅作用于配置、索引、任务和缓存 | [产品需求](../product-requirements.md)、[安全模型](../security.md)、只读容器与删除流程测试 |
| P0 | 用户在 Web 中创建多个媒体库，只选择已映射根目录下的子目录，并默认包含后代 | Docker 只声明一个 `/library` 媒体根和 `/app/data`；拒绝后代嵌套挂载；库根保存为安全相对路径；拒绝重叠；根路径在 MVP 中不可变 | [ADR-0002](../adr/0002-library-path-model.md)、[ADR-0004](../adr/0004-library-root-immutable.md)、[ADR-0009](../adr/0009-linux-openat2-single-media-root.md)、路径/mount/重叠测试 |
| P0 | 挂载离线、权限错误、取消或进程中断不能把可靠索引误判为空 | 文件系统是存在性与层级事实来源；SQLite 是派生状态；完整扫描使用 generation，只有全程成功才清理旧代次 | [ADR-0003](../adr/0003-scan-consistency.md)、故障注入和恢复测试 |
| P0 | 私人媒体不能因默认网络配置而匿名暴露 | 稳定版内建单管理员、会话、退出和 CSRF；认证完成后可直接服务受信 LAN HTTP，公网 TLS/代理由部署者提供 | [ADR-0005](../adr/0005-built-in-single-admin-auth.md)、[ADR-0010](../adr/0010-authenticated-lan-http.md)、[安全模型](../security.md)、认证与 transport 测试 |
| P0 | 路径、文件名、媒体编码和元数据均视为不可信输入 | 相对路径先经唯一词法策略；Linux 实际打开由根 FD 锚定的 `openat2` 原子执行并拒绝 symlink/mount crossing；原生媒体处理有超时、取消、大小限制和并发上限 | [安全模型](../security.md)、FS-01 与 FS-03 |
| P1 | 在四核、4 GiB 的主要验收环境中处理约 10 万媒体和 1 万目录，且交互请求不被后台工作饿死 | 流式遍历、有界队列、批量短事务、写入串行化、游标分页、虚拟化和按资源类别限流 | [测试策略](../testing-strategy.md)、FS-04；该规模是验收目标，不是已证明性能 |
| P1 | 自托管部署、升级、备份和故障排查保持简单 | 一个应用容器、一个 Go 服务端口、嵌入式 SPA、本地 SQLite、两个主要 volume；不引入额外服务 | [ADR-0001](../adr/0001-go-react-sqlite.md)、[部署草案](../deployment.md)、FS-05 |
| P1 | 大型列表在数据变化时仍能稳定翻页，媒体可以按浏览器协议流式读取 | 索引查询、稳定唯一 tie-breaker 的 keyset cursor、条件请求、`HEAD` 和单范围 Range；不在请求中递归文件系统 | [API 设计](../api-design.md)、游标与 HTTP 集成测试 |
| P1 | SQLite、索引和缓存故障可恢复，且不可重建的账号与设置可备份 | WAL 位于受支持的本地文件系统；迁移只追加；派生缓存可重建；数据库备份与恢复必须演练 | [数据模型](../data-model.md)、[部署草案](../deployment.md)、迁移和恢复门禁 |

## C4 L1：系统上下文 [T]

```mermaid
flowchart LR
    admin["管理员\n管理实例、媒体库并浏览媒体 [T]"]
    browser["浏览器\n运行同源 React SPA [T]"]
    proxy["可信反向代理 / TLS\n可选部署边界 [T]"]
    foliopath["FolioPath\n单管理员、自托管媒体目录浏览系统 [T]"]
    mediaHost["宿主机或 NAS 媒体目录\n文件系统事实来源 [T]"]
    backup["备份存储与恢复流程\n保护 /app/data [T]"]

    admin --> browser
    browser -->|"公网或不可信网络：HTTPS"| proxy
    proxy -->|"受信转发头；支持流式与 Range"| foliopath
    browser -->|"受信 LAN：经认证 HTTP / 同源请求"| foliopath
    mediaHost -->|"一个只读根映射到 /library；无后代挂载"| foliopath
    foliopath -->|"安全读取；离线时保留索引"| mediaHost
    foliopath -->|"停机或 SQLite 安全备份流程"| backup
```

系统之外的管理员和部署平台决定哪个单一宿主呈现根映射到 `/library`；FolioPath 只能在
这个容器内允许根之下工作，并拒绝其后代的嵌套 mount。Web 设置可以创建 `/library`
普通子目录对应的多个媒体库，但不能扩大 Docker 已授予的文件系统能力。

## C4 L2：容器与数据边界 [T]

```mermaid
flowchart TB
    subgraph client["用户设备"]
        spa["React SPA\n路由、查询缓存、虚拟化视图 [T]"]
    end

    subgraph image["FolioPath 应用容器：一个对外服务单元 [T]"]
        go["Go 应用进程\n嵌入 SPA、HTTP API、认证、能力服务、调度与有界 worker [T]"]
        native["受限媒体执行边界\nlibvips 库；ffprobe / ffmpeg 子进程 [T]"]
        go -->|"超时、取消、输入限制、并发配额"| native
    end

    subgraph data["容器挂载与持久边界"]
        library["/library\n只读媒体与目录 [T]"]
        sqlite["/app/data/foliopath.db\nSQLite WAL [T]"]
        cache["/app/data/cache\n可重建缩略图与封面 [T]"]
        temp["/app/data/tmp\n受控临时文件 [T]"]
    end

    spa <-->|"同源 /api/v1；媒体条件请求与 Range"| go
    go -->|"迁移、配置、索引、会话、任务"| sqlite
    go -->|"安全目录枚举与按 ID 打开"| library
    go -->|"缓存命中、配额与 LRU 元数据"| cache
    native -->|"读取不可信媒体"| library
    native -->|"先写临时文件"| temp
    native -->|"同目录原子替换派生文件"| cache
```

“一个应用进程”指一个 Go 服务进程和一个对外 HTTP 端口；FFmpeg/ffprobe 是由该进程受控启动的短生命周期子进程，libvips 是原生库边界，不是额外部署服务。SQLite、缓存和媒体目录是数据存储，不是可独立扩展的服务。未经 ADR，不增加 Redis、外部数据库、独立 worker、Nginx、GraphQL、WebSocket 或第二个可部署单元。

## 部署与信任边界

```mermaid
flowchart LR
    internet["浏览器 / 不可信网络"]
    proxy["TLS 与可信代理配置 [T]"]

    subgraph host["容器宿主机"]
        hostMedia["宿主媒体目录"]
        hostData["本地持久数据目录"]

        subgraph container["非 root FolioPath 容器 [T]"]
            http["Go HTTP 进程"]
            allowed["/library :ro"]
            appData["/app/data :rw"]
            tools["libvips / FFmpeg 边界"]
        end
    end

    internet --> proxy --> http
    hostMedia -->|"只读 volume"| allowed
    hostData <-->|"可写 volume"| appData
    http --> allowed
    http --> appData
    http --> tools
    tools --> allowed
    tools --> appData
```

| 边界 | 进入边界的数据 | 强制控制 | 失败语义 |
| --- | --- | --- | --- |
| 浏览器 → HTTP | Cookie、CSRF、ID、查询、游标、Range 和显示文本 | 同源会话、CSRF、限流、结构化校验、统一安全错误、请求 ID | 拒绝请求而不泄露 SQL、绝对路径、堆栈或原始 stderr |
| 代理 → 应用 | `Forwarded` / `X-Forwarded-*`、客户端地址、协议 | 只信任显式配置的代理来源；稳定网络暴露使用 TLS | 非受信来源的转发头不参与安全判断 |
| `/library` → 文件边界 | 路径组件、symlink、mount、权限、目录项、文件内容 | 单一只读根；Linux 从根 FD 使用 `openat2` 的 `BENEATH`、`NO_SYMLINKS`、`NO_XDEV` 原子解析；扫描不跟随目录 symlink 或跨越后代挂载 | 边界能力不可用时失败关闭；单库不可读时标记离线并保留可靠索引 |
| Go → 媒体工具 | 不可信媒体与解析参数 | 参数数组而非 shell；超时、取消、大小/像素/帧限制和全局并发上限 | 记录稳定错误，有限重试，不阻塞整个队列 |
| Go → `/app/data` | 账号、会话、配置、索引、任务、缓存和临时文件 | 明确权限、本地文件系统、WAL、短事务、磁盘余量和原子文件提交 | 不可写或迁移失败时不进入业务就绪；派生数据可重建 |

容器的目标 Compose、安全选项、权限、备份与升级流程由[部署草案](../deployment.md)维护；威胁和输入控制由[安全模型](../security.md)维护。本文不固定尚未验证的镜像 UID/GID、healthcheck 间隔、端口或资源数值。

## 关键动态流程 [T]

以下序列图表达所有权、提交点和失败传播；精确字段、状态码和枚举最终由 OpenAPI 与迁移固定。

### 启动与就绪

```mermaid
sequenceDiagram
    participant Runtime as 容器运行时
    participant App as internal/app
    participant Store as SQLite 适配器
    participant Jobs as 调度器与 worker
    participant HTTP as HTTP 服务

    Runtime->>App: 启动进程并提供启动配置
    App->>App: 校验监听、允许根、数据路径和安全启动参数
    App->>Store: 打开 /app/data，设置 WAL/外键/busy timeout，执行迁移
    alt 数据目录不可写、数据库打不开或迁移失败
        Store-->>App: 安全分类的启动错误
        App-->>Runtime: 保持 not-ready 并终止启动
    else 持久层就绪
        App->>Store: 恢复遗留 running 扫描/任务为可安全收敛状态
        App->>Jobs: 创建全局有界队列、调度器和根取消上下文
        App->>HTTP: 注册健康、受限初始化和业务路由
        App->>HTTP: 开始服务；数据库与迁移成功后 ready
        App->>Jobs: 请求启动校准/计划扫描
        Note over Jobs,Store: 单个媒体库离线只改变该库状态，不使整个实例 not-ready
    end
```

首次启动是否进入管理员初始化状态由持久认证状态决定；不存在默认密码。精确认证流程见[ADR-0005](../adr/0005-built-in-single-admin-auth.md)。

### 创建媒体库与首次完整扫描

```mermaid
sequenceDiagram
    participant UI as 设置界面
    participant API as HTTP / API
    participant Auth as auth 能力
    participant Library as library 能力
    participant Files as files 边界
    participant Store as SQLite 适配器
    participant Jobs as jobs 能力
    participant Scanner as scanner 能力

    UI->>API: 浏览 /library 下的直接子目录
    API->>Auth: 验证会话
    API->>Library: ListAllowedDirectories 查询
    Library->>Files: 通过 directory-lister 端口按相对 parent 安全枚举
    Files-->>Library: 名称、相对路径和文件边界状态
    Library-->>API: 应用重叠/可选业务规则后的目录结果
    API-->>UI: 安全 DTO，不含宿主路径

    UI->>API: 创建库（名称、rootRelPath、幂等键）
    API->>Auth: 验证会话与 CSRF
    API->>Library: CreateLibrary 命令
    Library->>Files: 规范化、解析真实根并取得安全身份
    Library->>Store: 序列化短事务：再次检查名称/相同/祖先/后代冲突，创建库与持久首次扫描请求
    alt 名称或根冲突
        Store-->>Library: 稳定 conflict；不提交任何记录
        Library-->>API: 领域冲突
        API-->>UI: 409 安全错误
    else 提交成功
        Store-->>Library: 媒体库与扫描引用
        Library->>Jobs: 提交后唤醒 worker（持久记录才是事实）
        Library-->>API: 媒体库与扫描引用
        API-->>UI: 返回已接受的资源状态
    end

    Jobs->>Store: 原子领取可运行扫描
    Jobs->>Scanner: 执行完整 generation
    Scanner->>Files: 再验证库根并流式遍历
    loop 有界批次
        Scanner->>Store: 短事务 upsert 目录/媒体和 lastSeenGeneration
        Scanner->>Jobs: 提交后登记需要的派生任务
    end
    alt 全部遍历与写入成功
        Scanner->>Store: 单一 finalize 事务：清旧代次、聚合计数、推进 currentGeneration、标记成功
    else 离线、部分不可读、取消、进程中断或写入失败
        Scanner->>Store: 记录安全失败/中断；绝不清理旧代次
    end
    UI->>API: 按条件/退避规则轮询扫描状态
```

目录遍历不能把完整树加载到内存，也不能按条目启动无界 goroutine。扫描状态、generation 和删除语义以[数据模型](../data-model.md)和[ADR-0003](../adr/0003-scan-consistency.md)为准；
创建、轮询与取消的精确 HTTP 结构由当前权威 `api/openapi.yaml` 固定，[API 设计](../api-design.md)
只解释动机与实现参数。

### 缩略图与视频封面

```mermaid
sequenceDiagram
    participant UI as 浏览器
    participant API as HTTP / API
    participant Thumb as thumbnail 能力
    participant Store as SQLite 适配器
    participant Jobs as jobs 能力
    participant Files as files 边界
    participant Tool as libvips / ffprobe / ffmpeg
    participant Cache as /app/data/cache

    UI->>API: GET thumbnail(assetId, variant)
    API->>Thumb: 读取允许变体的派生状态
    Thumb->>Store: 按 asset、源指纹和变换版本查询
    alt 缓存记录与文件均有效
        Thumb->>Cache: 安全打开缓存文件
        API-->>UI: 条件响应或缩略图流
    else 未生成、过期或损坏
        Thumb->>Jobs: 幂等确保一个派生任务
        API-->>UI: 返回 OpenAPI 固定的 pending/占位契约
        Jobs->>Store: 原子领取可运行派生任务
        Jobs->>Thumb: 调用已注册的 thumbnail handler
        Thumb->>Files: 按媒体 ID 对应相对路径安全打开源文件
        Thumb->>Tool: 带超时、取消、输入限制执行变换
        Tool->>Cache: 写随机临时文件并同步必要内容
        Tool->>Cache: 同目录原子替换最终缓存文件
        Thumb->>Store: 文件可用后才提交 ready、尺寸与字节数
        Note over Thumb,Store: 源指纹变化则丢弃旧结果；崩溃残留由幂等恢复/清理收敛
    end
```

缩略图文件不进入 SQLite。缓存键、默认配额和 LRU 语义以[数据模型](../data-model.md)及[架构总览](../architecture.md)为准；“未就绪”响应方式仍是 OpenAPI 落地前需要固定的技术参数。

### 原媒体条件请求与 Range

```mermaid
sequenceDiagram
    participant UI as 浏览器
    participant API as HTTP / API
    participant Media as media 能力
    participant Catalog as catalog 能力
    participant Files as files 边界

    UI->>API: GET/HEAD content(assetId) + 条件头/Range
    API->>API: 验证会话、方法和协议头
    API->>Media: 按 assetId 请求安全内容句柄
    Media->>Catalog: 取得 libraryId、相对路径、类型和源指纹
    Media->>Files: 再解析边界并安全打开当前文件
    Files-->>Media: 受控句柄、stat 与安全错误
    Media-->>API: 内容元数据与可流式读取句柄
    alt 校验器命中
        API-->>UI: 304
    else HEAD
        API-->>UI: 200 + 头，不传响应体
    else 合法单范围
        API-->>UI: 206 流式响应
    else 无效或不可满足范围
        API-->>UI: 416
    else 完整内容
        API-->>UI: 200 流式响应
    end
    UI--xAPI: 播放/导航取消
    API-->>Media: 传播请求取消并关闭句柄
```

MVP 不转码视频，也不因读取失败在请求路径中执行大范围索引清理。Range、缓存校验器、安全响应头和错误码的精确行为由 OpenAPI 与 HTTP 集成测试固定。

### 优雅停机与重启恢复

```mermaid
sequenceDiagram
    participant Runtime as 容器运行时
    participant App as internal/app
    participant HTTP as HTTP 服务
    participant Jobs as 调度器
    participant Workers as 扫描与媒体 worker
    participant Store as SQLite 适配器

    Runtime->>App: SIGTERM / 根上下文取消
    App->>HTTP: 立即切换 not-ready，停止接收新的业务工作
    App->>Jobs: 停止调度和领取新任务
    App->>Workers: 传播协作式取消
    App->>HTTP: 在有界期限内等待在途请求结束
    Workers->>Store: 提交已完成的安全小批次；不执行失败扫描的陈旧清理
    Workers->>Store: 将运行状态留给中断/租约恢复语义收敛
    Workers-->>App: 结束或达到停机期限
    App->>Store: 完成必要关闭并关闭数据库句柄
    App-->>Runtime: 退出
```

停机不能依赖无限等待，也不能强行把部分扫描标记成功。具体优雅期限、任务租约和强制退出后的恢复细节需要在 jobs 设计、部署验证与 FS-05 中固定。

## 本文有意不固定的事项

- Cookie、CSRF、密码哈希、会话期限和可信代理参数：由安全设计、OpenAPI 和认证测试固定。
- API 分页上限、幂等键期限、轮询退避、缩略图 pending 响应和 Range 的精确表达：由[API 设计](../api-design.md)列出的 spike 与 OpenAPI 固定。
- worker 数量、队列容量、内存/CPU 水位、p95 延迟和停机期限：由 FS-03～FS-05 与容量基线给出可重复证据后写回。
- 镜像 UID/GID、端口、healthcheck 间隔和只读根文件系统兼容性：由最终镜像测试和[部署草案](../deployment.md)固定。
- 未来 watcher、SSE、多用户、分享、转码或外部服务：不属于 MVP；若改变当前约束，必须先经过需求与 ADR。
