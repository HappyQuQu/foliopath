# FolioPath

> A folder-first, self-hosted photo and video browser.

FolioPath 是一个以真实文件夹结构为核心的自托管图片与视频浏览器。

只需将一个媒体根目录挂载到 Docker 容器中，再通过 Web 设置创建一个或多个媒体库。FolioPath 会扫描所选目录并生成缩略图，让你通过浏览器按照原始文件夹层级查看内容。它提供清晰的目录树、瀑布流布局，以及可以一次浏览当前目录及所有子目录内容的递归模式。

你的文件夹，就是你的相册。

> [!IMPORTANT]
> FolioPath 目前处于早期开发阶段，后端运行骨架已通过真实组合与测试容器 smoke，但尚无可用产品界面或发布镜像。FS-01 已在
> 原生 Linux amd64/arm64 验证 `openat2` 同设备、跨设备和 self-bind 边界及真实 HTTP test harness，
> FS-02 当前正确性范围、FS-03 双架构媒体链路、FS-04 Stage 0 容量范围和 FS-05 双架构
> 运行/恢复范围均通过；OpenAPI、生成类型、唯一 Web API 客户端和供应链 CI 已建立。
> 测试容器不是可供部署的正式镜像。
> Stage 0 Gate 已通过并只授权后端优先的 Stage 1；这不代表应用、业务 UI 或发布镜像已完成。

## Why FolioPath?

许多照片管理工具以时间线、数据库相册或 AI 分类为核心，真实目录结构往往只是附加功能。

FolioPath 选择另一条路线：

- 文件系统是媒体内容与目录层级的唯一事实来源
- 保留现有目录结构，无需复制、移动或导入照片
- 快速浏览拥有大量层级和文件的图片目录
- 通过递归模式穿透子文件夹，连续查看全部内容
- 在 Web 设置中创建和管理多个媒体库
- 使用 Docker 部署，适合 NAS、家庭服务器和个人设备

## Features

### Planned core features

- 🌲 清晰、可折叠的完整文件夹树，包括没有媒体的可读目录
- 📚 在 Web 设置中创建和管理多个媒体库
- 🖼️ 默认自适应网格，并可切换为记忆的瀑布流布局
- 🔭 递归浏览当前目录及所有子目录
- 📍 面包屑导航与媒体库内相对路径显示
- ⚡ 默认 10 GiB 可配置缩略图缓存、虚拟滚动和增量扫描
- 🔎 按文件名、类型、日期和路径搜索
- 🎞️ JPEG、PNG、WebP、GIF，以及 MP4、MOV、MKV 索引与封面
- 🌙 明暗主题
- 🌐 简体中文与英文，默认跟随浏览器
- 📱 响应式 Web 界面
- 🐳 Docker 与 Docker Compose 部署
- 🔒 默认支持只读媒体目录
- 🔐 稳定版内建首次设置的单管理员认证

### Possible future features

- EXIF 和媒体信息面板
- 收藏、评分和浏览历史
- SVG、HEIC/HEIF、AVIF、RAW 及更多格式的缩略图
- 文件系统变更监控
- 分享链接和细粒度访问控制
- 重复文件检测
- 地图与时间线视图

## Media Libraries

Docker 挂载只负责划定 FolioPath 可以读取的媒体根目录。具体媒体库在 Web 设置中创建，无需为每个媒体库修改 Compose 配置。

例如，将宿主机的 `/mnt/photos` 挂载为容器内的 `/library` 后，可以创建：

```text
家庭照片  → /library/family
工作素材  → /library/work
手机备份  → /library/mobile
```

媒体库可以选择 `/library` 本身或其任一安全后代，默认包含所选目录下的全部子目录，并保留原始层级。选择 `/library` 本身会与所有其他媒体库重叠，因此该实例不能再创建第二个库。FolioPath 将库根保存为相对于 `/library` 的路径，将目录和媒体保存为相对于具体库根的路径；删除媒体库只会移除索引、设置和派生缓存，不会修改原始文件。

为避免重复索引，首个版本默认不允许媒体库根目录互相包含。如果挂载暂时不可用，媒体库会被标记为离线，而不会被视为空目录并清除索引。

媒体库名称必须在实例内唯一且可以改名；首版不允许原地修改根路径，更换路径通过从 FolioPath 移除后重新创建完成。创建后立即扫描，应用启动时校准，并默认每 24 小时完整扫描；周期可以在设置中修改或关闭。

## Recursive View

递归模式是 FolioPath 的核心功能。

假设目录结构如下：

```text
Photos/
├── 2024/
│   ├── Beijing/
│   └── Shanghai/
└── 2025/
    ├── Tokyo/
    └── Kyoto/
```

选择 `Photos/2025/Tokyo` 时，只显示 Tokyo 目录中的内容。

选择 `Photos` 并开启递归模式时，将连续显示所有子目录中的图片和视频，同时保留每个文件的来源路径，方便随时定位回原目录。

