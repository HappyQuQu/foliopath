# FolioPath

> A folder-first, self-hosted photo and video browser.

FolioPath 是一个以真实文件夹结构为核心的自托管图片与视频浏览器。

只需将一个媒体根目录挂载到 Docker 容器中，再通过 Web 设置创建一个或多个媒体库。FolioPath 会扫描所选目录并生成缩略图，让你通过浏览器按照原始文件夹层级查看内容。它提供清晰的目录树、瀑布流布局，以及可以一次浏览当前目录及所有子目录内容的递归模式。

你的文件夹，就是你的相册。

> [!IMPORTANT]
> FolioPath 的 Stage 0～4 核心产品切片已完成，Stage 5 发布加固正在进行。
> 当前根 `Dockerfile` 已能把真实 React SPA、Go API、SQLite、libvips 和 FFmpeg 构建进
> 同一非 root 候选镜像，并通过 linux/arm64 本地只读媒体/只读根/health/SIGTERM、
> MVP 媒体和安全 Compose smoke。可信代理与非回环应用边界已通过 `S5-003`；
> 本机 100k 媒体/10k 目录候选容量档以及 Chromium、Firefox、WebKit 自动化候选矩阵
> 已通过；原生 arm64 与 amd64 也已用两个不同的不可变候选 image ID 通过向前升级和
> 配对数据回滚。最终不可变 digest、代表性物理设备、
> 供应链漏洞处置和 Release Candidate Gate 尚未完成。
> 不要把当前 candidate
> 当作稳定发布版部署。

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

### MVP candidate capabilities

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

这些能力已经进入 Stage 1～4 的 Integrated Done 候选，不等于已经发布稳定镜像。
Stage 5 的平台、恢复、容量、浏览器、供应链与文档 Gate 仍决定能否发布。

### Possible future features

- EXIF 和媒体信息面板
- 收藏、评分和浏览历史
- SVG、HEIC/HEIF、AVIF、RAW 及更多格式的缩略图
- 文件系统变更监控
- 分享链接和细粒度访问控制
- 重复文件检测
- 地图与时间线视图

## Supported Media

MVP 的服务端索引与派生格式契约如下：

| 类型 | 扩展名 | 候选行为 |
| --- | --- | --- |
| 图片 | `.jpg`、`.jpeg`、`.png`、`.webp` | 索引、WebP 缩略图和原内容查看 |
| 动图 | `.gif` | 索引、静态缩略图和浏览器原内容动画 |
| 视频容器 | `.mp4`、`.mov`、`.mkv` | 索引、FFmpeg poster；兼容编码从原文件通过 HTTP Range 直接播放 |

扩展名匹配不区分大小写。视频不会转码；容器受支持不表示当前浏览器一定能解码其中的
video/audio codec，不兼容时会显示明确降级状态。SVG、HEIC/HEIF、AVIF 和 RAW 不属于
MVP 索引/缩略图契约。

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

> [!WARNING]
> 仓库中的 Compose 是 Stage 5 候选配置；尚无可作为稳定版部署的公开镜像。

目前没有可直接拉取的稳定镜像。只进行候选评估时，先从当前检出的源码构建一个明确本地
标签，再复制 [`.env.example`](.env.example) 为 `.env`，把 `FOLIOPATH_IMAGE` 改为该
标签，并填写媒体目录、数据目录和实际 TLS 代理的精确 CIDR：

```bash
docker build --build-arg VERSION=stage5-local -t foliopath:stage5-local .
docker compose up -d
```

本地 tag 不是不可变发布引用，只能用于当前源码候选验证。正式部署必须使用发布说明指定的
版本或 digest；当前还没有这样的稳定引用。

通过你配置的单跳 TLS 反向代理访问其 HTTPS 地址。应用端口仅发布到宿主机回环地址，
且没有合法代理声明的直连请求会失败关闭；不要把 `http://localhost:8080` 当作用户入口。

建议将一个足够大的共同目录以只读方式挂载到 `/library`：

```text
/mnt/photos:/library:ro
```

这样 FolioPath 可以在 Web 设置中选择 `/library` 下的子目录作为媒体库，但不能修改或删除原始内容。`/app/data` 用于保存设置、SQLite 索引和缩略图缓存。

`/library` 是唯一媒体挂载目标；它下面只能是普通目录，不能再嵌套 Docker volume、
bind mount 或其他挂载点。媒体位于多个独立宿主卷时，需要先由宿主机提供一个经过验证的
单一呈现根，再将该根一次性只读挂载到 `/library`。详细约束见
[ADR-0009](docs/adr/0009-linux-openat2-single-media-root.md)。

默认端口仅绑定本机回环地址并要求可信 TLS 反向代理。内建单管理员初始化、会话和退出登录
已经实现，但 Stage 5 发布 Gate 尚未通过，不得将当前候选版当作稳定服务暴露到局域网或互联网。

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

Stage 0～4 已通过各自 Gate。当前源码包含完整的 React SPA、单管理员认证、媒体库与扫描、
游标浏览与搜索、缩略图/缓存、图片查看器、原视频 Range 交付，以及真实 Go composition
root。根 `Dockerfile` 与 `compose.yaml` 是 Stage 5 候选，不是稳定发行物。

Stage 5 当前证据包括：安全候选容器与 Compose、可信代理边界、离线备份/恢复及失败关闭、
本机 linux/arm64 与指定原生 linux/amd64 的 100k/10k 产品容量档、100k 全量媒体与
cache 水位，以及 Chromium/Firefox/WebKit 候选自动化。仍阻断发布的项目包括最终不可变
digest、真实 Firefox、读屏/缩放/触摸和移动物理设备签署，以及候选供应链扫描中的
1 Critical / 8 High。详见[任务清单](docs/task-list.md)与
[Stage 5 Gate 记录](docs/gates/README.md)。当前统一
[Release Candidate 判断](docs/gates/MVP-2026-07-23/s5-release-candidate-current.md)
为 No-Go。

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
- [x] Stage 1 后端运行骨架、SQLite migration、health、单管理员认证与前端
- [x] Stage 2 安全媒体库、可靠扫描及产品 UI
- [x] Stage 3 目录/递归浏览、缩略图、网格/瀑布流及预览
- [x] Stage 4 搜索、过滤、图片查看器、原视频 Range 与降级状态
- [x] Stage 5 候选 Dockerfile、Compose、可信代理、恢复/失败关闭、容量和浏览器自动化基础
- [ ] 完成多架构镜像、真实升级、代表性设备、供应链和 RC 发布加固
- [ ] 关闭供应链、真实升级、代表性设备和 Release Candidate 阻断项
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
