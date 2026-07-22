# FolioPath 架构

## 状态

本文描述首个可用版本的目标架构。项目尚处于规划与早期开发阶段；改变本文中的核心约束前，应先在 `docs/adr/` 新增或替代 ADR。

## 目标与非目标

首版目标：

- 用一个 Docker 容器提供多个只读媒体库的管理和浏览。
- 保留真实目录层级，并高效支持当前目录与递归目录浏览。
- 在大型目录上以可控的内存、CPU 和磁盘占用完成扫描与缩略图生成。
- 将 Docker 配置限制为媒体根目录、数据目录和必要的网络设置，其余配置由 Web 界面管理。

首版不提供媒体上传、备份、文件整理、视频转码、AI 分类、水平扩展或多实例共享写入。

## 运行拓扑

```text
Browser
   │ REST / media Range
   ▼
FolioPath Go process
   ├── embedded React SPA
   ├── HTTP API
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
- 发布：基于 Debian slim 的多阶段、多架构 Docker 镜像。

## 代码布局

本节给出顶层布局；包职责、依赖方向和前端 feature 规则详见[项目目录与依赖约束](project-structure.md)。

```text
cmd/foliopath/          minimal process entry point
internal/app/           dependency composition and lifecycle
internal/api/           HTTP handlers, DTOs and middleware
internal/library/       library configuration and state
internal/catalog/       directories, assets, search and browsing
internal/scanner/       reconciliation scans and generations
internal/thumbnail/     image thumbnail jobs and cache
internal/media/         probing and original-media responses
internal/jobs/          bounded, restart-safe background work
internal/files/         allowed-root and path-security boundary
internal/store/sqlite/  SQLite adapters
internal/webassets/     Go embed wrapper and generated Vite output
api/                    OpenAPI source
migrations/             append-only SQL migrations
web/                    React application
tests/                  integration, browser and synthetic fixtures
deploy/                 container artifacts
```

业务能力包声明自己所需的接口，SQLite、文件系统和媒体工具作为适配器实现这些接口。`internal/api` 只能调用服务，不能直接查询数据库、解析真实路径或启动媒体进程；具体实现统一在 `internal/app` 组装。Vite 在生产构建时把产物写入 `internal/webassets/dist`，再由同目录的 Go 包通过 `go:embed` 嵌入。不要创建没有明确外部使用者的 `pkg/`，也不要创建通用 `utils`、`common` 或 `helpers` 包。

## 关键流程

### 创建媒体库

1. 用户在设置中浏览 `/library` 下服务端允许的目录。
2. API 接收名称和相对于 `/library` 的路径。
3. `internal/files` 规范化并解析真实路径，确认其仍在允许根目录内。
4. 服务拒绝相同或祖先/后代关系的重叠媒体库。
5. SQLite 保存媒体库配置并创建首次扫描任务。

### 扫描和索引

扫描器分批读取目录，通过有界队列识别文件，再由单一数据库写入路径批量提交。每次完整扫描分配 generation；只有根目录可读、遍历完整且所有索引写入成功后，才清理旧 generation。失败、取消或离线扫描保留旧索引。详细语义见 [ADR-0003](adr/0003-scan-consistency.md)。

### 浏览和搜索

浏览请求使用媒体库 ID、目录 ID、排序条件和不透明游标。后端只从持久索引查询，不在请求过程中递归遍历文件系统。列表使用稳定排序和唯一 ID 作为游标的最终比较项；前端通过 TanStack Query 追加页面，并通过 TanStack Virtual 只渲染可视区域。

### 媒体和缩略图

公开 URL 使用媒体 ID。服务从索引取得相对路径，再通过 `internal/files` 安全打开文件。原图和浏览器兼容视频使用条件请求与 Range 响应。缩略图按需进入全局有界队列，先写同目录临时文件，再原子替换缓存目标。

## 配置与部署边界

容器内路径固定为：

- `/library`：允许用户选择媒体库的只读根目录。
- `/app/data`：数据库、配置、任务状态和派生缓存。

默认部署只要求映射这两个目录。应用设置保存在 SQLite，不要求为新增 `/library` 下的媒体库修改 Compose。新增允许边界之外的宿主机目录仍需要先增加 Docker volume，这是预期的安全边界。

SQLite 数据目录必须使用容器宿主机可正确提供锁和同步语义的本地文件系统；不要把 WAL 数据库直接放在 SMB/NFS 上。首版发布 `linux/amd64` 与 `linux/arm64` 镜像。