普通目录默认按文件名自然升序；递归视图和搜索结果默认按文件修改时间倒序。搜索默认作用于当前媒体库，并可切换当前目录（可递归）或全部媒体库。

## Docker

> [!NOTE]
> 以下配置是计划中的部署方式。首个可用镜像发布后，将补充准确的镜像地址、端口和环境变量。

```yaml
services:
  foliopath:
    image: ghcr.io/YOUR_GITHUB_USERNAME/foliopath:VERSION
    container_name: foliopath
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - /mnt/photos:/library:ro
      - ./foliopath-data:/app/data
    environment:
      - TZ=Asia/Shanghai
```

启动服务：

```bash
docker compose up -d
```

然后访问：

```text
http://localhost:8080
```

建议将一个足够大的共同目录以只读方式挂载到 `/library`：

```text
/mnt/photos:/library:ro
```

这样 FolioPath 可以在 Web 设置中选择 `/library` 下的子目录作为媒体库，但不能修改或删除原始内容。`/app/data` 用于保存设置、SQLite 索引和缩略图缓存。

`/library` 是唯一媒体挂载目标；它下面只能是普通目录，不能再嵌套 Docker volume、
bind mount 或其他挂载点。媒体位于多个独立宿主卷时，需要先由宿主机提供一个经过验证的
单一呈现根，再将该根一次性只读挂载到 `/library`。详细约束见
[ADR-0009](docs/adr/0009-linux-openat2-single-media-root.md)。

默认端口仅绑定本机回环地址，适合配合认证反向代理使用。在单管理员认证功能完成前，不得将开发预览版直接暴露到局域网或互联网；首个稳定版将提供内建管理员初始化、会话和退出登录。

## Design Principles

1. **Folder first** — 文件夹层级是主要导航方式，不是附加功能。
2. **Non-destructive** — 默认不移动、不重命名、不修改原始媒体文件。
3. **Fast browsing** — 面向大型目录优化扫描、缩略图和滚动性能。
4. **Simple deployment** — 尽量通过一个 Docker Compose 配置完成部署。
5. **Progressive features** — 核心浏览功能保持轻量，高级功能按需启用。

## Technical Architecture

FolioPath 采用单体、单进程、单端口架构：

- **Backend:** Go 与标准库 `net/http`
- **Database:** SQLite；媒体索引是可重建的派生状态
- **Image processing:** libvips，通过 govips 生成缩略图
- **Video processing:** ffprobe 提取信息，FFmpeg 生成封面；首版不转码
- **Frontend:** React、TypeScript、Vite、TanStack Query 与 TanStack Virtual
- **API:** 同源 REST API、游标分页和媒体 ID
- **Authentication:** 首次设置的单管理员账号与安全 Cookie 会话
- **Deployment:** 前端产物嵌入 Go 服务，以单个 Docker 容器发布

完整入口参阅[项目文档索引](docs/README.md)。架构、数据模型、安全、API、界面、部署、测试和路线图都从该页进入；重要决策记录在 [`docs/adr`](docs/adr) 中。

项目使用[系统架构档案](docs/architecture/README.md)约束模块所有权、依赖、数据、任务、前端设计系统和
质量门禁。当前 MVP 范围由[版本 scope manifest](docs/releases/MVP-2026-07-23-scope.md)冻结；新能力默认进入后续版本。每个纵向切片按“需求/架构 → OpenAPI 与
数据契约 → 后端及集成证据 → 前端消费 → 发布验证”推进，不允许由页面临时实现反向定义系统行为。

## Development Status

仓库已经有固定的 Go toolchain、SQLite 初始迁移、路径边界与 generation 扫描实验代码、
权威 [`api/openapi.yaml`](api/openapi.yaml)，以及 Go 单元、契约、集成和显式容量测试；这批
代码仍是运行骨架、可行性和契约证据，不是可供用户部署的完整 FolioPath。正式
`cmd/foliopath` 入口、HTTP health/status、SQLite migration 和优雅停机已经建立，并由真实
composition root 与测试专用容器覆盖；当前还没有 React 产品前端、正式发布 Dockerfile、
认证实现或生产媒体处理链路。仓库已有隔离的 FS-05 probe/应用 smoke Dockerfile、契约生成
和 CI，原生双架构 runtime/recovery 与 SBOM jobs 已通过；这仍不是可发布应用证据。

- [FS-01 路径边界](docs/spikes/fs-01-path-boundary.md)：**Passed（Stage 0 范围）**。Darwin 与原生
  Linux amd64/arm64 路径矩阵、Linux `openat2` 的同设备/跨设备/self-bind mount 拒绝，以及真实
  HTTP test harness 已通过；生产 handler/auth 转入首个受保护 API Backend Gate，只读发布
  volume 与运行期 unmount 转入 FS-05/Release Gate。
