# FolioPath 架构

## 状态

本文是首个可用版本的架构概览。完整的系统上下文、模块所有权、交付治理、前端子系统、需求追踪和
自动化约束见[系统架构档案](architecture/README.md)。项目尚处于规划与早期开发阶段；改变本文中的
核心约束前，应先在 `docs/adr/` 新增或替代 ADR。

文档中的 **Target**、仓库实际存在的 **Current** 与 spike 的局部验证必须分开理解。当前已有部分
Go 能力、SQLite/generation、Darwin 与原生 Linux amd64/arm64 `openat2` 路径边界证据，
`api/openapi.yaml` 已成为 HTTP 结构权威，TypeScript 类型和唯一 Web API client 可确定性
生成，摘要锁、语义兼容检查与双架构 CI 工作流也已建立；首次原生 amd64/arm64 CI 已通过。运行进程、
生产 handler、认证、React 产品应用、完整媒体 adapter、容器与发布链路仍未实现。

## 架构驱动因素

按优先级约束方案取舍：

1. 原媒体只读、路径与认证边界失败关闭；
2. 离线、中断和部分失败不会破坏上次可靠索引或不可重建配置；
3. 一个容器、一个端口和两个主要 volume 的低运维成本；
4. 四核、4 GiB 目标环境上的有界并发、事务、缓存和前端渲染；
5. 文件夹语义、URL 与 API/数据契约长期一致；
6. linux/amd64、linux/arm64 和主流浏览器的可重复交付。

这些驱动因素的场景、视图和验证入口分别见[系统上下文](architecture/system-context.md)、
[模块边界](architecture/modules.md)与[架构适配度检查](architecture/fitness-functions.md)。

## 目标与非目标

首版目标：

- 用一个 Docker 容器提供多个只读媒体库的管理和浏览。
- 保留真实目录层级，并高效支持当前目录与递归目录浏览。
- 在大型目录上以可控的内存、CPU 和磁盘占用完成扫描与缩略图生成。
- 将 Docker 配置限制为媒体根目录、数据目录和必要的网络设置，其余配置由 Web 界面管理。
- 首个稳定版提供内建单管理员认证（见 [ADR-0005](adr/0005-built-in-single-admin-auth.md)）；主要容量验收档为约 10 万媒体、1 万目录、4 GiB 内存的四核 NAS/家庭服务器。

首版不提供媒体上传、备份、文件整理、视频转码、AI 分类、水平扩展或多实例共享写入。

## 运行拓扑

```text
Browser
   │ REST / media Range
   ▼
FolioPath Go process
   ├── embedded React SPA
   ├── HTTP API
   ├── single-admin authentication and sessions
   ├── library/catalog services
   ├── bounded scan and media jobs
   ├── SQLite ───────────────► /app/data/foliopath.db
   ├── thumbnail cache ──────► /app/data/cache
   └── read-only media ──────► /library
          ├── family
          ├── work
          └── mobile
```

运行时只有一个应用进程和一个 HTTP 端口。React 静态产物嵌入 Go 服务，不需要 Nginx、独立前端容器、Redis 或外部数据库。

## 技术栈

- 后端：Go，HTTP 使用标准库 `net/http`。
- 数据：SQLite WAL、`sqlc` 查询代码和 Goose 嵌入式迁移。
- 图片：libvips，通过 govips 探测和生成缩略图。
- 视频：`ffprobe` 获取元数据，FFmpeg 抽取封面；首版不转码。
- 前端：React、TypeScript、Vite、React Router、TanStack Query 和 TanStack Virtual。
- API：同源 `/api/v1` REST API、OpenAPI 契约和不透明游标分页。
- 发布：基于固定 digest Debian-family 构建层和 distroless final stage 的多阶段、
  多架构 Docker 镜像。

## 代码布局

本节给出顶层布局；包职责、依赖方向和前端 feature 规则详见[项目目录与依赖约束](project-structure.md)。

```text
cmd/foliopath/          minimal process entry point
internal/app/           dependency composition and lifecycle
internal/api/           HTTP handlers, DTOs and middleware
internal/auth/          administrator initialization, sessions and CSRF boundary
internal/library/       library configuration and state
internal/catalog/       directories, assets, search and browsing
internal/scanner/       reconciliation scans and generations
internal/thumbnail/     image thumbnail jobs and cache
internal/media/         probing and original-media responses
internal/jobs/          bounded, restart-safe background work
internal/pathpolicy/    pure relative-path lexical policy
internal/files/         allowed-root and path-security boundary
internal/store/sqlite/  SQLite adapters
internal/webassets/     Go embed wrapper and generated Vite output
api/                    OpenAPI source
migrations/             append-only SQL migrations
web/                    React application
tests/                  integration, browser and synthetic fixtures
deploy/                 container artifacts
```

