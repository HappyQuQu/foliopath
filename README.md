# FolioPath

> A folder-first, self-hosted photo and video browser.

FolioPath 是一个以真实文件夹结构为核心的自托管图片与视频浏览器。

只需将一个媒体根目录挂载到 Docker 容器中，再通过 Web 设置创建一个或多个媒体库。FolioPath 会扫描所选目录并生成缩略图，让你通过浏览器按照原始文件夹层级查看内容。它提供清晰的目录树、瀑布流布局，以及可以一次浏览当前目录及所有子目录内容的递归模式。

你的文件夹，就是你的相册。

> [!IMPORTANT]
> FolioPath 目前处于规划与早期开发阶段。本文档中的功能和配置可能会调整。

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

- 🌲 清晰、可折叠的文件夹树
- 📚 在 Web 设置中创建和管理多个媒体库
- 🖼️ 自适应网格与瀑布流布局
- 🔭 递归浏览当前目录及所有子目录
- 📍 面包屑导航与媒体库内相对路径显示
- ⚡ 缩略图缓存、虚拟滚动和增量扫描
- 🔎 按文件名、类型、日期和路径搜索
- 🎞️ 支持常见图片格式、GIF 和视频
- 🌙 明暗主题
- 📱 响应式 Web 界面
- 🐳 Docker 与 Docker Compose 部署
- 🔒 默认支持只读媒体目录

### Possible future features

- EXIF 和媒体信息面板
- 收藏、评分和浏览历史
- RAW、HEIC 及更多格式的缩略图
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

媒体库默认包含所选目录下的全部子目录，并保留原始目录层级。FolioPath 只保存媒体库内的相对路径；删除媒体库只会移除索引、设置和派生缓存，不会修改原始文件。

为避免重复索引，首个版本默认不允许媒体库根目录互相包含。如果挂载暂时不可用，媒体库会被标记为离线，而不会被视为空目录并清除索引。

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

## Docker

> [!NOTE]
> 以下配置是计划中的部署方式。首个可用镜像发布后，将补充准确的镜像地址、端口和环境变量。

```yaml
services:
  foliopath:
    image: ghcr.io/YOUR_GITHUB_USERNAME/foliopath:latest
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

默认端口仅绑定本机回环地址，适合配合认证反向代理使用。在身份认证功能完成前，不建议将服务直接暴露到互联网。

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
- **Deployment:** 前端产物嵌入 Go 服务，以单个 Docker 容器发布

完整入口参阅[项目文档索引](docs/README.md)。架构、数据模型、安全、API、界面、部署、测试和路线图都从该页进入；重要决策记录在 [`docs/adr`](docs/adr) 中。

## Roadmap

详细阶段、依赖和出口条件见[项目路线图](docs/roadmap.md)；编码前的条件 Go 结论与必做验证见[可行性研究](docs/feasibility-study.md)。

- [x] 确定技术栈和基础架构
- [ ] 项目脚手架、数据库迁移和基础 API
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