- [FS-02 SQLite 与扫描 generation](docs/spikes/fs-02-sqlite-generation.md)：**当前正确性范围通过**。真实文件 SQLite、Goose、WAL、故障/取消/离线/重启保留、原子 finalize 与跨媒体库隔离已有自动化证据；磁盘满、真实强杀、长期 WAL 压力及备份恢复仍未验证。
- [FS-03 媒体矩阵](docs/spikes/fs-03-media-matrix.md)：**Stage 0 范围通过，完整范围
  Conditional**。govips/FFmpeg fixture 已在原生双架构通过；生产任务隔离、更多敌意输入、
  浏览器和最终镜像仍由后续 Gate 验证。
- [FS-04 容量基线](docs/spikes/fs-04-capacity-baseline.md)：**扫描/索引子范围通过，整体
  Conditional**。Linux/arm64、四核/4 GiB 下完成 10 万媒体/1 万目录档并修复 finalize
  复杂度问题；代表性存储、FTS、媒体/缩略图、HTTP 和前端并发仍未验证。
- [FS-05 运行与恢复](docs/spikes/fs-05-runtime-recovery.md)：**Stage 0 范围通过**。原生
  双架构同 Dockerfile 已验证非 root/只读、health、退出、离线恢复、重复迁移与故障关闭。

项目已获准进入[后端优先的 Stage 1](docs/gates/MVP-2026-07-23/stage-0-current.md)，但上述
spike 结果不能解释为应用功能已可用或发布门槛已满足。

## Roadmap

详细阶段、依赖和出口条件见[项目路线图](docs/roadmap.md)；编码前的条件 Go 结论与必做验证见[可行性研究](docs/feasibility-study.md)。

- [x] 确定技术栈和基础架构
- [x] 确认 MVP 需求基线（RQ-001～RQ-014 全部采用 A）
- [x] FS-02 SQLite/generation 当前正确性验证
- [x] 第一版权威 OpenAPI 契约与离线契约检查
- [x] FS-01 原生 Linux amd64/arm64 `openat2` mount 边界与 Stage 0 HTTP harness 范围
- [x] FS-04 目标档的扫描/索引子范围
- [x] FS-05 原生双架构运行、恢复和失败关闭范围
- [x] Stage 0 SBOM/license 与风险复审，Gate 允许进入 Stage 1
- [x] Stage 1 后端运行骨架、SQLite migration、health、取消与真实应用容器 smoke
- [ ] 在对应 Backend/Release Gate 完成生产 handler/auth、只读发布 volume、运行期 unmount
  与长期 churn；不反向阻断 FS-01 Stage 0 可行性结论
- [ ] 在对应 Gate 完成 FS-03 生产媒体任务、更多敌意输入、浏览器与最终镜像矩阵
- [ ] 完成 FS-04 代表性存储与完整媒体/搜索/HTTP/前端容量验证
- [ ] 单管理员认证契约与后端
- [ ] 安全媒体根目录与多媒体库管理
- [ ] 目录扫描与媒体索引
- [ ] 缩略图生成和缓存
- [ ] 文件夹树与面包屑导航
- [ ] 网格与瀑布流布局
- [ ] 递归目录浏览
- [ ] 图片查看器与视频播放器
- [ ] Docker 镜像与 Compose 示例
- [ ] 搜索、排序和过滤
- [ ] 发布安全加固、认证和多架构镜像
- [ ] 文件系统实时监控
- [ ] 分享功能

## Inspiration

FolioPath 的产品方向受到以下开源项目启发：

- [Immich](https://github.com/immich-app/immich) — 高性能、自托管的照片与视频管理平台
- [FlowVision](https://github.com/netdcy/FlowVision) — 具有瀑布流、文件夹导航和递归浏览体验的 macOS 图片查看器

FolioPath 是独立项目，与 Immich 或 FlowVision 没有从属或官方关联。项目将专注于以文件夹为核心的 Web 浏览体验，而不是成为完整的照片备份平台或桌面文件管理器。

## Contributing

项目仍处于早期阶段，欢迎通过 Issue 参与功能讨论、交互设计、技术选型和测试。

在提交代码前，请先创建 Issue 描述问题或提案，以便保持实现方向一致。

## License

FolioPath 采用 [GNU Affero General Public License v3.0 or later](https://www.gnu.org/licenses/agpl-3.0.html)（`AGPL-3.0-or-later`）许可证。

你可以使用、研究、修改和分发 FolioPath。分发原始版本或修改版本时，需要依照许可证提供对应源代码；如果修改后的程序通过网络向用户提供服务，也需要向这些用户提供相应版本的源代码。

完整条款请参阅仓库中的 [`LICENSE`](LICENSE) 文件。

FolioPath 是独立实现的项目。它的产品方向受到 Immich 和 FlowVision 启发，但这并不表示 FolioPath 隶属于上述项目。贡献代码时，请勿提交来源不明、许可证不兼容，或直接复制自其他项目的代码。

---

**FolioPath** — Your folders, beautifully browsed.