业务能力包声明自己所需的接口，SQLite、文件系统和媒体工具作为适配器实现这些接口。纯
`internal/pathpolicy` 只拥有相对路径词法规则，不打开文件；真实身份与打开操作仍只属于
`internal/files`。`internal/api` 只能调用服务，不能直接查询数据库、解析真实路径或启动媒体进程；
具体实现统一在 `internal/app` 组装。Vite 在生产构建时把产物写入 `internal/webassets/dist`，再由同目录
的 Go 包通过 `go:embed` 嵌入。不要创建没有明确外部使用者的 `pkg/`，也不要创建通用 `utils`、
`common` 或 `helpers` 包。

每条业务规则、事务、任务状态机、错误映射和共享 UI 语义必须有唯一所有者；完整矩阵见
[模块、数据与运行时边界](architecture/modules.md)。Go import 方向从当前阶段起由 `make arch-check`
检查，前端依赖、token 和组件唯一性在前端脚手架建立时加入同一门禁。

## 关键流程

### 创建媒体库

1. 用户在设置中浏览 `/library` 下服务端允许的目录。
2. API 接收实例内唯一名称和相对于 `/library` 的路径；空路径唯一表示 `/library` 本身。
3. `internal/files` 规范化相对路径，并从已锚定的 `/library` 边界原子打开；Linux 同时
   拒绝 symlink 与后代 mount crossing。
4. 服务拒绝相同或祖先/后代关系的重叠媒体库。
5. SQLite 保存媒体库配置并创建首次扫描任务。MVP 允许改名但不允许原地修改根路径；更换路径必须移除后重新创建，详见 [ADR-0004](adr/0004-library-root-immutable.md)。

### 扫描和索引

扫描器分批读取目录，通过有界队列识别文件，再由单一数据库写入路径批量提交。每次完整扫描分配 generation；只有根目录可读、遍历完整且所有索引写入成功后，才清理旧 generation。失败、协作式取消或离线扫描保留旧索引。创建后与应用启动时触发校准，默认每 24 小时完整扫描且可在 UI 修改或关闭。扫描普通隐藏项，但跳过维护清单中的 NAS/系统派生目录和回收站，并记录跳过统计。详细语义见 [ADR-0003](adr/0003-scan-consistency.md)。

### 浏览和搜索

浏览请求使用媒体库 ID、目录 ID、排序条件和不透明游标。后端只从持久索引查询，不在请求过程中递归遍历文件系统。目录树包含所有可读取目录及直接/递归媒体计数。普通目录默认按自然文件名升序，递归与搜索默认按修改时间倒序；搜索默认当前媒体库，并可切换当前目录（可递归）与全部媒体库。列表使用稳定排序和唯一 ID 作为游标的最终比较项；前端通过 TanStack Query 追加页面，并通过 TanStack Virtual 只渲染可视区域。

### 媒体和缩略图

公开 URL 使用媒体 ID。服务从索引取得相对路径，再通过 `internal/files` 安全打开文件。JPEG、PNG、WebP、GIF 进入图片缩略图流程，MP4、MOV、MKV 进入视频探测/封面流程；只有浏览器兼容编码直接播放。原图和兼容视频使用条件请求与 Range 响应。缩略图按需进入全局有界队列，先写同目录临时文件，再原子替换缓存目标；默认 10 GiB 可配置配额达到水位时按 LRU 清理并保留安全余量。

Post-MVP/1 通过 CR-2026-010 将 AVI 接入同一视频探测、派生、资源预算与 Range 边界；
它不改变“不转码、只直接播放浏览器兼容 codec”的约束。

## 配置与部署边界

容器内路径固定为：

- `/library`：允许用户选择媒体库的只读根目录。
- `/app/data`：数据库、配置、任务状态和派生缓存。

默认部署只要求映射这两个目录。应用设置、单管理员账户和会话状态保存在 SQLite，不要求为
新增 `/library` 下的普通目录媒体库修改 Compose。应用设置还包括 24 小时默认扫描周期、
10 GiB 默认缓存配额和中英语言偏好。若要纳入当前允许边界之外的宿主机媒体，部署者必须调整
唯一 `/library` 根挂载或先建立单一宿主呈现根；不得在 `/library` 后代增加子 volume。
这是预期的安全边界，详见
[ADR-0009](adr/0009-linux-openat2-single-media-root.md)。

SQLite 数据目录必须使用容器宿主机可正确提供锁和同步语义的本地文件系统；不要把 WAL 数据库直接放在 SMB/NFS 上。首版发布 `linux/amd64` 与 `linux/arm64` 镜像。

## 实施与演进约束

项目采用[契约驱动、切片内后端优先](adr/0006-contract-driven-backend-first.md)：需求和失败语义先进入
Architecture Ready，随后固定 OpenAPI/数据契约；后端领域、适配器与集成证据通过后，业务前端才使用
生成客户端实现。应用壳、[共享前端设计系统](adr/0007-shared-frontend-system.md)和可丢弃原型可以并行，
但不得在长期 mock 上形成第二套业务语义。

MVP 功能范围已经冻结。新能力默认进入后续版本，范围变更、阶段/版本关系、Gate 和豁免规则见
[交付与架构治理](architecture/delivery-governance.md)。阶段完成不等于版本完成；阶段 0～5、PRD 验收、
发布测试和阻断风险全部满足后，才能称为稳定 MVP。
